package main

#Question: {...}
#Group: {...}
#Choices: {...}
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
			type:    "select_one"
			choices: #Choices & {
				list_name: "ages"
				choices: [
					{
						over_30: "English (en)": "Over 30"
					},
					{
						over_40: "English (en)": "Over 40"
					},
				]
			}
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
