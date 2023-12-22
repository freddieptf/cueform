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
					id: "age/label",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "How old is your father?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Hoe oud is jou pa?"},
					},
				},
				{
					id: "home_or_away/label",
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
			labels, err := getLabels("English (en)", instances[0].Files[0])
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
