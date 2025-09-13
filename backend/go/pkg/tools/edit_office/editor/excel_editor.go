package editor

import (
	"github.com/unidoc/unioffice/v2/measurement"
	"time"

	"github.com/unidoc/unioffice/v2/schema/soo/sml"
	"github.com/unidoc/unioffice/v2/spreadsheet"
	"github.com/unidoc/unioffice/v2/spreadsheet/reference"
)

// ExcelWorkbook wraps a unioffice spreadsheet workbook.
type ExcelWorkbook struct {
	wb *spreadsheet.Workbook
}

// Sheet wraps a unioffice sheet.
type Sheet struct {
	s spreadsheet.Sheet
}

// ExcelRow wraps a unioffice row.
type ExcelRow struct {
	r spreadsheet.Row
}

// ExcelCell wraps a unioffice cell.
type ExcelCell struct {
	c spreadsheet.Cell
}

// CellStyle wraps a unioffice cell style.
type CellStyle struct {
	cs spreadsheet.CellStyle
}

// Aliases for spreadsheet types
type (
	ST_HorizontalAlignment = sml.ST_HorizontalAlignment
	ST_VerticalAlignment   = sml.ST_VerticalAlignment
)

// --- 1. File Operations ---

// NewExcelWorkbook creates a new blank Excel workbook.
func NewExcelWorkbook() *ExcelWorkbook {
	return &ExcelWorkbook{wb: spreadsheet.New()}
}

// OpenExcelWorkbook opens an existing .xlsx file.
func OpenExcelWorkbook(path string) (*ExcelWorkbook, error) {
	wb, err := spreadsheet.Open(path)
	if err != nil {
		return nil, err
	}
	return &ExcelWorkbook{wb: wb}, nil
}

// SaveToFile saves the workbook to the specified path.
func (w *ExcelWorkbook) SaveToFile(path string) error {
	return w.wb.SaveToFile(path)
}

// --- 2. Sheet Management ---

// AddSheet adds a new sheet to the workbook.
func (w *ExcelWorkbook) AddSheet(name string) Sheet {
	sheet := w.wb.AddSheet()
	sheet.SetName(name)
	return Sheet{s: sheet}
}

// GetSheet returns a sheet by its name.
func (w *ExcelWorkbook) GetSheet(name string) (Sheet, error) {
	s, err := w.wb.GetSheet(name)
	if err != nil {
		return Sheet{}, err
	}
	return Sheet{s: s}, nil
}

// Sheets returns all sheets in the workbook.
func (w *ExcelWorkbook) Sheets() []Sheet {
	var sheets []Sheet
	for _, s := range w.wb.Sheets() {
		sheets = append(sheets, Sheet{s: s})
	}
	return sheets
}

// SetActiveSheet sets the active sheet tab.
func (w *ExcelWorkbook) SetActiveSheet(index int) {
	w.wb.SetActiveSheetIndex(uint32(index))
}

// --- 3. Row and Cell Operations ---

// Row returns a row by its 1-based index, creating it if it doesn't exist.
func (s Sheet) Row(rowNum int) ExcelRow {
	return ExcelRow{r: s.s.Row(uint32(rowNum))}
}

// Cell returns a cell by its name (e.g., "A1"), creating it if it doesn't exist.
func (s Sheet) Cell(cellRef string) ExcelCell {
	return ExcelCell{c: s.s.Cell(cellRef)}
}

// AddCell adds a new cell to the row.
func (r ExcelRow) AddCell() ExcelCell {
	return ExcelCell{c: r.r.AddCell()}
}

// SetString sets the cell's value to a string.
func (c ExcelCell) SetString(val string) {
	c.c.SetString(val)
}

// SetNumber sets the cell's value to a number.
func (c ExcelCell) SetNumber(val float64) {
	c.c.SetNumber(val)
}

// SetDate sets the cell's value to a date/time.
func (c ExcelCell) SetDate(t time.Time) {
	c.c.SetDate(t)
}

// SetFormula sets a formula for the cell.
func (c ExcelCell) SetFormula(formula string) {
	c.c.SetFormulaRaw(formula)
}

// GetString gets the string value of the cell.
func (c ExcelCell) GetString() string {
	return c.c.GetString()
}

// GetNumber gets the numeric value of the cell.
func (c ExcelCell) GetNumber() (float64, error) {
	return c.c.GetValueAsNumber()
}

// GetFormattedValue gets the displayed value of the cell.
func (c ExcelCell) GetFormattedValue() string {
	return c.c.GetFormattedValue()
}

// --- 4. Styling and Formatting ---

// AddCellStyle creates a new cell style.
func (w *ExcelWorkbook) AddCellStyle() CellStyle {
	return CellStyle{cs: w.wb.StyleSheet.AddCellStyle()}
}

// SetNumberFormat sets the number format for a style.
func (cs CellStyle) SetNumberFormat(format string) {
	cs.cs.SetNumberFormat(format)
}

// SetHorizontalAlignment sets the horizontal alignment for a style.
func (cs CellStyle) SetHorizontalAlignment(align ST_HorizontalAlignment) {
	cs.cs.SetHorizontalAlignment(align)
}

// SetVerticalAlignment sets the vertical alignment for a style.
func (cs CellStyle) SetVerticalAlignment(align ST_VerticalAlignment) {
	cs.cs.SetVerticalAlignment(align)
}

// SetStyle applies a style to a cell.
func (c ExcelCell) SetStyle(cs CellStyle) {
	c.c.SetStyle(cs.cs)
}

// --- 5. Advanced Operations ---

// MergeCells merges a range of cells.
func (s Sheet) MergeCells(fromRef, toRef string) {
	s.s.AddMergedCells(fromRef, toRef)
}

// SetFrozen freezes rows and/or columns.
func (s Sheet) SetFrozen(rows, cols int) {
	s.s.SetFrozen(rows > 0, cols > 0)
}

// SetColWidth sets the width of a column.
func (s Sheet) SetColWidth(colName string, width float64) error {
	colIdx := reference.ColumnToIndex(colName) // 只返回一个值
	col := s.s.Column(colIdx)
	col.SetWidth(measurement.Distance(width))
	return nil
}

// SetRowHeight sets the height of a row.
func (r ExcelRow) SetRowHeight(height float64) {
	r.r.SetHeight(measurement.Distance(height))
}
