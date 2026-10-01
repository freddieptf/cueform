package xlsform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// pyxform reads workbooks with openpyxl in read-only mode, which reads only the cells inside
// each sheet's declared dimension
func TestEncodeSheetDimensions(t *testing.T) {
	buf, err := NewEncoder().Encode("testdata/form_select.cue")
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatal(err)
	}
	for sheet, want := range map[string]string{"survey": "A1:C5", "choices": "A1:C3", "settings": "A1:D2"} {
		if have, err := f.GetSheetDimension(sheet); err != nil || have != want {
			t.Errorf("%s dimension: have %q (%v), want %q", sheet, have, err, want)
		}
	}
}

func TestEncode(t *testing.T) {
	testCases := []struct {
		file string
		// a substring of the expected error, empty if no error is expected
		err  string
		form *xlsForm
	}{
		{
			file: "testdata/form.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey: [][]string{
					{"text", "family_name", "What's your family name?"},
					{"begin_group", "father", "Father"},
					{"integer", "age", "How old is your father?"},
					{"end_group"},
				},
				settingColumnHeaders: []string{"form_title", "form_id", "default_language", "version"},
				settings: [][]string{
					{"test", "test_id", "English (en)", "1"},
				},
			},
		}, {
			file: "testdata/form_select.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey: [][]string{
					{"text", "family_name", "What's your family name?"},
					{"begin_group", "father", "Father"},
					{"select_one ages", "age", "How old is your father?"},
					{"end_group"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices: [][]string{
					{"ages", "over_30", "Over 30"},
					{"ages", "over_40", "Over 40"},
				},
				settingColumnHeaders: []string{"form_title", "form_id", "default_language", "version"},
				settings: [][]string{
					{"test", "test_id", "English (en)", "1"},
				},
			},
		}, {
			file: "testdata/form_spaced_group.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)"},
				survey: [][]string{
					{"begin group", "household", "Household"},
					{"begin repeat", "members", "Members"},
					{"text", "member_name", "Name"},
					{"end repeat"},
					{"end group"},
				},
			},
		}, {
			file: "testdata/form_unlabelled.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)", "calculation"},
				survey: [][]string{
					{"note", "intro", "Welcome"},
					{"start", "started"},
					{"calculate", "total", "", "1 + 1"},
				},
			},
		}, {
			file: "testdata/form_strings.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)", "required", "repeat_count", "read_only", "default"},
				survey: [][]string{
					{"begin_repeat", "members", "Members", "", "3"},
					{"decimal", "height", "Height", "yes", "", "no", "1.5"},
					{"select_one counties", "county", "County"},
					{"end_repeat"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)"},
				choices: [][]string{
					{"counties", "nairobi", "Nairobi"},
					{"counties", "kampala", "Kampala"},
				},
				settingColumnHeaders: []string{"form_title", "form_id", "default_language", "version"},
				settings: [][]string{
					{"test", "test_id", "English (en)", "2"},
				},
			},
		}, {
			// imports resolve from the form's own cue.mod, not the working directory
			file: "../../sample/composition/person_registration.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::en", "required", "appearance"},
				survey: [][]string{
					{"begin group", "person_registration", "Person Registration", "", "field-list"},
					{"text", "first_name", "First Name", "yes"},
					{"text", "middle_name", "Middle Name"},
					{"text", "last_name", "Last Name"},
					{"integer", "age", "Age"},
					{"end group"},
				},
			},
		}, {
			file: "testdata/form_bool.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)", "required", "read_only"},
				survey: [][]string{
					{"text", "family_name", "What's your family name?", "yes", "no"},
				},
			},
		}, {
			// numbers are written as CUE's exact decimal
			file: "testdata/form_number.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::English (en)", "repeat_count", "default", "parameters"},
				survey: [][]string{
					{"begin_repeat", "members", "Members", "3"},
					{"range", "height", "Height", "", "1.50", "start=0 end=3 step=0.5"},
					{"integer", "count", "Count", "", "0"},
					{"end_repeat"},
				},
				settingColumnHeaders: []string{"form_title", "form_id", "default_language", "version"},
				settings: [][]string{
					{"test", "test_id", "English (en)", "2026100101"},
				},
			},
		}, {
			file: "testdata/form_list_value.cue",
			err:  "family_name.default: xlsform values must be strings, bools or numbers",
		}, {
			file: "testdata/form_invalid_type.cue",
			err:  `family_name: "txt" is not a valid question type`,
		}, {
			file: "testdata/form_invalid_child.cue",
			err:  "father.children[0] does not match the schema",
		}, {
			file: "testdata/form_missing_label.cue",
			err:  "family_name does not match the schema",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			encoder := NewEncoder()
			f, err := encoder.Encode(tc.file)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("have %v but wanted an error containing %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			form, err := parseXLSForm(f)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(form, tc.form) {
				t.Fatalf("have\n%+v\nbut want\n%+v", form, tc.form)
			}
		})
	}
}
