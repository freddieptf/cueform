package main

#Question: {...}
#Choices: {...}
#Settings: {...}

fav: #Question & {
	type: "select_one"
	name: "fav"
	label: {"English (en)": "Favourite", "French (fr)": "Préféré"}
	choices: #Choices & {
		list_name: "fruit"
		choices: [
			{apple: {label: {"English (en)": "Apple", "French (fr)": "Pomme"}, image: "apple.png"}},
			{mango: {"English (en)": "Mango", "French (fr)": "Mangue"}},
		]
	}
}
form_settings: #Settings & {
	type:             "settings"
	default_language: "English (en)"
}
