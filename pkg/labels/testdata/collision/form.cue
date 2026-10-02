package main

#Question: {...}
#Group: {...}
#Settings: {...}

mother: #Group & {type: "begin_group", name: "mother", label: {"English (en)": "Mother"}, children: [
	#Question & {type: "integer", name: "age", label: {"English (en)": "Mother's age"}},
]}
father: #Group & {type: "begin_group", name: "father", label: {"English (en)": "Father"}, children: [
	#Question & {type: "integer", name: "age", label: {"English (en)": "Father's age"}},
]}
// labels.cue already has a "village/label" entry with other text
village: #Question & {type: "text", name: "village", label: {"English (en)": "Village name"}}
form_settings: #Settings & {type: "settings", default_language: "English (en)"}
