// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package model

import (
	"github.com/gsoultan/gexcel/style"
)

// Xf is one cellXfs / cellStyleXfs entry: indexes into the component tables
// plus the parts stored inline.
type Xf struct {
	FontID, FillID, BorderID, NumFmtID int
	XfID                               int // cellStyleXfs index (cellXfs only)
	Alignment                          style.Alignment
	Protection                         style.Protection
	QuotePrefix                        bool
	PivotButton                        bool
	ApplyFont, ApplyFill, ApplyBorder  bool
	ApplyNumFmt, ApplyAlignment        bool
	ApplyProtection                    bool
}

// CellStyle is a named cell style (cellStyles entry).
type CellStyle struct {
	Name          string
	XfID          int // cellStyleXfs index
	BuiltinID     int // -1 when not built-in
	CustomBuiltin bool
	Hidden        bool
	ILevel        int
}

// Styles is the workbook style table. It deduplicates fonts, fills,
// borders, number formats and cell formats; cells refer to CellXfs by index.
type Styles struct {
	Fonts        []style.Font
	Fills        []style.Fill
	Borders      []style.Border
	NumFmts      map[int]string // custom number formats by id (>= 164)
	CellXfs      []Xf
	CellStyleXfs []Xf
	CellStyles   []CellStyle
	Dxfs         []style.Style
	// TableStyles are custom table styles (name -> raw XML), preserved.
	TableStyles       []RawXML
	DefaultTableStyle string
	DefaultPivotStyle string
	// Colors keeps a custom indexed palette (colors/indexedColors) when present.
	IndexedColors []string
	MRUColors     []string
	Extensions    []RawXML

	fontIdx   map[style.Font]int
	fillIdx   map[string]int
	borderIdx map[style.Border]int
	numFmtIdx map[string]int
	xfIdx     map[Xf]int
	dxfIdx    map[string]int
}

// NewStyles returns the style table of a new workbook: the default font,
// the two mandatory fills, an empty border, the Normal cell style and one
// default cell format.
func NewStyles() *Styles {
	s := &Styles{NumFmts: map[int]string{}}
	s.index()
	s.fontID(style.Font{Name: style.DefaultFontName, Size: style.DefaultFontSize, Color: style.ThemeColor(style.ThemeDark1, 0), Family: 2, Scheme: "minor"})
	s.Fills = append(s.Fills, style.Fill{}, style.Fill{Pattern: style.PatternGray125})
	s.fillIdx[fillKey(style.Fill{})] = 0
	s.fillIdx[fillKey(style.Fill{Pattern: style.PatternGray125})] = 1
	s.borderID(style.Border{})
	s.CellStyleXfs = append(s.CellStyleXfs, Xf{})
	s.CellStyles = append(s.CellStyles, CellStyle{Name: "Normal", BuiltinID: 0})
	s.xfID(Xf{})
	return s
}

// Reindex rebuilds the lookup maps after the tables were filled directly
// (the reader does this).
func (s *Styles) Reindex() {
	s.index()
	for i, f := range s.Fonts {
		if _, ok := s.fontIdx[f]; !ok {
			s.fontIdx[f] = i
		}
	}
	for i, f := range s.Fills {
		if _, ok := s.fillIdx[fillKey(f)]; !ok {
			s.fillIdx[fillKey(f)] = i
		}
	}
	for i, b := range s.Borders {
		if _, ok := s.borderIdx[b]; !ok {
			s.borderIdx[b] = i
		}
	}
	for id, code := range s.NumFmts {
		if _, ok := s.numFmtIdx[code]; !ok {
			s.numFmtIdx[code] = id
		}
	}
	for i, x := range s.CellXfs {
		if _, ok := s.xfIdx[x]; !ok {
			s.xfIdx[x] = i
		}
	}
	for i, d := range s.Dxfs {
		if _, ok := s.dxfIdx[d.Key()]; !ok {
			s.dxfIdx[d.Key()] = i
		}
	}
}

func (s *Styles) index() {
	s.fontIdx = map[style.Font]int{}
	s.fillIdx = map[string]int{}
	s.borderIdx = map[style.Border]int{}
	s.numFmtIdx = map[string]int{}
	s.xfIdx = map[Xf]int{}
	s.dxfIdx = map[string]int{}
	if s.NumFmts == nil {
		s.NumFmts = map[int]string{}
	}
}

func fillKey(f style.Fill) string { return (style.Style{Fill: f}).Key() }

// DefaultFont returns font 0.
func (s *Styles) DefaultFont() style.Font {
	if len(s.Fonts) == 0 {
		return style.Font{Name: style.DefaultFontName, Size: style.DefaultFontSize}
	}
	return s.Fonts[0]
}

// normalizeFont fills the default name and size so equal-looking fonts
// share an entry.
func (s *Styles) normalizeFont(f style.Font) style.Font {
	def := s.DefaultFont()
	if f.Name == "" {
		f.Name = def.Name
		if f.Scheme == "" && def.Scheme != "" {
			f.Scheme = def.Scheme
		}
		if f.Family == 0 {
			f.Family = def.Family
		}
	}
	if f.Size == 0 {
		f.Size = def.Size
	}
	if f.Color.IsZero() && f.Name == def.Name {
		f.Color = def.Color
	}
	return f
}

func (s *Styles) fontID(f style.Font) int {
	if id, ok := s.fontIdx[f]; ok {
		return id
	}
	s.Fonts = append(s.Fonts, f)
	s.fontIdx[f] = len(s.Fonts) - 1
	return len(s.Fonts) - 1
}

// FontID returns the index of the font, adding it when new.
func (s *Styles) FontID(f style.Font) int { return s.fontID(s.normalizeFont(f)) }

// FillID returns the index of the fill, adding it when new.
func (s *Styles) FillID(f style.Fill) int {
	if f.Gradient == nil && f.Pattern == style.PatternNone && !f.Foreground.IsZero() {
		// A color without a pattern means solid.
		f.Pattern = style.PatternSolid
	}
	k := fillKey(f)
	if id, ok := s.fillIdx[k]; ok {
		return id
	}
	s.Fills = append(s.Fills, f)
	s.fillIdx[k] = len(s.Fills) - 1
	return len(s.Fills) - 1
}

func (s *Styles) borderID(b style.Border) int {
	if id, ok := s.borderIdx[b]; ok {
		return id
	}
	s.Borders = append(s.Borders, b)
	s.borderIdx[b] = len(s.Borders) - 1
	return len(s.Borders) - 1
}

// BorderID returns the index of the border, adding it when new.
func (s *Styles) BorderID(b style.Border) int { return s.borderID(b) }

// NumFmtID returns the id of a number format code: a built-in id when the
// code is built in, otherwise a custom id (164+), added when new.
func (s *Styles) NumFmtID(code string) int {
	if code == "" || code == style.NumFmtGeneral {
		return 0
	}
	if id, ok := style.BuiltinNumFmtID(code); ok {
		return id
	}
	if id, ok := s.numFmtIdx[code]; ok {
		return id
	}
	id := style.FirstCustomNumFmtID
	for {
		if _, taken := s.NumFmts[id]; !taken {
			break
		}
		id++
	}
	s.NumFmts[id] = code
	s.numFmtIdx[code] = id
	return id
}

// NumFmtCode returns the code of a number format id ("" for General).
func (s *Styles) NumFmtCode(id int) string {
	if id == 0 {
		return ""
	}
	if code, ok := s.NumFmts[id]; ok {
		return code
	}
	if code, ok := style.BuiltinNumFmt(id); ok {
		return code
	}
	return ""
}

func (s *Styles) xfID(x Xf) int {
	if id, ok := s.xfIdx[x]; ok {
		return id
	}
	s.CellXfs = append(s.CellXfs, x)
	s.xfIdx[x] = len(s.CellXfs) - 1
	return len(s.CellXfs) - 1
}

// Add returns the cell format index for a style, adding every component
// that is new. The zero style maps to index 0.
func (s *Styles) Add(st style.Style) int {
	if st.IsZero() {
		return 0
	}
	x := Xf{
		FontID:      s.FontID(st.Font),
		FillID:      s.FillID(st.Fill),
		BorderID:    s.BorderID(st.Border),
		NumFmtID:    s.NumFmtID(st.NumFmt),
		Alignment:   st.Alignment,
		Protection:  st.Protection,
		QuotePrefix: st.QuotePrefix,
		PivotButton: st.PivotButton,
	}
	if st.Named != "" {
		x.XfID = s.NamedStyleXf(st.Named)
	}
	x.ApplyFont = x.FontID != 0
	x.ApplyFill = x.FillID != 0
	x.ApplyBorder = x.BorderID != 0
	x.ApplyNumFmt = x.NumFmtID != 0
	x.ApplyAlignment = !st.Alignment.IsZero()
	x.ApplyProtection = !st.Protection.IsZero()
	return s.xfID(x)
}

// Get returns the style of a cell format index. Out-of-range ids give the
// default style.
func (s *Styles) Get(id int) style.Style {
	if id < 0 || id >= len(s.CellXfs) {
		return style.Style{}
	}
	x := s.CellXfs[id]
	st := style.Style{
		Alignment:   x.Alignment,
		Protection:  x.Protection,
		QuotePrefix: x.QuotePrefix,
		PivotButton: x.PivotButton,
		NumFmt:      s.NumFmtCode(x.NumFmtID),
	}
	if x.FontID >= 0 && x.FontID < len(s.Fonts) {
		st.Font = s.Fonts[x.FontID]
	}
	if x.FillID >= 0 && x.FillID < len(s.Fills) {
		st.Fill = s.Fills[x.FillID]
	}
	if x.BorderID >= 0 && x.BorderID < len(s.Borders) {
		st.Border = s.Borders[x.BorderID]
	}
	if x.XfID > 0 {
		for _, cs := range s.CellStyles {
			if cs.XfID == x.XfID {
				st.Named = cs.Name
				break
			}
		}
	}
	return st
}

// NamedStyleXf returns the cellStyleXfs index of a named style, creating a
// plain entry (derived from Normal) when the name is unknown.
func (s *Styles) NamedStyleXf(name string) int {
	for _, cs := range s.CellStyles {
		if cs.Name == name {
			return cs.XfID
		}
	}
	s.CellStyleXfs = append(s.CellStyleXfs, Xf{})
	id := len(s.CellStyleXfs) - 1
	s.CellStyles = append(s.CellStyles, CellStyle{Name: name, XfID: id, BuiltinID: builtinStyleID(name)})
	return id
}

// AddNamedStyle defines (or redefines) a named cell style with the given
// formatting. Cells use it through Style.Named.
func (s *Styles) AddNamedStyle(name string, st style.Style) {
	x := Xf{
		FontID:     s.FontID(st.Font),
		FillID:     s.FillID(st.Fill),
		BorderID:   s.BorderID(st.Border),
		NumFmtID:   s.NumFmtID(st.NumFmt),
		Alignment:  st.Alignment,
		Protection: st.Protection,
	}
	for i, cs := range s.CellStyles {
		if cs.Name == name {
			s.CellStyleXfs[cs.XfID] = x
			s.CellStyles[i].BuiltinID = builtinStyleID(name)
			return
		}
	}
	s.CellStyleXfs = append(s.CellStyleXfs, x)
	s.CellStyles = append(s.CellStyles, CellStyle{Name: name, XfID: len(s.CellStyleXfs) - 1, BuiltinID: builtinStyleID(name)})
}

// NamedStyle returns the formatting of a named cell style.
func (s *Styles) NamedStyle(name string) (style.Style, bool) {
	for _, cs := range s.CellStyles {
		if cs.Name == name && cs.XfID < len(s.CellStyleXfs) {
			x := s.CellStyleXfs[cs.XfID]
			st := style.Style{Alignment: x.Alignment, Protection: x.Protection, NumFmt: s.NumFmtCode(x.NumFmtID)}
			if x.FontID < len(s.Fonts) {
				st.Font = s.Fonts[x.FontID]
			}
			if x.FillID < len(s.Fills) {
				st.Fill = s.Fills[x.FillID]
			}
			if x.BorderID < len(s.Borders) {
				st.Border = s.Borders[x.BorderID]
			}
			return st, true
		}
	}
	return style.Style{}, false
}

// Built-in cell style ids (ECMA-376 18.8.3), by name.
var builtinStyles = map[string]int{
	"Normal": 0, "RowLevel_1": 1, "ColLevel_1": 2, "Comma": 3, "Currency": 4, "Percent": 5,
	"Comma [0]": 6, "Currency [0]": 7, "Hyperlink": 8, "Followed Hyperlink": 9, "Note": 10,
	"Warning Text": 11, "Title": 15, "Heading 1": 16, "Heading 2": 17, "Heading 3": 18, "Heading 4": 19,
	"Input": 20, "Output": 21, "Calculation": 22, "Check Cell": 23, "Linked Cell": 24, "Total": 25,
	"Good": 26, "Bad": 27, "Neutral": 28, "Accent1": 29, "20% - Accent1": 30, "40% - Accent1": 31,
	"60% - Accent1": 32, "Accent2": 33, "20% - Accent2": 34, "40% - Accent2": 35, "60% - Accent2": 36,
	"Accent3": 37, "20% - Accent3": 38, "40% - Accent3": 39, "60% - Accent3": 40, "Accent4": 41,
	"20% - Accent4": 42, "40% - Accent4": 43, "60% - Accent4": 44, "Accent5": 45, "20% - Accent5": 46,
	"40% - Accent5": 47, "60% - Accent5": 48, "Accent6": 49, "20% - Accent6": 50, "40% - Accent6": 51,
	"60% - Accent6": 52, "Explanatory Text": 53,
}

func builtinStyleID(name string) int {
	if id, ok := builtinStyles[name]; ok {
		return id
	}
	return -1
}

// AddDxf returns the index of a differential format, adding it when new.
func (s *Styles) AddDxf(st style.Style) int {
	k := st.Key()
	if id, ok := s.dxfIdx[k]; ok {
		return id
	}
	s.Dxfs = append(s.Dxfs, st)
	s.dxfIdx[k] = len(s.Dxfs) - 1
	return len(s.Dxfs) - 1
}

// Dxf returns a differential format by index.
func (s *Styles) Dxf(id int) style.Style {
	if id < 0 || id >= len(s.Dxfs) {
		return style.Style{}
	}
	return s.Dxfs[id]
}
