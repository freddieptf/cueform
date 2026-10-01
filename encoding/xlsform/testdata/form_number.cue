package main

#Question: {...}
#Group: {...}
#Settings: {...}

_sizes: ["s", "m", "l"]

members: #Group & {
	type: "begin_repeat"
	name: "members"
	label: "English (en)": "Members"
	repeat_count: len(_sizes)
	children: [
		#Question & {
			type:       "range"
			name:       "height"
			label: "English (en)": "Height"
			parameters: "start=0 end=3 step=0.5"
			default:    1.50
		},
		#Question & {
			type:    "integer"
			name:    "count"
			label: "English (en)": "Count"
			default: 0
		},
	]
}
form_settings: #Settings & {
	type:             "settings"
	form_title:       "test"
	form_id:          "test_id"
	version:          2026100101
	default_language: "English (en)"
}
