package main

#Question: {...}
#Choices: {...}

smokes: #Question & {
	type: "select_one"
	name: "smokes"
	label: en: "Do you smoke?"
	choices: #Choices & {
		list_name: "yes_no"
		choices: [{yes: en: "Yes"}, {no: en: "No"}]
	}
}
drinks: #Question & {
	type: "select_one"
	name: "drinks"
	label: en: "Do you drink?"
	choices: #Choices & {
		list_name: "yes_no"
		choices: [{yes: en: "Yes"}, {no: en: "No"}, {sometimes: en: "Sometimes"}]
	}
}
