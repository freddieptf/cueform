package schema

#Translatable: [string]: string

#Condition: {
    element: #Question
    expr: string
}

#QuestionType: string
#QuestionType: "integer" | "decimal" | "text" | "note" | "select_one" | "select_multiple" | "date" | "time" | "dateTime" | "hidden"

#Choices: {
    name: string
    choices: [string]: #Translatable
}

#Question: {
    name: string
    type: #QuestionType
    label: #Translatable
    hint?: #Translatable
    choices?: #Choices
    constraint?: string
    constraint_message?: #Translatable
    required?: bool
    required_message?: #Translatable
    relevant?: [...#Condition]
    choice_filter?: string
    read_only?: bool
    calculation?: string
    appearance?: string
}

#GroupAppearance: string
#GroupAppearance: "field-list" | "table-list"

#Group: {
    name: string
    label?: #Translatable
    relevant?: [...#Condition]
    begin_after?: #Question
    appearance?: #GroupAppearance
}

#SurveyConfiguration: {
    #Group
    children: [...#Group]
}

#FormSettings: {
    form_title?: string
    form_id?: string
    version?: string
    instance_name?: string
    default_language?: string
    public_key?: string
}