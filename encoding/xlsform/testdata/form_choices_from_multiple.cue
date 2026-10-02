package main

#Question: {...}
#Group: {...}

trees: #Group & {
	type: "begin_repeat"
	name: "trees"
	label: "Trees"
	children: [{type: "text", name: "tree_name", label: "Tree name"}]
}
best: #Question & {
	type:         "select_multiple"
	name:         "best"
	label:        "Best trees"
	choices_from: "${tree_name}"
}
