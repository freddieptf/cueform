package main

#Question: {...}

intro: #Question & {
	type: "note"
	name: "intro"
	label: "English (en)": "Welcome"
}
started: #Question & {
	type: "start"
	name: "started"
}
total: #Question & {
	type:        "calculate"
	name:        "total"
	calculation: "1 + 1"
}
