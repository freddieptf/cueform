package transform

import (
	"github.com/freddieptf/cueform/sample/scripting/data"
	"github.com/freddieptf/cueform/xlsform"
)

systemChoices: xlsform.#Choices & {
	list_name: "systems"
	choices: [
		for os in data.os {
			{
				"\(os)": en: os
			}
		},
	]
}
