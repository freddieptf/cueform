package main

#Question: {...}
#Settings: {...}

a: #Question & {type: "text", name: "a", label: {"English (en)": "Name", "Swahili (sw)": "Jina"}}
// the same English as a, but a different Swahili translation
b: #Question & {type: "text", name: "b", label: {"English (en)": "Name", "Swahili (sw)": "Jina la mtoto"}}
// the same translations as a
c: #Question & {type: "text", name: "c", label: {"English (en)": "Name", "Swahili (sw)": "Jina"}}
// the same translations as an entry already in labels.cue
v: #Question & {type: "text", name: "v", label: {"English (en)": "Village", "Swahili (sw)": "Kijiji"}}
form_settings: #Settings & {type: "settings", default_language: "English (en)"}
