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

type cueform struct {
	surveyElements []*cue.Value
	settings       *cue.Value
}

func parseCueForm(file string) (*cueform, error) {
	val, err := loadFile(file)
	if err != nil {
		return nil, err
	}
	form := &cueform{surveyElements: []*cue.Value{}}
	fieldIter, err := getIter(val)
	if err != nil {
		return nil, err
	}
	for fieldIter.Next() {
		element := fieldIter.Value()
		if l, _ := element.Label(); l == "form_settings" {
			form.settings = &element
		} else {
			form.surveyElements = append(form.surveyElements, &element)
		}
	}
	return form, nil
}

func (c *cueform) toSurvey() ([]map[string]string, []map[string]string, error) {
	survey := []map[string]string{}
	choices := []map[string]string{}
	for _, element := range c.surveyElements {
		err := elementToRows(element, &survey, &choices)
		if err != nil {
			return nil, nil, err
		}
	}
	return survey, choices, nil
}

func elementToRows(val *cue.Value, rows *[]map[string]string, choices *[]map[string]string) error {
	elementTypeVal := val.LookupPath(cue.ParsePath("type"))
	elementType, err := elementTypeVal.String()
	if err != nil {
		return err
	}

	row, err := fieldsToRow(val)
	if err != nil {
		return err
	}
	*rows = append(*rows, row)

	if strings.HasPrefix(elementType, "select_") {
		choiceStruct := val.LookupPath(cue.ParsePath("choices"))
		c, err := choiceStructToRows(&choiceStruct)
		if err != nil {
			return err
		}
		*choices = append(*choices, c...)
	}

	if strings.HasPrefix(elementType, "begin_") {
		children := val.LookupPath(cue.ParsePath("children"))
		if children.Exists() {
			iter, err := getIter(&children)
			if err != nil {
				return err
			}
			for iter.Next() {
				child := iter.Value()
				elementToRows(&child, rows, choices)
			}
		}
		endTag := fmt.Sprintf("end_%s", strings.TrimPrefix(elementType, "begin_"))
		*rows = append(*rows, map[string]string{"type": endTag})
	}
	return nil
}

func fieldsToRow(val *cue.Value) (map[string]string, error) {
	elIter, err := val.Fields()
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for elIter.Next() {
		key := elIter.Label()
		if key == "children" || key == "choices" {
			continue
		}
		if isTranslatableColumn(translatableCols, key) {
			langsIter, err := elIter.Value().Fields()
			if err != nil {
				return nil, err
			}
			for langsIter.Next() {
				labelHeader := fmt.Sprintf("%s::%s", key, langsIter.Label())
				result[labelHeader], err = langsIter.Value().String()
				if err != nil {
					return nil, err
				}
			}
		} else {
			keyVal, err := elIter.Value().String()
			if err != nil {
				return nil, err
			}
			if key == "type" && strings.HasPrefix(keyVal, "select_") {
				choiceStruct := val.LookupPath(cue.ParsePath("choices"))
				listName, err := choiceStruct.LookupPath(cue.ParsePath("list_name")).String()
				if err != nil {
					return nil, err
				}
				result[key] = fmt.Sprintf("%s %s", keyVal, listName)
			} else {
				result[key] = keyVal
			}
		}
	}
	return result, nil
}

func choiceStructToRows(val *cue.Value) ([]map[string]string, error) {
	el := val.Value()
	listName, err := el.LookupPath(cue.ParsePath("list_name")).String()
	if err != nil {
		return nil, err
	}
	choicesIter, err := el.LookupPath(cue.ParsePath("choices")).List()
	if err != nil {
		return nil, err
	}

	elements := []map[string]string{}
	for choicesIter.Next() {
		choiceIter, err := choicesIter.Value().Fields()
		if err != nil {
			return nil, err
		}
		for choiceIter.Next() {
			key := choiceIter.Label()
			element := map[string]string{}
			element["list_name"] = listName
			switch key {
			case "filterCategory":
			default:
				element["name"] = key
				choiceStructIter, err := choiceIter.Value().Fields()
				if err != nil {
					return nil, err
				}
				for choiceStructIter.Next() {
					labelKey := fmt.Sprintf("label::%s", choiceStructIter.Label())
					element[labelKey], err = choiceStructIter.Value().String()
					if err != nil {
						return nil, err
					}
				}
			}
			elements = append(elements, element)
		}
	}
	return elements, nil
}

func encode(file string) (map[string][][]string, error) {
	form, err := parseCueForm(file)
	if err != nil {
		return nil, err
	}

	survey, choices, err := form.toSurvey()
	if err != nil {
		return nil, err
	}
	surveyKeys := map[string]struct{}{}
	for _, row := range survey {
		for k := range row {
			surveyKeys[k] = struct{}{}
		}
	}
	surveyHeaders := getHeadersInOrder(surveyKeys, surveyColumns)
	surveyRows := [][]string{surveyHeaders}
	for _, element := range survey {
		row := make([]string, len(surveyHeaders))
		for key, val := range element {
			row[indexOf(surveyHeaders, key)] = val
		}
		surveyRows = append(surveyRows, row)
	}

	var choiceRows [][]string
	if len(choices) > 0 {
		choiceKeys := map[string]struct{}{}
		for _, row := range choices {
			for k := range row {
				choiceKeys[k] = struct{}{}
			}
		}
		choiceHeaders := getHeadersInOrder(choiceKeys, choiceColumns)
		choiceRows = append(choiceRows, choiceHeaders)
		for _, element := range choices {
			row := make([]string, len(choiceHeaders))
			for key, val := range element {
				row[indexOf(choiceHeaders, key)] = val
			}
			choiceRows = append(choiceRows, row)
		}
	}

	var settingRows [][]string
	if form.settings != nil {
		row, err := fieldsToRow(form.settings)
		if err != nil {
			return nil, err
		}
		delete(row, "type")
		settingHeaders := map[string]struct{}{}
		for k := range row {
			settingHeaders[k] = struct{}{}
		}
		headers := getHeadersInOrder(settingHeaders, settingColumns)
		settings := make([]string, len(headers))
		for k, v := range row {
			settings[indexOf(headers, k)] = v
		}
		settingRows = append(settingRows, headers, settings)
	}

	return map[string][][]string{surveySheetName: surveyRows, choiceSheetName: choiceRows, settingsSheetName: settingRows}, nil
}

func setDefaultColumnWidth(sheet string, f *excelize.File) {
	f.SetColWidth(sheet, "A", "ZZ", 30)
	switch sheet {
	case "survey":
		f.SetColWidth(sheet, "C", "C", 50)
	}
}

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

// Encode returns XLSForm equivalent of the CUE file at filePath
func (encoder *Encoder) Encode(filePath string) (*bytes.Buffer, error) {
	result, err := encode(filePath)
	if err != nil {
		return nil, err
	}
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()
	for _, sheet := range []string{surveySheetName, choiceSheetName, settingsSheetName} {
		if len(result[sheet]) > 0 {
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
	}
	formFile.DeleteSheet("Sheet1")
	return formFile.WriteToBuffer()
}
