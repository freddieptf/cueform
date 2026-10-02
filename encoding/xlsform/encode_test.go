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
				// filterCategory becomes a column a choice_filter can test
				choiceColumnHeaders: []string{"list_name", "name", "label::English (en)", "country"},
				choices: [][]string{
					{"counties", "nairobi", "Nairobi", "ke"},
					{"counties", "kampala", "Kampala", "ug"},
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
			// a list shared by several questions is written once; rank takes a list like the selects
			file: "testdata/form_shared_choices.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::en"},
				survey: [][]string{
					{"select_one yes_no", "smokes", "Do you smoke?"},
					{"select_multiple yes_no", "drinks", "Which do you drink?"},
					{"rank yes_no", "ranked", "Rank these"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::en"},
				choices: [][]string{
					{"yes_no", "yes", "Yes"},
					{"yes_no", "no", "No"},
				},
			},
		}, {
			// a cascading select: choice_filter tests the country column filterCategory writes
			file: "testdata/form_cascading.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::en", "choice_filter"},
				survey: [][]string{
					{"select_one countries", "country", "Country"},
					{"select_one cities", "city", "City", "country=${country}"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::en", "country"},
				choices: [][]string{
					{"countries", "ke", "Kenya"},
					{"countries", "ug", "Uganda"},
					{"cities", "nairobi", "Nairobi", "ke"},
					{"cities", "mombasa", "Mombasa", "ke"},
					{"cities", "kampala", "Kampala", "ug"},
				},
			},
		}, {
			// guidance_hint and media columns take translations, or one plain value
			file: "testdata/form_media.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::en", "label::fr", "hint::en", "hint::fr", "guidance_hint::en", "guidance_hint::fr", "image", "big-image::en", "big-image::fr", "audio::en", "audio::fr"},
				survey: [][]string{
					{"image", "photo", "Take a photo", "Prenez une photo", "Hold steady", "Restez immobile", "Only for staff", "Pour le personnel", "example.png", "large_en.png", "large_fr.png", "prompt_en.mp3", "prompt_fr.mp3"},
				},
			},
		}, {
			// a single-language form uses plain columns, with no ::lang
			file: "testdata/form_single_language.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label", "hint"},
				survey: [][]string{
					{"select_one yes_no", "smokes", "Do you smoke?", "Include e-cigarettes"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label"},
				choices: [][]string{
					{"yes_no", "yes", "Yes"},
					{"yes_no", "no", "No"},
				},
			},
		}, {
			file: "testdata/form_select_no_choices.cue",
			err:  "smokes: select_one needs choices, or choices_from naming a question in a repeat",
		}, {
			// names must be unique within their group; the first "age" child is fine
			file: "testdata/form_duplicate_name.cue",
			err:  `father.children[1]: name "age" is already used at father.children[0]; names must be unique`,
		}, {
			file: "testdata/form_duplicate_repeat.cue",
			err:  `father.children[0]: repeat name "children" is already used at mother.children[0]`,
		}, {
			// every settings column is optional
			file: "testdata/form_settings_minimal.cue",
			form: &xlsForm{
				surveyColumnHeaders:  []string{"type", "name", "label"},
				survey:               [][]string{{"note", "intro", "Welcome"}},
				settingColumnHeaders: []string{"form_title"},
				settings:             [][]string{{"Minimal"}},
			},
		}, {
			file: "testdata/form_settings_invalid.cue",
			err:  "form_settings does not match the schema: #Settings.default_language: conflicting values",
		}, {
			file: "testdata/form_empty_choices.cue",
			err:  "smokes does not match the schema: #Question.choices.choices: incompatible list lengths (0 and 1)",
		}, {
			// _from_file selects name their attached file in the type column
			file: "testdata/form_from_file.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label", "choice_filter", "parameters"},
				survey: [][]string{
					{"select_one_from_file cities.csv", "city", "City"},
					{"select_multiple_from_file people.xml", "person", "People", "district=${city}", "value=id label=full_name"},
				},
			},
		}, {
			file: "testdata/form_from_file_no_file.cue",
			err:  "city does not match the schema: #Question.file: field is required but not present",
		}, {
			file: "testdata/form_from_file_with_choices.cue",
			err:  "city: select_one_from_file reads its choices from file, so it can't have choices",
		}, {
			file: "testdata/form_file_on_select.cue",
			err:  "city.file: file is only for select_one_from_file and select_multiple_from_file",
		}, {
			file: "testdata/form_from_file_bad_extension.cue",
			err:  "city does not match the schema: #Question.file: invalid value",
		}, {
			// choices_from lists the answers to a question in a repeat
			file: "testdata/form_choices_from.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label"},
				survey: [][]string{
					{"begin_repeat", "trees", "Trees"},
					{"text", "tree_name", "Tree name"},
					{"end_repeat"},
					{"select_one ${tree_name}", "best", "Best tree"},
					{"rank ${tree_name}", "ranked", "Rank the trees"},
				},
			},
		}, {
			file: "testdata/form_choices_from_missing.cue",
			err:  `best.choices_from: no question is named "tree"`,
		}, {
			file: "testdata/form_choices_from_no_repeat.cue",
			err:  `best.choices_from: "tree_name" isn't inside a repeat`,
		}, {
			file: "testdata/form_choices_from_multiple.cue",
			err:  "best.choices_from: pyxform doesn't support choices_from on select_multiple",
		}, {
			// a choice with media is {label: ..., image: ...}; each field is a plain or ::lang column
			file: "testdata/form_choice_media.cue",
			form: &xlsForm{
				surveyColumnHeaders: []string{"type", "name", "label::en", "label::fr"},
				survey: [][]string{
					{"select_one fruit", "fav", "Favourite", "Préféré"},
				},
				choiceColumnHeaders: []string{"list_name", "name", "label::en", "label::fr", "image", "audio::en", "audio::fr"},
				choices: [][]string{
					{"fruit", "apple", "Apple", "Pomme", "apple.png", "apple_en.mp3", "apple_fr.mp3"},
					{"fruit", "mango", "Mango", "Mangue"},
				},
			},
		}, {
			file: "testdata/form_or_other.cue",
			err:  "favourite.or_other: or_other is not supported",
		}, {
			file: "testdata/form_reserved_filter.cue",
			err:  `city.choices.choices[0].filterCategory: "image" is a choices sheet column, not a filter`,
		}, {
			file: "testdata/form_conflicting_choices.cue",
			err:  `drinks.choices: choice list "yes_no" differs from the one at smokes.choices`,
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
