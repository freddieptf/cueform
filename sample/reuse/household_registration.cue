package reuse

import (
	"github.com/freddieptf/cueform/sample/reuse/shared"
	"github.com/freddieptf/cueform/schema/xlsform"
)

hh_registration: xlsform.#Group & {
	type:        "begin group"
	name:        "hh_registration"
	"label::en": "Household Registration"
	appearance:  "field-list"
	children:    [
			xlsform.#Question & {
			type:        "note"
			name:        "note_hh_details"
			"label::en": "Please provide the household head details below"
		},
	] + shared.name_questions + [
		xlsform.#Group & {
			type:         "begin repeat"
			name:         "hh_member_registration"
			"label::en":  "Household Member Registration"
			appearance:   "field-list"
			repeat_count: "3"
			children:     shared.name_questions + [
					xlsform.#Question & {
					type:        "integer"
					name:        "age"
					"label::en": "Age"
				},
			]
		},
	]
}
