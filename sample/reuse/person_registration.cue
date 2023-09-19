package reuse

import (
	"github.com/freddieptf/cueform/sample/reuse/shared"
	"github.com/freddieptf/cueform/schema/xlsform"
)

person_registration: xlsform.#Group & {
	type:        "begin group"
	name:        "person_registration"
	"label::en": "Person Registration"
	appearance:  "field-list"
	children:    shared.name_questions + [
			xlsform.#Question & {
			type:        "integer"
			name:        "age"
			"label::en": "Age"
		},
	]
}
