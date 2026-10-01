package xlsform

import (
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// excelize's GetRows returns the text Excel displays, but pyxform reads number cells by their
// stored value: a 3 shown as "3.00" is "3" to pyxform. cellReader reads numbers and dates the
// way pyxform 4.5.0 does. cells_test.go checks this against pyxform's own reading of fixtures.

var (
	excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

	// built-in number format ids that openpyxl treats as dates or times
	builtinDateFormats = map[int]bool{14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true, 45: true, 46: true, 47: true}

	// quoted text and [...] sections such as [Red] or [$-409], which are not date parts
	numFmtLiteralRe = regexp.MustCompile(`".*?"|\[[^\]]*\]`)
)

type cellReader struct {
	f          *excelize.File
	dateStyles map[int]bool
}

func newCellReader(f *excelize.File) *cellReader {
	return &cellReader{f: f, dateStyles: map[int]bool{}}
}

// rows returns the cell text of every row in sheet as pyxform reads it
func (r *cellReader) rows(sheet string) ([][]string, error) {
	shown, err := r.f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	raw, err := r.f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	for rowIdx, row := range raw {
		for colIdx, value := range row {
			if value == "" {
				continue
			}
			if rowIdx < len(shown) && colIdx < len(shown[rowIdx]) {
				row[colIdx] = shown[rowIdx][colIdx]
			}
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			if err != nil {
				return nil, err
			}
			cellType, err := r.f.GetCellType(sheet, cell)
			if err != nil {
				return nil, err
			}
			// text and bool cells read as they display; an unset type is a number
			if cellType != excelize.CellTypeNumber && cellType != excelize.CellTypeUnset {
				continue
			}
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			isDate, err := r.isDateCell(sheet, cell)
			if err != nil {
				return nil, err
			}
			if !isDate {
				row[colIdx] = strconv.FormatFloat(number, 'f', -1, 64)
				continue
			}
			row[colIdx] = dateText(number)
			// XLSForm values are text, so a date cell is usually text Excel converted, e.g. a hint
			// typed as 1/2. pyxform silently writes it as a Python date, and so do we, with a warning.
			header := ""
			if colIdx < len(raw[0]) {
				header = raw[0][colIdx]
			}
			log.Printf("warning: %s!%s (%s): Excel stored this as a date, which pyxform reads as %q; format the cell as text if that isn't what you meant", sheet, cell, header, row[colIdx])
		}
	}
	return raw, nil
}

func (r *cellReader) isDateCell(sheet, cell string) (bool, error) {
	styleID, err := r.f.GetCellStyle(sheet, cell)
	if err != nil {
		return false, err
	}
	if isDate, ok := r.dateStyles[styleID]; ok {
		return isDate, nil
	}
	style, err := r.f.GetStyle(styleID)
	if err != nil {
		return false, err
	}
	isDate := builtinDateFormats[style.NumFmt]
	if style.CustomNumFmt != nil {
		isDate = isDateFormat(*style.CustomNumFmt)
	}
	r.dateStyles[styleID] = isDate
	return isDate, nil
}

// dateText is how pyxform writes a date cell: "2024-01-15 00:00:00", or "13:30:00" for a time on
// its own
func dateText(serial float64) string {
	day := math.Floor(serial)
	seconds := math.Round((serial - day) * 24 * 60 * 60)
	t := excelEpoch.AddDate(0, 0, int(day)).Add(time.Duration(seconds) * time.Second)
	if day == 0 {
		return t.Format("15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}

// isDateFormat reports whether a number format shows a date or time, as openpyxl decides it:
// any d, m, h, y or s in the format's first section, outside quoted text and [...] sections
func isDateFormat(format string) bool {
	format, _, _ = strings.Cut(format, ";")
	return strings.ContainsAny(numFmtLiteralRe.ReplaceAllString(format, ""), "dmhysDMHYS")
}
