package labels

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue/load"
)

func TestBuildFiles(t *testing.T) {
	testCases := []struct {
		file      string
		outForm   string
		outLabels string
	}{
		{
			file: "testdata/form.cue",
			outLabels: `package main

_labels: {
	"family_name/label": {
		"English (en)":   "What's your family name?"
		"Afrikaans (af)": "Wat is jou familienaam?"
	}
	"father/label": {
		"English (en)":   "Father"
		"Afrikaans (af)": "Pa"
	}
	"age/label": {
		"English (en)":   "How old is your father?"
		"Afrikaans (af)": "Hoe oud is jou pa?"
	}
	"yes_no/yes": {
		"English (en)":   "Yes"
		"Afrikaans (af)": "Ja"
	}
	"yes_no/no": {
		"English (en)":   "No"
		"Afrikaans (af)": "Nee"
	}
	"home_or_away/label": {
		"English (en)":   "Is he home?"
		"Afrikaans (af)": "Is hy tuis?"
	}
}
`,
			outForm: `package main

_#Question: {...}
_#Choices: {...}
_#Group: {...}
_#Settings: {...}

family_name: _#Question & {
	type:  "text"
	name:  "family_name"
	label: _labels."family_name/label"
}
father: _#Group & {
	type:  "begin_group"
	name:  "father"
	label: _labels."father/label"
	children: [
		_#Question & {
			type:  "integer"
			name:  "age"
			label: _labels."age/label"
		},
		_#Question & {
			type: "select_one"
			choices: _#Choices & {
				list_name: "yes_no"
				choices: [
					{
						yes: _labels."yes_no/yes"
					},
					{
						no: _labels."yes_no/no"
					},
				]
			}
			name:  "home_or_away"
			label: _labels."home_or_away/label"
		},
	]
}
form_settings: _#Settings & {
	type:             "settings"
	form_title:       "test"
	form_id:          "test_id"
	version:          "1"
	default_language: "English (en)"
}
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			form, labels, err := extractLabels(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(string(form), tc.outForm) {
				t.Fatalf("have\n%q\nwant\n%q\n", string(form), tc.outForm)
			}
			if !reflect.DeepEqual(string(labels), tc.outLabels) {
				t.Fatalf("have\n%q\nwant\n%q\n", string(labels), tc.outLabels)
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
