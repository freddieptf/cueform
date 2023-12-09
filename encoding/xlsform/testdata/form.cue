package main

#Question: {...}
#Group: {...}
#Settings: {...}

family_name: #Question & {
	type: "text"
	name: "family_name"
	label: "English (en)": "What's your family name?"
}
father: #Group & {
	type: "begin_group"
	name: "father"
	label: "English (en)": "Father"
	children: [
		#Question & {
			type: "integer"
			name: "age"
			label: "English (en)": "How old is your father?"
		},
	]
}
form_settings: #Settings & {
	type:             "settings"
	form_title:       "test"
	form_id:          "test_id"
	version:          "1"
	default_language: "English (en)"
}
