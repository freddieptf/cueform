package main

#Question: {...}

city: #Question & {
	type: "select_one_from_file"
	name: "city"
	label: "City"
	file: "cities.csv"
}
person: #Question & {
	type:          "select_multiple_from_file"
	name:          "person"
	label:         "People"
	file:          "people.xml"
	choice_filter: "district=${city}"
	parameters:    "value=id label=full_name"
}
