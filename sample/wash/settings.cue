package wash

import "github.com/freddieptf/cueform/schema"

languages: ["en", "sw"]

group_mapping: [...schema.#SurveyConfiguration] & [
		{
		name: "wash"
		// children: [
		//     {
		//         name: "wash"
		//         begin_after: s.has_functional_latrine
		//     }
		// ]
	},
	{
		name: "summary"
	},
]
