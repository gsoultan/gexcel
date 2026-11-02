// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package model

import (
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

// Table is a structured table (ListObject) over a range whose first row
// holds the column headers.
type Table struct {
	ID          int
	Name        string
	DisplayName string
	Range       ref.Range
	Columns     []TableColumn
	Style       TableStyle
	HeaderRow   *bool // default true
	TotalsRow   bool
	// Insert row/column flags and comment.
	Comment        string
	TotalsRowCount int
	AutoFilter     *AutoFilter // nil means a default filter over the data; Disabled hides it
	NoAutoFilter   bool
	// Connection/XML tables are preserved raw.
	Raw []byte
}

// TableColumn is one column of a table.
type TableColumn struct {
	ID   int
	Name string
	// Totals row: a function (sum, average, count, countNums, max, min,
	// stdDev, var, custom, none) and/or a label or formula.
	TotalsRowFunction string
	TotalsRowLabel    string
	TotalsRowFormula  string
	// CalculatedColumnFormula fills every data cell of the column.
	CalculatedColumnFormula string
	DataDxfID               int
	HeaderDxfID             int
	TotalsDxfID             int
	UniqueName              string
	QueryTableFieldID       int
}

// TableStyle names a built-in table style ("TableStyleMedium2", ...) and
// its stripe/column options.
type TableStyle struct {
	Name              string
	ShowFirstColumn   bool
	ShowLastColumn    bool
	ShowRowStripes    bool
	ShowColumnStripes bool
}

// DefaultTableStyle is what Excel applies to a new table.
func DefaultTableStyle() TableStyle {
	return TableStyle{Name: "TableStyleMedium2", ShowRowStripes: true}
}

// CondFmtType selects the conditional formatting rule.
type CondFmtType string

// Conditional formatting rule types (ST_CfType).
const (
	CfCellIs            CondFmtType = "cellIs"
	CfExpression        CondFmtType = "expression"
	CfColorScale        CondFmtType = "colorScale"
	CfDataBar           CondFmtType = "dataBar"
	CfIconSet           CondFmtType = "iconSet"
	CfTop10             CondFmtType = "top10"
	CfUniqueValues      CondFmtType = "uniqueValues"
	CfDuplicateValues   CondFmtType = "duplicateValues"
	CfContainsText      CondFmtType = "containsText"
	CfNotContainsText   CondFmtType = "notContainsText"
	CfBeginsWith        CondFmtType = "beginsWith"
	CfEndsWith          CondFmtType = "endsWith"
	CfContainsBlanks    CondFmtType = "containsBlanks"
	CfNotContainsBlanks CondFmtType = "notContainsBlanks"
	CfContainsErrors    CondFmtType = "containsErrors"
	CfNotContainsErrors CondFmtType = "notContainsErrors"
	CfTimePeriod        CondFmtType = "timePeriod"
	CfAboveAverage      CondFmtType = "aboveAverage"
)

// CondFmtOperator is the comparison of a cellIs rule (ST_ConditionalFormattingOperator).
type CondFmtOperator string

// Conditional formatting operators.
const (
	CfOpBetween      CondFmtOperator = "between"
	CfOpNotBetween   CondFmtOperator = "notBetween"
	CfOpEqual        CondFmtOperator = "equal"
	CfOpNotEqual     CondFmtOperator = "notEqual"
	CfOpGreater      CondFmtOperator = "greaterThan"
	CfOpLess         CondFmtOperator = "lessThan"
	CfOpGreaterEq    CondFmtOperator = "greaterThanOrEqual"
	CfOpLessEq       CondFmtOperator = "lessThanOrEqual"
	CfOpContainsText CondFmtOperator = "containsText"
	CfOpNotContains  CondFmtOperator = "notContains"
	CfOpBeginsWith   CondFmtOperator = "beginsWith"
	CfOpEndsWith     CondFmtOperator = "endsWith"
)

// ConditionalFormat applies rules to ranges.
type ConditionalFormat struct {
	Ranges []ref.Range
	Rules  []CondFmtRule
	Pivot  bool
}

// CondFmtRule is one conditional formatting rule.
type CondFmtRule struct {
	Type       CondFmtType
	Priority   int // lower wins; the writer assigns sequential priorities when 0
	StopIfTrue bool
	// Style is the differential format applied when the rule matches
	// (cellIs, expression, text, top10, ... rules). DxfID is its index
	// in the style table after reading/before writing.
	Style    style.Style
	HasStyle bool
	DxfID    int
	Operator CondFmtOperator
	Formulas []string // 1 or 2 formulas without "="
	Text     string   // containsText and friends
	// top10
	Rank    int
	Bottom  bool
	Percent bool
	// aboveAverage
	AboveAverage *bool // nil = above (default true); false = below
	EqualAverage bool
	StdDev       int
	// timePeriod: today, yesterday, tomorrow, last7Days, thisMonth, lastMonth,
	// nextMonth, thisWeek, lastWeek, nextWeek
	TimePeriod string
	ColorScale *ColorScale
	DataBar    *DataBar
	IconSet    *IconSet
}

// CfValueType is how a color scale / data bar / icon set threshold is given.
type CfValueType string

// Threshold value types (ST_CfvoType).
const (
	CfvoNum        CfValueType = "num"
	CfvoPercent    CfValueType = "percent"
	CfvoMax        CfValueType = "max"
	CfvoMin        CfValueType = "min"
	CfvoFormula    CfValueType = "formula"
	CfvoPercentile CfValueType = "percentile"
	CfvoAutoMin    CfValueType = "autoMin"
	CfvoAutoMax    CfValueType = "autoMax"
)

// CfValue is a threshold (cfvo).
type CfValue struct {
	Type  CfValueType
	Value string // number, percent or formula; empty for min/max
	GTE   *bool  // greater-than-or-equal (icon sets, default true)
}

// ColorScale is a 2- or 3-color scale.
type ColorScale struct {
	Values []CfValue
	Colors []style.Color
}

// DataBar is a data bar rule.
type DataBar struct {
	Min, Max  CfValue
	Color     style.Color
	MinLength int // percent, 0 = 10
	MaxLength int // percent, 0 = 90
	ShowValue *bool
	// Excel 2010+ extensions.
	Gradient       *bool
	BorderColor    style.Color
	NegativeColor  style.Color
	NegativeBorder style.Color
	AxisColor      style.Color
	AxisPosition   string // "", "middle", "none"
	Direction      string // "", "leftToRight", "rightToLeft"
}

// IconSet is an icon set rule.
type IconSet struct {
	Set       string // "3TrafficLights1", "3Arrows", "4Rating", "5Quarters", ...
	Values    []CfValue
	ShowValue *bool
	Reverse   bool
	Percent   *bool
	// Custom icons per threshold (Excel 2010+): set name and index.
	Custom []CustomIcon
}

// CustomIcon is one icon of a custom icon set.
type CustomIcon struct {
	Set string
	ID  int
}

// AnchorKind is how a drawing object is positioned.
type AnchorKind uint8

// Anchor kinds.
const (
	AnchorTwoCell  AnchorKind = iota // from/to cells (moves and sizes with cells)
	AnchorOneCell                    // from cell plus size
	AnchorAbsolute                   // absolute position and size
)

// CellOffset is a cell corner with EMU offsets (914400 EMU per inch).
type CellOffset struct {
	Row, Col       int   // 1-based
	RowOff, ColOff int64 // EMU
}

// Anchor positions a drawing object.
type Anchor struct {
	Kind     AnchorKind
	From, To CellOffset
	// Width and Height in EMU (one-cell and absolute anchors; also written
	// as the shape extent).
	Width, Height int64
	// X and Y in EMU for absolute anchors.
	X, Y int64
	// EditAs: "", "oneCell", "absolute", "twoCell".
	EditAs string
}

// EMU conversions.
const (
	EMUPerInch  = 914400
	EMUPerPoint = 12700
	EMUPerPixel = 9525 // at 96 DPI
	EMUPerCm    = 360000
)

// Image is a picture placed on a sheet.
type Image struct {
	Name        string
	Description string
	Format      string // "png", "jpeg", "gif", "bmp", "tiff", "svg", "emf", "wmf", "webp"
	Data        []byte
	Anchor      Anchor
	Hyperlink   string
	LockAspect  bool
	// ID is the drawing object id, assigned by the writer.
	ID int
}

// DefinedName is a workbook or sheet-scoped name.
type DefinedName struct {
	Name     string
	RefersTo string // formula without "=", e.g. "Sheet1!$A$1:$B$2"
	// Local scopes the name to the sheet at SheetIndex (0-based); otherwise
	// the name is workbook-wide.
	Local      bool
	SheetIndex int
	Comment    string
	Hidden     bool
	Function   bool
	VBProc     bool
	// Builtin names: "_xlnm.Print_Area", "_xlnm.Print_Titles", "_xlnm._FilterDatabase", ...
}

// Builtin defined names.
const (
	NamePrintArea      = "_xlnm.Print_Area"
	NamePrintTitles    = "_xlnm.Print_Titles"
	NameFilterDatabase = "_xlnm._FilterDatabase"
	NameCriteria       = "_xlnm.Criteria"
	NameExtract        = "_xlnm.Extract"
	NameSheetTitle     = "_xlnm.Sheet_Title"
	NameConsolidate    = "_xlnm.Consolidate_Area"
	NameDatabase       = "_xlnm.Database"
)
