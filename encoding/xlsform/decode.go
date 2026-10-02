package xlsform

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
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
	ErrInvalidLabel        = errors.New("found translatable column with no language code")
	ErrUnsupportedSheet    = errors.New("found a sheet cueform doesn't support")
	ErrUnsupportedType     = errors.New("found a question type cueform doesn't support")
	// pyxform adds the "other" choice to the shared list, so every question using it shows it
	ErrOrOther = errors.New(`or_other is not supported; add an "other" choice and a text question with relevant, as the XLSForm spec recommends`)

	surveySheetName   = "survey"
	choiceSheetName   = "choices"
	settingsSheetName = "settings"

	requiredSurveySheetColumns = []string{"type", "name", "label"}
	requiredChoiceSheetColumns = []string{"list_name", "name", "label"}

	// columns the schema types as string | bool
	boolColumns = []string{"required", "read_only"}
	// settings pyxform reads as yes/no flags (aliases.yes_no); auto_send and auto_delete are
	// copied into the XForm as written, so they stay strings
	settingBoolColumns = []string{"allow_choice_duplicates", "clean_text_values", "omit_instanceID", "client_editable", "add_none_option"}
	// matches pyxform's aliases.BINDING_CONVERSIONS (v4.5.0); any other value is an XPath expression
	boolValues = map[string]bool{
		"yes": true, "Yes": true, "YES": true, "true": true, "True": true, "TRUE": true,
		"no": false, "No": false, "NO": false, "false": false, "False": false, "FALSE": false,
	}
)

type Decoder struct {
	schemaPkg string
}

// NewDecoder returns a new decoder that uses pkg as the xlsform schema definition package
func NewDecoder(pkg string) *Decoder {
	return &Decoder{schemaPkg: pkg}
}

// UsePkg changes the package we import schema definitions from
// the package should ofcourse contain all the required element schema definitions
func (d *Decoder) UsePkg(schemaPkg string) {
	d.schemaPkg = schemaPkg
}

// Decode returns the CUE encoding of r
func (d *Decoder) Decode(r io.Reader) ([]byte, error) {
	form, err := parseXLSForm(r)
	if err != nil {
		return nil, err
	}
	file, err := form.toAstFile(ast.NewImport(nil, d.schemaPkg))
	if err != nil {
		return nil, err
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
	if err := checkSheets(f); err != nil {
		return nil, err
	}
	form := xlsForm{}
	cells := newCellReader(f)
	if surveyRows, err := sheetRows(f, cells, surveySheetName); err != nil {
		return nil, fmt.Errorf("%v: %w", err, ErrInvalidXLSForm)
	} else {
		if err := validXLSFormSheet(surveySheetName, surveyRows); err != nil {
			return nil, err
		}
		form.surveyColumnHeaders = surveyRows[0]
		if len(surveyRows) > 1 {
			form.survey = surveyRows[1:]
		}
		if ti := slices.Index(form.surveyColumnHeaders, "type"); ti >= 0 {
			for _, row := range form.survey {
				if ti < len(row) {
					row[ti] = canonicalSelectType(row[ti])
				}
			}
		}
	}
	if choiceRows, err := sheetRows(f, cells, choiceSheetName); err != nil {
		if !errors.Is(err, excelize.ErrSheetNotExist{SheetName: choiceSheetName}) {
			return nil, err
		}
		// choices is not required
		log.Println(err)

	} else {
		if err := validXLSFormSheet(choiceSheetName, choiceRows); err != nil {
			return nil, err
		}
		form.choiceColumnHeaders = choiceRows[0]
		for _, header := range form.choiceColumnHeaders {
			if base, _, _ := strings.Cut(header, "::"); base == "media" {
				log.Printf("warning: choices column %q is dropped: write media as image, audio or video columns", header)
			}
		}
		if len(choiceRows) > 1 {
			form.choices = choiceRows[1:]
		}
	}
	if settingsRows, err := sheetRows(f, cells, settingsSheetName); err != nil {
		if !errors.Is(err, excelize.ErrSheetNotExist{SheetName: settingsSheetName}) {
			return nil, err
		}
		// settings is not required
		log.Println(err)
	} else {
		if err := validXLSFormSheet(settingsSheetName, settingsRows); err != nil {
			return nil, err
		}
		form.settingColumnHeaders = settingsRows[0]
		if len(settingsRows) > 1 {
			form.settings = settingsRows[1:]
		}
	}
	return &form, nil
}

// checkSheets rejects or warns about sheets pyxform reads but cueform doesn't. Other sheets are
// ignored by pyxform too, so nothing is lost.
func checkSheets(f *excelize.File) error {
	for _, sheet := range f.GetSheetList() {
		switch strings.ToLower(sheet) {
		case "entities":
			return fmt.Errorf("%w: %s", ErrUnsupportedSheet, sheet)
		case "external_choices", "osm":
			log.Printf("warning: the %s sheet is dropped: it isn't supported", sheet)
		}
	}
	return nil
}

// sheetRows reads a sheet, found case-insensitively as pyxform does, with its header row renamed
// to cueform's column names
func sheetRows(f *excelize.File, cells *cellReader, sheet string) ([][]string, error) {
	name := sheet
	for _, s := range f.GetSheetList() {
		if strings.EqualFold(s, sheet) {
			name = s
			break
		}
	}
	rows, err := cells.rows(name)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		if rows[0], err = canonicalHeaders(sheet, rows[0]); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// validXLSFormSheet validates that the work sheet has the required columns
func validXLSFormSheet(sheet string, rows [][]string) error {
	if len(rows) <= 0 {
		log.Printf("%s is empty", sheet)
		return ErrInvalidXLSForm
	}
	required := map[string][]string{surveySheetName: requiredSurveySheetColumns, choiceSheetName: requiredChoiceSheetColumns}[sheet]
	for _, col := range required {
		if !slices.ContainsFunc(rows[0], func(header string) bool { return isColumn(header, col) }) {
			return fmt.Errorf("%w: the %s sheet has no %s column", ErrInvalidXLSFormSheet, sheet, col)
		}
	}
	return nil
}

// isColumn reports whether a header is the column col; a translatable column also counts with a
// ::lang suffix, as a translated form may have only label::en
func isColumn(header, col string) bool {
	if header == col {
		return true
	}
	base, _, translated := strings.Cut(header, "::")
	return translated && base == col && IsTranslatableColumn(col)
}

func (form *xlsForm) toAstFile(i *ast.ImportSpec) (*ast.File, error) {
	importInfo, err := astutil.ParseImportSpec(i)
	if err != nil {
		return nil, err
	}
	choiceMap, err := form.choicesToAst(importInfo)
	if err != nil {
		return nil, err
	}
	root := ast.NewStruct()
	_, err = form.surveyToAst(importInfo, root, 0, choiceMap, nil)
	if err != nil {
		return nil, err
	}
	decls := []ast.Decl{&ast.Package{Name: ast.NewIdent("main")}, &ast.ImportDecl{Specs: []*ast.ImportSpec{i}}}
	for _, c := range root.Elts[0].(*ast.Field).Value.(*ast.ListLit).Elts {
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
		decls = append(decls, &ast.Field{Label: nameValue, Value: v})
	}
	settings := form.settingsToAst(importInfo)
	if settings != nil {
		decls = append(decls, settings)
	}
	return &ast.File{Decls: decls}, nil
}

// choicesToAst converts rows from the choice sheet to CUE expressions
func (form *xlsForm) choicesToAst(importInfo astutil.ImportInfo) (map[string]ast.Expr, error) {
	if len(form.choices) == 0 {
		return nil, nil
	}
	choiceAsts := make(map[string]ast.Expr)
	for _, list := range extractChoices(form.choiceColumnHeaders, form.choices) {
		choiceStruct, err := buildChoiceStruct(list.name, form.choiceColumnHeaders, list.rows)
		if err != nil {
			return nil, err
		}
		choiceAsts[list.name] = newConjuctionOnNewLine(importInfo, "Choices", choiceStruct, false)
	}
	return choiceAsts, nil
}

type choiceRows struct {
	name string
	rows [][]string
}

// extractChoices groups the choice sheet rows by list_name, in the order the lists first appear
func extractChoices(columns []string, rows [][]string) []choiceRows {
	listNameIdx := slices.Index(columns, "list_name")
	lists := []choiceRows{}
	index := map[string]int{}
	for i, row := range rows {
		if !slices.ContainsFunc(row, func(c string) bool { return c != "" }) {
			continue
		}
		listName := ""
		if listNameIdx < len(row) {
			listName = row[listNameIdx]
		}
		if listName == "" {
			// pyxform skips the row too; the header is sheet row 1
			log.Printf("warning: choices row %d is dropped: it has no list_name", i+2)
			continue
		}
		if _, ok := index[listName]; !ok {
			index[listName] = len(lists)
			lists = append(lists, choiceRows{name: listName})
		}
		lists[index[listName]].rows = append(lists[index[listName]].rows, row)
	}
	return lists
}

// buildChoiceStruct builds a CUE struct from rows describing a choice
func buildChoiceStruct(choiceListName string, columns []string, rows [][]string) (*ast.StructLit, error) {
	entries := &ast.ListLit{Rbrack: token.Newline.Pos()}
	choice := ast.NewStruct(&ast.Field{Label: ast.NewIdent("list_name"), Value: ast.NewString(choiceListName)}, &ast.Field{Label: ast.NewIdent("choices"), Value: entries})
	translated := translatedColumns(columns)
	for _, row := range rows {
		choiceEntry := &ast.Field{}
		filters := ast.NewStruct()
		// the choice's label and media, by column, each a plain value or a {lang: text} struct
		texts := map[string]ast.Expr{}
		for idx, colVal := range row {
			col := columns[idx]
			base, _, _ := strings.Cut(col, "::")
			switch {
			case colVal != "" && isChoiceFilterColumn(col):
				filters.Elts = append(filters.Elts, &ast.Field{Label: ast.NewString(col), Value: ast.NewString(colVal)})
			case col == "name":
				choiceEntry.Label = ast.NewIdent(colVal)
			case base == "label" || (colVal != "" && slices.Contains(choiceTextColumns, base)):
				if col == base && !translated[base] {
					texts[base] = ast.NewString(colVal)
					continue
				}
				_, lang, err := columnLang(col, translated)
				if err != nil {
					return nil, err
				}
				if texts[base] == nil {
					texts[base] = ast.NewStruct()
				}
				lit := texts[base].(*ast.StructLit)
				lit.Elts = append(lit.Elts, &ast.Field{Label: &ast.Ident{Name: lang, NamePos: token.Newline.Pos()}, Value: ast.NewString(colVal)})
			}
		}
		if len(texts) == 1 && texts["label"] != nil {
			choiceEntry.Value = texts["label"]
		} else if len(texts) > 0 {
			// media makes the choice {label: ..., image: ...}
			details := ast.NewStruct()
			for _, col := range append([]string{"label"}, choiceTextColumns...) {
				if texts[col] != nil {
					details.Elts = append(details.Elts, &ast.Field{Label: ast.NewString(col), Value: texts[col]})
				}
			}
			choiceEntry.Value = details
		}
		entry := ast.NewStruct(choiceEntry)
		if len(filters.Elts) > 0 {
			entry.Elts = append(entry.Elts, &ast.Field{Label: ast.NewIdent("filterCategory"), Value: filters})
		}
		entry.Lbrace = token.Newline.Pos()
		entries.Elts = append(entries.Elts, entry)
	}
	return choice, nil
}

// groupRowRe matches pyxform's group rows: begin or end, then _ or a space, then group or repeat.
// An "end" question type, which records when the form was finished, doesn't match.
var groupRowRe = regexp.MustCompile(`^(begin|end)[ _](group|repeat)$`)

// pyxform's older group kinds, which cueform doesn't support
var legacyGroupRowRe = regexp.MustCompile(`^(begin|end)[ _](lgroup|loop|looped group)$`)

// openGroup is the group or repeat whose rows surveyToAst is reading
type openGroup struct {
	kind, name string
	row        int
}

// surveyToAst converts survey rows to valid survey exprs. We use the passed in struct n as the root level node which holds all the top level elements in the survey sheet.
// open is the group the rows belong to, or nil at the top level.
func (form *xlsForm) surveyToAst(importInfo astutil.ImportInfo, n *ast.StructLit, idx int, choiceMap map[string]ast.Expr, open *openGroup) (int, error) {
	elList := &ast.ListLit{Rbrack: token.Newline.Pos()}
	n.Elts = append(n.Elts, &ast.Field{Label: ast.NewIdent("children"), Value: elList})
	// names must be unique within their group, repeat or survey, as pyxform requires; at the top
	// level each name is also a CUE field, so a duplicate would merge two questions
	names := map[string]int{}
	for {
		if idx > len(form.survey)-1 {
			if open != nil {
				return idx, fmt.Errorf("%w: survey row %d begins %s %q, which has no end_%s row", ErrInvalidXLSForm, open.row, open.kind, open.name, open.kind)
			}
			return idx, nil
		}
		row := form.survey[idx]
		// the header is sheet row 1
		rowNumber := idx + 2
		idx++
		if !slices.ContainsFunc(row, func(c string) bool { return c != "" }) {
			continue
		}
		elementType := form.surveyCell(row, "type")
		if elementType == "" {
			return idx, fmt.Errorf("%w: survey row %d has no type", ErrInvalidXLSForm, rowNumber)
		}
		if qtype, _, _ := strings.Cut(elementType, " "); qtype == "select_one_external" {
			return idx, fmt.Errorf("%w: survey row %d: select_one_external", ErrUnsupportedType, rowNumber)
		}
		if legacyGroupRowRe.MatchString(elementType) {
			return idx, fmt.Errorf("%w: survey row %d: %q; use begin_group or begin_repeat", ErrUnsupportedType, rowNumber, elementType)
		}
		groupRow := groupRowRe.FindStringSubmatch(elementType)
		if groupRow != nil && groupRow[1] == "end" {
			if open == nil || open.kind != groupRow[2] {
				return idx, fmt.Errorf("%w: survey row %d: %q has no matching begin_%s", ErrInvalidXLSForm, rowNumber, elementType, groupRow[2])
			}
			return idx, nil
		}
		name := form.surveyCell(row, "name")
		if name == "" {
			return idx, fmt.Errorf("%w: survey row %d (%s) has no name", ErrInvalidXLSForm, rowNumber, elementType)
		}
		if first, ok := names[name]; ok {
			return idx, fmt.Errorf("%w: survey row %d: name %q is already used in row %d; names must be unique within their group, repeat or survey", ErrInvalidXLSForm, rowNumber, name, first)
		}
		names[name] = rowNumber
		if open == nil && name == "form_settings" && len(form.settings) > 0 {
			return idx, fmt.Errorf("%w: survey row %d: the name %q is taken by the settings sheet", ErrInvalidXLSForm, rowNumber, name)
		}
		if groupRow != nil {
			group, err := buildSurveyElement(true, form.surveyColumnHeaders, row, choiceMap)
			if err != nil {
				return idx, err
			}
			idx, err = form.surveyToAst(importInfo, group, idx, choiceMap, &openGroup{kind: groupRow[2], name: name, row: rowNumber})
			if err != nil {
				return idx, err
			}
			elList.Elts = append(elList.Elts, newConjuction(importInfo, "Group", group))
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
	translated := translatedColumns(columnHeaders)
	for idx, header := range columnHeaders {
		if idx >= len(row) || row[idx] == "" {
			continue
		}
		qtype, choice, hasList := strings.Cut(row[idx], " ")
		if header == "type" && hasList && (usesChoices(qtype) || isFromFile(qtype)) {
			list, suffix, _ := strings.Cut(strings.TrimSpace(choice), " ")
			if suffix != "" {
				return nil, fmt.Errorf("%w: type %q", ErrOrOther, row[idx])
			}
			if strings.HasPrefix(list, "${") {
				element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(qtype)}, &ast.Field{Label: ast.NewIdent("choices_from"), Value: ast.NewString(list)})
				continue
			}
			if isFromFile(qtype) {
				element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(qtype)}, &ast.Field{Label: ast.NewIdent("file"), Value: ast.NewString(list)})
				continue
			}
			choices, ok := choiceMap[list]
			if !ok {
				return nil, fmt.Errorf("%w: type %q uses choice list %q, which the choices sheet doesn't have", ErrInvalidXLSForm, row[idx], list)
			}
			element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(qtype)}, &ast.Field{Label: ast.NewIdent("choices"), Value: choices})
		} else if IsTranslatableColumn(header) && (strings.Contains(header, ":") || translated[header]) {
			col, lang, err := columnLang(header, translated)
			if err != nil {
				return nil, err
			}
			if translatables[col] == nil {
				labels := ast.NewStruct()
				element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(col), Value: labels})
				translatables[col] = labels
			}
			translatables[col].Elts = append(translatables[col].Elts, &ast.Field{Label: &ast.Ident{Name: lang, NamePos: token.Newline.Pos()}, Value: ast.NewString(row[idx])})
		} else if b, ok := boolValues[row[idx]]; ok && slices.Contains(boolColumns, header) {
			element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewBool(b)})
		} else {
			element.Elts = append(element.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(row[idx])})
		}
	}
	return &element, nil
}

// translatedColumns returns the translatable columns that have at least one ::lang header
func translatedColumns(headers []string) map[string]bool {
	translated := map[string]bool{}
	for _, header := range headers {
		if col, _, ok := strings.Cut(header, "::"); ok && IsTranslatableColumn(col) {
			translated[col] = true
		}
	}
	return translated
}

// columnLang splits a translatable header into its column and language. A plain column in a
// sheet that also has ::lang columns for it is the "default" language, as pyxform reads it.
func columnLang(header string, translated map[string]bool) (string, string, error) {
	if translated[header] {
		return header, "default", nil
	}
	return GetLangFromCol(header)
}

// surveyCell returns a survey row's value in a column, or "" when the row is shorter than the header
func (form *xlsForm) surveyCell(row []string, col string) string {
	if i := slices.Index(form.surveyColumnHeaders, col); i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}

// settingBoolValue is pyxform's aliases.yes_no: the survey sheet's yes/no spellings plus true()
// and false()
func settingBoolValue(value string) (bool, bool) {
	switch value {
	case "true()":
		return true, true
	case "false()":
		return false, true
	}
	b, ok := boolValues[value]
	return b, ok
}

func (form *xlsForm) settingsToAst(importInfo astutil.ImportInfo) *ast.Field {
	// pyxform uses the first settings row and ignores the rest
	var row []string
	for i, r := range form.settings {
		if !slices.ContainsFunc(r, func(c string) bool { return c != "" }) {
			continue
		}
		if row != nil {
			// the header is sheet row 1
			log.Printf("warning: settings row %d is ignored: only the first settings row is used", i+2)
			continue
		}
		row = r
	}
	if row == nil {
		return nil
	}
	settings := ast.NewStruct(&ast.Field{Label: ast.NewIdent("type"), Value: ast.NewString("settings")})
	for idx, header := range form.settingColumnHeaders {
		// a row shorter than the header leaves the last settings empty
		if idx >= len(row) || row[idx] == "" {
			continue
		}
		var value ast.Expr = ast.NewString(row[idx])
		if b, ok := settingBoolValue(row[idx]); ok && slices.Contains(settingBoolColumns, header) {
			value = ast.NewBool(b)
		}
		settings.Elts = append(settings.Elts, &ast.Field{Label: ast.NewIdent(header), Value: value})
	}
	return &ast.Field{Label: ast.NewIdent("form_settings"), Value: newConjuction(importInfo, "Settings", settings)}
}

func (form *xlsForm) WriteToBuffer() (*bytes.Buffer, error) {
	formFile := excelize.NewFile()
	defer func() {
		if err := formFile.Close(); err != nil {
			log.Println(err)
		}
	}()
	err := writeSheet(formFile, surveySheetName, form.surveyColumnHeaders, form.survey)
	if err != nil {
		return nil, err
	}
	if len(form.choices) > 0 {
		err = writeSheet(formFile, choiceSheetName, form.choiceColumnHeaders, form.choices)
		if err != nil {
			return nil, err
		}
	}
	if len(form.settings) > 0 {
		err = writeSheet(formFile, settingsSheetName, form.settingColumnHeaders, form.settings)
		if err != nil {
			return nil, err
		}
	}
	formFile.DeleteSheet("Sheet1")
	return formFile.WriteToBuffer()
}

func writeSheet(f *excelize.File, sheet string, headers []string, rows [][]string) error {
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	setDefaultColumnWidth(sheet, f)
	if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
		return err
	}
	for idx, row := range rows {
		err = f.SetSheetRow(sheet, fmt.Sprintf("A%d", idx+2), &row)
		if err != nil {
			return err
		}
	}
	// excelize leaves the dimension at A1, and openpyxl's read-only mode, which pyxform uses,
	// trusts it and reads only that cell
	lastCell, err := excelize.CoordinatesToCellName(len(headers), len(rows)+1)
	if err != nil {
		return err
	}
	return f.SetSheetDimension(sheet, "A1:"+lastCell)
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
