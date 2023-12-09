package xlsform

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
)

func TestParseXLSForm(t *testing.T) {
	testCases := []struct {
		file string
		want *xlsForm
		err  error
	}{
		{
			file: "testdata/empty.xlsx",
			want: &xlsForm{
				surveyColumnHeaders:  []string{"type", "name", "label::English (en)"},
				choiceColumnHeaders:  []string{"list_name", "name", "label::English (en)"},
				settingColumnHeaders: []string{"form_title", "form_id", "version", "default_language"},
			},
			err: nil,
		},
		{
			file: "testdata/invalid.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/validate_choices_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/validate_survey_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/valid.xlsx",
			want: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey: [][]string{
					{"select_one one_two", "fav_num", "Select one or two, now."},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices: [][]string{
					{"one_two", "one", "ONE"},
					{"one_two", "two", "TWO"},
				},
				settingColumnHeaders: []string{
					"form_title", "form_id", "version", "default_language",
				},
				settings: [][]string{
					{"Test Form", "test", "1", "English (en)"},
				},
			},
			err: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			contentReader, err := os.Open(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parseXLSForm(contentReader)
			if !errors.Is(err, tc.err) {
				t.Fatalf("have %s, want %s", err, tc.err)
			}
			if !reflect.DeepEqual(f, tc.want) {
				t.Fatalf("have %+v, want %+v", f, tc.want)
			}
		})
	}
}

func TestBuildSurveyElement(t *testing.T) {
	testCases := []struct {
		colHeaders []string
		row        []string
		want       string
		err        error
	}{
		{
			colHeaders: []string{"type", "name", "label"},
			row:        []string{"note", "test", "test"},
			err:        ErrInvalidLabel,
		},
		{
			colHeaders: []string{"type", "name", "label:lang(en)"},
			row:        []string{"note", "test", "test"},
			err:        ErrInvalidLabel,
		},
		{
			colHeaders: []string{"type", "name", "label::lang (en)", "required_message"},
			row:        []string{"note", "test", "test", "test"},
			err:        ErrInvalidLabel,
		},
		{
			colHeaders: []string{"type", "name", "label::lang (en)"},
			row:        []string{"note", "test", "test"},
			want: `{
	type: "note"
	name: "test"
	label: "lang (en)": "test"
}`,
			err: nil,
		},
	}
	choiceMap := make(map[string]ast.Expr)
	for _, tc := range testCases {
		result, err := buildSurveyElement(true, tc.colHeaders, tc.row, choiceMap)
		if err != nil && !errors.Is(tc.err, ErrInvalidLabel) {
			t.Fatalf("have %s but want %s", err, tc.err)
		}
		if err == nil {
			b, _ := format.Node(result, format.Simplify())
			str := string(b)
			if tc.want != str {
				t.Fatalf("have\n%q\nwant\n%q", str, tc.want)
			}
		}
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		file string
		want string
		err  error
	}{
		{
			file: "testdata/sample.xlsx",
			want: `package main

import "test"

family_name:
	test.#Question & {
		type: "text"
		name: "family_name"
		label: {
			"English (en)":  "What's your family name?"
			"Testlang (tl)": "Test Test"
		}
	}
father:
	test.#Group & {
		type: "begin_group"
		name: "father"
		label: "English (en)": "Father"
		children: [
			test.#Question & {
				type: "phone number"
				name: "phone_number"
				label: {
					"English (en)":  "What's your father's phone number?"
					"Testlang (tl)": "Test Test Test"
				}
			},
			test.#Question & {
				type: "integer"
				name: "age"
				label: "English (en)": "How old is your father?"
			},
			test.#Group & {
				type: "begin_group"
				name: "next_of_kin"
				label: "English (en)": "Father’s Next of Kin"
				children: [
					test.#Question & {
						type:    "select_one"
						choices: test.#Choices & {
							list_name: "yes_no"
							choices: [
								{
									yes: "English (en)": "Yes"
								},
								{
									no: "English (en)": "No"
								},
							]
						}
						name: "has_next_of_kin"
						label: "English (en)": "Does your Father have a Next of Kin?"
					},
				]
			},
		]
	}
form_settings:
	test.#Settings & {
		type:       "settings"
		form_title: "Sample"
		form_id:    "sample"
	}
`,
			err: nil,
		},
	}
	decoder, _ := NewDecoder("test")
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			content, err := os.Open(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			r, err := decoder.Decode(content)
			if err != tc.err {
				t.Fatalf("have %s, want %s", err, tc.err)
			}
			if have := string(r); tc.want != "" && have != tc.want {
				t.Fatalf("%s\n\nhave %q\nwant %q", have, have, tc.want)
			}
		})
	}
}
