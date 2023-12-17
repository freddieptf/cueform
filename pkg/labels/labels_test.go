package labels

import (
	"reflect"
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
	bMZn: {
		"English (en)":   "What's your family name?"
		"Afrikaans (af)": "Wat is jou familienaam?"
	}
	UkLW: {
		"English (en)":   "Father"
		"Afrikaans (af)": "Pa"
	}
	gbHJ: {
		"English (en)":   "How old is your father?"
		"Afrikaans (af)": "Hoe oud is jou pa?"
	}
}
`,
			outForm: `package main

_#Question: {...}
_#Group: {...}
_#Settings: {...}

family_name: _#Question & {
	type:  "text"
	name:  "family_name"
	label: _labels."bMZn"
}
father: _#Group & {
	type:  "begin_group"
	name:  "father"
	label: _labels."UkLW"
	children: [
		_#Question & {
			type:  "integer"
			name:  "age"
			label: _labels."gbHJ"
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
					id: "bMZn",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "What's your family name?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Wat is jou familienaam?"},
					},
				},
				{
					id: "UkLW",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "Father"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Pa"},
					},
				},
				{
					id: "gbHJ",
					labels: []label{
						{lang: "English (en)", langCode: "en", text: "How old is your father?"},
						{lang: "Afrikaans (af)", langCode: "af", text: "Hoe oud is jou pa?"},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			instances := load.Instances([]string{tc.file}, &load.Config{})
			labels, err := getLabels("English (en)", instances[0].Files[0])
			if err != tc.err {
				t.Fatalf("have %s but want %s", err, tc.err)
			}
			if !reflect.DeepEqual(labels, tc.result) {
				t.Fatalf("have\n%+v\nwant\n%+v\n", labels, tc.result)
			}
		})
	}
}
