package registration

import "github.com/freddieptf/cueform/schema/xlsform"

person_registration: xlsform.#Group & {
	type: "begin group"
	name: "person_registration"
	label: en: "Person Registration"
	appearance: "field-list"
	children:   questions
}
