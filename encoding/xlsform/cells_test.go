package xlsform

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/freddieptf/cueform/schema"
	"github.com/xuri/excelize/v2"
)

// testdata/pyxform/cells.xlsx was written by openpyxl, and cells.json holds pyxform 4.5.0's
// reading of its survey "default" column. testdata/pyxform/cells.py regenerates both.
const pyxformWorkbook = "testdata/pyxform/cells.xlsx"

func pyxformReading(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("testdata/pyxform/cells.json")
	if err != nil {
		t.Fatal(err)
	}
	var reading []string
	if err := json.Unmarshal(b, &reading); err != nil {
		t.Fatal(err)
	}
	return reading
}

func TestCellsMatchPyxform(t *testing.T) {
	f, err := os.Open(pyxformWorkbook)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	form, err := parseXLSForm(f)
	if err != nil {
		t.Fatal(err)
	}
	want := pyxformReading(t)
	if len(form.survey) != len(want) {
		t.Fatalf("have %d rows, pyxform read %d", len(form.survey), len(want))
	}
	for i, row := range form.survey {
		if have := row[3]; have != want[i] {
			t.Errorf("row %d (%s): have %q, pyxform reads %q", i+2, row[2], have, want[i])
		}
	}
}

// TestRoundTripMatchesPyxform runs pyxform itself, so it is skipped unless CUEFORM_PYXFORM_PYTHON
// names a Python with pyxform==4.5.0 installed. It checks cells.json is still what pyxform reads,
// and that decoding to CUE and encoding again gives a workbook pyxform reads the same way.
func TestRoundTripMatchesPyxform(t *testing.T) {
	python := os.Getenv("CUEFORM_PYXFORM_PYTHON")
	if python == "" {
		t.Skip("set CUEFORM_PYXFORM_PYTHON to a Python with pyxform==4.5.0 to run")
	}
	pyxformRead := func(path string) []string {
		t.Helper()
		out, err := exec.Command(python, "testdata/pyxform/cells.py", "read", path).Output()
		if err != nil {
			t.Fatalf("pyxform read %s: %v", path, err)
		}
		var values []string
		if err := json.Unmarshal(out, &values); err != nil {
			t.Fatal(err)
		}
		return values
	}

	want := pyxformRead(pyxformWorkbook)
	if !reflect.DeepEqual(want, pyxformReading(t)) {
		t.Fatal("cells.json is out of date; rerun testdata/pyxform/cells.py write")
	}

	// a CUE module where the decoded form can import the schema
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "cue.mod", "pkg", "github.com", "freddieptf", "cueform", "xlsform")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		filepath.Join(dir, "cue.mod", "module.cue"): []byte("module: \"roundtrip.test\"\nlanguage: version: \"v0.17.1\"\n"),
		filepath.Join(pkgDir, "schema.cue"):         schema.XLSForm,
	}
	for path, content := range files {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := os.Open(pyxformWorkbook)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	src, err := NewDecoder("github.com/freddieptf/cueform/xlsform").Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	cuePath := filepath.Join(dir, "cells.cue")
	if err := os.WriteFile(cuePath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	buf, err := NewEncoder().Encode(cuePath)
	if err != nil {
		t.Fatal(err)
	}
	encoded := filepath.Join(dir, "cells.xlsx")
	if err := os.WriteFile(encoded, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if have := pyxformRead(encoded); !reflect.DeepEqual(have, want) {
		t.Fatalf("pyxform reads the re-encoded workbook differently\nhave %q\nwant %q", have, want)
	}
}

// expected values were produced by openpyxl 3.1.5's is_date_format
func TestDateFormats(t *testing.T) {
	testCases := []struct {
		format string
		want   bool
	}{
		{`General`, false},
		{`0.00`, false},
		{`#,##0.000`, false},
		{`0.00E+00`, false},
		{`0.00 "kg"`, false},
		{`#,##0.00_);[Red](#,##0.00)`, false},
		{`[$-409]0.00`, false},
		{`@`, false},
		{`yyyy-mm-dd`, true},
		{`dd/mm/yyyy hh:mm`, true},
		{`[$-409]d-mmm-yy`, true},
		{`h:mm AM/PM`, true},
	}
	for _, tc := range testCases {
		if have := isDateFormat(tc.format); have != tc.want {
			t.Errorf("isDateFormat(%q) = %v; want %v", tc.format, have, tc.want)
		}
	}
}

// excelize writes workbooks differently from openpyxl (its own style ids, numbers rounded to 15
// digits), so this checks the reader on one excelize built
func TestCellsFromExcelize(t *testing.T) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	custom := func(code string) *excelize.Style { return &excelize.Style{CustomNumFmt: &code} }
	cells := []struct {
		value any
		style *excelize.Style
		want  string
	}{
		{3, &excelize.Style{NumFmt: 2}, "3"},
		{0.5, &excelize.Style{NumFmt: 10}, "0.5"},
		{1234.5, custom("#,##0.000"), "1234.5"},
		{"3.00", nil, "3.00"},
		{true, nil, "TRUE"},
		// a locale format id excelize knows, which openpyxl does not treat as a date
		{45306, &excelize.Style{NumFmt: 27}, "45306"},
	}
	for i, c := range cells {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetCellValue(sheet, cell, c.value); err != nil {
			t.Fatal(err)
		}
		if c.style != nil {
			id, err := f.NewStyle(c.style)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellStyle(sheet, cell, cell, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := newCellReader(saved).rows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cells {
		if have := rows[i][0]; have != c.want {
			t.Errorf("row %d (%v): have %q, want %q", i+1, c.value, have, c.want)
		}
	}
}

// XLSForm values are text, so a date cell is usually text Excel converted (a hint typed as 1/2,
// a date default). It decodes as pyxform reads it, with a warning naming the cell.
func TestDateCells(t *testing.T) {
	workbook := func(sheet string, headers []string, value any, numFmt int) *bytes.Buffer {
		t.Helper()
		f := excelize.NewFile()
		if _, err := f.NewSheet("survey"); err != nil {
			t.Fatal(err)
		}
		f.SetSheetRow("survey", "A1", &[]string{"type", "name", "label::en"})
		f.SetSheetRow("survey", "A2", &[]string{"note", "intro", "Hello"})
		if sheet != "survey" {
			if _, err := f.NewSheet(sheet); err != nil {
				t.Fatal(err)
			}
		}
		f.SetSheetRow(sheet, "A1", &headers)
		cell, _ := excelize.CoordinatesToCellName(len(headers), 2)
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
		style, err := f.NewStyle(&excelize.Style{NumFmt: numFmt})
		if err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellStyle(sheet, cell, cell, style); err != nil {
			t.Fatal(err)
		}
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		return buf
	}
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	for _, tc := range []struct {
		name    string
		sheet   string
		headers []string
		value   any
		numFmt  int
		want    string
		warning string
	}{
		{"date default", "survey", []string{"type", "name", "label::en", "default"}, 45306, 14, "2024-01-15 00:00:00", "survey!D2 (default)"},
		{"time default", "survey", []string{"type", "name", "label::en", "default"}, 0.5625, 20, "13:30:00", "survey!D2 (default)"},
		{"hint typed as 1/2", "survey", []string{"type", "name", "label::en", "hint::en"}, 46024, 16, "2026-01-02 00:00:00", "survey!D2 (hint::en)"},
		{"version", "settings", []string{"form_title", "version"}, 45306, 14, "2024-01-15 00:00:00", "settings!B2 (version)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs.Reset()
			form, err := parseXLSForm(workbook(tc.sheet, tc.headers, tc.value, tc.numFmt))
			if err != nil {
				t.Fatal(err)
			}
			rows := form.survey
			if tc.sheet == "settings" {
				rows = form.settings
			}
			if have := rows[len(rows)-1][len(tc.headers)-1]; have != tc.want {
				t.Errorf("have %q, want %q", have, tc.want)
			}
			if !strings.Contains(logs.String(), "warning: "+tc.warning) {
				t.Errorf("no warning for %s in %q", tc.warning, logs.String())
			}
		})
	}
}
