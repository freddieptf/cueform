package main

#Question: {...}
#Choices: {...}

_fruit: #Choices & {
	list_name: "fruit"
	choices: [{apple: en: "Apple"}, {mango: en: "Mango"}]
}
favourite: #Question & {
	type:     "select_one"
	name:     "favourite"
	label: en: "Favourite fruit"
	choices:  _fruit
	or_other: true
}
eaten: #Question & {
	type:     "select_multiple"
	name:     "eaten"
	label: en: "Fruit eaten today"
	choices:  _fruit
	or_other: false
}
