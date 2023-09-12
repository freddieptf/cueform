package xlsform

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"github.com/xuri/excelize/v2"
)

func makeFormDir(parent, formName string) (string, error) {
	dir := filepath.Join(parent, formName)
	if err := os.Mkdir(dir, fs.ModePerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", err
	}
	return dir, nil
}

func buildChoiceField(columnHeader []string, choiceKey string, rows [][]string) (*ast.StructLit, error) {
	entries := ast.NewList()
	choice := ast.NewStruct(&ast.Field{Label: ast.NewIdent("list_name"), Value: ast.NewString(choiceKey)}, &ast.Field{Label: ast.NewIdent("choices"), Value: entries})
	for _, row := range rows {
		choiceEntry := &ast.Field{}
		for idx, colVal := range row {
			if columnHeader[idx] == "name" {
				choiceEntry.Label = ast.NewIdent(colVal)
			} else if strings.HasPrefix(columnHeader[idx], "label::") {
				if choiceEntry.Value == nil {
					choiceEntry.Value = ast.NewStruct()
				}
				choiceEntry.Value.(*ast.StructLit).Elts = append(choiceEntry.Value.(*ast.StructLit).Elts,
					&ast.Field{Label: ast.NewIdent(strings.TrimPrefix(columnHeader[idx], "label::")), Value: ast.NewString(colVal)})
			}
		}
		entries.Elts = append(entries.Elts, ast.NewStruct(choiceEntry))
	}
	return choice, nil
}

func buildChoiceFile(pkg string, rows [][]string) (*ast.File, error) {
	file := &ast.File{}
	file.Decls = append(file.Decls, &ast.Package{Name: ast.NewIdent(pkg)})

	schemaImportSpec := ast.NewImport(nil, "github.com/freddieptf/cueform/schema/xlsform")
	file.Decls = append(file.Decls, &ast.ImportDecl{Specs: []*ast.ImportSpec{schemaImportSpec}})
	info, err := astutil.ParseImportSpec(schemaImportSpec)
	if err != nil {
		log.Fatal(err)
	}
	schemaPkg := ast.NewIdent(info.Ident)

	columnHeaders := rows[0]
	listNameIdx := indexOf(columnHeaders, "list_name")
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

	for choiceKey, rows := range choices {
		choiceStruct, err := buildChoiceField(columnHeaders, choiceKey, rows)
		if err != nil {
			return nil, err
		}
		file.Decls = append(file.Decls, &ast.Field{
			Label: ast.NewIdent(choiceKey),
			Value: ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: schemaPkg, Sel: ast.NewIdent("#Choices")}, choiceStruct),
		})
	}
	return file, nil
}

type groupField struct {
	name string
	lit  *ast.StructLit
}

func buildGroupField(schemaPkg *ast.Ident, columnHeaders []string, total int, rows [][]string) (int, *groupField) {
	typeColumnIdx := indexOf(columnHeaders, "type")

	group := &ast.StructLit{}
	groupRow := rows[0]
	for idx, header := range columnHeaders {
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
		if row[typeColumnIdx] == "begin group" {
			fmt.Println("found nested", row[1])
			nTotal, nested := buildGroupField(schemaPkg, columnHeaders, total, rows[idx:])
			childrenList.Elts = append(childrenList.Elts, ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: schemaPkg, Sel: ast.NewIdent("#Group")}, nested.lit))
			idx += nTotal
			total = idx
		} else if row[typeColumnIdx] == "end group" {
			break
		} else {
			childStruct := ast.StructLit{}
			for idx, header := range columnHeaders {
				if idx >= len(row) || row[idx] == "" {
					continue
				}
				childStruct.Elts = append(childStruct.Elts, &ast.Field{Label: ast.NewIdent(header), Value: ast.NewString(row[idx])})
			}
			childrenList.Elts = append(childrenList.Elts, ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: schemaPkg, Sel: ast.NewIdent("#Question")}, &childStruct))
			total++
		}
	}
	return total, &groupField{name: groupRow[indexOf(columnHeaders, "name")], lit: group}
}

func buildGroupFields(schemaPkg *ast.Ident, columnHeaders []string, rows [][]string) []*groupField {
	typeColumnIdx := indexOf(columnHeaders, "type")
	idx, start := 0, -1
	groupTrackz := []string{}
	fields := []*groupField{}
	for {
		if idx >= len(rows) {
			break
		}
		row := rows[idx]
		if len(row) == 0 {
			idx++
			continue
		}
		if row[typeColumnIdx] == "begin group" {
			if start == -1 {
				start = idx
			}
			groupTrackz = append(groupTrackz, "begin group")
		} else if row[typeColumnIdx] == "end group" {
			if groupTrackz[len(groupTrackz)-1] == "begin group" {
				groupTrackz = groupTrackz[:len(groupTrackz)-1]
			}
			if len(groupTrackz) == 0 {
				fmt.Println("processing group:", rows[start][1])
				_, group := buildGroupField(schemaPkg, columnHeaders, 0, rows[start:idx])
				fields = append(fields, group)
				start = -1
			}
		} else {

		}
		idx++
	}
	return fields
}

func buildSurveyFile(pkg string, rows [][]string) (*ast.File, error) {
	file := &ast.File{}
	file.Decls = append(file.Decls, &ast.Package{Name: ast.NewIdent(pkg)})

	schemaImportSpec := ast.NewImport(nil, "github.com/freddieptf/cueform/schema/xlsform")
	file.Decls = append(file.Decls, &ast.ImportDecl{Specs: []*ast.ImportSpec{schemaImportSpec}})
	info, err := astutil.ParseImportSpec(schemaImportSpec)
	if err != nil {
		log.Fatal(err)
	}
	schemaPkg := ast.NewIdent(info.Ident)
	nlSchemaPkg := &ast.Ident{NamePos: token.Newline.Pos(), Name: info.Ident}

	columnHeaders := rows[0]
	for _, group := range buildGroupFields(nlSchemaPkg, columnHeaders, rows[1:]) {
		field := &ast.Field{
			Label: ast.NewIdent(group.name),
			Value: ast.NewBinExpr(token.AND, &ast.SelectorExpr{X: schemaPkg, Sel: ast.NewIdent("#Group")}, group.lit),
		}
		file.Decls = append(file.Decls, field)
	}

	return file, nil
}

func writeFile(path, name string, file *ast.File) error {
	out, err := format.Node(file, format.Simplify())
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(path, fmt.Sprintf("%s.cue", name)), out, fs.ModePerm)
}

func Decode(outDir, path string) error {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(err)
		}
	}()
	sheets := file.GetSheetList()
	for _, sheet := range sheets {
		switch sheet {
		case "survey":
			fallthrough
		case "choices":
			fallthrough
		case "settings":
			fmt.Println("found", sheet)
		default:
			fmt.Println("unexpected sheet", sheet)
		}
	}
	parentDir, err := makeFormDir(outDir, strings.TrimSuffix(filepath.Base(path), ".xlsx"))
	if err != nil {
		return err
	}
	choiceRows, err := file.GetRows("choices")
	if err != nil {
		return err
	}
	choiceFile, err := buildChoiceFile(filepath.Base(parentDir), choiceRows)
	if err != nil {
		return err
	}
	err = writeFile(parentDir, "choices", choiceFile)
	if err != nil {
		return err
	}

	surveyRows, err := file.GetRows("survey")
	if err != nil {
		return err
	}

	surveyFile, err := buildSurveyFile(filepath.Base(parentDir), surveyRows)
	if err != nil {
		return err
	}
	err = writeFile(parentDir, "survey", surveyFile)
	if err != nil {
		return err
	}

	return nil
}
