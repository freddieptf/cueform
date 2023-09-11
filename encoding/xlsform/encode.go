package xlsform

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"

	"cuelang.org/go/cue"
	"github.com/xuri/excelize/v2"
)

func getIter(val *cue.Value) (*cue.Iterator, error) {
	switch val.Eval().Kind() {
	case cue.StructKind:
		if iter, err := val.Fields(cue.Concrete(true)); err != nil {
			return nil, err
		} else {
			return iter, nil
		}
	case cue.ListKind:
		if iter, err := val.List(); err != nil {
			return nil, err
		} else {
			return &iter, nil
		}
	default:
		return nil, fmt.Errorf("no %+v", val)
	}
}

func fillSurveyElements(fieldKeys map[string]struct{}, vals *[]map[string]string, val *cue.Value) error {
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
			} else {
				fieldKeys[fieldKey] = struct{}{}
				element[fieldKey], err = elIter.Value().String()
				if err != nil {
					return err
				}
			}
		}
		*vals = append(*vals, element)
		children := el.LookupPath(cue.ParsePath("children"))
		if children.Exists() {
			err = fillSurveyElements(fieldKeys, vals, &children)
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
		}
	}
	return nil
}

func fillChoicesElement(fieldKeys map[string]struct{}, vals *[]map[string]string, val *cue.Value) error {
	fieldIter, err := getIter(val)
	if err != nil {
		return err
	}
	for fieldIter.Next() {
		el := fieldIter.Value()
		listName, err := el.LookupPath(cue.ParsePath("list_name")).String()
		if err != nil {
			return err
		}
		fieldKeys["list_name"] = struct{}{}
		choicesIter, err := el.LookupPath(cue.ParsePath("choices")).List()
		if err != nil {
			return err
		}
		for choicesIter.Next() {
			choiceIter, err := choicesIter.Value().Fields()
			if err != nil {
				return err
			}
			for choiceIter.Next() {
				key := choiceIter.Label()
				element := map[string]string{}
				element["list_name"] = listName
				switch key {
				case "filterCategory":
				default:
					fieldKeys["name"] = struct{}{}
					element["name"] = key
					choiceStructIter, err := choiceIter.Value().Fields()
					if err != nil {
						return err
					}
					for choiceStructIter.Next() {
						labelKey := fmt.Sprintf("label::%s", choiceStructIter.Label())
						fieldKeys[labelKey] = struct{}{}
						element[labelKey], err = choiceStructIter.Value().String()
						if err != nil {
							return err
						}
					}
				}
				*vals = append(*vals, element)
			}
		}
	}
	return nil
}

var (
	langRe        = regexp.MustCompile(`::.+`)
	surveyColumns = []string{"type", "name", "label", "required", "required_message", "relevant", "constraint", "constraint_message", "hint", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns = []string{"list_name", "name", "label"}
)

func getHeadersInOrder(headers map[string]struct{}, parentList []string) []string {
	columnHeaders := []string{}
	for _, header := range parentList {
		if _, ok := headers[header]; ok {
			columnHeaders = append(columnHeaders, header)
			delete(headers, header)
		} else {
			for key := range headers {
				if header == langRe.ReplaceAllString(key, "") {
					columnHeaders = append(columnHeaders, key)
					delete(headers, key)
					break
				}
			}
		}
	}
	moarFields := []string{}
	for key := range headers {
		moarFields = append(moarFields, key)
	}
	sort.Strings(moarFields)
	columnHeaders = append(columnHeaders, moarFields...)
	return columnHeaders
}

func readSurveyFile(path string) ([][]string, error) {
	val, err := loadFile(path)
	if err != nil {
		return nil, err
	}
	headerMap := make(map[string]struct{})
	elements := []map[string]string{}
	err = fillSurveyElements(headerMap, &elements, val)
	if err != nil {
		return nil, err
	}
	columnHeaders := getHeadersInOrder(headerMap, surveyColumns)
	rows := [][]string{columnHeaders}
	for _, element := range elements {
		row := make([]string, len(columnHeaders))
		for key, val := range element {
			row[indexOf(columnHeaders, key)] = val
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func readChoicesFile(path string) ([][]string, error) {
	val, err := loadFile(path)
	if err != nil {
		return nil, err
	}
	headerMap := make(map[string]struct{})
	elements := []map[string]string{}
	err = fillChoicesElement(headerMap, &elements, val)
	if err != nil {
		return nil, err
	}
	columnHeaders := getHeadersInOrder(headerMap, choiceColumns)
	rows := [][]string{columnHeaders}
	for _, element := range elements {
		row := make([]string, len(columnHeaders))
		for key, val := range element {
			row[indexOf(columnHeaders, key)] = val
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func Encode(formDir string) error {
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()
	for _, sheet := range []string{"survey", "choices"} {
		_, err := formFile.NewSheet(sheet)
		if err != nil {
			return err
		}
	}
	surveyRows, err := readSurveyFile(filepath.Join(formDir, "survey.cue"))
	if err != nil {
		return err
	}
	for idx, row := range surveyRows {
		err = formFile.SetSheetRow(
			"survey",
			fmt.Sprintf("A%d", idx+1),
			&row,
		)
		if err != nil {
			return err
		}
	}
	choiceRows, err := readChoicesFile(filepath.Join(formDir, "choices.cue"))
	if err != nil {
		return err
	}
	for idx, row := range choiceRows {
		err = formFile.SetSheetRow(
			"choices",
			fmt.Sprintf("A%d", idx+1),
			&row,
		)
		if err != nil {
			return err
		}
	}
	return formFile.SaveAs(fmt.Sprintf("%s.xlsx", filepath.Base(formDir)))
}
