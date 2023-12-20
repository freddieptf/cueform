package xlsform

#Translatable: [string]: string
#QuestionType: "select_one" | "select_multiple" | "select_one_from_file" | "select_multiple_from_file" | "select_one_external" |
	"rank" | "text" | "integer" | "decimal" | "date" | "time" | "dateTime" | "geopoint" | "image" | "audio" | "background-audio" | "video" | "file" | "note" |
	"barcode" | "acknowledge" | "calculate" | "geotrace" | "geoshape"

#Question: {
	type:                #QuestionType
	name:                string
	label:               #Translatable
	constraint?:         string
	constraint_message?: #Translatable
	hint?:               #Translatable
	required?:           string
	required_message?:   #Translatable
	relevant?:           string
	choices?:            #Choices
	choice_filter?:      string
	read_only?:          string
	calculation?:        string
	appearance?:         string
	...
}

#GroupAppearance: "field-list" | "table-list"
#GroupType:       "begin_group" | "begin_repeat" | "begin group" | "begin repeat"
#Group: {
	type:        #GroupType
	name:        string
	label:       #Translatable
	relevant?:   string
	appearance?: #GroupAppearance
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
