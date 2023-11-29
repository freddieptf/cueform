package xlsform

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
)

func TestValidXLSFormSheet(t *testing.T) {
	testCases := []struct {
		name          string
		columnHeaders [][]string
		sheet         string
		want          error
	}{
		{
			name:          "fail if no survey columns",
			columnHeaders: [][]string{},
			sheet:         surveySheetName,
			want:          ErrInvalidXLSForm,
		},
		{
			name:          "fail if no choice columns",
			columnHeaders: [][]string{},
			sheet:         choiceSheetName,
			want:          ErrInvalidXLSForm,
		},
		{
			name:          "fail if invalid survey columns",
			columnHeaders: [][]string{{"hello"}, {"world"}},
			sheet:         surveySheetName,
			want:          ErrInvalidXLSFormSheet,
		},
		{
			name:          "fail if invalid choice columns",
			columnHeaders: [][]string{{"hello"}, {"world"}},
			sheet:         choiceSheetName,
			want:          ErrInvalidXLSFormSheet,
		},
		{
			name:          "pass if correct survey columns",
			columnHeaders: [][]string{requiredSurveySheetColumns},
			sheet:         surveySheetName,
			want:          nil,
		},
		{
			name:          "pass if correct choice columns",
			columnHeaders: [][]string{requiredChoiceSheetColumns},
			sheet:         choiceSheetName,
			want:          nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validXLSFormSheet(tc.sheet, tc.columnHeaders)
			if err != tc.want {
				t.Fatalf("have \"%s\", wanted \"%v\"", err, tc.want)
			}
		})
	}

}

func TestParseXLSForm(t *testing.T) {
	testCases := []struct {
		file string
		want *xlsForm
		err  error
	}{
		{
			file: "testdata/empty.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/empty_survey_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/empty_choices_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/empty_valid_survey_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/empty_valid_choices_sheet.xlsx",
			want: nil,
			err:  ErrInvalidXLSForm,
		},
		{
			file: "testdata/empty_valid.xlsx",
			want: &xlsForm{
				surveyColumnHeaders: requiredSurveySheetColumns,
				choiceColumnHeaders: requiredChoiceSheetColumns,
				settingColumnHeaders: []string{
					"form_title", "form_id", "version", "default_language",
				},
			},
			err: nil,
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

func TestExtractChoices(t *testing.T) {
	testCases := []struct {
		choices [][]string
		want    map[string][][]string
	}{
		{
			choices: [][]string{
				{"yes_no", "yes", "Yes"},
				{"yes_no", "no", "No"},
				{},
				{"one_two", "one", "1"},
				{"one_two", "two", "2"},
			},
			want: map[string][][]string{
				"yes_no": {
					{"yes_no", "yes", "Yes"},
					{"yes_no", "no", "No"},
				},
				"one_two": {
					{"one_two", "one", "1"},
					{"one_two", "two", "2"},
				},
			},
		},
	}
	for tIdx, tc := range testCases {
		t.Run(fmt.Sprintf("%d\n", tIdx), func(t *testing.T) {
			result := extractChoices(requiredChoiceSheetColumns, tc.choices)
			if !reflect.DeepEqual(result, tc.want) {
				t.Fatalf("have %v but want %v", result, tc.want)
			}
		})
	}
}

func TestBuildChoiceField(t *testing.T) {
	testCases := []struct {
		choiceName string
		columns    []string
		rows       [][]string
		want       string
		err        error
	}{
		{
			choiceName: "one_two",
			columns:    []string{"list_name", "name", "label::English (en)"},
			rows: [][]string{
				{"one_two", "one", "1"},
				{"one_two", "two", "2"},
			},
			want: `{
	list_name: "one_two"
	choices: [
		{
			one: {
				"English (en)": "1"
			}
		},
		{
			two: {
				"English (en)": "2"
			}
		},
	]
}`,
			err: nil,
		},
		{
			choiceName: "one_two",
			columns:    []string{"list_name", "name", "label"},
			rows: [][]string{
				{"one_two", "one", "1"},
				{"one_two", "two", "2"},
			},
			want: "",
			err:  ErrInvalidLabel,
		},
	}
	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			result, err := buildChoiceStruct(tc.choiceName, tc.columns, tc.rows)
			var structStr string
			if result != nil {
				x, _ := format.Node(result)
				structStr = strings.TrimSpace(string(x))
			}
			if structStr != tc.want && tc.err != err {
				t.Fatalf("have { result: %q, err: %s }\nwant { result: %q, err: %s }", structStr, err, tc.want, tc.err)
			}
		})
	}
}

func TestChoicesToAst(t *testing.T) {
	testCases := []struct {
		name string
		form *xlsForm
		want map[string]string
		err  error
	}{
		{
			name: "zero rows",
			form: &xlsForm{
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices:             [][]string{},
			},
			want: nil,
			err:  nil,
		},
		{
			name: "invalid choice label",
			form: &xlsForm{
				choiceColumnHeaders: []string{"list_name", "name", "label"},
				choices: [][]string{
					{"yes_no", "yes", "Yes"},
					{"yes_no", "no", "No"},
				},
			},
			want: nil,
			err:  ErrInvalidLabel,
		},
		{
			name: "choice expression map",
			form: &xlsForm{
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices: [][]string{
					{"yes_no", "yes", "Yes"},
					{"yes_no", "no", "No"},
				},
			},
			want: map[string]string{
				"yes_no": `test.#Choices & {
	list_name: "yes_no"
	choices: [
		{
			yes: {
				"English (en)": "Yes"
			}
		},
		{
			no: {
				"English (en)": "No"
			}
		},
	]
}`,
			},
		},
	}
	importInfo, _ := astutil.ParseImportSpec(ast.NewImport(nil, "test"))
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.form.choicesToAst(importInfo)
			if err != tc.err {
				t.Fatalf("have %s\nwant %s", err, tc.err)
			}
			if len(result) != len(tc.want) {
				t.Fatalf("result has length of %d, want %d", len(result), len(tc.want))
			}
			for k, v := range result {
				vS, _ := format.Node(v)
				have := strings.TrimSpace(string(vS))
				if want := tc.want[k]; have != want {
					t.Fatalf("\nhave %q\nwant %q", have, want)
				}
			}
		})
	}
}

func TestBuildSurveyElement(t *testing.T) {
	testCases := []struct {
		form *xlsForm
		want string
		err  error
	}{

		{
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey:              [][]string{{"note", "test_note", "This is a test note"}},
			},
			want: `{
	type: "note"
	name: "test_note"
	label: {
		"English (en)": "This is a test note"
	}
}`,
			err: nil,
		},
		{
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "", "", "label::English (en)"},
				survey:              [][]string{{"note", "test_note", "", "", "This is a test note"}},
			},
			want: `{
	type: "note"
	name: "test_note"
	label: {
		"English (en)": "This is a test note"
	}
}`,
			err: nil,
		},
		{
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey:              [][]string{{"select_one one_choice", "test", "Select one, now!"}},
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices: [][]string{
					{"one_choice", "one", "ONE"},
				},
			},
			want: `{
	type:    "select_one"
	choices: test.#Choices & {
		list_name: "one_choice"
		choices: [
			{
				one: {
					"English (en)": "ONE"
				}
			},
		]
	}
	name: "test"
	label: {
		"English (en)": "Select one, now!"
	}
}`,
			err: nil,
		},
		{
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label"},
				survey:              [][]string{{"select_one one_choice", "test", "Select one, now!"}},
			},
			want: ``,
			err:  ErrInvalidLabel,
		},
	}
	importInfo, _ := astutil.ParseImportSpec(ast.NewImport(nil, "test"))
	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("case %d", idx), func(t *testing.T) {
			choiceMap, _ := tc.form.choicesToAst(importInfo)
			el, err := buildSurveyElement(false, tc.form.surveyColumnHeaders, tc.form.survey[0], choiceMap)
			if !errors.Is(err, tc.err) {
				t.Fatalf("have %s, want %s", err, tc.err)
			}
			if tc.err == nil {
				elF, _ := format.Node(el)
				if have := string(elF); have != tc.want {
					t.Fatalf("%s\n\nhave %q\nwant %q", have, have, tc.want)
				}
			}
		})
	}
}

func TestSurveyToAst(t *testing.T) {
}

func TestSettingsToAst(t *testing.T) {
	testCases := []struct {
		form *xlsForm
		want string
	}{
		{
			form: &xlsForm{
				settingColumnHeaders: []string{"form_title", "form_id", "version", "default_language"},
				settings:             [][]string{{"form", "id", "version", "English"}},
			},
			want: `form_settings: test.#Settings & {
	type:             "settings"
	form_title:       "form"
	form_id:          "id"
	version:          "version"
	default_language: "English"
}`,
		},
	}
	importInfo, _ := astutil.ParseImportSpec(ast.NewImport(nil, "test"))
	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			settingsField := tc.form.settingsToAst(importInfo)
			sb, err := format.Node(settingsField, format.Simplify())
			if err != nil {
				t.Fatalf("ahah %s", err)
			}
			if have := string(sb); have != tc.want {
				t.Fatalf("%s\n\nhave %q\nwant %q", have, have, tc.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		file string
		want string
		err  error
	}{
		{
			file: "testdata/group.xlsx",
			want: `package main

import "test"

family_name:
	test.#Question & {
		type:            "text"
		name:            "family_name"
		"label:English": "What's your family name?"
	}
father:
	test.#Group & {
		type:            "begin_group"
		name:            "father"
		"label:English": "Father"
		children: [
			test.#Question & {
				type:            "phone number"
				name:            "phone_number"
				"label:English": "What's your father's phone number?"
			},
			test.#Question & {
				type:            "integer"
				name:            "age"
				"label:English": "How old is your father?"
			},
		]
	}
form_settings:
	test.#Settings & {
		type:             "settings"
		form_title:       "test"
		form_id:          "test_id"
		version:          "1"
		default_language: "English (en)"
	}
`,
			err: nil,
		},
		{
			file: "testdata/group_nested.xlsx",
			want: `package main

import "test"

family_name:
	test.#Question & {
		type:            "text"
		name:            "family_name"
		"label:English": "What's your family name?"
	}
father:
	test.#Group & {
		type:            "begin_group"
		name:            "father"
		"label:English": "Father"
		children: [
			test.#Question & {
				type:            "phone number"
				name:            "phone_number"
				"label:English": "What's your father's phone number?"
			},
			test.#Question & {
				type:            "integer"
				name:            "age"
				"label:English": "How old is your father?"
			},
			test.#Group & {
				type:            "begin_group"
				name:            "next_of_kin"
				"label:English": "Father’s Next of Kin"
				children: [
					test.#Question & {
						type:            "phone number"
						name:            "nok_phone_number"
						"label:English": "What's your father's next of kin phone number?"
					},
				]
			},
		]
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
