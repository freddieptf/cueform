package shared

import "github.com/freddieptf/cueform/schema/xlsform"

name_questions: [
	xlsform.#Question & {
		type:        "text"
		name:        "first_name"
		"label::en": "First Name"
		required:    "yes"
	},
	xlsform.#Question & {
		type:        "text"
		name:        "middle_name"
		"label::en": "Middle Name"
	},
	xlsform.#Question & {
		type:        "text"
		name:        "last_name"
		"label::en": "Last Name"
	},
]
