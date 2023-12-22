package cueform

#Translatable: [string]: string

#Question: {
	type:           string
	name:           string
	label:          #Translatable
	constraint?:    string
	required?:      string
	relevant?:      string
	choices?:       #Choices
	choice_filter?: string
	read_only?:     bool
	calculation?:   string
	appearance?:    string
	...
}

#Group: {
	type:        string
	name:        string
	label:       #Translatable
	relevant?:   string
	appearance?: string
	children?: [...]
	...
}

#Choice: {
	[string]: #Translatable
	filterCategory?: [string]: string
}

#Choices: {
	list_name: string
	choices: [...#Choice]
}

#Settings: {
	form_title:       string
	form_id:          string
	public_key?:      string
	submission_url?:  string
	default_language: string
	style?:           string
	version:          string
	instance_name?:   string
	...
}
