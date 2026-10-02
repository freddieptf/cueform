package main

import "list"

#Question: {...}
#Group: {...}
#Choices: {...}
#Settings: {...}

// a helper question, used by reference below
_tree_name: #Question & {type: "text", name: "tree_name", label: {"English (en)": "Tree name"}}

// a choice list on its own, used by reference below
_yes_no: #Choices & {list_name: "yes_no", choices: [{yes: {"English (en)": "Yes"}}, {no: {"English (en)": "No"}}]}

trees: #Group & {
	type: "begin_repeat"
	name: "trees"
	label: {"English (en)": "Trees"}
	children: list.Concat([[
		// a plain struct, not #Question & {...}
		{type: "integer", name: "height", label: {"English (en)": "Height"}},
	], [_tree_name]])
}
fruit: #Question & {
	type: "select_one"
	name: "fruit"
	label: {"English (en)": "Has fruit?"}
	choices: _yes_no
}
form_settings: #Settings & {type: "settings", default_language: "English (en)"}
