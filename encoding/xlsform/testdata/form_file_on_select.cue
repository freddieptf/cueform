package main

#Question: {...}
#Choices: {...}

city: #Question & {
	type: "select_one"
	name: "city"
	label: "City"
	file: "cities.csv"
	choices: #Choices & {list_name: "cities", choices: [{nairobi: "Nairobi"}]}
}
