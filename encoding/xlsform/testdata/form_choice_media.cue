package main

#Question: {...}
#Choices: {...}

fav: #Question & {
	type: "select_one"
	name: "fav"
	label: {en: "Favourite", fr: "Préféré"}
	choices: #Choices & {
		list_name: "fruit"
		choices: [
			{apple: {label: {en: "Apple", fr: "Pomme"}, image: "apple.png", audio: {en: "apple_en.mp3", fr: "apple_fr.mp3"}}},
			{mango: {en: "Mango", fr: "Mangue"}},
		]
	}
}
