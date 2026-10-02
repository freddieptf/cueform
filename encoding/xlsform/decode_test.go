package xlsform

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"github.com/xuri/excelize/v2"
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

// extra choices sheet columns, which a choice_filter tests, decode as filterCategory; media makes
// the choice {label: ..., image: ...}
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
			nairobi: {
				label: en: "Nairobi"
				image: "nairobi.png"
			}
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
			// a select from earlier answers names a ${question} rather than a choice list
			colHeaders: []string{"type", "name", "label"},
			row:        []string{"select_one ${tree_name}", "test", "Best tree"},
			want: `{
	type:         "select_one"
	choices_from: "${tree_name}"
	name:         "test"
	label:        "Best tree"
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

func TestValidXLSFormSheetColumns(t *testing.T) {
	for _, tc := range []struct {
		sheet   string
		headers []string
		missing string
	}{
		{surveySheetName, []string{"type", "name", "label"}, ""},
		// a translated form may have only label::lang columns
		{surveySheetName, []string{"type", "name", "label::en", "label::fr"}, ""},
		// a column that only starts with a required name isn't that column
		{surveySheetName, []string{"type", "name_foo", "label"}, "name"},
		{surveySheetName, []string{"types", "name", "label"}, "type"},
		{choiceSheetName, []string{"list_name", "name", "label_x"}, "label"},
	} {
		err := validXLSFormSheet(tc.sheet, [][]string{tc.headers})
		if tc.missing == "" {
			if err != nil {
				t.Errorf("%q: have %v, want no error", tc.headers, err)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidXLSFormSheet) || !strings.Contains(err.Error(), "has no "+tc.missing+" column") {
			t.Errorf("%q: have %v, want a missing %s column", tc.headers, err, tc.missing)
		}
	}
}

// sheets and columns pyxform reads but cueform doesn't are an error (entities) or a warning, so
// an import never loses them silently; sheets pyxform ignores are ignored here too
func TestUnsupportedSheets(t *testing.T) {
	workbook := func(extra string, choiceHeaders []string) *bytes.Buffer {
		f := excelize.NewFile()
		f.SetSheetName("Sheet1", "survey")
		f.SetSheetRow("survey", "A1", &[]string{"type", "name", "label"})
		f.SetSheetRow("survey", "A2", &[]string{"note", "intro", "Hello"})
		if choiceHeaders != nil {
			f.NewSheet("choices")
			f.SetSheetRow("choices", "A1", &choiceHeaders)
			f.SetSheetRow("choices", "A2", &[]string{"fruit", "apple", "Apple", "apple.png"})
		}
		if extra != "" {
			f.NewSheet(extra)
			f.SetSheetRow(extra, "A1", &[]string{"list_name", "label"})
		}
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		return buf
	}
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	if _, err := parseXLSForm(workbook("Entities", nil)); !errors.Is(err, ErrUnsupportedSheet) {
		t.Errorf("entities: have %v, want %v", err, ErrUnsupportedSheet)
	}
	for _, tc := range []struct {
		extra         string
		choiceHeaders []string
		warning       string
	}{
		{"external_choices", nil, "the external_choices sheet is dropped"},
		{"", []string{"list_name", "name", "label", "media::image"}, `choices column "media::image" is dropped`},
		{"", []string{"list_name", "name", "label", "image"}, ""},
		{"notes", nil, ""},
	} {
		logs.Reset()
		if _, err := parseXLSForm(workbook(tc.extra, tc.choiceHeaders)); err != nil {
			t.Fatal(err)
		}
		if tc.warning == "" && strings.Contains(logs.String(), "warning") {
			t.Errorf("%s: unexpected warning %q", tc.extra, logs.String())
		}
		if tc.warning != "" && !strings.Contains(logs.String(), tc.warning) {
			t.Errorf("no warning %q in %q", tc.warning, logs.String())
		}
	}
}

// malformed rows used to panic; they now decode or fail naming the sheet row
func TestDecodeMalformedRows(t *testing.T) {
	decode := func(survey [][]string, settings [][]string) ([]byte, error) {
		f := excelize.NewFile()
		f.SetSheetName("Sheet1", "survey")
		for i, row := range survey {
			f.SetSheetRow("survey", fmt.Sprintf("A%d", i+1), &row)
		}
		if settings != nil {
			f.NewSheet("settings")
			for i, row := range settings {
				f.SetSheetRow("settings", fmt.Sprintf("A%d", i+1), &row)
			}
		}
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		return NewDecoder("x").Decode(buf)
	}

	// a settings row shorter than its header
	src, err := decode([][]string{{"type", "name", "label"}, {"note", "n", "N"}}, [][]string{{"form_title", "form_id", "version"}, {"T"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `form_title: "T"`) || strings.Contains(string(src), "form_id") {
		t.Errorf("short settings row: have\n%s", src)
	}

	for _, tc := range []struct {
		name   string
		survey [][]string
		want   string
	}{
		// the row ends before its type cell
		{"no type", [][]string{{"name", "label", "type"}, {"n", "N"}}, "survey row 2 has no type"},
		{"no name", [][]string{{"type", "name", "label"}, {"note", "a", "A"}, {"text", "", "No name"}}, "survey row 3 (text) has no name"},
		{"no name in a group", [][]string{{"type", "name", "label"}, {"begin_group", "g", "G"}, {"text", "", "Q"}, {"end_group"}}, "survey row 3 (text) has no name"},
	} {
		_, err := decode(tc.survey, nil)
		if !errors.Is(err, ErrInvalidXLSForm) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: have %v, want %q", tc.name, err, tc.want)
		}
	}
}

// only begin/end group and repeat rows open and close groups; an "end" question records when the
// form was finished
func TestDecodeGroupRows(t *testing.T) {
	decode := func(rows [][]string) (string, error) {
		f := excelize.NewFile()
		f.SetSheetName("Sheet1", "survey")
		f.SetSheetRow("survey", "A1", &[]string{"type", "name", "label"})
		for i, row := range rows {
			f.SetSheetRow("survey", fmt.Sprintf("A%d", i+2), &row)
		}
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		src, err := NewDecoder("x").Decode(buf)
		return string(src), err
	}

	src, err := decode([][]string{{"begin_group", "g", "G"}, {"end", "finished"}, {"text", "after", "After"}, {"end_group"}})
	if err != nil {
		t.Fatal(err)
	}
	want := `children: [
			x.#Question & {
				type: "end"
				name: "finished"
			},
			x.#Question & {
				type:  "text"
				name:  "after"
				label: "After"
			},
		]`
	if !strings.Contains(src, want) {
		t.Errorf("the end question and the row after it should stay in the group:\n%s", src)
	}

	for _, tc := range []struct {
		name string
		rows [][]string
		want string
	}{
		{"stray end", [][]string{{"text", "a", "A"}, {"end_group"}, {"text", "b", "B"}}, `survey row 3: "end_group" has no matching begin_group`},
		{"unclosed", [][]string{{"begin_group", "g", "G"}, {"text", "a", "A"}}, `survey row 2 begins group "g", which has no end_group row`},
		{"mismatched", [][]string{{"begin group", "g", "G"}, {"text", "a", "A"}, {"end repeat"}}, `survey row 4: "end repeat" has no matching begin_repeat`},
	} {
		if _, err := decode(tc.rows); !errors.Is(err, ErrInvalidXLSForm) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: have %v, want %q", tc.name, err, tc.want)
		}
	}

	// pyxform's older group kinds and select_one_external aren't supported
	for _, typ := range []string{"begin loop", "begin_lgroup", "begin looped group", "end loop", "select_one_external cities"} {
		if _, err := decode([][]string{{typ, "g", "G"}}); !errors.Is(err, ErrUnsupportedType) || !strings.Contains(err.Error(), "survey row 2") {
			t.Errorf("%q: have %v, want %v", typ, err, ErrUnsupportedType)
		}
	}
}

// pyxform reads these settings as yes/no flags; auto_send is copied as written, so stays a string
func TestDecodeSettingFlags(t *testing.T) {
	form := &xlsForm{
		settingColumnHeaders: []string{"form_title", "allow_choice_duplicates", "omit_instanceID", "clean_text_values", "auto_send", "style"},
		settings:             [][]string{{"T", "yes", "TRUE", "false()", "true", "no"}},
	}
	b, err := format.Node(form.settingsToAst(astutil.ImportInfo{Ident: "x"}).Value, format.Simplify())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"allow_choice_duplicates: true", "omit_instanceID:         true", "clean_text_values:       false", `auto_send:               "true"`, `style:                   "no"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("want %s in\n%s", want, b)
		}
	}
}

// lists keep the order they first appear in; a row with no list_name is skipped, as pyxform does
func TestExtractChoices(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	columns := []string{"name", "label", "list_name"}
	rows := [][]string{
		{"yes", "Yes", "yes_no"},
		{"apple", "Apple", "fruit"},
		{"mango", "Mango"},
		{"no", "No", "yes_no"},
		{},
		{"zebra", "Zebra", "animals"},
	}
	var have []string
	for _, list := range extractChoices(columns, rows) {
		for _, row := range list.rows {
			have = append(have, list.name+"/"+row[0])
		}
	}
	if want := []string{"yes_no/yes", "yes_no/no", "fruit/apple", "animals/zebra"}; !reflect.DeepEqual(have, want) {
		t.Errorf("have %q, want %q", have, want)
	}
	if !strings.Contains(logs.String(), "choices row 4 is dropped: it has no list_name") {
		t.Errorf("no warning for the row without a list_name in %q", logs.String())
	}
}

// pyxform uses the first settings row; the rest are ignored with a warning
func TestDecodeSettingsRows(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	form := &xlsForm{
		settingColumnHeaders: []string{"form_title", "form_id"},
		settings:             [][]string{{}, {"First", "first_id"}, {"Second", "second_id"}},
	}
	b, err := format.Node(form.settingsToAst(astutil.ImportInfo{Ident: "x"}).Value, format.Simplify())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `form_title: "First"`) || strings.Contains(string(b), "Second") {
		t.Errorf("want only the first non-empty row in\n%s", b)
	}
	if !strings.Contains(logs.String(), "settings row 4 is ignored") {
		t.Errorf("no warning for the second row in %q", logs.String())
	}
	if (&xlsForm{settingColumnHeaders: []string{"form_title"}}).settingsToAst(astutil.ImportInfo{Ident: "x"}) != nil {
		t.Error("a settings sheet with no data rows should give no form_settings")
	}
}

// names must be unique within their group, repeat or survey; top-level names are CUE fields
func TestDecodeDuplicateNames(t *testing.T) {
	decode := func(survey [][]string, withSettings bool) error {
		f := excelize.NewFile()
		f.SetSheetName("Sheet1", "survey")
		f.SetSheetRow("survey", "A1", &[]string{"type", "name", "label"})
		for i, row := range survey {
			f.SetSheetRow("survey", fmt.Sprintf("A%d", i+2), &row)
		}
		if withSettings {
			f.NewSheet("settings")
			f.SetSheetRow("settings", "A1", &[]string{"form_title"})
			f.SetSheetRow("settings", "A2", &[]string{"T"})
		}
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewDecoder("x").Decode(buf)
		return err
	}

	// the same name in different groups is fine
	if err := decode([][]string{{"text", "age", "Age"}, {"begin_group", "father", "Father"}, {"integer", "age", "Age"}, {"end_group"}}, false); err != nil {
		t.Errorf("same name in different groups: %v", err)
	}
	for _, tc := range []struct {
		name         string
		survey       [][]string
		withSettings bool
		want         string
	}{
		{"top level", [][]string{{"text", "age", "Age"}, {"note", "n", "N"}, {"integer", "age", "Age again"}}, false, `survey row 4: name "age" is already used in row 2`},
		{"in a group", [][]string{{"begin_group", "g", "G"}, {"text", "a", "A"}, {"text", "a", "A"}, {"end_group"}}, false, `survey row 4: name "a" is already used in row 3`},
		{"form_settings", [][]string{{"text", "form_settings", "Q"}}, true, `the name "form_settings" is taken by the settings sheet`},
	} {
		if err := decode(tc.survey, tc.withSettings); !errors.Is(err, ErrInvalidXLSForm) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: have %v, want %q", tc.name, err, tc.want)
		}
	}
	// without a settings sheet, form_settings is an ordinary name
	if err := decode([][]string{{"text", "form_settings", "Q"}}, false); err != nil {
		t.Errorf("form_settings without settings: %v", err)
	}
}
