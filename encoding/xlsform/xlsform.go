package xlsform

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/xuri/excelize/v2"
)

func loadFile(path string) (*cue.Value, error) {
	ctx := cuecontext.New()
	bis := load.Instances([]string{path}, &load.Config{ModuleRoot: "."})
	bi := bis[0]
	// check for errors on the instance
	// these are typically parsing errors
	if bi.Err != nil {
		return nil, fmt.Errorf("Error during load: %w", bi.Err)
	}
	// Use cue.Context.BuildInstance to turn
	// a build.Instance into a cue.Value
	value := ctx.BuildInstance(bi)
	if value.Err() != nil {
		return nil, fmt.Errorf("Error during build: %w", value.Err())
	}
	// Validate the value
	err := value.Validate(cue.Concrete(true))
	if err != nil {
		return nil, fmt.Errorf("Error during validation: %v", errors.Details(err, nil))
	}
	return &value, nil
}

type condition struct {
	element string
	expr    string
}

type survey struct {
	survey        string
	label         map[string]string
	after_element *question
	depends_on    []*condition
}

type surveyConfig struct {
	survey   *survey
	children []*survey
}

type formConfig struct {
	languages []string
	surveys   []*surveyConfig
}

type question struct {
	survey             string
	name               string
	questionType       string
	label              map[string]string
	hint               map[string]string
	choiceKey          *string
	required           bool
	requiredMessages   map[string]string
	constraint         *string
	constraintMessages map[string]string
	depends_on         []*condition
	choiceFilter       string
	calculation        string
	appearance         string
}

func newQuestion(survey string) *question {
	return &question{
		survey:             survey,
		label:              make(map[string]string),
		hint:               make(map[string]string),
		requiredMessages:   make(map[string]string),
		constraintMessages: make(map[string]string),
	}
}

func fillTranslatable(translations map[string]string, val cue.Value) error {
	vals, err := val.Fields(cue.Concrete(true))
	if err != nil {
		return err
	}
	for vals.Next() {
		translations[vals.Label()], err = vals.Value().String()
		if err != nil {
			return err
		}
	}
	return nil
}

func fillQuestionFields(question *question, label string, val cue.Value) (err error) {
	switch label {
	case "name":
		question.name, err = val.String()
	case "type":
		question.questionType, err = val.String()
	case "label":
		err = fillTranslatable(question.label, val)
	case "hint":
		err = fillTranslatable(question.hint, val)
	case "constraint":
		var constraint string
		constraint, err = val.String()
		if constraint != "" {
			question.constraint = &constraint
		}
	case "constraint_message":
		err = fillTranslatable(question.constraintMessages, val)
	case "required":
		question.required, err = val.Bool()
	case "required_message":
		err = fillTranslatable(question.requiredMessages, val)
	case "choice_filter":
		question.choiceFilter, err = val.String()
	case "appearance":
		question.appearance, err = val.String()
	case "calculation":
		question.calculation, err = val.String()
	case "choices":
		var choiceKey string
		choiceKey, err = val.LookupPath(cue.ParsePath("name")).String()
		question.choiceKey = &choiceKey
	}
	return
}

func fillSurveyFields(survey *survey, label string, val cue.Value) (err error) {
	switch label {
	case "name":
		survey.survey, err = val.String()
		return
	case "begin_after":
		iter, err := val.Fields()
		if err != nil {
			return err
		}
		question := newQuestion(survey.survey)
		for iter.Next() {
			fillQuestionFields(question, iter.Label(), iter.Value())
		}
		survey.after_element = question
	}
	return nil
}

func readFormConfiguration(path string) (*formConfig, error) {
	value, err := loadFile(path)
	if err != nil {
		return nil, err
	}
	languages := []string{}
	languageValue := value.LookupPath(cue.ParsePath("languages"))
	languageIter, err := languageValue.List()
	if err != nil {
		return nil, err
	}
	for languageIter.Next() {
		lang, err := languageIter.Value().String()
		if err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}
	groupMapping := value.LookupPath(cue.ParsePath("group_mapping"))
	iter, err := groupMapping.List()
	if err != nil {
		return nil, err
	}
	surveyConfs := []*surveyConfig{}
	for iter.Next() {
		surveyStruct := iter.Value()
		surveyStructIter, err := surveyStruct.Fields()
		if err != nil {
			return nil, err
		}
		surveyConf := &surveyConfig{survey: &survey{}, children: []*survey{}}
		for surveyStructIter.Next() {
			structLabel := surveyStructIter.Label()
			if structLabel == "children" {
				childrenIter, err := surveyStructIter.Value().List()
				if err != nil {
					return nil, err
				}
				for childrenIter.Next() {
					if childStructIter, err := childrenIter.Value().Fields(); err != nil {
						return nil, err
					} else {
						child := &survey{}
						for childStructIter.Next() {
							err = fillSurveyFields(child, childStructIter.Label(), childStructIter.Value())
							if err != nil {
								return nil, err
							}
						}
						surveyConf.children = append(surveyConf.children, child)
					}
				}
			} else {
				fillSurveyFields(surveyConf.survey, structLabel, surveyStructIter.Value())
			}
		}
		surveyConfs = append(surveyConfs, surveyConf)
	}
	return &formConfig{languages: languages, surveys: surveyConfs}, nil
}

type choiceEntry struct {
	name string
	text map[string]string
}

type choice struct {
	key    string
	labels []choiceEntry
}

func readChoice(val cue.Value) (*choice, error) {
	choiceKey, err := val.LookupPath(cue.ParsePath("name")).String()
	if err != nil {
		return nil, err
	}
	choice := &choice{key: choiceKey, labels: []choiceEntry{}}
	choicesIter, err := val.LookupPath(cue.ParsePath("choices")).Fields()
	if err != nil {
		return nil, err
	}
	for choicesIter.Next() {
		c := choicesIter.Label()
		iter, err := choicesIter.Value().Fields()
		if err != nil {
			return nil, err
		}
		translate := map[string]string{}
		for iter.Next() {
			translate[iter.Label()], err = iter.Value().String()
			if err != nil {
				return nil, err
			}
		}
		choice.labels = append(choice.labels, choiceEntry{name: c, text: translate})
	}
	return choice, nil
}

func readSurveyQuestions(survey, path string) ([]*question, []*choice, error) {
	value, err := loadFile(path)
	if err != nil {
		return nil, nil, err
	}
	questions := []*question{}
	fieldIter, err := value.Fields(cue.Concrete(true))
	if err != nil {
		return nil, nil, err
	}
	choices := []*choice{}
	for fieldIter.Next() {
		questionStruct := fieldIter.Value()
		questionStructIter, err := questionStruct.Fields()
		if err != nil {
			return nil, nil, err
		}
		question := newQuestion(survey)
		for questionStructIter.Next() {
			label := questionStructIter.Label()
			val := questionStructIter.Value()
			fillQuestionFields(question, label, val)
			if label == "choices" {
				choice, err := readChoice(val)
				if err != nil {
					return nil, nil, err
				}
				choices = append(choices, choice)
				fmt.Printf("%+v\n", choice)
			}
		}
		questions = append(questions, question)
	}
	return questions, choices, nil
}

func getSurveyQuestions(formDir string, survey *survey) ([]*question, []*choice, error) {
	questions, choices, err := readSurveyQuestions(survey.survey, filepath.Join(formDir, "survey", fmt.Sprintf("%s.cue", survey.survey)))
	if err != nil {
		return nil, nil, err
	}
	return questions, choices, nil
}

type headerKey int

const (
	BeginGroupHeader headerKey = iota
	EndGroupHeader
)

type GroupHeader struct {
	key    headerKey
	survey *survey
}

func indexOf(arr []interface{}, val interface{}) int {
	for idx, item := range arr {
		if reflect.DeepEqual(item, val) {
			return idx
		}
	}
	return -1
}

func getSurveyColumnHeaders(languages []string) []string {
	labelHeaders := []string{}
	hintHeaders := []string{}
	constraintMsgHeaders := []string{"constraint"}
	requiredMsgHeaders := []string{"required"}
	for _, lang := range languages {
		labelHeaders = append(labelHeaders, fmt.Sprintf("label::%s", lang))
		hintHeaders = append(hintHeaders, fmt.Sprintf("hint::%s", lang))
		constraintMsgHeaders = append(constraintMsgHeaders, fmt.Sprintf("constraint_message::%s", lang))
		requiredMsgHeaders = append(requiredMsgHeaders, fmt.Sprintf("required_message::%s", lang))
	}
	xlsformHeaders := []string{"type", "name"}
	xlsformHeaders = append(xlsformHeaders, labelHeaders...)
	xlsformHeaders = append(xlsformHeaders, requiredMsgHeaders...)
	xlsformHeaders = append(xlsformHeaders, "relevant")
	xlsformHeaders = append(xlsformHeaders, constraintMsgHeaders...)
	xlsformHeaders = append(xlsformHeaders, hintHeaders...)
	xlsformHeaders = append(xlsformHeaders, "choice_filter", "read_only", "calculation", "appearance")
	return xlsformHeaders
}

func fillSurveyRowData(columns []string, q *question) *[]interface{} {
	row := make([]interface{}, len(columns))
	for idx, header := range columns {
		if strings.HasPrefix(header, "label") {
			lang := strings.TrimPrefix(header, "label::")
			row[idx] = q.label[lang]
		} else if strings.HasPrefix(header, "hint") {
			lang := strings.TrimPrefix(header, "label::")
			row[idx] = q.hint[lang]
		} else {
			switch header {
			case "type":
				if strings.HasPrefix(q.questionType, "select_") {
					row[idx] = fmt.Sprintf("%s %s", q.questionType, *q.choiceKey)
				} else {
					row[idx] = q.questionType
				}
			case "name":
				row[idx] = q.name
			case "constraint":
				if q.constraint != nil {
					row[idx] = q.constraint
				}
			case "required":
				if q.required {
					row[idx] = "yes"
				}
			case "relevant":
				row[idx] = ""
			case "choice_filter":
				row[idx] = q.choiceFilter
			case "read_only":
				row[idx] = ""
			case "calculation":
				row[idx] = q.calculation
			case "appearance":
				row[idx] = q.appearance
			}
		}
	}
	return &row
}

func buildSurveyRows(formDir string, languages []string, surveys []*surveyConfig) ([]interface{}, map[string]*choice, error) {
	var rows []interface{}
	choiceMap := make(map[string]*choice)
	for _, surveyConf := range surveys {
		rows = append(rows, GroupHeader{BeginGroupHeader, surveyConf.survey})
		questions, choices, err := getSurveyQuestions(formDir, surveyConf.survey)
		if err != nil {
			return nil, nil, err
		}
		for _, q := range questions {
			rows = append(rows, q)
		}
		for _, choice := range choices {
			choiceMap[choice.key] = choice
		}
		for _, nestedSurvey := range surveyConf.children {
			nestedQuestions, choices, err := getSurveyQuestions(formDir, nestedSurvey)
			if err != nil {
				return nil, nil, err
			}
			for _, choice := range choices {
				choiceMap[choice.key] = choice
			}
			if nestedSurvey.after_element != nil {
				nestedStart := indexOf(rows, nestedSurvey.after_element)
				var newRows []interface{}
				newRows = append(newRows, rows[:nestedStart+1]...)
				newRows = append(newRows, GroupHeader{BeginGroupHeader, nestedSurvey})
				for _, q := range nestedQuestions {
					newRows = append(newRows, q)
				}
				newRows = append(newRows, GroupHeader{EndGroupHeader, nestedSurvey})
				newRows = append(newRows, rows[nestedStart+1:]...)
				rows = newRows
			} else {
				rows = append(rows, GroupHeader{BeginGroupHeader, nestedSurvey})
				for _, q := range nestedQuestions {
					rows = append(rows, q)
				}
				rows = append(rows, GroupHeader{EndGroupHeader, nestedSurvey})
			}
		}
		rows = append(rows, GroupHeader{EndGroupHeader, surveyConf.survey})
	}
	columnHeaders := getSurveyColumnHeaders(languages)
	surveyRows := []interface{}{&columnHeaders}
	for _, row := range rows {
		switch row := row.(type) {
		case *question:
			surveyRows = append(surveyRows, fillSurveyRowData(columnHeaders, row))
		case GroupHeader:
			var groupType string
			if row.key == BeginGroupHeader {
				groupType = "begin group"
			} else {
				groupType = "end group"
			}
			surveyRows = append(surveyRows, &[]interface{}{groupType, row.survey.survey, row.survey.survey})
		}
	}
	return surveyRows, choiceMap, nil
}

func buildChoiceRows(languages []string, choiceMap map[string]*choice) []interface{} {
	columnHeaders := []string{"list_name", "name"}
	for _, lang := range languages {
		columnHeaders = append(columnHeaders, fmt.Sprintf("label::%s", lang))
	}
	rows := []interface{}{&columnHeaders}
	for _, choice := range choiceMap {
		for _, entry := range choice.labels {
			row := make([]interface{}, len(columnHeaders))
			for idx, header := range columnHeaders {
				if header == "list_name" {
					row[idx] = choice.key
				} else if header == "name" {
					row[idx] = entry.name
				} else if strings.HasPrefix(header, "label::") {
					lang := strings.TrimPrefix(header, "label::")
					row[idx] = entry.text[lang]
				}
			}
			rows = append(rows, &row)
		}
	}
	return rows
}

func buildFormRows(formDir string) ([]interface{}, []interface{}, error) {
	settings, err := readFormConfiguration(filepath.Join(formDir, "settings.cue"))
	if err != nil {
		return nil, nil, err
	}
	surveyRows, choiceMap, err := buildSurveyRows(formDir, settings.languages, settings.surveys)
	if err != nil {
		return nil, nil, err
	}
	choiceRows := buildChoiceRows(settings.languages, choiceMap)
	if err != nil {
		return nil, nil, err
	}
	return surveyRows, choiceRows, nil
}

// encode to XLSForm standard
func Encode(formDir string) error {
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	surveyRows, choiceRows, err := buildFormRows(formDir)
	if err != nil {
		return err
	}

	for _, sheet := range []string{"survey", "choices"} {
		_, err := formFile.NewSheet(sheet)
		if err != nil {
			return err
		}
	}

	for idx, row := range surveyRows {
		err = formFile.SetSheetRow(
			"survey",
			fmt.Sprintf("A%d", idx+1),
			row,
		)
		if err != nil {
			return err
		}
	}

	for idx, row := range choiceRows {
		err = formFile.SetSheetRow(
			"choices",
			fmt.Sprintf("A%d", idx+1),
			row,
		)
		if err != nil {
			return err
		}
	}

	return formFile.SaveAs(fmt.Sprintf("%s.xlsx", filepath.Base(formDir)))
}
