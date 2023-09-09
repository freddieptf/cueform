package xlsform

#Question: {
	type:           string
	name:           string
	constraint?:    string
	required?:      bool
	relevant?:      string
	choice_filter?: string
	read_only?:     bool
	calculation?:   string
	appearance?:    string
	...
}

#GroupAppearance: "field-list" | "table-list"

#Group: {
	type:        string
	name:        string
	relevant?:   string
	appearance?: #GroupAppearance
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
