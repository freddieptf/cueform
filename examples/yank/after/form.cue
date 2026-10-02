// ../before/form.cue after `cueform yank labels form.cue`: each translated value is now a
// reference into labels.cue, which the encoder loads with this file.
package main

import "github.com/freddieptf/cueform/xlsform"

intro: xlsform.#Question & {
	type: "note"
	name: "intro"
	// a single-language value isn't a translation, so yank leaves it here
	label: "Household survey, version 2"
}

household: xlsform.#Group & {
	type:  "begin_group"
	name:  "household"
	label: _labels."household/label"
	children: [
		xlsform.#Question & {
			type:  "integer"
			name:  "head_age"
			label: _labels."head_age/label"
			hint:  _labels."head_age/hint"
		},
		xlsform.#Question & {
			type:  "select_one"
			name:  "water"
			label: _labels."water/label"
			choices: xlsform.#Choices & {
				list_name: "water_sources"
				choices: [
					{tap: _labels."water_sources/tap"},
					// a choice with media: yank takes its label and leaves the image
					{well: {
						label: _labels."water_sources/well"
						image: "well.png"
					}},
				]
			}
		},
	]
}

member: xlsform.#Group & {
	type:  "begin_repeat"
	name:  "member"
	label: _labels."member/label"
	children: [
		// the same translations as head_age's label, so both use one entry in labels.cue
		xlsform.#Question & {
			type:  "integer"
			name:  "member_age"
			label: _labels."head_age/label"
		},
	]
}

form_settings: xlsform.#Settings & {
	type:             "settings"
	form_title:       "Household survey"
	form_id:          "household_survey"
	version:          "2"
	default_language: "English (en)"
}
