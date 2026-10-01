package main

#Question: {...}
#Choices: {...}

city: #Question & {
	type: "select_one"
	name: "city"
	label: en: "City"
	choices: #Choices & {
		list_name: "cities"
		choices: [{nairobi: en: "Nairobi", filterCategory: image: "nairobi.png"}]
	}
}
