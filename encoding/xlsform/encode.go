package xlsform

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"github.com/xuri/excelize/v2"
)

var (
	langRe        = regexp.MustCompile(`::.+`)
	surveyColumns = []string{"type", "name", "label", "required", "required_message", "relevant", "repeat_count", "constraint", "constraint_message", "hint", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns = []string{"list_name", "name", "label"}
)

func (e *Encoder) fillSurveyElements(fieldKeys map[string]struct{}, vals *[]map[string]string, val *cue.Value) error {
	fieldIter, err := getIter(val)
	if err != nil {
		return err
	}
	for fieldIter.Next() {
		el := fieldIter.Value()
		elIter, err := el.Fields()
		if err != nil {
			return err
		}
		element := make(map[string]string)
		for elIter.Next() {
			fieldKey := elIter.Label()
			if fieldKey == "children" {
				continue
			} else if fieldKey == "choices" {
				continue
			} else {
				fieldKeys[fieldKey] = struct{}{}
				keyVal, err := elIter.Value().String()
				if err != nil {
					return err
				}
				if fieldKey == "type" && strings.HasPrefix(keyVal, "select_") {
					choiceStruct := el.LookupPath(cue.ParsePath("choices"))
					listName, err := e.fillChoicesElement(&choiceStruct)
					if err != nil {
						return err
					}
					element[fieldKey] = fmt.Sprintf("%s %s", keyVal, listName)
				} else {
					element[fieldKey] = keyVal
				}

			}
		}
		*vals = append(*vals, element)
		children := el.LookupPath(cue.ParsePath("children"))
		if children.Exists() {
			err = e.fillSurveyElements(fieldKeys, vals, &children)
			if err != nil {
				return err
			}
		}
		if elType, err := el.LookupPath(cue.ParsePath("type")).String(); err != nil {
			return fmt.Errorf("%s %+v", err, val)
		} else {
			if elType == "begin group" {
				*vals = append(*vals, map[string]string{"type": "end group"})
			}
			if elType == "begin repeat" {
				*vals = append(*vals, map[string]string{"type": "end repeat"})
			}
		}
	}
	return nil
}

func (e *Encoder) fillChoicesElement(val *cue.Value) (string, error) {
	el := val.Value()
	listName, err := el.LookupPath(cue.ParsePath("list_name")).String()
	if err != nil {
		return "", err
	}
	e.choiceFieldKeys["list_name"] = struct{}{}
	choicesIter, err := el.LookupPath(cue.ParsePath("choices")).List()
	if err != nil {
		return "", err
	}
	for choicesIter.Next() {
		choiceIter, err := choicesIter.Value().Fields()
		if err != nil {
			return "", err
		}
		for choiceIter.Next() {
			key := choiceIter.Label()
			element := map[string]string{}
			element["list_name"] = listName
			switch key {
			case "filterCategory":
			default:
				e.choiceFieldKeys["name"] = struct{}{}
				element["name"] = key
				choiceStructIter, err := choiceIter.Value().Fields()
				if err != nil {
					return "", err
				}
				for choiceStructIter.Next() {
					labelKey := fmt.Sprintf("label::%s", choiceStructIter.Label())
					e.choiceFieldKeys[labelKey] = struct{}{}
					element[labelKey], err = choiceStructIter.Value().String()
					if err != nil {
						return "", err
					}
				}
			}
			e.choices = append(e.choices, element)
		}
	}
	return listName, nil
}

func (e *Encoder) encode() (map[string][][]string, error) {
	var (
		headerMap = make(map[string]struct{})
		elements  = []map[string]string{}
	)
	e.choiceFieldKeys = map[string]struct{}{}
	val, err := loadFile(e.module, e.filePath)
	if err != nil {
		return nil, err
	}
	err = e.fillSurveyElements(headerMap, &elements, val)
	if err != nil {
		return nil, err
	}
	surveyHeaders := getHeadersInOrder(headerMap, surveyColumns)
	surveyRows := [][]string{surveyHeaders}
	for _, element := range elements {
		row := make([]string, len(surveyHeaders))
		for key, val := range element {
			row[indexOf(surveyHeaders, key)] = val
		}
		surveyRows = append(surveyRows, row)
	}
	choiceHeaders := getHeadersInOrder(e.choiceFieldKeys, choiceColumns)
	choiceRows := [][]string{choiceHeaders}
	for _, element := range e.choices {
		row := make([]string, len(choiceHeaders))
		for key, val := range element {
			row[indexOf(choiceHeaders, key)] = val
		}
		choiceRows = append(choiceRows, row)
	}

	return map[string][][]string{"survey": surveyRows, "choices": choiceRows}, nil
}

func setDefaultColumnWidth(sheet string, f *excelize.File) {
	f.SetColWidth(sheet, "A", "ZZ", 30)
	switch sheet {
	case "survey":
		f.SetColWidth(sheet, "C", "C", 50)
	}
}

type Encoder struct {
	filePath        string
	module          string
	choiceFieldKeys map[string]struct{}
	choices         []map[string]string
}

func NewEncoder(filePath string) *Encoder {
	return &Encoder{filePath: filePath, module: ""}
}

func (e *Encoder) UseModule(path string) {
	e.module = path
}

func (e *Encoder) Encode() (*bytes.Buffer, error) {
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()
	result, err := e.encode()
	if err != nil {
		return nil, err
	}
	for _, sheet := range []string{"survey", "choices"} {
		_, err := formFile.NewSheet(sheet)
		if err != nil {
			return nil, err
		}
		setDefaultColumnWidth(sheet, formFile)
		for idx, row := range result[sheet] {
			err = formFile.SetSheetRow(sheet, fmt.Sprintf("A%d", idx+1), &row)
			if err != nil {
				return nil, err
			}
		}
	}
	formFile.DeleteSheet("Sheet1")
	return formFile.WriteToBuffer()
}
