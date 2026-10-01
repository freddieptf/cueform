package main

#Question: {...}
#Choices: {...}

_yes_no: #Choices & {
	list_name: "yes_no"
	choices: [{yes: en: "Yes"}, {no: en: "No"}]
}
smokes: #Question & {
	type: "select_one"
	name: "smokes"
	label: en: "Do you smoke?"
	choices: _yes_no
}
drinks: #Question & {
	type: "select_multiple"
	name: "drinks"
	label: en: "Which do you drink?"
	choices: _yes_no
}
ranked: #Question & {
	type: "rank"
	name: "ranked"
	label: en: "Rank these"
	choices: _yes_no
}
