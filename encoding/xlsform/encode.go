package xlsform

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"github.com/freddieptf/cueform/schema"
	"github.com/xuri/excelize/v2"
)

var (
	langRe           = regexp.MustCompile(`(?P<column>[\w-]+)::(?P<lang>.+)`)
	TranslatableCols = []string{"label", "required_message", "constraint_message", "hint", "guidance_hint", "image", "big-image", "audio", "video"}
	// translatable columns that are also commonly written once, without a language
	untranslatedCols = []string{"guidance_hint", "image", "big-image", "audio", "video"}
	surveyColumns    = []string{"type", "name", "label", "required", "required_message", "relevant", "repeat_count", "constraint", "constraint_message", "hint", "guidance_hint", "image", "big-image", "audio", "video", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns    = []string{"list_name", "name", "label"}
	settingColumns   = []string{"form_title", "form_id", "public_key", "submission_url", "default_language", "style", "version", "instance_name"}
)

type CueForm struct {
	SurveyElements []*cue.Value
	Settings       *cue.Value
}

func ParseCueForm(file string) (*CueForm, error) {
	val, err := LoadValue(file)
	if err != nil {
		return nil, err
	}
	return parseCueFormFromVal(val)
}

func parseCueFormFromVal(val *cue.Value) (*CueForm, error) {
	form := &CueForm{SurveyElements: []*cue.Value{}}
	fieldIter, err := getIter(val)
	if err != nil {
		return nil, err
	}
	for fieldIter.Next() {
		element := fieldIter.Value()
		if l, _ := element.Label(); l == "form_settings" {
			form.Settings = &element
		} else {
			form.SurveyElements = append(form.SurveyElements, &element)
		}
	}
	return form, nil
}

func (c *CueForm) toXLSForm() (*xlsForm, error) {
	survey := []map[string]string{}
	choices := []map[string]string{}
	state := &encodeState{
		surveyColHeaders: make(map[string]struct{}),
		choiceColHeaders: make(map[string]struct{}),
		choiceLists:      make(map[string]choiceList),
	}
	if len(c.SurveyElements) > 0 {
		// the schema must be built in the same context as the form for Unify to work
		s := c.SurveyElements[0].Context().CompileBytes(schema.XLSForm, cue.Filename("xlsform/schema.cue"))
		if s.Err() != nil {
			return nil, fmt.Errorf("error compiling schema: %s", errors.Details(s.Err(), nil))
		}
		state.question = s.LookupPath(cue.MakePath(cue.Def("Question")))
		state.questionType = s.LookupPath(cue.MakePath(cue.Def("QuestionType")))
		state.group = s.LookupPath(cue.MakePath(cue.Def("Group")))
		state.groupType = s.LookupPath(cue.MakePath(cue.Def("GroupType")))
	}

	for _, element := range c.SurveyElements {
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
			row[slices.Index(orderSurveyColHeaders, key)] = val
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
				row[slices.Index(orderedChoiceColHeaders, key)] = val
			}
			choiceRows = append(choiceRows, row)
		}
		form.choiceColumnHeaders = orderedChoiceColHeaders
		form.choices = choiceRows
	}

	if c.Settings != nil {
		settingHeaders := map[string]struct{}{}
		row, err := fieldsToRow(c.Settings, settingHeaders)
		if err != nil {
			return nil, err
		}
		delete(row, "type")
		delete(settingHeaders, "type")
		orderedSettingColHeaders := getHeadersInOrder(settingHeaders, settingColumns)
		settings := make([]string, len(orderedSettingColHeaders))
		for k, v := range row {
			settings[slices.Index(orderedSettingColHeaders, k)] = v
		}
		form.settingColumnHeaders = orderedSettingColHeaders
		form.settings = [][]string{settings}
	}

	return form, nil
}

type encodeState struct {
	surveyColHeaders map[string]struct{}
	choiceColHeaders map[string]struct{}
	question         cue.Value
	questionType     cue.Value
	group            cue.Value
	groupType        cue.Value
	// choice lists already written, by list_name
	choiceLists map[string]choiceList
}

type choiceList struct {
	rows []map[string]string
	path cue.Path
}

func isGroupType(elementType string) bool {
	return strings.HasPrefix(elementType, "begin_") || strings.HasPrefix(elementType, "begin ")
}

// usesChoices reports whether a question type takes a choice list, written as "<type> <list_name>"
func usesChoices(elementType string) bool {
	return strings.HasPrefix(elementType, "select_") || elementType == "rank"
}

func (e *encodeState) elementToRows(val *cue.Value, rows *[]map[string]string, choices *[]map[string]string) error {
	elementTypeVal := val.LookupPath(cue.ParsePath("type"))
	elementType, err := elementTypeVal.String()
	if err != nil {
		return fmt.Errorf("%s: %s", val.Path(), errors.Details(err, nil))
	}

	def, typeDef, kind := e.question, e.questionType, "question"
	if isGroupType(elementType) {
		def, typeDef, kind = e.group, e.groupType, "group"
	}
	// checked on its own because a failed disjunction reports every alternative
	if err := typeDef.Unify(elementTypeVal).Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("%s: %q is not a valid %s type", val.Path(), elementType, kind)
	}
	if err := def.Unify(*val).Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("%s does not match the schema: %s", val.Path(), errors.Details(err, nil))
	}

	row, err := fieldsToRow(val, e.surveyColHeaders)
	if err != nil {
		return err
	}
	*rows = append(*rows, row)

	if usesChoices(elementType) {
		choiceStruct := val.LookupPath(cue.ParsePath("choices"))
		c, err := choiceStructToRows(&choiceStruct, e.choiceColHeaders)
		if err != nil {
			return err
		}
		listName := c[0]["list_name"]
		// questions often share a list; pyxform rejects a list whose rows appear twice
		if first, ok := e.choiceLists[listName]; ok {
			if !reflect.DeepEqual(first.rows, c) {
				return fmt.Errorf("%s: choice list %q differs from the one at %s; give one of them another list_name", choiceStruct.Path(), listName, first.path)
			}
		} else {
			e.choiceLists[listName] = choiceList{rows: c, path: choiceStruct.Path()}
			*choices = append(*choices, c...)
		}
	}

	if isGroupType(elementType) {
		children := val.LookupPath(cue.ParsePath("children"))
		if children.Exists() {
			iter, err := getIter(&children)
			if err != nil {
				return err
			}
			for iter.Next() {
				child := iter.Value()
				if err := e.elementToRows(&child, rows, choices); err != nil {
					return err
				}
			}
		}
		// keeps the separator, so "begin group" closes with "end group"
		endTag := "end" + strings.TrimPrefix(elementType, "begin")
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
		// a translatable field is a {lang: text} struct; media and guidance_hint may also be plain
		if IsTranslatableColumn(key) && elIter.Value().Kind() == cue.StructKind {
			if err := addTranslations(result, keys, key, elIter.Value()); err != nil {
				return nil, err
			}
		} else {
			keyVal, err := valueToCell(elIter.Value())
			if err != nil {
				return nil, err
			}
			if key == "type" && usesChoices(keyVal) {
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

// addTranslations decodes a translatable value ({lang: text}) into one "col::lang" cell per language
func addTranslations(row map[string]string, keys map[string]struct{}, col string, val cue.Value) error {
	var translations map[string]string
	if err := val.Decode(&translations); err != nil {
		return fmt.Errorf("%s: %s", val.Path(), errors.Details(err, nil))
	}
	for lang, text := range translations {
		header := fmt.Sprintf("%s::%s", col, lang)
		row[header] = text
		keys[header] = struct{}{}
	}
	return nil
}

// valueToCell writes strings exactly as they are, bools as yes/no and numbers as plain decimal text
func valueToCell(val cue.Value) (string, error) {
	switch val.Kind() {
	case cue.BoolKind:
		b, err := val.Bool()
		if err != nil {
			return "", err
		}
		if b {
			return "yes", nil
		}
		return "no", nil
	case cue.IntKind, cue.FloatKind:
		// CUE's exact decimal, so 1.50 keeps its trailing zero
		j, err := val.MarshalJSON()
		if err != nil {
			return "", err
		}
		return string(j), nil
	}
	s, err := val.String()
	if err != nil {
		return "", fmt.Errorf("%s: xlsform values must be strings, bools or numbers: %s", val.Path(), errors.Details(err, nil))
	}
	return s, nil
}

func choiceStructToRows(val *cue.Value, keys map[string]struct{}) ([]map[string]string, error) {
	listName, err := val.LookupPath(cue.ParsePath("list_name")).String()
	if err != nil {
		return nil, err
	}
	choicesIter, err := val.LookupPath(cue.ParsePath("choices")).List()
	if err != nil {
		return nil, err
	}

	elements := []map[string]string{}
	for choicesIter.Next() {
		entry := choicesIter.Value()
		// filterCategory becomes extra columns, which a question's choice_filter can test
		filters := map[string]string{}
		if f := entry.LookupPath(cue.ParsePath("filterCategory")); f.Exists() {
			if err := f.Decode(&filters); err != nil {
				return nil, fmt.Errorf("%s: %s", f.Path(), errors.Details(err, nil))
			}
			for col := range filters {
				if !isChoiceFilterColumn(col) {
					return nil, fmt.Errorf("%s: %q is a choices sheet column, not a filter", f.Path(), col)
				}
			}
		}
		// iterated rather than decoded so choices keep their source order
		choiceIter, err := entry.Fields()
		if err != nil {
			return nil, err
		}
		for choiceIter.Next() {
			key := choiceIter.Label()
			if key == "filterCategory" {
				continue
			}
			element := map[string]string{"list_name": listName, "name": key}
			keys["list_name"] = struct{}{}
			keys["name"] = struct{}{}
			if err := addTranslations(element, keys, "label", choiceIter.Value()); err != nil {
				return nil, err
			}
			for col, value := range filters {
				element[col] = value
				keys[col] = struct{}{}
			}
			elements = append(elements, element)
		}
	}
	return elements, nil
}

// isChoiceFilterColumn reports whether a choices sheet column is free for choice_filter data,
// rather than one XLSForm defines for choices (name, labels, media)
func isChoiceFilterColumn(col string) bool {
	base, _, _ := strings.Cut(col, "::")
	switch base {
	case "list_name", "name", "label", "image", "big-image", "audio", "video", "media":
		return false
	}
	return true
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
	source, err := ParseCueForm(filePath)
	if err != nil {
		return nil, err
	}
	xlsform, err := source.toXLSForm()
	if err != nil {
		return nil, err
	}
	return xlsform.WriteToBuffer()
}
