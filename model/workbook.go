// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package model

import (
	"strings"
	"time"

	"github.com/gsoultan/gexcel/style"
)

// Workbook is the whole in-memory workbook.
type Workbook struct {
	Sheets       []*Sheet
	Styles       *Styles
	Theme        *style.Theme
	DefinedNames []DefinedName
	Properties   Properties
	CustomProps  []CustomProperty
	Date1904     bool
	Calc         CalcProperties
	View         WorkbookView
	Protection   *WorkbookProtection
	// CodeName is the VBA project name of the workbook (ThisWorkbook).
	CodeName string
	// VBAProject holds xl/vbaProject.bin for macro-enabled workbooks; the
	// writer emits an .xlsm package when it is set.
	VBAProject []byte
	// ExternalLinks are the externalLink parts formulas refer to as [1], [2]...
	ExternalLinks []*ExternalLink
	// Preserved parts (custom XML, connections, metadata, ...) written back verbatim.
	Preserved []*RawPart
	// Extensions preserves workbook-level <extLst> children.
	Extensions []RawXML
	// PivotCaches are the workbook-level pivot cache definitions by id.
	PivotCaches map[int]*RawPart
	// RichValue parts (Excel 365 data types) preserved for round trips.
	RichData []*RawPart
	// Metadata is xl/metadata.xml (dynamic array markers); regenerated when nil.
	Metadata            []byte
	FilterPrivacy       bool
	HidePivotFieldList  bool
	ShowInkAnnotation   *bool
	UpdateLinks         string // "", "never", "always", "userSet"
	DefaultThemeVersion int
	AbsPath             string
}

// Properties are the core and extended document properties.
type Properties struct {
	Title          string
	Subject        string
	Creator        string
	Keywords       string
	Description    string
	LastModifiedBy string
	Revision       string
	Category       string
	ContentStatus  string
	Language       string
	Identifier     string
	Version        string
	Created        time.Time
	Modified       time.Time
	LastPrinted    time.Time
	// Extended (app.xml).
	Application       string
	AppVersion        string
	Company           string
	Manager           string
	DocSecurity       int
	HyperlinkBase     string
	Template          string
	ScaleCrop         bool
	LinksUpToDate     bool
	SharedDoc         bool
	HyperlinksChanged bool
}

// CustomProperty is a user-defined document property. Value is a string,
// float64, int, bool or time.Time.
type CustomProperty struct {
	Name  string
	Value any
}

// CalcProperties mirror the workbook calcPr element.
type CalcProperties struct {
	CalcID          int
	CalcMode        string // "", "auto", "autoNoTable", "manual"
	FullCalcOnLoad  bool
	RefModeR1C1     bool
	Iterate         bool
	IterateCount    int
	IterateDelta    float64
	NoFullPrecision bool
	CalcCompleted   *bool
	CalcOnSave      *bool
	ConcurrentCalc  *bool
	ForceFullCalc   bool
}

// WorkbookView mirrors bookViews/workbookView.
type WorkbookView struct {
	ActiveTab                 int
	FirstSheet                int
	Visibility                string // "", "hidden", "veryHidden"
	Minimized                 bool
	HideHorizontalScroll      bool
	HideVerticalScroll        bool
	HideSheetTabs             bool
	XWindow, YWindow          int
	WindowWidth, WindowHeight int
	TabRatio                  int
	AutoFilterDateGrouping    *bool
}

// WorkbookProtection locks the structure and windows.
type WorkbookProtection struct {
	LockStructure                                    bool
	LockWindows                                      bool
	LockRevision                                     bool
	Password                                         string // plain text, hashed by the writer
	Algorithm                                        string
	HashValue                                        string
	SaltValue                                        string
	SpinCount                                        int
	LegacyHash                                       string
	RevisionPassword                                 string
	RevisionsAlgorithm, RevisionsHash, RevisionsSalt string
	RevisionsSpin                                    int
}

// ExternalLink is an external workbook referenced by formulas. The part is
// preserved verbatim; SheetNames lists the sheets it exposes.
type ExternalLink struct {
	Target     string // path/URL of the other workbook
	SheetNames []string
	Raw        *RawPart
}

// New returns an empty workbook with default styles and theme.
func New() *Workbook {
	return &Workbook{Styles: NewStyles(), Theme: style.DefaultTheme(), Properties: Properties{
		Application: "Microsoft Excel", AppVersion: "16.0300",
	}}
}

// Sheet returns the sheet with the name (case-insensitively) or nil.
func (w *Workbook) Sheet(name string) *Sheet {
	for _, s := range w.Sheets {
		if strings.EqualFold(s.Name, name) {
			return s
		}
	}
	return nil
}

// SheetIndex returns the index of the sheet with the name, or -1.
func (w *Workbook) SheetIndex(name string) int {
	for i, s := range w.Sheets {
		if strings.EqualFold(s.Name, name) {
			return i
		}
	}
	return -1
}

// AddSheet appends a new worksheet.
func (w *Workbook) AddSheet(name string) *Sheet {
	s := NewSheet(name)
	w.Sheets = append(w.Sheets, s)
	return s
}

// DefinedName returns the workbook-scoped name, or nil.
func (w *Workbook) DefinedName(name string) *DefinedName {
	for i := range w.DefinedNames {
		d := &w.DefinedNames[i]
		if strings.EqualFold(d.Name, name) && !d.Local {
			return d
		}
	}
	return nil
}

// LocalDefinedName returns the name scoped to the sheet, or nil.
func (w *Workbook) LocalDefinedName(name string, sheetIndex int) *DefinedName {
	for i := range w.DefinedNames {
		d := &w.DefinedNames[i]
		if strings.EqualFold(d.Name, name) && d.Local && d.SheetIndex == sheetIndex {
			return d
		}
	}
	return nil
}

// Max limits.
const (
	MaxSheetNameLen = 31
	MaxCellChars    = 32_767
	MaxColWidth     = 255
	MaxRowHeight    = 409
)

// ValidSheetName reports whether the name is acceptable to Excel: 1..31
// characters, none of []:*?/\, not starting or ending with an apostrophe,
// and not "History".
func ValidSheetName(name string) bool {
	if name == "" || len([]rune(name)) > MaxSheetNameLen || strings.EqualFold(name, "History") {
		return false
	}
	if strings.ContainsAny(name, `[]:*?/\`) {
		return false
	}
	if name[0] == '\'' || name[len(name)-1] == '\'' {
		return false
	}
	return true
}
