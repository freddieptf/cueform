package scripting

import (
	"github.com/freddieptf/cueform/sample/scripting/transform"
	"github.com/freddieptf/cueform/schema/xlsform"
)

systems: xlsform.#Group & {
	type: "begin group"
	name: "system_group"
	label: en: "System Survey"
	appearance: "field-list"
	children: [
		xlsform.#Question & {
			type:    "select_one"
			choices: transform.systemChoices
			name:    "system_used"
			label: en: "What system are you using enketo on today?"
			required: "yes"
			required_message: en: "You cannot skip this"
		},
	]
}
