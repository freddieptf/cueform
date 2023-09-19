package xlsform

import (
	"fmt"
	"io"
	"log"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"github.com/xuri/excelize/v2"
)

type Decoder struct {
	r io.Reader
	s state
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	decoder := &Decoder{r: r, s: state{}}
	decoder.UseSchema("github.com/freddieptf/cueform", "schema/xlsform")
	return decoder, nil
}

// experimental still, but at some point we'll need to use other schemas as long as they have definitions for Group,Choices and Question
func (d *Decoder) UseSchema(module, pkg string) error {
	if pkg == "" {
		d.s.schemaImportInfo = nil
		return nil
	}
	d.s.module, d.s.pkg = module, pkg
	importInfo, err := astutil.ParseImportSpec(ast.NewImport(nil, fmt.Sprintf("%s/%s", d.s.module, d.s.pkg)))
	if err != nil {
		return err
	}
	d.s.schemaImportInfo = &importInfo
	return nil
}

func (d *Decoder) Decode() ([]byte, error) {
	file, err := excelize.OpenReader(d.r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(err)
		}
	}()
	sheets := file.GetSheetList()
	for _, sheet := range []string{"survey", "choices", "settings"} {
		if indexOf(sheets, sheet) == -1 {
			return nil, fmt.Errorf("missing sheet %s", sheet)
		}
	}
	if choiceRows, err := file.GetRows("choices"); err != nil {
		return nil, err
	} else {
		if err := d.s.readChoiceSheet(choiceRows); err != nil {
			return nil, err
		}
	}
	var (
		surveyRows  [][]string
		surveyBytes []byte
	)
	if surveyRows, err = file.GetRows("survey"); err != nil {
		return nil, err
	}
	if fields, err := d.s.readSurveySheet(surveyRows); err != nil {
		return nil, err
	} else {
		surveyBytes, err = d.s.getFileBytesFromNamedExpr(fields)
		if err != nil {
			return nil, err
		}
	}
	return surveyBytes, nil
}

//state, nice
type state struct {
	module, pkg         string
	schemaImportInfo    *astutil.ImportInfo
	surveyColumnHeaders []string
	choiceColumnHeaders []string
	choices             map[string]ast.Expr
	nameColumnIdx       int
	typeColumnIdx       int
}

// builds a complete choice struct, requires only the rows describing one choice
func (s *state) buildChoiceField(choiceKey string, rows [][]string) (*ast.StructLit, error) {
	entries := &ast.ListLit{Rbrack: token.Newline.Pos()}
	choice := ast.NewStruct(&ast.Field{Label: ast.NewIdent("list_name"), Value: ast.NewString(choiceKey)}, &ast.Field{Label: ast.NewIdent("choices"), Value: entries})
	for _, row := range rows {
		choiceEntry := &ast.Field{}
		for idx, colVal := range row {
			if s.choiceColumnHeaders[idx] == "name" {
				choiceEntry.Label = ast.NewIdent(colVal)
			} else if strings.HasPrefix(s.choiceColumnHeaders[idx], "label::") {
				if choiceEntry.Value == nil {
					choiceEntry.Value = ast.NewStruct()
				}
				choiceEntry.Value.(*ast.StructLit).Elts = append(choiceEntry.Value.(*ast.StructLit).Elts,
					&ast.Field{Label: ast.NewIdent(strings.TrimPrefix(s.choiceColumnHeaders[idx], "label::")), Value: ast.NewString(colVal)})
			}
		}
		entry := ast.NewStruct(choiceEntry)
		entry.Lbrace = token.Newline.Pos()
		entries.Elts = append(entries.Elts, entry)
	}
	return choice, nil
}

// we return choices in the format map[list_name]array of rows specific to list_name
func (s *state) buildChoiceMap(rows [][]string) map[string][][]string {
	listNameIdx := indexOf(s.choiceColumnHeaders, "list_name")
	choices := make(map[string][][]string)
	for _, row := range rows[1:] {
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

func (s *state) newConjuctionOnNewLine(def string, sl ast.Expr, newLine bool) ast.Expr {
	if s.schemaImportInfo != nil {
		i := &ast.Ident{Name: s.schemaImportInfo.Ident}
		if newLine {
			i.NamePos = token.Newline.Pos()
		}
		return ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: i, Sel: ast.NewIdent(fmt.Sprintf("#%s", def))}, sl)
	} else {
		i := &ast.Ident{Name: fmt.Sprintf("#%s", def)}
		if newLine {
			i.NamePos = token.Newline.Pos()
		}
		return ast.NewBinExpr(token.AND, i, sl)
	}
}

func (s *state) newConjuction(def string, sl ast.Expr) ast.Expr {
	return s.newConjuctionOnNewLine(def, sl, true)
}

func (s *state) readChoiceSheet(rows [][]string) error {
	if len(rows) == 0 { // choices can be empty
		return nil
	}
	s.choiceColumnHeaders = rows[0]
	s.choices = make(map[string]ast.Expr)
	for choiceKey, rows := range s.buildChoiceMap(rows) {
		choiceStruct, err := s.buildChoiceField(choiceKey, rows)
		if err != nil {
			return err
		}
		s.choices[choiceKey] = s.newConjuctionOnNewLine("Choices", choiceStruct, false)
	}
	return nil
}

func (s *state) buildQuestionStruct(nl bool, row []string) ast.Expr {
	question := ast.StructLit{}
	for idx, header := range s.surveyColumnHeaders {
		if idx >= len(row) || row[idx] == "" {
			continue
		}
		if header == "type" && strings.HasPrefix(row[idx], "select_") {
			raw := strings.SplitAfterN(row[idx], " ", 2)
			qtype, choice := strings.TrimSpace(raw[0]), strings.TrimSpace(raw[1])
			question.Elts = append(question.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(qtype)}, &ast.Field{Label: ast.NewIdent("choices"), Value: s.choices[choice]})
		} else {
			question.Elts = append(question.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(row[idx])})
		}
	}
	return &question
}

type namedExpr struct {
	name string
	expr ast.Expr
}

// a group within a group within a group within a group
func (s *state) buildGroupField(total int, rows [][]string) (int, *namedExpr) {
	group := &ast.StructLit{}
	groupRow := rows[0]
	for idx, header := range s.surveyColumnHeaders {
		if idx >= len(groupRow) || groupRow[idx] == "" {
			continue
		}
		group.Elts = append(group.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(groupRow[idx])})
	}
	childrenList := &ast.ListLit{Rbrack: token.Newline.Pos()}
	group.Elts = append(group.Elts, &ast.Field{Label: ast.NewIdent("children"), Value: childrenList})
	for idx := 1; idx < len(rows); idx++ {
		row := rows[idx]
		if len(row) == 0 {
			continue
		}
		if strings.HasPrefix(row[s.typeColumnIdx], "begin ") {
			nTotal, nested := s.buildGroupField(total, rows[idx:])
			childrenList.Elts = append(childrenList.Elts, s.newConjuction("Group", nested.expr))
			idx += nTotal
			total = idx
		} else if strings.HasPrefix(row[s.typeColumnIdx], "end ") {
			break
		} else {
			childrenList.Elts = append(childrenList.Elts, s.newConjuction("Question", s.buildQuestionStruct(true, row)))
			total++
		}
	}
	return total, &namedExpr{
		name: groupRow[s.nameColumnIdx],
		expr: group,
	}
}

func (s *state) readSurveySheet(rows [][]string) ([]*namedExpr, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("found empty survey sheet")
	}
	s.surveyColumnHeaders = rows[0]
	s.typeColumnIdx = indexOf(s.surveyColumnHeaders, "type")
	s.nameColumnIdx = indexOf(s.surveyColumnHeaders, "name")
	var (
		idx, start  = 0, -1
		groupTrackz = []string{}
		fields      = []*namedExpr{}
	)
	rows = rows[1:]
	for {
		if idx >= len(rows) {
			break
		}
		row := rows[idx]
		if len(row) == 0 {
			idx++
			continue
		}
		if strings.HasPrefix(row[s.typeColumnIdx], "begin ") {
			if start == -1 {
				start = idx
			}
			groupTrackz = append(groupTrackz, row[s.typeColumnIdx])
		} else if strings.HasPrefix(row[s.typeColumnIdx], "end ") {
			if strings.HasPrefix(groupTrackz[len(groupTrackz)-1], "begin ") {
				groupTrackz = groupTrackz[:len(groupTrackz)-1]
			}
			if len(groupTrackz) == 0 {
				_, group := s.buildGroupField(0, rows[start:idx])
				fields = append(fields, &namedExpr{group.name, s.newConjuctionOnNewLine("Group", group.expr, false)})
				start = -1
			}
		} else {
			if start == -1 && len(row) > 0 {
				// found rows not in a group, assume they are questions
				fields = append(fields, &namedExpr{name: row[s.nameColumnIdx], expr: s.newConjuctionOnNewLine("Question", s.buildQuestionStruct(false, row), false)})
			}
		}
		idx++
	}
	return fields, nil
}

func (s *state) getFileBytesFromNamedExpr(fields []*namedExpr) ([]byte, error) {
	file := &ast.File{}
	file.Decls = append(file.Decls, &ast.Package{Name: ast.NewIdent("main")})
	if s.schemaImportInfo != nil {
		file.Decls = append(file.Decls, &ast.ImportDecl{Specs: []*ast.ImportSpec{ast.NewImport(nil, fmt.Sprintf("%s/%s", s.module, s.pkg))}})
	}
	for _, field := range fields {
		file.Decls = append(file.Decls, &ast.Field{Label: ast.NewIdent(field.name), Value: field.expr})
	}
	return format.Node(file, format.Simplify())
}
