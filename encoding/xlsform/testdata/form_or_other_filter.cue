package main

#Question: {...}
#Choices: {...}

city: #Question & {
	type:          "select_one"
	name:          "city"
	label: en:     "City"
	choice_filter: "country=${country}"
	or_other:      true
	choices: #Choices & {
		list_name: "cities"
		choices: [{nairobi: en: "Nairobi", filterCategory: country: "ke"}]
	}
}
