package main

#Question: {...}
#Group: {...}

age: #Question & {
	type: "integer"
	name: "age"
	label: "Age"
}
father: #Group & {
	type: "begin_group"
	name: "father"
	label: "Father"
	children: [
		// the same name in another group is fine
		#Question & {type: "integer", name: "age", label: "Father's age"},
		#Question & {type: "integer", name: "age", label: "Age again"},
	]
}
