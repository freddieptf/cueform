package survey

import (
   "github.com/freddieptf/cueform/schema"
   "github.com/freddieptf/cueform/schema/xlsform"
   choice "github.com/freddieptf/cueform/sample/wash/survey:choices"
)

has_functional_latrine: schema.#Question & {
   name: "has_functional_latrine"
   type: "select_one"
   choices: choice.yes_no
   label: {
      en: "Does ${place_name} have a functional latrine?"
   }
   required: true
}

n_latrine: xlsform.#Note & {
   name: "n_latrine"
   label: {
      en: "<span style=\"color:blue;\">**Teach on the importance of using latrines to promote environmental hygiene and to prevent diseases.**</span>"
   }
   relevant: "${\(has_functional_latrine.name)}=\"no\""
}