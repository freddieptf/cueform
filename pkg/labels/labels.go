package labels

import (
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/load"
	"github.com/freddieptf/cueform/encoding/xlsform"
	"github.com/sqids/sqids-go"
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

func extractLabels(formPath string) (formFile []byte, labelsFile []byte, err error) {
	instances := load.Instances([]string{formPath}, &load.Config{})
	form := instances[0].Files[0]
	val := cuecontext.New().BuildInstance(instances[0])
	var defaultLang string
	defaultLang, err = getDefaultLang(&val)
	if err != nil {
		return
	}
	var labels []elementLabel
	labels, err = getLabels(defaultLang, form)
	if err != nil {
		return
	}
	labelsAstFile, err := buildLabelsFile(labels)
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

func getDefaultLang(val *cue.Value) (string, error) {
	form, err := xlsform.ParseCueFormFromVal(val)
	if err != nil {
		return "", err
	}
	if defVal := form.Settings.LookupPath(cue.ParsePath("default_language")); !defVal.Exists() {
		return "", fmt.Errorf("no default lang defined")
	} else {
		return defVal.String()
	}
}

func getLabels(defaultLang string, form *ast.File) ([]elementLabel, error) {
	labelExtractor := newExtractor()
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

func buildLabelsFile(labels []elementLabel) (*ast.File, error) {
	labelMapAst := ast.NewStruct()
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
	trackUniq map[string]string
	elements  []elementLabel
	idGener   *sqids.Sqids
}

func newExtractor() *extractor {
	return &extractor{trackUniq: make(map[string]string), elements: []elementLabel{}}
}

func (e *extractor) newId() (string, error) {
	var err error
	if e.idGener == nil {
		e.idGener, err = sqids.New(sqids.Options{MinLength: 4})
		if err != nil {
			return "", err
		}
	}
	return e.idGener.Encode([]uint64{uint64(len(e.elements))})
}

func (e *extractor) extractLabels(defaultLang string, node *ast.BinaryExpr) error {
	elStruct := node.Y.(*ast.StructLit)
	for _, f := range elStruct.Elts {
		name, _, err := ast.LabelName(f.(*ast.Field).Label)
		if err != nil {
			return err
		}
		labels := elementLabel{labels: []label{}}
		if xlsform.IsTranslatableColumn(name) {
			labelStruct := f.(*ast.Field).Value.(*ast.StructLit)
			for _, ls := range labelStruct.Elts {
				lang, _, err := ast.LabelName(ls.(*ast.Field).Label)
				if err != nil {
					return err
				}
				text := ls.(*ast.Field).Value.(*ast.BasicLit).Value
				text, err = literal.Unquote(text)
				if err != nil {
					return err
				}
				match := langCodeRe.FindStringSubmatch(lang)
				if len(match) != 3 {
					return xlsform.ErrInvalidLabel
				}
				labels.labels = append(labels.labels, label{lang: lang, langCode: match[2], text: text})
			}
			defaultText, err := getDefaultText(defaultLang, labels)
			if err != nil {
				return err
			}
			if _, exists := e.trackUniq[defaultText]; !exists {
				var err error
				labels.id, err = e.newId()
				if err != nil {
					return err
				}
				e.trackUniq[defaultText] = labels.id
				e.elements = append(e.elements, labels)
			}
			f.(*ast.Field).Value = &ast.SelectorExpr{X: ast.NewIdent("_labels"), Sel: ast.NewString(e.trackUniq[defaultText])}
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
