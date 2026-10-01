package main

#Question: {...}
#Choices: {...}

favourite: #Question & {
	type:     "select_one"
	name:     "favourite"
	label: en: "Favourite fruit"
	or_other: true
	choices: #Choices & {
		list_name: "fruit"
		choices: [{apple: en: "Apple"}, {mango: en: "Mango"}]
	}
}
