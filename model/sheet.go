// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package model

import (
	"sort"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

// SheetState is the sheet visibility.
type SheetState string

// Sheet visibility states.
const (
	SheetVisible    SheetState = ""
	SheetHidden     SheetState = "hidden"
	SheetVeryHidden SheetState = "veryHidden"
)

// SheetKind distinguishes worksheets from chart sheets and macro sheets.
type SheetKind uint8

// Sheet kinds.
const (
	Worksheet   SheetKind = iota
	Chartsheet            // a sheet that holds one chart (Charts[0]) and no cells
	MacroSheet            // Excel 4 macro sheet, preserved raw in Raw
	DialogSheet           // preserved raw in Raw
)

// Sheet is one worksheet.
type Sheet struct {
	Name     string
	Kind     SheetKind
	State    SheetState
	TabColor style.Color
	CodeName string // VBA code name

	Rows map[int]*Row
	Cols []Col

	// Defaults for rows and columns (sheetFormatPr).
	DefaultRowHeight  float64 // points; 0 = 15
	DefaultColWidth   float64 // characters; 0 = 8.43 (baseColWidth 8)
	CustomRowHeight   bool
	ZeroHeight        bool // rows hidden by default
	OutlineLevelRow   int
	OutlineLevelCol   int
	ThickTop          bool
	ThickBottom       bool
	SummaryBelow      *bool // outline summary rows below detail (default true)
	SummaryRight      *bool
	ApplyStyles       bool
	FitToPage         bool
	FilterMode        bool
	EnableCondFmtCalc *bool
	Published         bool
	TransitionEval    bool

	Merges             []ref.Range
	Hyperlinks         []Hyperlink
	Comments           []Comment
	DataValidations    []DataValidation
	ConditionalFormats []ConditionalFormat
	Tables             []*Table
	AutoFilter         *AutoFilter
	Images             []*Image
	Charts             []*chart.Chart
	Shapes             []*Shape
	Sparklines         []SparklineGroup
	PivotTables        []*PivotTable

	View         SheetView
	PageSetup    PageSetup
	PageMargins  PageMargins
	PrintOptions PrintOptions
	HeaderFooter HeaderFooter
	// PrintArea and print titles are stored as defined names in the
	// workbook (_xlnm.Print_Area, _xlnm.Print_Titles); these fields are the
	// decoded form and win over DefinedNames when set.
	PrintArea      []ref.Range
	PrintTitleRows *ref.Range // whole-row range
	PrintTitleCols *ref.Range // whole-column range
	RowBreaks      []PageBreak
	ColBreaks      []PageBreak
	Protection     *SheetProtection
	// IgnoredErrors lists ranges with suppressed green-triangle warnings.
	IgnoredErrors []IgnoredError
	// Dimension as read (ref attribute); the writer recomputes it.
	Dimension ref.Range
	// Raw keeps the part bytes of chart/macro/dialog sheets that are
	// preserved rather than modelled, and their relationships.
	Raw *RawPart
	// Extensions preserves <extLst> children the model does not cover.
	Extensions []RawXML
}

// Col describes a run of columns (Min..Max inclusive).
type Col struct {
	Min, Max int
	// Width in characters of the default font's max digit width; 0 means
	// the sheet default.
	Width        float64
	CustomWidth  bool
	BestFit      bool
	Hidden       bool
	StyleID      int
	OutlineLevel int
	Collapsed    bool
	Phonetic     bool
}

// PageBreak is a manual page break before the row or column Index+1 (a
// break at row 10 means page 2 starts at row 11).
type PageBreak struct {
	Index  int
	Max    int // extent of the break (Excel writes 16383 / 1048575)
	Manual bool
}

// IgnoredError suppresses error checking on ranges.
type IgnoredError struct {
	Ranges             []ref.Range
	EvalError          bool
	TwoDigitTextYear   bool
	NumberStoredAsTxt  bool
	Formula            bool
	FormulaRange       bool
	UnlockedFormula    bool
	EmptyCellRef       bool
	ListDataValidation bool
	CalculatedColumn   bool
}

// RawPart is a package part kept verbatim.
type RawPart struct {
	Name        string // part name, e.g. "xl/chartsheets/sheet1.xml"
	ContentType string
	Data        []byte
	Rels        []Relationship
	// Related holds parts reached through Rels (drawings, media), keyed by part name.
	Related map[string]*RawPart
}

// Relationship is an OPC relationship.
type Relationship struct {
	ID         string
	Type       string
	Target     string
	TargetMode string // "" or "External"
}

// RawXML is an XML fragment preserved as read.
type RawXML struct {
	URI  string // ext uri, when it is an <ext> element
	Data []byte
}

// NewSheet returns an empty worksheet.
func NewSheet(name string) *Sheet {
	return &Sheet{Name: name, Rows: map[int]*Row{}}
}

// Row returns the row, creating it when missing.
func (s *Sheet) Row(index int) *Row {
	if s.Rows == nil {
		s.Rows = map[int]*Row{}
	}
	r, ok := s.Rows[index]
	if !ok {
		r = &Row{Index: index}
		s.Rows[index] = r
	}
	return r
}

// LookupRow returns the row or nil.
func (s *Sheet) LookupRow(index int) *Row { return s.Rows[index] }

// Cell returns the cell, creating the row and cell when missing.
func (s *Sheet) Cell(row, col int) *Cell { return s.Row(row).Cell(col) }

// Lookup returns the cell or nil.
func (s *Sheet) Lookup(row, col int) *Cell {
	r := s.Rows[row]
	if r == nil {
		return nil
	}
	return r.Cells[col]
}

// SortedRows returns the rows in index order.
func (s *Sheet) SortedRows() []*Row {
	out := make([]*Row, 0, len(s.Rows))
	for _, r := range s.Rows {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// UsedRange returns the smallest range covering every cell, or ok=false
// for an empty sheet.
func (s *Sheet) UsedRange() (ref.Range, bool) {
	minR, minC, maxR, maxC := 0, 0, 0, 0
	for _, r := range s.Rows {
		if len(r.Cells) == 0 {
			continue
		}
		if minR == 0 || r.Index < minR {
			minR = r.Index
		}
		if r.Index > maxR {
			maxR = r.Index
		}
		if c := r.MinCol(); minC == 0 || c < minC {
			minC = c
		}
		if c := r.MaxCol(); c > maxC {
			maxC = c
		}
	}
	if maxR == 0 {
		return ref.Range{}, false
	}
	return ref.NewRange(minR, minC, maxR, maxC), true
}

// ColInfo returns the Col entry covering the column, or nil.
func (s *Sheet) ColInfo(col int) *Col {
	for i := range s.Cols {
		if col >= s.Cols[i].Min && col <= s.Cols[i].Max {
			return &s.Cols[i]
		}
	}
	return nil
}

// SetCol sets attributes for a single column, splitting an existing run
// that covers it. fn receives the column's current settings to modify.
func (s *Sheet) SetCol(col int, fn func(*Col)) {
	var out []Col
	var cur *Col
	for _, c := range s.Cols {
		if col < c.Min || col > c.Max {
			out = append(out, c)
			continue
		}
		if c.Min < col {
			left := c
			left.Max = col - 1
			out = append(out, left)
		}
		mid := c
		mid.Min, mid.Max = col, col
		out = append(out, mid)
		cur = &out[len(out)-1]
		if c.Max > col {
			right := c
			right.Min = col + 1
			out = append(out, right)
		}
	}
	if cur == nil {
		out = append(out, Col{Min: col, Max: col})
		cur = &out[len(out)-1]
	}
	fn(cur)
	sort.Slice(out, func(i, j int) bool { return out[i].Min < out[j].Min })
	s.Cols = out
}

// Hyperlink is a cell hyperlink. Either Target (external URL or file, with
// an optional "#fragment") or Location (a reference inside the workbook)
// is set.
type Hyperlink struct {
	Range    ref.Range
	Target   string
	Location string
	Tooltip  string
	Display  string
}

// Comment is a cell note (legacy comment) or a threaded comment.
type Comment struct {
	Ref    ref.Cell
	Author string
	Text   string
	Rich   []RichRun
	// Visible shows the note permanently.
	Visible bool
	// Box position and size in pixels for the legacy VML shape; zero picks a default.
	Width, Height float64
	// Threaded comments (Excel 365) carry an id, a timestamp and replies.
	Threaded bool
	ID       string
	Created  string // ISO 8601
	Done     bool
	Replies  []CommentReply
}

// CommentReply is a reply in a threaded comment.
type CommentReply struct {
	Author  string
	Text    string
	ID      string
	Created string
}

// DataValidationType selects the validation rule.
type DataValidationType string

// Data validation types.
const (
	ValidationNone    DataValidationType = ""
	ValidationWhole   DataValidationType = "whole"
	ValidationDecimal DataValidationType = "decimal"
	ValidationList    DataValidationType = "list"
	ValidationDate    DataValidationType = "date"
	ValidationTime    DataValidationType = "time"
	ValidationTextLen DataValidationType = "textLength"
	ValidationCustom  DataValidationType = "custom"
)

// DataValidationOperator compares the value with Formula1/Formula2.
type DataValidationOperator string

// Data validation operators.
const (
	OpBetween    DataValidationOperator = "" // default
	OpNotBetween DataValidationOperator = "notBetween"
	OpEqual      DataValidationOperator = "equal"
	OpNotEqual   DataValidationOperator = "notEqual"
	OpLess       DataValidationOperator = "lessThan"
	OpLessEq     DataValidationOperator = "lessThanOrEqual"
	OpGreater    DataValidationOperator = "greaterThan"
	OpGreaterEq  DataValidationOperator = "greaterThanOrEqual"
)

// DataValidation restricts what may be entered in a range.
type DataValidation struct {
	Ranges   []ref.Range
	Type     DataValidationType
	Operator DataValidationOperator
	// Formula1 and Formula2 are formulas without "=": a range, a value, or
	// for lists a quoted comma-separated set ("\"A,B,C\"").
	Formula1, Formula2 string
	AllowBlank         bool
	ShowDropDown       bool // suppresses the in-cell dropdown for lists (Excel's inverted flag)
	ShowInputMessage   bool
	ShowErrorMessage   bool
	ErrorStyle         string // "stop" (default), "warning", "information"
	ErrorTitle, Error  string
	PromptTitle        string
	Prompt             string
	IMEMode            string
}

// AutoFilter is the filter dropdown on a range, with optional column filters.
type AutoFilter struct {
	Range     ref.Range
	Columns   []FilterColumn
	SortState *SortState
}

// FilterColumn is the filter on one column of an autofilter or table. ColID
// is 0-based within the filter range.
type FilterColumn struct {
	ColID        int
	HiddenButton bool
	ShowButton   *bool
	// Values filters (filters/filter val="..."); Blank includes blank cells.
	Values []string
	Blank  bool
	// DateGroups filters by date granularity.
	DateGroups []DateGroupItem
	// Custom filters: up to two conditions joined by And.
	Custom      []CustomFilter
	And         bool
	Top10       *Top10Filter
	Dynamic     *DynamicFilter
	ColorFilter *ColorFilter
	IconFilter  *IconFilter
}

// DateGroupItem is one dateGroupItem of a filter.
type DateGroupItem struct {
	Grouping             string // year, month, day, hour, minute, second
	Year, Month, Day     int
	Hour, Minute, Second int
}

// CustomFilter is one customFilter condition.
type CustomFilter struct {
	Operator string // equal, notEqual, greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual
	Value    string
}

// Top10Filter is a top/bottom N filter.
type Top10Filter struct {
	Top       bool
	Percent   bool
	Val       float64
	FilterVal float64
}

// DynamicFilter is a relative date or average filter.
type DynamicFilter struct {
	Type   string // today, yesterday, thisMonth, aboveAverage, ...
	Val    float64
	MaxVal float64
}

// ColorFilter filters by cell or font color through a dxf.
type ColorFilter struct {
	DxfID     int
	CellColor bool
}

// IconFilter filters by icon set member.
type IconFilter struct {
	IconSet string
	IconID  int
}

// SortState records how a range was last sorted.
type SortState struct {
	Range         ref.Range
	CaseSensitive bool
	ColumnSort    bool
	Conditions    []SortCondition
}

// SortCondition is one sort key.
type SortCondition struct {
	Range      ref.Range
	Descending bool
	SortBy     string // "", "cellColor", "fontColor", "icon"
	DxfID      int
	IconSet    string
	IconID     int
	CustomList string
}

// SheetView holds the window settings of a sheet.
type SheetView struct {
	// Gridlines, headings and zeros are shown by default, so the fields
	// are inverted to keep the zero value as the default.
	HideGridLines    bool
	HideHeadings     bool
	HideZeros        bool
	ShowFormulas     bool
	HideRuler        bool
	HideOutline      bool
	HideWhiteSpace   bool
	RightToLeft      bool
	TabSelected      bool
	ZoomScale        int // percent; 0 = 100
	ZoomNormal       int
	ZoomPageLayout   int
	ZoomPageBreak    int
	Type             string // "", "pageBreakPreview", "pageLayout"
	TopLeftCell      string
	GridLineColor    int // indexed color
	WindowProtection bool
	// Panes: Freeze freezes rows above/columns left of the split; Split
	// gives a movable split at the position in twips (1/20 pt).
	Freeze     *Pane
	Split      *Pane
	Selections []Selection
}

// Pane describes frozen or split panes.
type Pane struct {
	Rows, Cols     int     // frozen rows and columns (Freeze), or
	XSplit, YSplit float64 // split positions in twips (Split)
	TopLeftCell    string
	ActivePane     string // bottomRight, topRight, bottomLeft, topLeft
}

// Selection is the active cell and selected ranges of a pane.
type Selection struct {
	Pane       string
	ActiveCell string
	Ranges     []ref.Range
}

// Orientation of printed pages.
type Orientation string

// Page orientations.
const (
	OrientationDefault   Orientation = ""
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// PageSetup holds the print page settings (pageSetup element).
type PageSetup struct {
	Orientation        Orientation
	PaperSize          int // 1 Letter, 9 A4, ... (ST_PaperSize)
	Scale              int // percent; 0 = 100
	FitToWidth         int // pages wide; used when the sheet's FitToPage is set
	FitToHeight        int
	FirstPageNumber    int
	UseFirstPage       bool
	BlackAndWhite      bool
	Draft              bool
	CellComments       string // "", "asDisplayed", "atEnd"
	Errors             string // "", "displayed", "blank", "dash", "NA"
	PageOrder          string // "", "downThenOver", "overThenDown"
	HorizontalDPI      int
	VerticalDPI        int
	Copies             int
	UsePrinterDefaults bool
}

// PageMargins in inches; zero means the Excel default (0.7/0.75/0.3).
type PageMargins struct {
	Left, Right, Top, Bottom, Header, Footer float64
}

// DefaultPageMargins are Excel's "Normal" margins.
func DefaultPageMargins() PageMargins {
	return PageMargins{Left: 0.7, Right: 0.7, Top: 0.75, Bottom: 0.75, Header: 0.3, Footer: 0.3}
}

// IsZero reports whether no margin is set.
func (m PageMargins) IsZero() bool { return m == PageMargins{} }

// PrintOptions holds printOptions.
type PrintOptions struct {
	GridLines          bool
	GridLinesSet       bool
	Headings           bool
	HorizontalCentered bool
	VerticalCentered   bool
}

// HeaderFooter holds the header/footer strings in Excel's code format
// (&L, &C, &R sections, &P page number, &N pages, &D date, &A sheet name, ...).
type HeaderFooter struct {
	OddHeader, OddFooter     string
	EvenHeader, EvenFooter   string
	FirstHeader, FirstFooter string
	DifferentOddEven         bool
	DifferentFirst           bool
	NoAlignWithMargins       bool
	NoScaleWithDoc           bool
}

// IsZero reports whether no header or footer is set.
func (h HeaderFooter) IsZero() bool { return h == HeaderFooter{} }

// SheetProtection locks the sheet. Password is either plain text (hashed by
// the writer with SHA-512 and a random salt) or the legacy 16-bit hash in
// LegacyHash. The boolean flags allow the named action on a protected
// sheet (Excel's attributes are inverted: locked by default).
type SheetProtection struct {
	Password          string
	Algorithm         string // "SHA-512" when hashed, "" for plain Password
	HashValue         string // base64
	SaltValue         string // base64
	SpinCount         int
	LegacyHash        string // 4 hex digits
	Sheet             bool   // protect sheet contents (always true when protecting)
	Objects           bool
	Scenarios         bool
	FormatCells       bool
	FormatColumns     bool
	FormatRows        bool
	InsertColumns     bool
	InsertRows        bool
	InsertHyperlinks  bool
	DeleteColumns     bool
	DeleteRows        bool
	SelectLockedCells bool // allow selecting locked cells (default allowed; set NoSelect* to forbid)
	NoSelectLocked    bool
	NoSelectUnlocked  bool
	Sort              bool
	AutoFilter        bool
	PivotTables       bool
}

// SparklineGroup is a group of sparklines (x14 extension).
type SparklineGroup struct {
	Type       string // "line" (default), "column", "stacked"
	Sparklines []Sparkline
	// Appearance.
	LineWeight                                          float64
	ShowMarkers                                         bool
	ShowHigh                                            bool
	ShowLow                                             bool
	ShowFirst                                           bool
	ShowLast                                            bool
	ShowNegative                                        bool
	DisplayXAxis                                        bool
	DisplayHidden                                       bool
	DisplayEmptyCellsAs                                 string // "gap" (default), "zero", "span"
	MinAxisType, MaxAxisType                            string // "individual" (default), "group", "custom"
	ManualMin, ManualMax                                float64
	RightToLeft                                         bool
	DateAxis                                            string // range formula for the date axis
	ColorSeries, ColorNegative, ColorAxis, ColorMarkers style.Color
	ColorFirst, ColorLast, ColorHigh, ColorLow          style.Color
}

// Sparkline is one sparkline: data formula and target cell.
type Sparkline struct {
	Formula string // e.g. "Sheet1!A1:F1"
	Ref     ref.Cell
}

// Shape is a drawing shape (text box, rectangle, ...) with minimal
// modelling; Raw keeps the drawing XML for round trips.
type Shape struct {
	Name   string
	Anchor Anchor
	Text   string
	Preset string // "rect", "ellipse", ...
	Fill   style.Color
	Line   style.Color
	Raw    []byte
}

// PivotTable keeps a pivot table and its cache as raw parts until the
// model covers them.
type PivotTable struct {
	Name    string
	Table   *RawPart
	Cache   *RawPart
	Records *RawPart
}

// Clone deep-copies the sheet's cells, rows, columns and settings. Images,
// charts, tables and raw parts are copied by value one level deep so the
// copy can be edited without touching the original.
func (s *Sheet) Clone() *Sheet {
	cp := *s
	cp.Rows = make(map[int]*Row, len(s.Rows))
	for i, r := range s.Rows {
		nr := *r
		nr.Cells = make(map[int]*Cell, len(r.Cells))
		for c, cell := range r.Cells {
			nc := *cell
			if cell.Formula != nil {
				f := *cell.Formula
				nc.Formula = &f
			}
			if cell.Value.Rich != nil {
				nc.Value.Rich = append([]RichRun(nil), cell.Value.Rich...)
			}
			nr.Cells[c] = &nc
		}
		cp.Rows[i] = &nr
	}
	cp.Cols = append([]Col(nil), s.Cols...)
	cp.Merges = append([]ref.Range(nil), s.Merges...)
	cp.Hyperlinks = append([]Hyperlink(nil), s.Hyperlinks...)
	cp.Comments = append([]Comment(nil), s.Comments...)
	cp.DataValidations = append([]DataValidation(nil), s.DataValidations...)
	cp.ConditionalFormats = append([]ConditionalFormat(nil), s.ConditionalFormats...)
	cp.Sparklines = append([]SparklineGroup(nil), s.Sparklines...)
	cp.PrintArea = append([]ref.Range(nil), s.PrintArea...)
	cp.RowBreaks = append([]PageBreak(nil), s.RowBreaks...)
	cp.ColBreaks = append([]PageBreak(nil), s.ColBreaks...)
	cp.IgnoredErrors = append([]IgnoredError(nil), s.IgnoredErrors...)
	cp.Extensions = append([]RawXML(nil), s.Extensions...)
	cp.Tables = nil
	for _, t := range s.Tables {
		nt := *t
		nt.Columns = append([]TableColumn(nil), t.Columns...)
		nt.ID = 0
		nt.Name, nt.DisplayName = "", "" // the writer assigns unique names
		if t.DisplayName != "" {
			nt.Name, nt.DisplayName = t.Name+"_copy", t.DisplayName+"_copy"
		}
		cp.Tables = append(cp.Tables, &nt)
	}
	cp.Images = nil
	for _, im := range s.Images {
		ni := *im
		cp.Images = append(cp.Images, &ni)
	}
	cp.Charts = nil
	for _, ch := range s.Charts {
		nc := *ch
		nc.Series = append([]chart.Series(nil), ch.Series...)
		cp.Charts = append(cp.Charts, &nc)
	}
	cp.Shapes = append([]*Shape(nil), s.Shapes...)
	cp.PivotTables = nil // pivot tables bind to their cache; not copied
	if s.AutoFilter != nil {
		af := *s.AutoFilter
		cp.AutoFilter = &af
	}
	if s.Protection != nil {
		p := *s.Protection
		cp.Protection = &p
	}
	if s.View.Freeze != nil {
		f := *s.View.Freeze
		cp.View.Freeze = &f
	}
	if s.View.Split != nil {
		f := *s.View.Split
		cp.View.Split = &f
	}
	cp.View.Selections = append([]Selection(nil), s.View.Selections...)
	if s.PrintTitleRows != nil {
		r := *s.PrintTitleRows
		cp.PrintTitleRows = &r
	}
	if s.PrintTitleCols != nil {
		r := *s.PrintTitleCols
		cp.PrintTitleCols = &r
	}
	return &cp
}
