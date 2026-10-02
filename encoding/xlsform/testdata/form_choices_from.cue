package main

#Question: {...}
#Group: {...}

_tree_name: #Question & {type: "text", name: "tree_name", label: "Tree name"}

trees: #Group & {
	type: "begin_repeat"
	name: "trees"
	label: "Trees"
	children: [_tree_name]
}
best: #Question & {
	type:  "select_one"
	name:  "best"
	label: "Best tree"
	// interpolation keeps the reference in step with the question's name
	choices_from: "${\(_tree_name.name)}"
}
ranked: #Question & {
	type:         "rank"
	name:         "ranked"
	label:        "Rank the trees"
	choices_from: "${tree_name}"
}
