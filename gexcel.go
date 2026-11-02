// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package gexcel reads and writes Microsoft Excel workbooks (.xlsx, .xlsm,
// .xltx) with no external dependencies.
//
// The package exposes its API through interfaces (Workbook, Sheet, Row,
// Cell) so that callers can mock it and so that the implementation can
// evolve without breaking users. Everything is backed by the plain data
// structures in package model, which remain reachable through the Model()
// methods for advanced use.
//
//	wb := gexcel.New()
//	sh, _ := wb.AddSheet("Report")
//	sh.Cell("A1").Set("Region").SetStyle(style.Style{Font: style.Font{Bold: true}})
//	sh.Cell("B2").Set(1234.5).SetNumFmt("#,##0.00")
//	sh.Cell("B3").SetFormula("SUM(B2:B2)")
//	err := wb.SaveFile("report.xlsx")
package gexcel

import (
	"errors"
	"io"
	"iter"
	"time"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/reader"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/gexcel/writer"
)

// Errors returned by the package.
var (
	ErrSheetExists    = errors.New("gexcel: sheet already exists")
	ErrSheetNotFound  = errors.New("gexcel: sheet not found")
	ErrInvalidName    = errors.New("gexcel: invalid sheet name")
	ErrInvalidRef     = errors.New("gexcel: invalid cell reference")
	ErrLastSheet      = errors.New("gexcel: a workbook needs at least one sheet")
	ErrUnsupportedVal = errors.New("gexcel: unsupported value type")
)

// Workbook is an open spreadsheet document.
type Workbook interface {
	// Sheets lists the sheets in tab order.
	Sheets() []Sheet
	// Sheet finds a sheet by name (case-insensitive, as Excel does).
	Sheet(name string) (Sheet, bool)
	// SheetAt returns the sheet at a 0-based position, or nil.
	SheetAt(i int) Sheet
	// AddSheet appends a worksheet.
	AddSheet(name string) (Sheet, error)
	// RemoveSheet deletes a sheet by name.
	RemoveSheet(name string) error
	// MoveSheet changes a sheet's tab position.
	MoveSheet(name string, to int) error
	// CopySheet duplicates a sheet (cells, styles, settings) under a new name.
	CopySheet(from, to string) (Sheet, error)
	// SetActiveSheet picks the tab shown when the file opens.
	SetActiveSheet(name string) error

	// Styles is the shared style registry; Cell.SetStyle uses it for you.
	Styles() *model.Styles
	// Theme is the document theme colors and fonts.
	Theme() *style.Theme
	// Properties are the core document properties (title, author, dates).
	Properties() *model.Properties

	// DefinedNames lists workbook and sheet-scoped names.
	DefinedNames() []model.DefinedName
	// SetDefinedName adds or replaces a workbook-scoped name.
	SetDefinedName(name, refersTo string)
	// SetSheetDefinedName adds or replaces a name scoped to a sheet.
	SetSheetDefinedName(sheet, name, refersTo string) error
	// RemoveDefinedName deletes a workbook-scoped name.
	RemoveDefinedName(name string)

	// Date1904 reports the date system in use.
	Date1904() bool
	// SetFullCalcOnLoad asks Excel to recalculate every formula when it opens the file.
	SetFullCalcOnLoad(bool)

	// Calculate evaluates every formula with the built-in engine and stores
	// the results as cached values. It returns formula.ErrCycle for
	// circular references; other errors become Excel error values.
	Calculate() error

	// Save writes the workbook as .xlsx.
	Save(w io.Writer, opts ...writer.Options) error
	// SaveFile writes the workbook to a file path.
	SaveFile(name string, opts ...writer.Options) error
	// Validate reports problems that would make Excel repair the file.
	Validate() error

	// Model exposes the underlying data for advanced manipulation.
	Model() *model.Workbook
}

// Sheet is one worksheet.
type Sheet interface {
	Name() string
	SetName(name string) error
	// Index is the 0-based tab position.
	Index() int
	// Hidden hides or shows the tab.
	SetHidden(bool)
	Hidden() bool
	SetTabColor(style.Color)

	// Cell returns the cell at an A1 reference ("B3"), creating it if needed.
	// An invalid reference yields a detached cell whose Err is set.
	Cell(ref string) Cell
	// CellAt returns the cell at 1-based row and column.
	CellAt(row, col int) Cell
	// Lookup returns the cell only if it exists.
	Lookup(ref string) (Cell, bool)
	// Row returns a row (creating it if needed).
	Row(index int) Row
	// Rows iterates existing rows in order.
	Rows() iter.Seq[Row]
	// Cells iterates every existing cell row by row.
	Cells() iter.Seq[Cell]
	// Range iterates every cell of an A1 range (creating missing cells).
	Range(rng string) iter.Seq[Cell]
	// UsedRange is the bounding range of non-empty cells.
	UsedRange() (ref.Range, bool)
	// MaxRow and MaxCol are the last used row and column (0 when empty).
	MaxRow() int
	MaxCol() int

	// Set writes a value at an A1 reference; see Cell.Set for accepted types.
	Set(ref string, v any) Cell
	// SetRow writes consecutive values starting at an A1 reference.
	SetRow(ref string, values ...any) error
	// SetRows writes a 2-D block starting at an A1 reference.
	SetRows(ref string, rows [][]any) error
	// Value reads a cell's typed value; an empty Value for missing cells.
	Value(ref string) model.Value
	// AppendRow adds values after the last used row.
	AppendRow(values ...any) Row

	// Merge joins the cells of an A1 range; Unmerge splits them again.
	Merge(rng string) error
	Unmerge(rng string) error
	Merges() []ref.Range

	// SetColWidth sets the width in characters for columns from..to (1-based).
	SetColWidth(from, to int, width float64)
	SetColHidden(from, to int, hidden bool)
	SetColStyle(from, to int, st style.Style)
	SetRowHeight(row int, height float64)
	// SetStyle applies a style to an A1 cell or range.
	SetStyle(rng string, st style.Style) error
	// Freeze freezes the top rows and left columns.
	Freeze(rows, cols int)
	// SetZoom sets the view zoom in percent.
	SetZoom(pct int)
	SetGridLines(show bool)

	// AddHyperlink links a cell or range to a URL ("https://...", "mailto:")
	// or a location in the workbook ("Sheet2!A1" with location=true).
	AddHyperlink(rng, target, tooltip string) error
	AddInternalLink(rng, location, tooltip string) error
	// AddComment attaches a note to a cell.
	AddComment(ref, author, text string) error
	// AddThreadedComment attaches an Excel 365 threaded comment.
	AddThreadedComment(ref, author, text string, replies ...model.CommentReply) error
	// AddImage places an image (png, jpeg, gif, ...) with its top-left at a cell.
	AddImage(ref string, img model.Image) error
	// AddChart places a chart covering the range given by from..to cells.
	AddChart(from, to string, c *chart.Chart) error
	// AddTable turns a range into a structured table (first row = headers).
	AddTable(rng, name string, opts ...TableOption) (*model.Table, error)
	// AddDataValidation restricts input in a range.
	AddDataValidation(rng string, dv model.DataValidation) error
	// AddConditionalFormat adds formatting rules for a range.
	AddConditionalFormat(rng string, rules ...model.CondFmtRule) error
	// AddSparklines adds a sparkline group.
	AddSparklines(group model.SparklineGroup)
	// SetAutoFilter puts filter buttons on a header row.
	SetAutoFilter(rng string) error
	// Protect locks the sheet; an empty password locks without one.
	Protect(password string, allow ...Permission)
	Unprotect()

	// PageSetup, PageMargins and HeaderFooter configure printing.
	PageSetup() *model.PageSetup
	PageMargins() *model.PageMargins
	HeaderFooter() *model.HeaderFooter
	SetPrintArea(rng string) error
	SetPrintTitleRows(from, to int)
	SetPrintTitleCols(from, to int)

	// Workbook returns the owning workbook.
	Workbook() Workbook
	// Model exposes the underlying sheet data.
	Model() *model.Sheet
}

// Row is a worksheet row.
type Row interface {
	Index() int
	// Cell returns the cell in the column (1-based), creating it if needed.
	Cell(col int) Cell
	// Cells iterates the existing cells in column order.
	Cells() iter.Seq[Cell]
	// Values returns the values from column 1 to the last used column.
	Values() []model.Value
	// Set writes consecutive values from column 1.
	Set(values ...any) Row
	SetHeight(points float64) Row
	SetHidden(bool) Row
	SetStyle(style.Style) Row
	SetOutlineLevel(level int, collapsed bool) Row
	Model() *model.Row
}

// Cell is a worksheet cell. Setters return the cell so calls can chain.
type Cell interface {
	// Ref is the cell's A1 address.
	Ref() string
	Row() int
	Col() int
	// Err is non-nil when the cell was obtained with an invalid reference.
	Err() error

	// Set writes a value. Accepted types: nil (clears), string, bool, all
	// int and uint sizes, float32/64, time.Time (serial number, date
	// format applied if the cell has none), time.Duration (elapsed time
	// format), []model.RichRun or model.Value, fmt.Stringer, and error
	// (stored as the matching Excel error code or #VALUE!). Other types
	// are stored through fmt.Sprint.
	Set(v any) Cell
	// SetFormula stores a formula without the leading "=". The cached
	// value is cleared; Excel calculates it on open.
	SetFormula(formula string) Cell
	// SetArrayFormula stores a legacy or dynamic array formula covering rng.
	SetArrayFormula(formula, rng string, dynamic bool) Cell
	// SetFormulaResult stores a formula with a cached result so that readers
	// that do not calculate still show a value.
	SetFormulaResult(formula string, result any) Cell
	// SetRich writes rich text runs.
	SetRich(runs ...model.RichRun) Cell
	// Clear removes the value and formula, keeping the style.
	Clear() Cell

	// Value is the typed cell value (cached result for formulas).
	Value() model.Value
	// Formula is the formula text without "=", or "".
	Formula() string
	// String returns the value as text (numbers unformatted).
	String() string
	// Formatted renders the value with the cell's number format, as Excel shows it.
	Formatted() string
	Float() (float64, bool)
	Int() (int64, bool)
	Bool() (bool, bool)
	// Time converts a date serial number to time.Time.
	Time() (time.Time, bool)
	// IsDate reports whether the cell has a date/time number format.
	IsDate() bool
	IsEmpty() bool
	// Any returns the value as a Go value: nil, string, float64, bool,
	// time.Time (date-formatted numbers) or an error code string.
	Any() any

	// Style returns the resolved style; SetStyle replaces it.
	Style() style.Style
	SetStyle(style.Style) Cell
	// MergeStyle overlays the non-zero parts of st onto the current style.
	MergeStyle(st style.Style) Cell
	SetNumFmt(code string) Cell
	SetFont(style.Font) Cell
	SetFill(style.Fill) Cell
	SetBorder(style.Border) Cell
	SetAlignment(style.Alignment) Cell
	SetBold(bool) Cell
	SetColor(style.Color) Cell
	// SetNamedStyle applies a named cell style such as "Heading 1" or "Currency".
	SetNamedStyle(name string) Cell
	StyleID() int

	SetHyperlink(target, tooltip string) Cell
	Hyperlink() (model.Hyperlink, bool)
	SetComment(author, text string) Cell
	Comment() (model.Comment, bool)

	Sheet() Sheet
	Model() *model.Cell
}

// Permission names an action still allowed on a protected sheet.
type Permission string

// Permissions for Sheet.Protect.
const (
	AllowFormatCells      Permission = "formatCells"
	AllowFormatColumns    Permission = "formatColumns"
	AllowFormatRows       Permission = "formatRows"
	AllowInsertColumns    Permission = "insertColumns"
	AllowInsertRows       Permission = "insertRows"
	AllowInsertHyperlinks Permission = "insertHyperlinks"
	AllowDeleteColumns    Permission = "deleteColumns"
	AllowDeleteRows       Permission = "deleteRows"
	AllowSort             Permission = "sort"
	AllowAutoFilter       Permission = "autoFilter"
	AllowPivotTables      Permission = "pivotTables"
	AllowObjects          Permission = "objects"
	AllowScenarios        Permission = "scenarios"
)

// TableOption tunes Sheet.AddTable.
type TableOption func(*model.Table)

// TableStyle picks a built-in table style name ("TableStyleLight9", ...).
func TableStyle(name string) TableOption {
	return func(t *model.Table) { t.Style.Name = name }
}

// TableTotals adds a totals row with a function per column name
// ("sum", "average", "count", "min", "max", "none").
func TableTotals(funcs map[string]string) TableOption {
	return func(t *model.Table) {
		t.TotalsRow = true
		for i := range t.Columns {
			if f, ok := funcs[t.Columns[i].Name]; ok {
				t.Columns[i].TotalsRowFunction = f
			}
		}
	}
}

// TableNoFilter hides the filter buttons.
func TableNoFilter() TableOption {
	return func(t *model.Table) { t.NoAutoFilter = true }
}

// TableStripes toggles banded rows and columns.
func TableStripes(rows, cols bool) TableOption {
	return func(t *model.Table) { t.Style.ShowRowStripes, t.Style.ShowColumnStripes = rows, cols }
}

// Options for Open.
type Options = reader.Options

// New creates an empty workbook with one sheet named "Sheet1".
func New() Workbook {
	wb := model.New()
	wb.AddSheet("Sheet1")
	return Wrap(wb)
}

// NewEmpty creates a workbook with no sheets; add one before saving.
func NewEmpty() Workbook { return Wrap(model.New()) }

// Wrap exposes an existing model through the Workbook interface.
func Wrap(wb *model.Workbook) Workbook { return &workbook{m: wb} }

// Open reads a workbook from a file.
func Open(name string, opts ...Options) (Workbook, error) {
	wb, err := reader.ReadFile(name, opts...)
	if err != nil {
		return nil, err
	}
	return Wrap(wb), nil
}

// OpenReader reads a workbook from a ReaderAt of known size.
func OpenReader(r io.ReaderAt, size int64, opts ...Options) (Workbook, error) {
	wb, err := reader.Read(r, size, opts...)
	if err != nil {
		return nil, err
	}
	return Wrap(wb), nil
}

// OpenBytes reads a workbook held in memory.
func OpenBytes(data []byte, opts ...Options) (Workbook, error) {
	wb, err := reader.ReadBytes(data, opts...)
	if err != nil {
		return nil, err
	}
	return Wrap(wb), nil
}

// Stream reads rows of very large files one at a time; see reader.Stream.
type Stream = reader.Stream

// StreamFile opens a file for row streaming.
func StreamFile(name string, opts ...Options) (*Stream, error) {
	return reader.OpenStreamFile(name, opts...)
}

// StreamReader opens a ReaderAt for row streaming.
func StreamReader(r io.ReaderAt, size int64, opts ...Options) (*Stream, error) {
	return reader.OpenStream(r, size, opts...)
}

// OpenStream reads all of r into memory and opens it.
func OpenStream(r io.Reader, opts ...Options) (Workbook, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return OpenBytes(data, opts...)
}
