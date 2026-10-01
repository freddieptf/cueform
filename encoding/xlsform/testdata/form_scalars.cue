package main

#Question: {...}
#Group: {...}
#Choices: {...}
#Settings: {...}

members: #Group & {
	type: "begin_repeat"
	name: "members"
	label: "English (en)": "Members"
	repeat_count: 3
	children: [
		#Question & {
			type:     "decimal"
			name:     "height"
			label: "English (en)": "Height"
			required:  true
			read_only: false
			default:   1.5
		},
		#Question & {
			type: "select_one"
			name: "county"
			label: "English (en)": "County"
			choices: #Choices & {
				list_name: "counties"
				choices: [
					{
						nairobi: "English (en)": "Nairobi"
						filterCategory: country: "ke"
					},
					{
						kampala: "English (en)": "Kampala"
						filterCategory: country: "ug"
					},
				]
			}
		},
	]
}
form_settings: #Settings & {
	type:             "settings"
	form_title:       "test"
	form_id:          "test_id"
	version:          2
	default_language: "English (en)"
}
