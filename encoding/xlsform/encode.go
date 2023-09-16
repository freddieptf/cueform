package xlsform

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"regexp"

	"cuelang.org/go/cue"
	"github.com/xuri/excelize/v2"
)

var (
	langRe        = regexp.MustCompile(`::.+`)
	surveyColumns = []string{"type", "name", "label", "required", "required_message", "relevant", "constraint", "constraint_message", "hint", "choice_filter", "read_only", "calculation", "appearance", "default"}
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
		}
	}
	return nil
}

func (e *Encoder) fillChoicesElement(fieldKeys map[string]struct{}, vals *[]map[string]string, val *cue.Value) error {
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

func (e *Encoder) readFile(file string) ([][]string, error) {
	var (
		headerMap     = make(map[string]struct{})
		elements      = []map[string]string{}
		columnHeaders = []string{}
	)
	val, err := loadFile(e.module, filepath.Join(e.formDir, fmt.Sprintf("%s.cue", file)))
	if err != nil {
		return nil, err
	}
	switch file {
	case "survey":
		err = e.fillSurveyElements(headerMap, &elements, val)
		if err != nil {
			return nil, err
		}
		columnHeaders = getHeadersInOrder(headerMap, surveyColumns)
	case "choices":
		err = e.fillChoicesElement(headerMap, &elements, val)
		if err != nil {
			return nil, err
		}
		columnHeaders = getHeadersInOrder(headerMap, choiceColumns)
	}
	results := [][]string{columnHeaders}
	for _, element := range elements {
		row := make([]string, len(columnHeaders))
		for key, val := range element {
			row[indexOf(columnHeaders, key)] = val
		}
		results = append(results, row)
	}
	return results, nil
}

func setDefaultColumnWidth(sheet string, f *excelize.File) {
	f.SetColWidth(sheet, "A", "ZZ", 30)
	switch sheet {
	case "survey":
		f.SetColWidth(sheet, "C", "C", 50)
	}
}

type Encoder struct {
	formDir string
	module  string
}

func NewEncoder(formDir string) *Encoder {
	return &Encoder{formDir: formDir, module: ""}
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
	for _, sheet := range []string{"survey", "choices"} {
		_, err := formFile.NewSheet(sheet)
		if err != nil {
			return nil, err
		}
		setDefaultColumnWidth(sheet, formFile)
		rows, err := e.readFile(sheet)
		if err != nil {
			return nil, err
		}
		for idx, row := range rows {
			err = formFile.SetSheetRow(sheet, fmt.Sprintf("A%d", idx+1), &row)
			if err != nil {
				return nil, err
			}
		}
	}
	formFile.DeleteSheet("Sheet1")
	return formFile.WriteToBuffer()
}
