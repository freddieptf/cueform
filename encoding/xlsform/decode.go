package xlsform

import (
	"errors"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"github.com/xuri/excelize/v2"
)

var (
	ErrInvalidXLSForm      = errors.New("xlsform structure is incorrect")
	ErrInvalidXLSFormSheet = errors.New("found xlsform sheet missing a required column")
	ErrInvalidLabel        = errors.New("found label column with no language code")

	surveySheetName   = "survey"
	choiceSheetName   = "choices"
	settingsSheetName = "settings"

	requiredSurveySheetColumns = []string{"type", "name", "label"}
	requiredChoiceSheetColumns = []string{"list_name", "name", "label"}
)

type Decoder struct {
	importSpec *ast.ImportSpec
	importInfo astutil.ImportInfo
}

func NewDecoder(pkg string) (*Decoder, error) {
	decoder := &Decoder{}
	err := decoder.UsePkg(pkg)
	return decoder, err
}

func (d *Decoder) UsePkg(pkg string) (err error) {
	d.importSpec = ast.NewImport(nil, pkg)
	d.importInfo, err = astutil.ParseImportSpec(d.importSpec)
	return err
}

func (d *Decoder) Decode(r io.Reader) ([]byte, error) {
	form, err := parseXLSForm(r)
	if err != nil {
		return nil, err
	}
	choiceMap, err := form.choicesToAst(d.importInfo)
	if err != nil {
		return nil, err
	}
	root := ast.NewStruct()
	_, err = form.surveyToAst(d.importInfo, root, 0, choiceMap)
	if err != nil {
		return nil, err
	}
	file := d.surveyAstToFile(root)
	settings := form.settingsToAst(d.importInfo)
	if settings != nil {
		file.Decls = append(file.Decls, settings)
	}
	return format.Node(file, format.Simplify())
}

type xlsForm struct {
	// contains all rows in the survey sheet
	surveyColumnHeaders []string
	survey              [][]string
	// contains all rows in the choices sheet
	choiceColumnHeaders []string
	choices             [][]string
	// contains all rows in the settings sheet
	settingColumnHeaders []string
	settings             [][]string
}

// parseXLSForm parses the xls file into an XLSForm struct
func parseXLSForm(r io.Reader) (*xlsForm, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()
	form := xlsForm{}
	if surveyRows, err := f.GetRows(surveySheetName); err != nil {
		return nil, fmt.Errorf("%v: %w", err, ErrInvalidXLSForm)
	} else {
		if err := validXLSFormSheet(surveySheetName, surveyRows); err != nil {
			return nil, err
		}
		form.surveyColumnHeaders = surveyRows[0]
		if len(surveyRows) > 1 {
			form.survey = surveyRows[1:]
		}
	}
	if choiceRows, err := f.GetRows(choiceSheetName); err != nil {
		return nil, fmt.Errorf("%v: %w", err, ErrInvalidXLSForm)
	} else {
		if err := validXLSFormSheet(choiceSheetName, choiceRows); err != nil {
			return nil, err
		}
		form.choiceColumnHeaders = choiceRows[0]
		if len(choiceRows) > 1 {
			form.choices = choiceRows[1:]
		}
	}
	if settingsRows, err := f.GetRows(settingsSheetName); err != nil {
		if !errors.Is(err, excelize.ErrSheetNotExist{SheetName: settingsSheetName}) {
			return nil, err
		}
		// settings is not required
		log.Println(err)
	} else {
		form.settingColumnHeaders = settingsRows[0]
		if len(settingsRows) > 1 {
			form.settings = settingsRows[1:]
		}
	}
	return &form, nil
}

// validXLSFormSheet validates that the work sheet has the required columns
func validXLSFormSheet(sheet string, rows [][]string) error {
	if len(rows) <= 0 {
		log.Printf("%s is empty", sheet)
		return ErrInvalidXLSForm
	}
	columnHeaders := rows[0]
	if sheet == surveySheetName {
		for _, requiredCol := range requiredSurveySheetColumns {
			match := slices.ContainsFunc(columnHeaders, func(s string) bool {
				_, found := strings.CutPrefix(s, requiredCol)
				return found
			})
			if !match {
				fmt.Println("no match")
				return ErrInvalidXLSFormSheet
			}
		}
	} else if sheet == choiceSheetName {
		for _, requiredCol := range requiredChoiceSheetColumns {
			match := slices.ContainsFunc(columnHeaders, func(s string) bool {
				_, found := strings.CutPrefix(s, requiredCol)
				return found
			})
			if !match {
				fmt.Println("no match")
				return ErrInvalidXLSFormSheet
			}
		}
	}
	return nil
}

// choicesToAst converts rows from the choice sheet to CUE expressions
func (form *xlsForm) choicesToAst(importInfo astutil.ImportInfo) (map[string]ast.Expr, error) {
	if len(form.choices) == 0 {
		return nil, nil
	}
	choiceAsts := make(map[string]ast.Expr)
	for choiceKey, rows := range extractChoices(form.choiceColumnHeaders, form.choices) {
		choiceStruct, err := buildChoiceStruct(choiceKey, form.choiceColumnHeaders, rows)
		if err != nil {
			return nil, err
		}
		choiceAsts[choiceKey] = newConjuctionOnNewLine(importInfo, "Choices", choiceStruct, false)
	}
	return choiceAsts, nil
}

// extractChoices transform the choice sheet rows to a map with the key being the choice list_name and
// the value being an array of all rows specific to the choice with the list_name
func extractChoices(columns []string, rows [][]string) map[string][][]string {
	listNameIdx := indexOf(columns, "list_name")
	choices := make(map[string][][]string)
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		choice := choices[row[listNameIdx]]
		if choice == nil {
			choice = [][]string{}
		}
		choice = append(choice, row)
		choices[row[listNameIdx]] = choice
	}
	return choices
}

// buildChoiceStruct builds a CUE struct from rows describing a choice
func buildChoiceStruct(choiceListName string, columns []string, rows [][]string) (*ast.StructLit, error) {
	entries := &ast.ListLit{Rbrack: token.Newline.Pos()}
	choice := ast.NewStruct(&ast.Field{Label: ast.NewIdent("list_name"), Value: ast.NewString(choiceListName)}, &ast.Field{Label: ast.NewIdent("choices"), Value: entries})
	for _, row := range rows {
		choiceEntry := &ast.Field{}
		for idx, colVal := range row {
			if columns[idx] == "name" {
				choiceEntry.Label = ast.NewIdent(colVal)
			} else if columns[idx] == "label" {
				return nil, ErrInvalidLabel
			} else if strings.HasPrefix(columns[idx], "label::") {
				if choiceEntry.Value == nil {
					choiceEntry.Value = ast.NewStruct()
				}
				label := &ast.Field{Label: &ast.Ident{Name: strings.TrimPrefix(columns[idx], "label::"), NamePos: token.Newline.Pos()}, Value: ast.NewString(colVal)}
				choiceEntry.Value.(*ast.StructLit).Elts = append(choiceEntry.Value.(*ast.StructLit).Elts, label)
			}
		}
		entry := ast.NewStruct(choiceEntry)
		entry.Lbrace = token.Newline.Pos()
		entries.Elts = append(entries.Elts, entry)
	}
	return choice, nil
}

// surveyToAst converts survey rows to valid survey exprs. We use the passed in struct n as the root level node which holds all the top level elements in the survey sheet
func (form *xlsForm) surveyToAst(importInfo astutil.ImportInfo, n *ast.StructLit, idx int, choiceMap map[string]ast.Expr) (int, error) {
	elList := &ast.ListLit{Rbrack: token.Newline.Pos()}
	n.Elts = append(n.Elts, &ast.Field{Label: ast.NewIdent("children"), Value: elList})
	for {
		if idx > len(form.survey)-1 {
			return idx, nil
		}
		row := form.survey[idx]
		if len(row) == 0 {
			// skip empty rows
			idx++
			continue
		}
		idx++
		elementType := row[indexOf(form.surveyColumnHeaders, "type")]
		if strings.HasPrefix(elementType, "begin") {
			group, err := buildSurveyElement(true, form.surveyColumnHeaders, row, choiceMap)
			if err != nil {
				return idx, err
			}
			idx, err = form.surveyToAst(importInfo, group, idx, choiceMap)
			if err != nil {
				return idx, err
			}
			elList.Elts = append(elList.Elts, newConjuction(importInfo, "Group", group))
		} else if strings.HasPrefix(elementType, "end") {
			return idx, nil
		} else {
			el, err := buildSurveyElement(false, form.surveyColumnHeaders, row, choiceMap)
			if err != nil {
				return idx, err
			}
			elList.Elts = append(elList.Elts, newConjuction(importInfo, "Question", el))
		}
	}
}

func buildSurveyElement(nl bool, columnHeaders []string, row []string, choiceMap map[string]ast.Expr) (*ast.StructLit, error) {
	element := ast.StructLit{}
	translatables := map[string]*ast.StructLit{}
	for idx, header := range columnHeaders {
		if idx >= len(row) || row[idx] == "" {
			continue
		}
		if header == "type" && strings.HasPrefix(row[idx], "select_") {
			raw := strings.SplitAfterN(row[idx], " ", 2)
			qtype, choice := strings.TrimSpace(raw[0]), strings.TrimSpace(raw[1])
			element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(qtype)}, &ast.Field{Label: ast.NewIdent("choices"), Value: choiceMap[choice]})
		} else if indexOf(translatableCols, header) != -1 {
			return nil, fmt.Errorf("missing lang code %s: %w", header, ErrInvalidLabel)
		} else if match := langRe.FindStringSubmatch(header); len(match) == 3 && indexOf(translatableCols, match[1]) != -1 {
			match := langRe.FindStringSubmatch(header)
			col, lang := match[1], match[2]
			if translatables[col] == nil {
				labels := ast.NewStruct()
				element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(col), Value: labels})
				translatables[col] = labels
			}
			translatables[col].Elts = append(translatables[col].Elts, &ast.Field{Label: &ast.Ident{Name: lang, NamePos: token.Newline.Pos()}, Value: ast.NewString(row[idx])})
		} else {
			element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(row[idx])})
		}
	}
	return &element, nil
}

// surveyAstToFile converts the root survey struct to a proper CUE file
func (d *Decoder) surveyAstToFile(root *ast.StructLit) *ast.File {
	file := &ast.File{}
	file.Decls = append(file.Decls, &ast.Package{Name: ast.NewIdent("main")})
	file.Decls = append(file.Decls, &ast.ImportDecl{Specs: []*ast.ImportSpec{d.importSpec}})
	topLevelElements := root.Elts[0].(*ast.Field).Value.(*ast.ListLit)
	for _, c := range topLevelElements.Elts {
		v := c.(*ast.BinaryExpr)
		if len(v.Y.(*ast.StructLit).Elts) <= 1 {
			continue
		}
		var nameField *ast.Field
		for _, el := range v.Y.(*ast.StructLit).Elts {
			switch v := el.(type) {
			case *ast.Field:
				if name, _, _ := ast.LabelName(v.Label); name == "name" {
					nameField = v
					break
				}
			}
		}
		nameValue := nameField.Value.(*ast.BasicLit)
		file.Decls = append(file.Decls, &ast.Field{Label: nameValue, Value: v})

	}
	return file
}

func (form *xlsForm) settingsToAst(importInfo astutil.ImportInfo) *ast.Field {
	if len(form.settings) != 1 {
		return nil
	}
	settings := ast.NewStruct(&ast.Field{Label: ast.NewIdent("type"), Value: ast.NewString("settings")})
	for idx, header := range form.settingColumnHeaders {
		settings.Elts = append(settings.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(form.settings[0][idx])})
	}
	return &ast.Field{Label: ast.NewIdent("form_settings"), Value: newConjuction(importInfo, "Settings", settings)}
}

func newConjuctionOnNewLine(info astutil.ImportInfo, def string, sl ast.Expr, newLine bool) ast.Expr {
	i := &ast.Ident{Name: info.Ident}
	if newLine {
		i.NamePos = token.Newline.Pos()
	}
	return ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: i, Sel: ast.NewIdent(fmt.Sprintf("#%s", def))}, sl)
}

func newConjuction(info astutil.ImportInfo, def string, sl ast.Expr) ast.Expr {
	return newConjuctionOnNewLine(info, def, sl, true)
}
