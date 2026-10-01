package main

#Question: {...}
#Group: {...}

household: #Group & {
	type: "begin group"
	name: "household"
	label: "English (en)": "Household"
	children: [
		#Group & {
			type: "begin repeat"
			name: "members"
			label: "English (en)": "Members"
			children: [
				#Question & {
					type: "text"
					name: "member_name"
					label: "English (en)": "Name"
				},
			]
		},
	]
}
