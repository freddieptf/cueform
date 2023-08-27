package choices

import (
    "github.com/freddieptf/cueform/schema"
)


yes_no: schema.#Choices & {
    name: "yes_no"
    choices: {
        yes: {
            "en": "Yes"
            "sw": "Ndio"
        }
        no: {
          "en": "No"
        }
    }
}