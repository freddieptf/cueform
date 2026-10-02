package main

#Question: {...}
#Settings: {...}

photo: #Question & {
	type: "image"
	name: "photo"
	label: {en: "Take a photo", fr: "Prenez une photo"}
	hint: {en: "Hold steady", fr: "Restez immobile"}
	guidance_hint: {en: "Only for staff", fr: "Pour le personnel"}
	image:       "example.png"
	"big-image": {en: "large_en.png", fr: "large_fr.png"}
	audio: {en: "prompt_en.mp3", fr: "prompt_fr.mp3"}
}

form_settings: #Settings & {type: "settings", default_language: "en"}
