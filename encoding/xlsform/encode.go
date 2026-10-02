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
	surveyColumns    = []string{"type", "name", "label", "required", "required_message", "relevant", "repeat_count", "constraint", "constraint_message", "hint", "guidance_hint", "image", "big-image", "audio", "video", "choice_filter", "read_only", "calculation", "appearance", "default"}
	choiceColumns    = []string{"list_name", "name", "label", "image", "big-image", "audio", "video"}
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

// anyValue returns a value of the form, for its cue.Context
func (c *CueForm) anyValue() *cue.Value {
	if len(c.SurveyElements) > 0 {
		return c.SurveyElements[0]
	}
	return c.Settings
}

func (c *CueForm) toXLSForm() (*xlsForm, error) {
	survey := []map[string]string{}
	choices := []map[string]string{}
	state := &encodeState{
		surveyColHeaders: make(map[string]struct{}),
		choiceColHeaders: make(map[string]struct{}),
		choiceLists:      make(map[string]choiceList),
		repeatNames:      make(map[string]cue.Path),
		inRepeat:         make(map[string]bool),
	}
	var settings cue.Value
	if form := c.anyValue(); form != nil {
		// the schema must be built in the same context as the form for Unify to work
		s := form.Context().CompileBytes(schema.XLSForm, cue.Filename("xlsform/schema.cue"))
		if s.Err() != nil {
			return nil, fmt.Errorf("error compiling schema: %s", errors.Details(s.Err(), nil))
		}
		state.question = s.LookupPath(cue.MakePath(cue.Def("Question")))
		state.questionType = s.LookupPath(cue.MakePath(cue.Def("QuestionType")))
		state.group = s.LookupPath(cue.MakePath(cue.Def("Group")))
		state.groupType = s.LookupPath(cue.MakePath(cue.Def("GroupType")))
		settings = s.LookupPath(cue.MakePath(cue.Def("Settings")))
	}

	topLevel := map[string]cue.Path{}
	for _, element := range c.SurveyElements {
		err := state.elementToRows(element, &survey, &choices, topLevel)
		if err != nil {
			return nil, err
		}
	}
	for _, ref := range state.references {
		inRepeat, ok := state.inRepeat[ref.name]
		if !ok {
			return nil, fmt.Errorf("%s: no question is named %q", ref.path, ref.name)
		}
		if !inRepeat {
			return nil, fmt.Errorf("%s: %q isn't inside a repeat, so its answers can't be choices", ref.path, ref.name)
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
		if err := settings.Unify(*c.Settings).Validate(cue.Concrete(true)); err != nil {
			return nil, fmt.Errorf("%s does not match the schema: %s", c.Settings.Path(), errors.Details(err, nil))
		}
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
	// repeat names must be unique in the whole form
	repeatNames map[string]cue.Path
	// every question name, and whether it is inside a repeat
	inRepeat    map[string]bool
	repeatDepth int
	// choices_from references, checked once every name is known
	references []reference
}

type choiceList struct {
	rows []map[string]string
	path cue.Path
}

func isGroupType(elementType string) bool {
	return strings.HasPrefix(elementType, "begin_") || strings.HasPrefix(elementType, "begin ")
}

// usesChoices reports whether a question type takes a list from the choices sheet, written as
// "<type> <list_name>"
func usesChoices(elementType string) bool {
	return (strings.HasPrefix(elementType, "select_") && !isFromFile(elementType)) || elementType == "rank"
}

// isFromFile reports whether a question type takes its choices from a form attachment, written as
// "<type> <file>"
func isFromFile(elementType string) bool {
	return strings.HasPrefix(elementType, "select_") && strings.HasSuffix(elementType, "_from_file")
}

type reference struct {
	name string
	path cue.Path
}

// choiceSource returns what follows the type in the type column: the choice list's name, the
// attached file a _from_file select reads, or the ${question} whose repeated answers are the choices
func choiceSource(val *cue.Value, elementType string) (string, error) {
	choices := val.LookupPath(cue.ParsePath("choices"))
	file := val.LookupPath(cue.ParsePath("file"))
	from := val.LookupPath(cue.ParsePath("choices_from"))
	if from.Exists() {
		switch {
		case elementType == "select_multiple":
			// pyxform 4.5.0 fails with a KeyError: https://github.com/XLSForm/pyxform/issues/773
			return "", fmt.Errorf("%s: pyxform doesn't support choices_from on select_multiple", from.Path())
		case elementType != "select_one" && elementType != "rank":
			return "", fmt.Errorf("%s: choices_from is only for select_one and rank", from.Path())
		case choices.Exists():
			return "", fmt.Errorf("%s: a question can't have both choices and choices_from", from.Path())
		}
		return from.String()
	}
	switch {
	case isFromFile(elementType):
		if choices.Exists() {
			return "", fmt.Errorf("%s: %s reads its choices from file, so it can't have choices", val.Path(), elementType)
		}
		return file.String()
	case usesChoices(elementType):
		if file.Exists() {
			return "", fmt.Errorf("%s: file is only for select_one_from_file and select_multiple_from_file", file.Path())
		}
		if !choices.Exists() {
			if elementType == "select_one" || elementType == "rank" {
				return "", fmt.Errorf("%s: %s needs choices, or choices_from naming a question in a repeat", val.Path(), elementType)
			}
			return "", fmt.Errorf("%s: %s needs choices", val.Path(), elementType)
		}
		return choices.LookupPath(cue.ParsePath("list_name")).String()
	case file.Exists():
		return "", fmt.Errorf("%s: file is only for select_one_from_file and select_multiple_from_file", file.Path())
	}
	return "", nil
}

// siblings holds the names used so far in the element's group, repeat or survey; pyxform
// requires names to be unique there
func (e *encodeState) elementToRows(val *cue.Value, rows *[]map[string]string, choices *[]map[string]string, siblings map[string]cue.Path) error {
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

	name, err := val.LookupPath(cue.ParsePath("name")).String()
	if err != nil {
		return fmt.Errorf("%s: %s", val.Path(), errors.Details(err, nil))
	}
	if first, ok := siblings[name]; ok {
		return fmt.Errorf("%s: name %q is already used at %s; names must be unique within their group, repeat or survey", val.Path(), name, first)
	}
	siblings[name] = val.Path()
	e.inRepeat[name] = e.repeatDepth > 0
	if strings.HasSuffix(elementType, "repeat") {
		if first, ok := e.repeatNames[name]; ok {
			return fmt.Errorf("%s: repeat name %q is already used at %s; repeat names must be unique in the form", val.Path(), name, first)
		}
		e.repeatNames[name] = val.Path()
	}

	row, err := fieldsToRow(val, e.surveyColHeaders)
	if err != nil {
		return err
	}
	source, err := choiceSource(val, elementType)
	if err != nil {
		return err
	}
	if strings.HasPrefix(source, "${") {
		e.references = append(e.references, reference{name: strings.TrimSuffix(strings.TrimPrefix(source, "${"), "}"), path: val.LookupPath(cue.ParsePath("choices_from")).Path()})
	}
	if source != "" {
		row["type"] += " " + source
	}
	*rows = append(*rows, row)

	if usesChoices(elementType) && val.LookupPath(cue.ParsePath("choices")).Exists() {
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
			groupNames := map[string]cue.Path{}
			isRepeat := strings.HasSuffix(elementType, "repeat")
			if isRepeat {
				e.repeatDepth++
			}
			for iter.Next() {
				child := iter.Value()
				if err := e.elementToRows(&child, rows, choices, groupNames); err != nil {
					return err
				}
			}
			if isRepeat {
				e.repeatDepth--
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
		// the choice source is part of the type column, which elementToRows writes
		if key == "children" || key == "choices" || key == "file" || key == "choices_from" {
			continue
		}
		if key == "or_other" {
			return nil, fmt.Errorf("%s: %w", elIter.Value().Path(), ErrOrOther)
		}
		// a translatable field is a {lang: text} struct, or one plain value for a single-language form
		if IsTranslatableColumn(key) && elIter.Value().Kind() == cue.StructKind {
			if err := addTranslations(result, keys, key, elIter.Value()); err != nil {
				return nil, err
			}
		} else {
			keyVal, err := valueToCell(elIter.Value())
			if err != nil {
				return nil, err
			}
			result[key] = keyVal
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

// writeText writes a #Text value: a plain value to the column col, or {lang: text} to col::lang
func writeText(row map[string]string, keys map[string]struct{}, col string, val cue.Value) error {
	if val.Kind() == cue.StructKind {
		return addTranslations(row, keys, col, val)
	}
	text, err := valueToCell(val)
	if err != nil {
		return err
	}
	row[col] = text
	keys[col] = struct{}{}
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
			value := choiceIter.Value()
			if !value.LookupPath(cue.ParsePath("label")).Exists() {
				if err := writeText(element, keys, "label", value); err != nil {
					return nil, err
				}
			} else {
				// {label: ..., image: ...}: each field is its own column
				fields, err := value.Fields()
				if err != nil {
					return nil, err
				}
				for fields.Next() {
					if err := writeText(element, keys, fields.Selector().Unquoted(), fields.Value()); err != nil {
						return nil, err
					}
				}
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

// media columns a choice can have besides its label, written in a {label: ..., image: ...} choice
var choiceTextColumns = []string{"image", "big-image", "audio", "video"}

// isChoiceFilterColumn reports whether a choices sheet column is free for choice_filter data,
// rather than one XLSForm defines for choices (name, labels, media)
func isChoiceFilterColumn(col string) bool {
	base, _, _ := strings.Cut(col, "::")
	switch base {
	case "list_name", "name", "label":
		return false
	}
	// media:: is pyxform's older spelling of the media columns
	return base != "media" && !slices.Contains(choiceTextColumns, base)
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
