package xlsform

#Question: {
	type:           string
	name:           string
	constraint?:    string
	required?:      string
	relevant?:      string
	choice_filter?: string
	read_only?:     bool
	calculation?:   string
	appearance?:    string
	...
}

#Group: {
	type:        string
	name:        string
	relevant?:   string
	appearance?: string
	children?: [...]
	...
}

#Choice: {
	[string]: [string]:        string
	filterCategory?: [string]: string
}

#Choices: {
	name: string
	choices: [...#Choice]
}
