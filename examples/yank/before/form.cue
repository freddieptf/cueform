// Run `cueform yank labels form.cue` here to move the translated text into labels.cue.
// ../after holds the result.
package main

import "github.com/freddieptf/cueform/xlsform"

intro: xlsform.#Question & {
	type: "note"
	name: "intro"
	// a single-language value isn't a translation, so yank leaves it here
	label: "Household survey, version 2"
}

household: xlsform.#Group & {
	type: "begin_group"
	name: "household"
	label: {
		"English (en)": "Household"
		"Swahili (sw)": "Kaya"
	}
	children: [
		xlsform.#Question & {
			type: "integer"
			name: "head_age"
			label: {
				"English (en)": "Age"
				"Swahili (sw)": "Umri"
			}
			hint: {
				"English (en)": "In completed years"
				"Swahili (sw)": "Kwa miaka kamili"
			}
		},
		xlsform.#Question & {
			type: "select_one"
			name: "water"
			label: {
				"English (en)": "Main source of water"
				"Swahili (sw)": "Chanzo kikuu cha maji"
			}
			choices: xlsform.#Choices & {
				list_name: "water_sources"
				choices: [
					{tap: {
						"English (en)": "Tap"
						"Swahili (sw)": "Bomba"
					}},
					// a choice with media: yank takes its label and leaves the image
					{well: {
						label: {
							"English (en)": "Well"
							"Swahili (sw)": "Kisima"
						}
						image: "well.png"
					}},
				]
			}
		},
	]
}

member: xlsform.#Group & {
	type: "begin_repeat"
	name: "member"
	label: {
		"English (en)": "Household member"
		"Swahili (sw)": "Mwanakaya"
	}
	children: [
		// the same translations as head_age's label, so both use one entry in labels.cue
		xlsform.#Question & {
			type: "integer"
			name: "member_age"
			label: {
				"English (en)": "Age"
				"Swahili (sw)": "Umri"
			}
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
