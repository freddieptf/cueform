package xlsform

import "list"

#Translatable: [string]: string

// Question types shown to the person filling in the form; they need a label.
#LabelledQuestionType: "select_one" | "select_multiple" | "select_one_from_file" | "select_multiple_from_file" | "select_one_external" |
	"rank" | "text" | "integer" | "decimal" | "range" | "date" | "time" | "dateTime" | "geopoint" | "image" | "audio" | "video" | "file" | "note" |
	"barcode" | "acknowledge" | "geotrace" | "geoshape"

// Question types that are never displayed, so a label is optional.
#UnlabelledQuestionTypes: ["calculate", "hidden", "background-audio", "xml-external", "csv-external",
	"start", "end", "today", "deviceid", "phonenumber", "username", "email", "audit", "start-geopoint", "subscriberid", "simserial"]

#QuestionType: #LabelledQuestionType | or(#UnlabelledQuestionTypes)

#Question: {
	type:                #QuestionType
	name:                string
	label?:              #Translatable
	constraint?:         string
	constraint_message?: #Translatable
	hint?:               #Translatable
	guidance_hint?:      string | #Translatable
	image?:              string | #Translatable
	"big-image"?:        string | #Translatable
	audio?:              string | #Translatable
	video?:              string | #Translatable
	required?:           string | bool
	required_message?:   #Translatable
	relevant?:           string
	choices?:            #Choices
	choice_filter?:      string
	read_only?:          string | bool
	calculation?:        string
	appearance?:         string
	if !list.Contains(#UnlabelledQuestionTypes, type) {
		label!: #Translatable
	}
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
