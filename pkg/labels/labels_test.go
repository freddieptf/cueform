package labels

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue/load"
	"golang.org/x/tools/txtar"
)

func TestBuildFiles(t *testing.T) {
	testCases := []struct {
		file   string
		result string
	}{
		{
			file:   "testdata/form.cue",
			result: "testdata/form.txtar",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			data, err := txtar.ParseFile(tc.result)
			if err != nil {
				t.Fatal(err)
			}
			result, err := ExtractLabels(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(string(result.Form), string(data.Files[1].Data)) {
				t.Fatalf("have\n%q\nwant\n%q\n", string(result.Form), string(data.Files[1].Data))
			}
			if !reflect.DeepEqual(string(result.Labels), string(data.Files[0].Data)) {
				t.Fatalf("have\n%q\nwant\n%q\n", string(result.Labels), string(data.Files[0].Data))
			}
		})
	}
}

func TestExtractLabels(t *testing.T) {
	testCases := []struct {
		file   string
		result []elementLabel
		err    error
	}{
		{
			file: "testdata/form.cue",
			result: []elementLabel{
				{
					id: "family_name/label",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "What's your family name?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Wat is jou familienaam?"},
					},
				},
				{
					id: "father/label",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "Father"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Pa"},
					},
				},
				{
					id: "father/age/label",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "How old is your father?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Hoe oud is jou pa?"},
					},
				},
				{
					id: "father/home_or_away/label",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "Is he home?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Is hy tuis?"},
					},
				},
				{
					id: "yes_no/yes",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "Yes"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Ja"},
					},
				},
				{
					id: "yes_no/no",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "No"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Nee"},
					},
				},
			},
		},
	}
	sortElements := func(els []elementLabel) {
		sort.SliceStable(els, func(i, j int) bool {
			return strings.Compare(els[i].id, els[j].id) > 0
		})
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			instances := load.Instances([]string{tc.file}, &load.Config{})
			labels, err := getLabels(instances[0].Files[0], nil)
			if err != tc.err {
				t.Fatalf("have %s but want %s", err, tc.err)
			}
			sortElements(labels)
			sortElements(tc.result)
			if !reflect.DeepEqual(labels, tc.result) {
				t.Fatalf("have\n%+v\nwant\n%+v\n", labels, tc.result)
			}
		})
	}
}

// a choice with media is {label: ..., image: ...}; only its label is yanked
func TestExtractLabelsChoiceMedia(t *testing.T) {
	result, err := ExtractLabels("testdata/media/form.cue")
	if err != nil {
		t.Fatal(err)
	}
	if want := `{apple: {label: _labels."fruit/apple", image: "apple.png"}}`; !strings.Contains(string(result.Form), want) {
		t.Errorf("form: want %s in\n%s", want, result.Form)
	}
	if want := `"fruit/apple": {`; !strings.Contains(string(result.Labels), want) {
		t.Errorf("labels: want %s in\n%s", want, result.Labels)
	}
}

// values share an entry only when every translation matches, including entries in labels.cue
func TestExtractLabelsDedupe(t *testing.T) {
	result, err := ExtractLabels("testdata/dedupe/form.cue")
	if err != nil {
		t.Fatal(err)
	}
	form, labels := string(result.Form), string(result.Labels)
	for _, want := range []string{`name: "a", label: _labels."a/label"`, `name: "b", label: _labels."b/label"`, `name: "c", label: _labels."a/label"`, `name: "v", label: _labels."old/label"`} {
		if !strings.Contains(form, want) {
			t.Errorf("form: want %s in\n%s", want, form)
		}
	}
	if !strings.Contains(labels, `"Swahili (sw)": "Jina la mtoto"`) {
		t.Errorf("labels: b's own translation is missing\n%s", labels)
	}
	if n := strings.Count(labels, "Kijiji"); n != 1 {
		t.Errorf("labels: the existing entry appears %d times\n%s", n, labels)
	}
}

// ids are paths, so repeated names in different groups don't collide; a taken id gets a suffix
func TestExtractLabelsIDs(t *testing.T) {
	result, err := ExtractLabels("testdata/collision/form.cue")
	if err != nil {
		t.Fatal(err)
	}
	form := string(result.Form)
	for _, want := range []string{`_labels."mother/age/label"`, `_labels."father/age/label"`, `_labels."village/label-2"`} {
		if !strings.Contains(form, want) {
			t.Errorf("form: want %s in\n%s", want, form)
		}
	}
	if !strings.Contains(string(result.Labels), `"village/label": "English (en)": "Village"`) {
		t.Errorf("labels: the existing entry changed\n%s", result.Labels)
	}
}

// elements written as plain structs, inside list.Concat, or defined once and referenced are all
// yanked where they are defined; references themselves are left alone
func TestExtractLabelsShapes(t *testing.T) {
	result, err := ExtractLabels("testdata/shapes/form.cue")
	if err != nil {
		t.Fatal(err)
	}
	form := string(result.Form)
	for _, want := range []string{
		`name: "tree_name", label: _labels."tree_name/label"`,
		`{yes: _labels."yes_no/yes"}`,
		`name: "height", label: _labels."trees/height/label"`,
		`label: _labels."trees/label"`,
		"choices: _yes_no",
		"[_tree_name]",
	} {
		if !strings.Contains(form, want) {
			t.Errorf("want %s in\n%s", want, form)
		}
	}
}

// yank doesn't need form_settings or a default language
func TestExtractLabelsNoSettings(t *testing.T) {
	result, err := ExtractLabels("testdata/no_settings/form.cue")
	if err != nil {
		t.Fatal(err)
	}
	if want := `label: _labels."q/label"`; !strings.Contains(string(result.Form), want) {
		t.Errorf("want %s in\n%s", want, result.Form)
	}
}
