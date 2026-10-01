package main

#Question: {...}
#Group: {...}

mother: #Group & {
	type: "begin_group"
	name: "mother"
	label: "Mother"
	children: [{type: "begin_repeat", name: "children", label: "Children"}]
}
father: #Group & {
	type: "begin_group"
	name: "father"
	label: "Father"
	children: [{type: "begin_repeat", name: "children", label: "Children"}]
}
