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
	langRe           = regexp.MustCompile(`(?P<column>\w+)::(?P<lang>.+)`)
	translatableCols = []string{"label", "required_message", "constraint_message", "hint"}
	surveyColumns    = []string{"type", "name", "label", "required", "required_message", "relevant", "repeat_count", "constraint", "constraint_message", "hint", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns    = []string{"list_name", "name", "label"}
	settingColumns   = []string{"form_title", "form_id", "public_key", "submission_url", "default_language", "style", "version", "instance_name"}
)

type encodeState struct {
	choiceFieldKeys map[string]struct{}
	choices         []map[string]string
	settings        [][]string
}

func (e *encodeState) fillSurveyElements(fieldKeys map[string]struct{}, vals *[]map[string]string, val *cue.Value) error {
	fieldIter, err := getIter(val)
	if err != nil {
		return err
	}
	for fieldIter.Next() {
		el := fieldIter.Value()
		if l, _ := el.Label(); l == "form_settings" {
			err = e.fillSettingsElement(&el)
			if err != nil {
				return err
			}
			continue
		}
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
			} else if indexOf(translatableCols, fieldKey) != -1 {
				translatableIter, err := elIter.Value().Fields()
				if err != nil {
					return err
				}
				for translatableIter.Next() {
					labelHeader := fmt.Sprintf("%s::%s", fieldKey, translatableIter.Label())
					fieldKeys[labelHeader] = struct{}{}
					element[labelHeader], err = translatableIter.Value().String()
					if err != nil {
						return err
					}
				}
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

func (e *encodeState) fillChoicesElement(val *cue.Value) (string, error) {
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

func (e *encodeState) fillSettingsElement(val *cue.Value) error {
	iter, err := val.Fields()
	if err != nil {
		return err
	}
	fieldKeys := map[string]struct{}{}
	row := map[string]string{}
	for iter.Next() {
		l := iter.Label()
		if l == "type" {
			continue
		}
		fieldKeys[l] = struct{}{}
		v, err := iter.Value().String()
		if err != nil {
			return err
		}
		row[l] = v
	}
	settingsRow := make([]string, len(row))
	headers := getHeadersInOrder(fieldKeys, settingColumns)
	for k, v := range row {
		settingsRow[indexOf(headers, k)] = v
	}
	e.settings = [][]string{headers, settingsRow}
	return nil
}

func (e *encodeState) encode(module, file string) (map[string][][]string, error) {
	var (
		headerMap = make(map[string]struct{})
		elements  = []map[string]string{}
	)
	e.choiceFieldKeys = map[string]struct{}{}
	val, err := loadFile(module, file)
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

	return map[string][][]string{"survey": surveyRows, "choices": choiceRows, "settings": e.settings}, nil
}

func setDefaultColumnWidth(sheet string, f *excelize.File) {
	f.SetColWidth(sheet, "A", "ZZ", 30)
	switch sheet {
	case "survey":
		f.SetColWidth(sheet, "C", "C", 50)
	}
}

type Encoder struct {
	filePath string
	module   string
	e        encodeState
}

func NewEncoder(filePath string) *Encoder {
	return &Encoder{filePath: filePath, module: ""}
}

func (encoder *Encoder) UseModule(path string) {
	encoder.module = path
}

func (encoder *Encoder) Encode() (*bytes.Buffer, error) {
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()
	result, err := encoder.e.encode(encoder.module, encoder.filePath)
	if err != nil {
		return nil, err
	}
	for _, sheet := range []string{"survey", "choices", "settings"} {
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
