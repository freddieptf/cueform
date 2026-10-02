package xlsform

import (
	"errors"
	"os"
	"reflect"
	"strings"
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

// extra choices sheet columns, which a choice_filter tests, decode as filterCategory
func TestBuildChoiceStructFilters(t *testing.T) {
	columns := []string{"list_name", "name", "label::en", "country", "image"}
	rows := [][]string{
		{"cities", "nairobi", "Nairobi", "ke", "nairobi.png"},
		{"cities", "other", "Other"},
	}
	choice, err := buildChoiceStruct("cities", columns, rows)
	if err != nil {
		t.Fatal(err)
	}
	b, err := format.Node(choice, format.Simplify())
	if err != nil {
		t.Fatal(err)
	}
	want := `{
	list_name: "cities"
	choices: [
		{
			nairobi: en: "Nairobi"
			filterCategory: country: "ke"
		},
		{
			other: en: "Other"
		},
	]
}`
	if have := string(b); have != want {
		t.Fatalf("have\n%s\nwant\n%s", have, want)
	}
}

// a plain label column is a single-language label; next to label::lang columns it is "default"
func TestBuildChoiceStructPlainLabels(t *testing.T) {
	for _, tc := range []struct {
		columns []string
		row     []string
		want    string
	}{
		{[]string{"list_name", "name", "label"}, []string{"yes_no", "yes", "Yes"}, `yes: "Yes"`},
		{[]string{"list_name", "name", "label", "label::fr"}, []string{"yes_no", "yes", "Yes", "Oui"}, "yes: {\n\t\t\t\tdefault: \"Yes\"\n\t\t\t\tfr:      \"Oui\"\n\t\t\t}"},
	} {
		choice, err := buildChoiceStruct("yes_no", tc.columns, [][]string{tc.row})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := format.Node(choice, format.Simplify())
		if !strings.Contains(string(b), tc.want) {
			t.Errorf("%v: have\n%s\nwant it to contain\n%s", tc.columns, b, tc.want)
		}
	}
}

// a select whose list isn't on the choices sheet used to decode to a nil value and panic
func TestBuildSurveyElementMissingList(t *testing.T) {
	_, err := buildSurveyElement(false, []string{"type", "name", "label"}, []string{"select_one cities", "q", "Q"}, map[string]ast.Expr{})
	if !errors.Is(err, ErrInvalidXLSForm) || !strings.Contains(err.Error(), `choice list "cities"`) {
		t.Fatalf("have %v, want %v naming the list", err, ErrInvalidXLSForm)
	}
}

func TestBuildSurveyElementOrOther(t *testing.T) {
	for _, typ := range []string{"select_one yes_no or_other", "select_multiple yes_no or specify other", "rank yes_no or other"} {
		_, err := buildSurveyElement(false, []string{"type", "name", "label::en"}, []string{typ, "q", "Q"}, map[string]ast.Expr{})
		if !errors.Is(err, ErrOrOther) {
			t.Errorf("%q: have %v, want %v", typ, err, ErrOrOther)
		}
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
			// a single-language form has plain columns
			colHeaders: []string{"type", "name", "label", "hint", "required_message"},
			row:        []string{"note", "test", "Name", "Full name", "Required"},
			want: `{
	type:             "note"
	name:             "test"
	label:            "Name"
	hint:             "Full name"
	required_message: "Required"
}`,
		},
		{
			// next to translations, a plain column is pyxform's "default" language
			colHeaders: []string{"type", "name", "label", "label::fr", "hint"},
			row:        []string{"note", "test", "Name", "Nom", "Full name"},
			want: `{
	type: "note"
	name: "test"
	label: {
		default: "Name"
		fr:      "Nom"
	}
	hint: "Full name"
}`,
		},
		{
			colHeaders: []string{"type", "name", "label:lang(en)"},
			row:        []string{"note", "test", "test"},
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
		{
			// only exact yes/no spellings become bools; expressions, true() and other columns stay strings
			colHeaders: []string{"type", "name", "label::lang (en)", "required", "read_only", "relevant", "default"},
			row:        []string{"text", "test", "test", "YES", "false", "yes", "no"},
			want: `{
	type: "text"
	name: "test"
	label: "lang (en)": "test"
	required:  true
	read_only: false
	relevant:  "yes"
	default:   "no"
}`,
		},
		{
			// guidance_hint and media columns decode as translations, or as one plain value
			colHeaders: []string{"type", "name", "label::en", "guidance_hint::en", "image", "big-image::en", "audio::fr", "hint_extra"},
			row:        []string{"image", "test", "test", "staff only", "a.png", "big.png", "a.mp3", "x"},
			want: `{
	type: "image"
	name: "test"
	label: en: "test"
	guidance_hint: en: "staff only"
	image: "a.png"
	"big-image": en: "big.png"
	audio: fr: "a.mp3"
	hint_extra: "x"
}`,
		},
		{
			// a _from_file select names an attached file rather than a choice list
			colHeaders: []string{"type", "name", "label", "parameters"},
			row:        []string{"select_one_from_file cities.csv", "test", "City", "value=id"},
			want: `{
	type:       "select_one_from_file"
	file:       "cities.csv"
	name:       "test"
	label:      "City"
	parameters: "value=id"
}`,
		},
		{
			// rank takes a choice list like the select types
			colHeaders: []string{"type", "name", "label::lang (en)"},
			row:        []string{"rank yes_no", "test", "test"},
			want: `{
	type:    "rank"
	choices: yes_no
	name:    "test"
	label: "lang (en)": "test"
}`,
		},
		{
			colHeaders: []string{"type", "name", "label::lang (en)", "required", "read_only"},
			row:        []string{"text", "test", "test", "${age} >= 18", "true()"},
			want: `{
	type: "text"
	name: "test"
	label: "lang (en)": "test"
	required:  "${age} >= 18"
	read_only: "true()"
}`,
		},
	}
	choiceMap := map[string]ast.Expr{"yes_no": ast.NewIdent("yes_no")}
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
						type: "select_one"
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
	decoder := NewDecoder("test")
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
