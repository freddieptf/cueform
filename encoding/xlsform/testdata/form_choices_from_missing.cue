package main

#Question: {...}

best: #Question & {
	type:         "select_one"
	name:         "best"
	label:        "Best tree"
	choices_from: "${tree}"
}
