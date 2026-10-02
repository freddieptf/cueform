package labels

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
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
	defaultLang, err := getDefaultLang(formPath)
	if err != nil {
		return nil, err
	}
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
	form, labels, err := extractLabels(defaultLang, formFile, labelFile)
	if err != nil {
		return nil, err
	}
	return &Result{Form: form, Labels: labels}, nil
}

func extractLabels(defaultLang string, form, labels *ast.File) (formFile []byte, labelsFile []byte, err error) {
	if form == nil {
		err = errors.New("did not find form file")
		return
	}
	elementLabels, err := getLabels(defaultLang, form, existingLabels(labels))
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

func getDefaultLang(formPath string) (string, error) {
	form, err := xlsform.ParseCueForm(formPath)
	if err != nil {
		return "", err
	}
	if defVal := form.Settings.LookupPath(cue.ParsePath("default_language")); !defVal.Exists() {
		return "", fmt.Errorf("no default lang defined")
	} else {
		return defVal.String()
	}
}

func getLabels(defaultLang string, form *ast.File, existing []elementLabel) ([]elementLabel, error) {
	labelExtractor := newExtractor()
	// reuse entries already in labels.cue, so running yank again doesn't duplicate them
	for _, l := range existing {
		labelExtractor.trackUniq[translationKey(l.labels)] = l.id
	}
	for _, el := range form.Decls {
		switch v := el.(type) {
		case *ast.Field:
			name, _, err := ast.LabelName(v.Label)
			if err != nil {
				return nil, err
			}
			// naaaaah
			if strings.HasPrefix(name, "#") || strings.HasPrefix(name, "_#") || strings.HasPrefix(name, "_") {
				continue
			}
			err = labelExtractor.extractLabels(defaultLang, v.Value.(*ast.BinaryExpr))
			if err != nil {
				return nil, err
			}
		default:
			// something something
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
	elements  []elementLabel
}

// reference returns the _labels reference for a translated value, adding an entry with the given
// id unless the same translations already have one
func (e *extractor) reference(labels elementLabel, id string) ast.Expr {
	key := translationKey(labels.labels)
	if _, exists := e.trackUniq[key]; !exists {
		labels.id = id
		e.trackUniq[key] = id
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
	return &extractor{trackUniq: make(map[string]string), elements: []elementLabel{}}
}

func (e *extractor) extractLabels(defaultLang string, node *ast.BinaryExpr) error {
	elStruct := node.Y.(*ast.StructLit)
	elName, err := getElementName(elStruct)
	if err != nil {
		log.Println(err)
		return nil
	}
	for _, f := range elStruct.Elts {
		name, _, err := ast.LabelName(f.(*ast.Field).Label)
		if err != nil {
			return err
		}
		if xlsform.IsTranslatableColumn(name) {
			labels := elementLabel{labels: []label{}}
			var labelStruct *ast.StructLit
			switch v := f.(*ast.Field).Value.(type) {
			case *ast.StructLit:
				labelStruct = v
			default:
				continue
			}
			for _, ls := range labelStruct.Elts {
				label, err := getLabelFromField(ls.(*ast.Field))
				if err != nil {
					return err
				}
				labels.labels = append(labels.labels, label)
			}
			if _, err := getDefaultText(defaultLang, labels); err != nil {
				return err
			}
			f.(*ast.Field).Value = e.reference(labels, fmt.Sprintf("%s/%s", elName, name))
		} else if name == "choices" {
			switch v := f.(*ast.Field).Value.(type) {
			case *ast.BinaryExpr:
				err = e.extractLabels(defaultLang, v)
				if err != nil {
					return err
				}
			case *ast.ListLit:
				for _, choice := range v.Elts {
					for _, c := range choice.(*ast.StructLit).Elts {
						labels := elementLabel{labels: []label{}}
						key, _, err := ast.LabelName(c.(*ast.Field).Label)
						if err != nil {
							return err
						}
						if key == "filterCategory" {
							continue
						}
						target := c.(*ast.Field)
						labelStruct, ok := target.Value.(*ast.StructLit)
						if !ok {
							continue
						}
						// a choice with media is {label: ..., image: ...}; only its label is yanked
						if details := choiceLabelField(labelStruct); details != nil {
							target = details
							if labelStruct, ok = details.Value.(*ast.StructLit); !ok {
								continue
							}
						}
						for _, l := range labelStruct.Elts {
							label, err := getLabelFromField(l.(*ast.Field))
							if err != nil {
								return err
							}
							labels.labels = append(labels.labels, label)
						}
						if _, err := getDefaultText(defaultLang, labels); err != nil {
							return err
						}
						target.Value = e.reference(labels, fmt.Sprintf("%s/%s", elName, key))
					}
				}
			}
		} else if name == "children" {
			children := f.(*ast.Field).Value.(*ast.ListLit)
			for _, child := range children.Elts {
				err := e.extractLabels(defaultLang, child.(*ast.BinaryExpr))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
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

func getElementName(el *ast.StructLit) (string, error) {
	for _, f := range el.Elts {
		name, _, err := ast.LabelName(f.(*ast.Field).Label)
		if err != nil {
			return "", err
		}
		if name == "name" || name == "list_name" {
			text := f.(*ast.Field).Value.(*ast.BasicLit).Value
			elName, err := literal.Unquote(text)
			if err != nil {
				return "", err
			}
			return elName, nil
		}
	}
	return "", errors.New("missing name")
}

func getLabelFromField(field *ast.Field) (label, error) {
	lang, _, err := ast.LabelName(field.Label)
	if err != nil {
		return label{}, err
	}
	text := field.Value.(*ast.BasicLit).Value
	text, err = literal.Unquote(text)
	if err != nil {
		return label{}, err
	}
	match := langCodeRe.FindStringSubmatch(lang)
	if len(match) != 3 {
		return label{}, xlsform.ErrInvalidLabel
	}
	return label{lang: lang, langCode: match[2], text: text}, nil
}

func getDefaultText(defaultLang string, label elementLabel) (string, error) {
	var defaultText string
	for _, l := range label.labels {
		if l.lang != defaultLang {
			continue
		}
		defaultText = l.text
		break
	}
	if defaultText == "" {
		return "", fmt.Errorf("found labels with no entry for default lang: %+v", label.labels)
	}
	return defaultText, nil
}
