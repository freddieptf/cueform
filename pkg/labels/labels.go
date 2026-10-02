package labels

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
	"github.com/freddieptf/cueform/encoding/xlsform"
)

var (
	langCodeRe = regexp.MustCompile(`(?P<lang>\w+)\s*\((?P<code>\w+)\)`)
)

type label struct {
	text     string
	lang     string
	langCode string
}

type elementLabel struct {
	id     string
	labels []label
}

type Result struct {
	Form   []byte
	Labels []byte
}

func ExtractLabels(formPath string) (*Result, error) {
	instances, err := xlsform.LoadInstance(formPath)
	if err != nil {
		return nil, err
	}
	var (
		formFile  *ast.File
		labelFile *ast.File
	)
	for _, file := range instances[0].Files {
		if filepath.Base(file.Filename) == filepath.Base(formPath) {
			formFile = file
		} else if filepath.Base(file.Filename) == "labels.cue" {
			labelFile = file
		}
	}
	form, labels, err := extractLabels(formFile, labelFile)
	if err != nil {
		return nil, err
	}
	return &Result{Form: form, Labels: labels}, nil
}

func extractLabels(form, labels *ast.File) (formFile []byte, labelsFile []byte, err error) {
	if form == nil {
		err = errors.New("did not find form file")
		return
	}
	elementLabels, err := getLabels(form, existingLabels(labels))
	if err != nil {
		return
	}
	labelsAstFile, err := buildLabelsFile(labels, elementLabels)
	if err != nil {
		return
	}
	labelsFile, err = format.Node(labelsAstFile, format.Simplify(), format.TabIndent(true))
	if err != nil {
		return
	}
	formFile, err = format.Node(form, format.Simplify(), format.TabIndent(true))
	if err != nil {
		return
	}
	return
}

func getLabels(form *ast.File, existing []elementLabel) ([]elementLabel, error) {
	labelExtractor := newExtractor()
	// reuse entries already in labels.cue, so running yank again doesn't duplicate them
	for _, l := range existing {
		labelExtractor.trackUniq[translationKey(l.labels)] = l.id
		labelExtractor.usedIDs[l.id] = true
	}
	for _, el := range form.Decls {
		field, ok := el.(*ast.Field)
		if !ok {
			continue
		}
		name, _, err := ast.LabelName(field.Label)
		if err != nil {
			return nil, err
		}
		// definitions are schemas, not elements; hidden fields such as helper questions are yanked
		// where they are defined, since references to them are skipped
		if strings.HasPrefix(name, "#") || strings.HasPrefix(name, "_#") || name == "_labels" {
			continue
		}
		if err := labelExtractor.extractLabels(field.Value, ""); err != nil {
			return nil, err
		}
	}
	return labelExtractor.elements, nil
}

func buildLabelsFile(file *ast.File, labels []elementLabel) (*ast.File, error) {
	var labelMapAst *ast.StructLit
	if file != nil {
		for _, decl := range file.Decls {
			switch v := decl.(type) {
			case *ast.Field:
				name, _, err := ast.LabelName(v.Label)
				if err != nil {
					return nil, err
				}
				if name == "_labels" {
					labelMapAst = v.Value.(*ast.StructLit)
				}
			}
		}
	} else {
		labelMapAst = ast.NewStruct()
	}
	for _, l := range labels {
		labelStruct := ast.NewStruct()
		for _, label := range l.labels {
			labelStruct.Elts = append(labelStruct.Elts, &ast.Field{Label: ast.NewIdent(label.lang), Value: ast.NewString((label.text))})
		}
		labelMapAst.Elts = append(labelMapAst.Elts, &ast.Field{Label: ast.NewIdent(l.id), Value: labelStruct})
	}
	decls := []ast.Decl{&ast.Package{Name: ast.NewIdent("main")}, &ast.Field{Label: ast.NewIdent("_labels"), Value: labelMapAst}}
	return &ast.File{Decls: decls}, nil
}

type extractor struct {
	// entry id by translationKey, so identical translations share one entry
	trackUniq map[string]string
	usedIDs   map[string]bool
	elements  []elementLabel
}

// reference returns the _labels reference for a translated value, adding an entry with the given
// id unless the same translations already have one
func (e *extractor) reference(labels elementLabel, id string) ast.Expr {
	key := translationKey(labels.labels)
	if _, exists := e.trackUniq[key]; !exists {
		// an id already used for other translations gets a numbered suffix
		unique := id
		for n := 2; e.usedIDs[unique]; n++ {
			unique = fmt.Sprintf("%s-%d", id, n)
		}
		labels.id = unique
		e.trackUniq[key] = unique
		e.usedIDs[unique] = true
		e.elements = append(e.elements, labels)
	}
	return &ast.SelectorExpr{X: ast.NewIdent("_labels"), Sel: ast.NewString(e.trackUniq[key])}
}

// translationKey identifies a value by all of its translations: two values share an entry only
// when every language has the same text
func translationKey(labels []label) string {
	pairs := make([]string, len(labels))
	for i, l := range labels {
		pairs[i] = l.lang + "\x00" + l.text
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\x01")
}

// existingLabels reads the entries already in labels.cue
func existingLabels(file *ast.File) []elementLabel {
	if file == nil {
		return nil
	}
	var entries []elementLabel
	for _, decl := range file.Decls {
		field, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		if name, _, _ := ast.LabelName(field.Label); name != "_labels" {
			continue
		}
		labelMap, ok := field.Value.(*ast.StructLit)
		if !ok {
			continue
		}
		for _, el := range labelMap.Elts {
			entry, ok := el.(*ast.Field)
			if !ok {
				continue
			}
			id, _, _ := ast.LabelName(entry.Label)
			translations, ok := entry.Value.(*ast.StructLit)
			if !ok {
				continue
			}
			l := elementLabel{id: id}
			for _, t := range translations.Elts {
				tf, ok := t.(*ast.Field)
				if !ok {
					continue
				}
				lit, ok := tf.Value.(*ast.BasicLit)
				if !ok {
					continue
				}
				lang, _, _ := ast.LabelName(tf.Label)
				text, err := literal.Unquote(lit.Value)
				if err != nil {
					continue
				}
				l.labels = append(l.labels, label{lang: lang, text: text})
			}
			entries = append(entries, l)
		}
	}
	return entries
}

func newExtractor() *extractor {
	return &extractor{trackUniq: make(map[string]string), usedIDs: make(map[string]bool), elements: []elementLabel{}}
}

// extractLabels yanks an element's translations. path is the names of the groups it is in, so ids
// such as "mother/age/label" stay distinct when names repeat in different groups. Values that
// aren't struct literals, such as references to questions defined elsewhere, are skipped: their
// translations live where they are defined.
func (e *extractor) extractLabels(node ast.Expr, path string) error {
	elStruct := elementStruct(node)
	if elStruct == nil {
		return nil
	}
	// a choice list defined on its own, such as _yes_no: #Choices & {...}
	if hasField(elStruct, "list_name") {
		return e.extractChoices(node)
	}
	elName := getElementName(elStruct)
	if elName == "" {
		// not an element, such as form_settings or a helper value
		return nil
	}
	elPath := elName
	if path != "" {
		elPath = path + "/" + elName
	}
	for _, el := range elStruct.Elts {
		f, ok := el.(*ast.Field)
		if !ok {
			continue
		}
		name, _, err := ast.LabelName(f.Label)
		if err != nil {
			return err
		}
		switch {
		case xlsform.IsTranslatableColumn(name):
			labels, ok, err := translations(f.Value)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			f.Value = e.reference(labels, fmt.Sprintf("%s/%s", elPath, name))
		case name == "choices":
			if err := e.extractChoices(f.Value); err != nil {
				return err
			}
		case name == "children":
			for _, child := range elementsIn(f.Value) {
				if err := e.extractLabels(child, elPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// extractChoices yanks the labels of a choice list. Choice lists are named across the whole form,
// so their ids are "list/choice".
func (e *extractor) extractChoices(node ast.Expr) error {
	list := elementStruct(node)
	if list == nil {
		return nil
	}
	listName := getElementName(list)
	for _, el := range list.Elts {
		f, ok := el.(*ast.Field)
		if !ok || listName == "" {
			continue
		}
		if name, _, _ := ast.LabelName(f.Label); name != "choices" {
			continue
		}
		for _, entry := range elementsIn(f.Value) {
			entryStruct := elementStruct(entry)
			if entryStruct == nil {
				continue
			}
			for _, c := range entryStruct.Elts {
				target, ok := c.(*ast.Field)
				if !ok {
					continue
				}
				key, _, err := ast.LabelName(target.Label)
				if err != nil {
					return err
				}
				if key == "filterCategory" {
					continue
				}
				// a choice with media is {label: ..., image: ...}; only its label is yanked
				if details, ok := target.Value.(*ast.StructLit); ok {
					if labelField := choiceLabelField(details); labelField != nil {
						target = labelField
					}
				}
				labels, ok, err := translations(target.Value)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				target.Value = e.reference(labels, fmt.Sprintf("%s/%s", listName, key))
			}
		}
	}
	return nil
}

func hasField(s *ast.StructLit, name string) bool {
	for _, el := range s.Elts {
		if f, ok := el.(*ast.Field); ok {
			if n, _, _ := ast.LabelName(f.Label); n == name {
				return true
			}
		}
	}
	return false
}

// elementStruct returns the struct literal of an element written as {...}, #Question & {...} or
// (...), or nil for anything else, such as a reference
func elementStruct(expr ast.Expr) *ast.StructLit {
	switch v := expr.(type) {
	case *ast.StructLit:
		return v
	case *ast.ParenExpr:
		return elementStruct(v.X)
	case *ast.BinaryExpr:
		if s := elementStruct(v.Y); s != nil {
			return s
		}
		return elementStruct(v.X)
	}
	return nil
}

// elementsIn returns the elements of a list literal, including lists passed to calls such as
// list.Concat([[...], other])
func elementsIn(expr ast.Expr) []ast.Expr {
	switch v := expr.(type) {
	case *ast.ListLit:
		var elements []ast.Expr
		for _, el := range v.Elts {
			if inner, ok := el.(*ast.ListLit); ok {
				elements = append(elements, elementsIn(inner)...)
			} else {
				elements = append(elements, el)
			}
		}
		return elements
	case *ast.CallExpr:
		var elements []ast.Expr
		for _, arg := range v.Args {
			elements = append(elements, elementsIn(arg)...)
		}
		return elements
	case *ast.ParenExpr:
		return elementsIn(v.X)
	}
	return nil
}

// translations reads a {lang: "text"} struct literal; ok is false for a plain value or anything
// that isn't literal text, which is left in the form
func translations(expr ast.Expr) (elementLabel, bool, error) {
	lit, ok := expr.(*ast.StructLit)
	if !ok || len(lit.Elts) == 0 {
		return elementLabel{}, false, nil
	}
	labels := elementLabel{labels: []label{}}
	for _, el := range lit.Elts {
		f, ok := el.(*ast.Field)
		if !ok {
			return elementLabel{}, false, nil
		}
		l, ok, err := getLabelFromField(f)
		if err != nil || !ok {
			return elementLabel{}, false, err
		}
		labels.labels = append(labels.labels, l)
	}
	return labels, true, nil
}

// choiceLabelField returns the label field of a {label: ..., image: ...} choice, or nil
func choiceLabelField(choice *ast.StructLit) *ast.Field {
	for _, el := range choice.Elts {
		if f, ok := el.(*ast.Field); ok {
			if name, _, _ := ast.LabelName(f.Label); name == "label" {
				return f
			}
		}
	}
	return nil
}

// getElementName returns an element's name, or a choice list's list_name, or "" if it has neither
// as literal text
func getElementName(el *ast.StructLit) string {
	for _, e := range el.Elts {
		f, ok := e.(*ast.Field)
		if !ok {
			continue
		}
		if name, _, _ := ast.LabelName(f.Label); name != "name" && name != "list_name" {
			continue
		}
		if lit, ok := f.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if elName, err := literal.Unquote(lit.Value); err == nil {
				return elName
			}
		}
	}
	return ""
}

// getLabelFromField reads one translation; ok is false unless it is literal text, and a language
// key not written as "Name (code)" is an error
func getLabelFromField(field *ast.Field) (label, bool, error) {
	lang, _, err := ast.LabelName(field.Label)
	if err != nil {
		return label{}, false, nil
	}
	lit, ok := field.Value.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return label{}, false, nil
	}
	text, err := literal.Unquote(lit.Value)
	if err != nil {
		return label{}, false, nil
	}
	match := langCodeRe.FindStringSubmatch(lang)
	if len(match) != 3 {
		return label{}, false, xlsform.ErrInvalidLabel
	}
	return label{lang: lang, langCode: match[2], text: text}, true, nil
}
