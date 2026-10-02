package main

#Question: {...}

tree_name: #Question & {type: "text", name: "tree_name", label: "Tree name"}
best: #Question & {
	type:         "select_one"
	name:         "best"
	label:        "Best tree"
	choices_from: "${tree_name}"
}
