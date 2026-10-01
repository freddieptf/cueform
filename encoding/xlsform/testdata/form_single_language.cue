package main

#Question: {...}
#Choices: {...}

smokes: #Question & {
	type:  "select_one"
	name:  "smokes"
	label: "Do you smoke?"
	hint:  "Include e-cigarettes"
	choices: #Choices & {
		list_name: "yes_no"
		choices: [{yes: "Yes"}, {no: "No"}]
	}
}
