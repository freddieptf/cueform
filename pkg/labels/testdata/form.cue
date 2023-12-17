package main

_#Question: {...}
_#Group: {...}
_#Settings: {...}

family_name: _#Question & {
	type: "text"
	name: "family_name"
	label: {
		"English (en)":   "What's your family name?"
		"Afrikaans (af)": "Wat is jou familienaam?"
	}
}
father: _#Group & {
	type: "begin_group"
	name: "father"
	label: {
		"English (en)":   "Father"
		"Afrikaans (af)": "Pa"
	}
	children: [
		_#Question & {
			type: "integer"
			name: "age"
			label: {
				"English (en)":   "How old is your father?"
				"Afrikaans (af)": "Hoe oud is jou pa?"
			}
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
