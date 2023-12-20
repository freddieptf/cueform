package main

_#Question: {...}
_#Choices: {...}
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
		_#Question & {
			type:    "select_one"
			choices: _#Choices & {
				list_name: "yes_no"
				choices: [
					{
						yes: {
							"English (en)":   "Yes"
							"Afrikaans (af)": "Ja"
						}
					},
					{
						no: {
							"English (en)":   "No"
							"Afrikaans (af)": "Nee"
						}
					},
				]
			}
			name: "home_or_away"
			label: {
				"English (en)":   "Is he home?"
				"Afrikaans (af)": "Is hy tuis?"
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
