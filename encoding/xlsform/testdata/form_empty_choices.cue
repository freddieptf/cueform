package main

#Question: {...}
#Choices: {...}

smokes: #Question & {
	type: "select_one"
	name: "smokes"
	label: "Do you smoke?"
	choices: #Choices & {
		list_name: "yes_no"
		choices: []
	}
}
