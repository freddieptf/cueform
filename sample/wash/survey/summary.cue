package survey

import (
    "github.com/freddieptf/cueform/schema/xlsform"
)

summary_note: xlsform.#Note & {
   name: "s_note_id"
   label: {
      "en": "Hello Summary"
   }
}

summary_note_2: xlsform.#Note & {
   name: "s_note_id_2"
   label: {
      "en": "Other Summary Note"
   }
}