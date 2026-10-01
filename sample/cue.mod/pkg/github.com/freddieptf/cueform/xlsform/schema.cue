package xlsform

import "list"

#Translatable: [string]: string

// a single-language value, or one value per language
#Text: string | #Translatable

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
	label?:              #Text
	constraint?:         string
	constraint_message?: #Text
	hint?:               #Text
	guidance_hint?:      #Text
	image?:              #Text
	"big-image"?:        #Text
	audio?:              #Text
	video?:              #Text
	required?:           string | bool
	required_message?:   #Text
	relevant?:           string
	choices?:            #Choices
	choice_filter?:      string
	read_only?:          string | bool
	calculation?:        string
	appearance?:         string
	if !list.Contains(#UnlabelledQuestionTypes, type) {
		label!: #Text
	}
	if list.Contains(["select_one", "select_multiple", "rank"], type) {
		choices!: #Choices
	}
	...
}

#GroupAppearance: "field-list" | "table-list"
#GroupType:       "begin_group" | "begin_repeat" | "begin group" | "begin repeat"
#Group: {
	type:        #GroupType
	name:        string
	label:       #Text
	relevant?:   string
	appearance?: #GroupAppearance
	children?: [...]
	...
}

#Choice: {
	[string]: #Text
	filterCategory?: [string]: string
}

#Choices: {
	list_name: string
	choices: [...#Choice]
}

// every settings column is optional in XLSForm
#Settings: {
	form_title?:       string
	form_id?:          string
	public_key?:       string
	submission_url?:   string
	default_language?: string
	style?:            string
	version?:          string | number
	instance_name?:    string
	...
}
