package xlsform

import (
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		file string
		err  error
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
			err: nil,
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
			err: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			encoder := NewEncoder()
			f, err := encoder.Encode(tc.file)
			if err != tc.err {
				t.Fatalf("have %s but wanted %s", err, tc.err)
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
