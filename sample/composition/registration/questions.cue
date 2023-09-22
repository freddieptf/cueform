package registration

import "github.com/freddieptf/cueform/schema/xlsform"

first_name: xlsform.#Question & {
	type:        "text"
	name:        "first_name"
	"label::en": "First Name"
	required:    "yes"
}

middle_name: xlsform.#Question & {
	type: "text"
	name: "middle_name"
	label: en: "Middle Name"
}

last_name: xlsform.#Question & {
	type: "text"
	name: "last_name"
	label: en: "Last Name"
}

age: xlsform.#Question & {
	type: "integer"
	name: "age"
	label: en: "Age"
}

questions: [
	first_name,
	middle_name,
	last_name,
	age,
]
