package xlsform

import "list"

// one value per language; "label" is not a language, so a choice's {label: ...} details are
// never read as translations
#Translatable: {[!="label"]: string}

// a single-language value, or one value per language
#Text: string | #Translatable

// Question types shown to the person filling in the form; they need a label.
#LabelledQuestionType: "select_one" | "select_multiple" | "select_one_from_file" | "select_multiple_from_file" |
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
	// the attached CSV, XML or GeoJSON file a _from_file select reads its choices from
	file?: =~"\\.(csv|xml|geojson)$"
	// a ${question} inside a repeat, whose answers become the choices
	choices_from?: =~"^\\$\\{[A-Za-z_][A-Za-z0-9_.-]*\\}$"
	choice_filter?:      string
	read_only?:          string | bool
	calculation?:        string
	appearance?:         string
	if !list.Contains(#UnlabelledQuestionTypes, type) {
		label!: #Text
	}
	if list.Contains(["select_one_from_file", "select_multiple_from_file"], type) {
		file!: _
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

// a choice is its label, or its label with media: {label: "Apple", image: "apple.png"}
#Choice: {
	[!="filterCategory"]: #Text | #ChoiceDetails
	filterCategory?: [string]: string
}

#ChoiceDetails: {
	label!:       #Text
	image?:       #Text
	"big-image"?: #Text
	audio?:       #Text
	video?:       #Text
}

#Choices: {
	list_name: string
	// pyxform rejects a select whose list has no choices
	choices: [#Choice, ...#Choice]
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
	// yes/no flags; a bool is written as yes or no
	allow_choice_duplicates?: string | bool
	clean_text_values?:       string | bool
	omit_instanceID?:         string | bool
	client_editable?:         string | bool
	add_none_option?:         string | bool
	// copied into the XForm as written, so they must be the text "true" or "false"
	auto_send?:   "true" | "false"
	auto_delete?: "true" | "false"
	...
}
