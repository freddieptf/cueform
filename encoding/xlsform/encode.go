package xlsform

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"github.com/xuri/excelize/v2"
)

var (
	langRe           = regexp.MustCompile(`(?P<column>\w+)::(?P<lang>.+)`)
	TranslatableCols = []string{"label", "required_message", "constraint_message", "hint"}
	surveyColumns    = []string{"type", "name", "label", "required", "required_message", "relevant", "repeat_count", "constraint", "constraint_message", "hint", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns    = []string{"list_name", "name", "label"}
	settingColumns   = []string{"form_title", "form_id", "public_key", "submission_url", "default_language", "style", "version", "instance_name"}
)

type cueform struct {
	surveyElements []*cue.Value
	settings       *cue.Value
}

func parseCueForm(file string) (*cueform, error) {
	val, err := LoadFile(file)
	if err != nil {
		return nil, err
	}
	return parseCueFormFromVal(val)
}

func parseCueFormFromVal(val *cue.Value) (*cueform, error) {
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

func (c *cueform) toXLSForm() (*xlsForm, error) {
	survey := []map[string]string{}
	choices := []map[string]string{}
	state := &encodeState{
		surveyColHeaders: make(map[string]struct{}),
		choiceColHeaders: make(map[string]struct{}),
	}

	for _, element := range c.surveyElements {
		err := state.elementToRows(element, &survey, &choices)
		if err != nil {
			return nil, err
		}
	}

	orderSurveyColHeaders := getHeadersInOrder(state.surveyColHeaders, surveyColumns)
	surveyRows := [][]string{}
	for _, element := range survey {
		row := make([]string, len(orderSurveyColHeaders))
		for key, val := range element {
			row[indexOf(orderSurveyColHeaders, key)] = val
		}
		surveyRows = append(surveyRows, row)
	}

	form := &xlsForm{surveyColumnHeaders: orderSurveyColHeaders, survey: surveyRows}

	if len(choices) > 0 {
		orderedChoiceColHeaders := getHeadersInOrder(state.choiceColHeaders, choiceColumns)
		choiceRows := [][]string{}
		for _, element := range choices {
			row := make([]string, len(orderedChoiceColHeaders))
			for key, val := range element {
				row[indexOf(orderedChoiceColHeaders, key)] = val
			}
			choiceRows = append(choiceRows, row)
		}
		form.choiceColumnHeaders = orderedChoiceColHeaders
		form.choices = choiceRows
	}

	if c.settings != nil {
		settingHeaders := map[string]struct{}{}
		row, err := fieldsToRow(c.settings, settingHeaders)
		if err != nil {
			return nil, err
		}
		delete(row, "type")
		delete(settingHeaders, "type")
		orderedSettingColHeaders := getHeadersInOrder(settingHeaders, settingColumns)
		settings := make([]string, len(orderedSettingColHeaders))
		for k, v := range row {
			settings[indexOf(orderedSettingColHeaders, k)] = v
		}
		form.settingColumnHeaders = orderedSettingColHeaders
		form.settings = [][]string{settings}
	}

	return form, nil
}

type encodeState struct {
	surveyColHeaders map[string]struct{}
	choiceColHeaders map[string]struct{}
}

func (e *encodeState) elementToRows(val *cue.Value, rows *[]map[string]string, choices *[]map[string]string) error {
	elementTypeVal := val.LookupPath(cue.ParsePath("type"))
	elementType, err := elementTypeVal.String()
	if err != nil {
		return err
	}

	row, err := fieldsToRow(val, e.surveyColHeaders)
	if err != nil {
		return err
	}
	*rows = append(*rows, row)

	if strings.HasPrefix(elementType, "select_") {
		choiceStruct := val.LookupPath(cue.ParsePath("choices"))
		c, err := choiceStructToRows(&choiceStruct, e.choiceColHeaders)
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
				e.elementToRows(&child, rows, choices)
			}
		}
		endTag := fmt.Sprintf("end_%s", strings.TrimPrefix(elementType, "begin_"))
		*rows = append(*rows, map[string]string{"type": endTag})
	}
	return nil
}

func fieldsToRow(val *cue.Value, keys map[string]struct{}) (map[string]string, error) {
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
		if isTranslatableColumn(TranslatableCols, key) {
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
				keys[labelHeader] = struct{}{}
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
			keys[key] = struct{}{}
		}
	}
	return result, nil
}

func choiceStructToRows(val *cue.Value, keys map[string]struct{}) ([]map[string]string, error) {
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
			keys["list_name"] = struct{}{}
			switch key {
			case "filterCategory":
			default:
				element["name"] = key
				keys["name"] = struct{}{}
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
					keys[labelKey] = struct{}{}
				}
			}
			elements = append(elements, element)
		}
	}
	return elements, nil
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
	source, err := parseCueForm(filePath)
	if err != nil {
		return nil, err
	}
	xlsform, err := source.toXLSForm()
	if err != nil {
		return nil, err
	}
	return xlsform.WriteToBuffer()
}
