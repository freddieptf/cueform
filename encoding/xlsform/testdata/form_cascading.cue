package main

#Question: {...}
#Choices: {...}

country: #Question & {
	type: "select_one"
	name: "country"
	label: en: "Country"
	choices: #Choices & {
		list_name: "countries"
		choices: [{ke: en: "Kenya"}, {ug: en: "Uganda"}]
	}
}
city: #Question & {
	type:          "select_one"
	name:          "city"
	label: en:     "City"
	choice_filter: "country=${country}"
	choices: #Choices & {
		list_name: "cities"
		choices: [
			{nairobi: en: "Nairobi", filterCategory: country: "ke"},
			{mombasa: en: "Mombasa", filterCategory: country: "ke"},
			{kampala: en: "Kampala", filterCategory: country: "ug"},
		]
	}
}
