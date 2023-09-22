package composition

import (
	"github.com/freddieptf/cueform/sample/composition/registration"
	"github.com/freddieptf/cueform/schema/xlsform"
)

hh_registration: xlsform.#Group & {
	type:        "begin group"
	name:        "hh_registration"
	label: en: "Household Registration"
	appearance:  "field-list"
	children:    [
			{
			type:        "note"
			name:        "note_hh_details"
			label: en: "Please provide the household head details below"
		},
	] + registration.questions
}

hh_member_registration: xlsform.#Group & {
	type:         "begin repeat"
	name:         "hh_member_registration"
	label: en:  "Household Member Registration"
	appearance:   "field-list"
	repeat_count: "3"
	children:     registration.questions
}
