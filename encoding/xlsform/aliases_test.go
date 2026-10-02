package xlsform

import (
	"errors"
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestCanonicalHeaders(t *testing.T) {
	headers := []string{"Type", "Name", "Label :: English (en)", "Relevance", "readonly", "Calculate", "Constraint Message::fr", "country", "Country Code"}
	want := []string{"type", "name", "label::English (en)", "relevant", "read_only", "calculation", "constraint_message::fr", "country", "Country Code"}
	have, err := canonicalHeaders(surveySheetName, headers)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Fatalf("have %q\nwant %q", have, want)
	}

	have, err = canonicalHeaders(choiceSheetName, []string{"List Name", "Value", "Caption"})
	if err != nil || !reflect.DeepEqual(have, []string{"list_name", "name", "label"}) {
		t.Fatalf("choices: have %q, %v", have, err)
	}
	have, err = canonicalHeaders(settingsSheetName, []string{"set_form_title", "Form ID", "omit_instanceID"})
	if err != nil || !reflect.DeepEqual(have, []string{"form_title", "form_id", "omit_instanceID"}) {
		t.Fatalf("settings: have %q, %v", have, err)
	}

	// pyxform rejects two headers for the same column
	if _, err := canonicalHeaders(surveySheetName, []string{"type", "relevant", "relevance"}); !errors.Is(err, ErrInvalidXLSFormSheet) {
		t.Fatalf("have %v, want %v", err, ErrInvalidXLSFormSheet)
	}
}

func TestCanonicalSelectType(t *testing.T) {
	for value, want := range map[string]string{
		"select one fruit":                 "select_one fruit",
		"Select All That Apply fruit":      "select_multiple fruit",
		"select one from file cities.csv":  "select_one_from_file cities.csv",
		"select multiple from file c.xml":  "select_multiple_from_file c.xml",
		"select1 fruit":                    "select_one fruit",
		"select_one fruit":                 "select_one fruit",
		"text":                             "text",
		"select one":                       "select one",
		"add select one prompt using list": "select_one list",
	} {
		if have := canonicalSelectType(value); have != want {
			t.Errorf("canonicalSelectType(%q) = %q, want %q", value, have, want)
		}
	}
}

// pyxform finds sheets case-insensitively and accepts the aliases above
func TestParseXLSFormAliases(t *testing.T) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Survey")
	f.SetSheetRow("Survey", "A1", &[]string{"Type", "Name", "Label", "Relevance"})
	f.SetSheetRow("Survey", "A2", &[]string{"select one fruit", "fav", "Favourite", "true()"})
	f.NewSheet("Choices")
	f.SetSheetRow("Choices", "A1", &[]string{"List Name", "Name", "Label"})
	f.SetSheetRow("Choices", "A2", &[]string{"fruit", "apple", "Apple"})
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	form, err := parseXLSForm(buf)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"type", "name", "label", "relevant"}; !reflect.DeepEqual(form.surveyColumnHeaders, want) {
		t.Errorf("survey headers: have %q, want %q", form.surveyColumnHeaders, want)
	}
	if have := form.survey[0][0]; have != "select_one fruit" {
		t.Errorf("type: have %q, want %q", have, "select_one fruit")
	}
	if want := []string{"list_name", "name", "label"}; !reflect.DeepEqual(form.choiceColumnHeaders, want) {
		t.Errorf("choice headers: have %q, want %q", form.choiceColumnHeaders, want)
	}
}
