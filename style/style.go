// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package style holds the value types that describe how a cell looks: font,
// fill, border, alignment, protection and number format, combined in Style.
//
// All types are plain values; the zero value of every part means "the
// workbook default" (Calibri 11, no fill, no border, general alignment,
// locked, General number format), so a Style literal only needs the parts
// that differ. Styles are compared by value, and the writer deduplicates
// them into the OOXML style tables.
package style

import (
	"fmt"
	"strings"
)

// ColorKind says how a Color is specified.
type ColorKind uint8

const (
	ColorNone    ColorKind = iota // no color (automatic / inherited)
	ColorRGB                      // explicit ARGB
	ColorTheme                    // theme color index with tint
	ColorIndexed                  // legacy palette index
	ColorAuto                     // "auto" (system window text / background)
)

// Color is a spreadsheet color: an ARGB value, a theme color, or a legacy
// indexed color. The zero value is "no color".
type Color struct {
	Kind    ColorKind
	ARGB    string  // "FFRRGGBB", upper case, for ColorRGB
	Theme   int     // theme index 0..11 for ColorTheme
	Tint    float64 // -1..1 lightening (positive) or darkening (negative)
	Indexed int     // palette index for ColorIndexed
}

// RGB returns an opaque color from "RRGGBB", "#RRGGBB" or "AARRGGBB".
// Invalid input yields the zero Color.
func RGB(hex string) Color {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	switch len(hex) {
	case 6:
		hex = "FF" + hex
	case 8:
	default:
		return Color{}
	}
	for _, c := range hex {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return Color{}
		}
	}
	return Color{Kind: ColorRGB, ARGB: strings.ToUpper(hex)}
}

// RGBA returns a color from channel values.
func RGBA(r, g, b, a uint8) Color {
	return Color{Kind: ColorRGB, ARGB: fmt.Sprintf("%02X%02X%02X%02X", a, r, g, b)}
}

// ThemeColor returns a theme color (see the Theme* constants) with a tint.
func ThemeColor(index int, tint float64) Color {
	return Color{Kind: ColorTheme, Theme: index, Tint: tint}
}

// Indexed returns a legacy palette color (0..65).
func Indexed(i int) Color { return Color{Kind: ColorIndexed, Indexed: i} }

// Auto returns the automatic color.
func Auto() Color { return Color{Kind: ColorAuto} }

// IsZero reports whether the color is unset.
func (c Color) IsZero() bool { return c.Kind == ColorNone }

// Theme color indexes as SpreadsheetML numbers them: 0 is Light 1 (the
// window background), 1 is Dark 1 (window text), then Light 2, Dark 2 and
// the six accents. The theme XML lists dk1 before lt1; Theme.Resolve maps
// between the two orders.
const (
	ThemeLight1 = iota
	ThemeDark1
	ThemeLight2
	ThemeDark2
	ThemeAccent1
	ThemeAccent2
	ThemeAccent3
	ThemeAccent4
	ThemeAccent5
	ThemeAccent6
	ThemeHyperlink
	ThemeFollowedHyperlink
)

// Common colors.
var (
	Black  = RGB("000000")
	White  = RGB("FFFFFF")
	Red    = RGB("FF0000")
	Green  = RGB("00B050")
	Blue   = RGB("0070C0")
	Yellow = RGB("FFFF00")
	Gray   = RGB("808080")
)

// Underline styles.
type Underline string

const (
	UnderlineNone             Underline = ""
	UnderlineSingle           Underline = "single"
	UnderlineDouble           Underline = "double"
	UnderlineSingleAccounting Underline = "singleAccounting"
	UnderlineDoubleAccounting Underline = "doubleAccounting"
)

// VertAlign is the font vertical alignment (superscript/subscript).
type VertAlign string

const (
	VertAlignBaseline    VertAlign = ""
	VertAlignSuperscript VertAlign = "superscript"
	VertAlignSubscript   VertAlign = "subscript"
)

// Font describes a cell or rich-text run font. Name "" and Size 0 mean the
// workbook default font (Calibri 11 unless the theme says otherwise).
type Font struct {
	Name      string
	Size      float64 // points
	Bold      bool
	Italic    bool
	Underline Underline
	Strike    bool
	Outline   bool
	Shadow    bool
	Condense  bool
	Extend    bool
	Color     Color
	VertAlign VertAlign
	Family    int    // 0 unknown, 1 roman, 2 swiss, 3 modern, 4 script, 5 decorative
	Charset   int    // 0 ANSI, 1 default, 2 symbol, ... (-1 means unset)
	Scheme    string // "major", "minor" or ""
}

// DefaultFontName and DefaultFontSize are the Office defaults.
const (
	DefaultFontName = "Calibri"
	DefaultFontSize = 11.0
)

// IsZero reports whether the font is the default.
func (f Font) IsZero() bool { return f == Font{} }

// PatternType is a fill pattern.
type PatternType string

// Fill patterns.
const (
	PatternNone            PatternType = ""
	PatternSolid           PatternType = "solid"
	PatternMediumGray      PatternType = "mediumGray"
	PatternDarkGray        PatternType = "darkGray"
	PatternLightGray       PatternType = "lightGray"
	PatternDarkHorizontal  PatternType = "darkHorizontal"
	PatternDarkVertical    PatternType = "darkVertical"
	PatternDarkDown        PatternType = "darkDown"
	PatternDarkUp          PatternType = "darkUp"
	PatternDarkGrid        PatternType = "darkGrid"
	PatternDarkTrellis     PatternType = "darkTrellis"
	PatternLightHorizontal PatternType = "lightHorizontal"
	PatternLightVertical   PatternType = "lightVertical"
	PatternLightDown       PatternType = "lightDown"
	PatternLightUp         PatternType = "lightUp"
	PatternLightGrid       PatternType = "lightGrid"
	PatternLightTrellis    PatternType = "lightTrellis"
	PatternGray125         PatternType = "gray125"
	PatternGray0625        PatternType = "gray0625"
)

// GradientType selects linear or path (rectangular) gradients.
type GradientType string

const (
	GradientLinear GradientType = ""
	GradientPath   GradientType = "path"
)

// GradientStop is one color stop of a gradient fill.
type GradientStop struct {
	Position float64 // 0..1
	Color    Color
}

// Gradient describes a gradient fill.
type Gradient struct {
	Type   GradientType
	Degree float64 // linear: angle in degrees
	// Path gradients: the inner rectangle as fractions of the cell.
	Left, Right, Top, Bottom float64
	Stops                    []GradientStop
}

// Fill is a cell background. For a solid fill set Pattern to PatternSolid
// and Foreground to the color (Excel's convention); Background is the
// second color of a pattern. A non-nil Gradient takes precedence.
type Fill struct {
	Pattern    PatternType
	Foreground Color
	Background Color
	Gradient   *Gradient
}

// Solid returns a solid fill of the color.
func Solid(c Color) Fill { return Fill{Pattern: PatternSolid, Foreground: c} }

// IsZero reports whether the fill is empty.
func (f Fill) IsZero() bool {
	return f.Pattern == PatternNone && f.Foreground.IsZero() && f.Background.IsZero() && f.Gradient == nil
}

// LineStyle is a border line style.
type LineStyle string

// Border line styles.
const (
	LineNone             LineStyle = ""
	LineThin             LineStyle = "thin"
	LineMedium           LineStyle = "medium"
	LineThick            LineStyle = "thick"
	LineDashed           LineStyle = "dashed"
	LineDotted           LineStyle = "dotted"
	LineDouble           LineStyle = "double"
	LineHair             LineStyle = "hair"
	LineMediumDashed     LineStyle = "mediumDashed"
	LineDashDot          LineStyle = "dashDot"
	LineMediumDashDot    LineStyle = "mediumDashDot"
	LineDashDotDot       LineStyle = "dashDotDot"
	LineMediumDashDotDot LineStyle = "mediumDashDotDot"
	LineSlantDashDot     LineStyle = "slantDashDot"
)

// Line is one border edge.
type Line struct {
	Style LineStyle
	Color Color
}

// Border holds the edges of a cell.
type Border struct {
	Left, Right, Top, Bottom Line
	Diagonal                 Line
	DiagonalUp, DiagonalDown bool
	Outline                  bool // Excel sets outline="1" on most borders; cosmetic
}

// All returns a border with the same line on the four edges.
func All(l Line) Border { return Border{Left: l, Right: l, Top: l, Bottom: l} }

// Thin returns a thin border in the color on all four sides.
func Thin(c Color) Border { return All(Line{Style: LineThin, Color: c}) }

// IsZero reports whether no edge is set.
func (b Border) IsZero() bool { return b == Border{} }

// HAlign is the horizontal alignment.
type HAlign string

const (
	HAlignGeneral          HAlign = ""
	HAlignLeft             HAlign = "left"
	HAlignCenter           HAlign = "center"
	HAlignRight            HAlign = "right"
	HAlignFill             HAlign = "fill"
	HAlignJustify          HAlign = "justify"
	HAlignCenterContinuous HAlign = "centerContinuous"
	HAlignDistributed      HAlign = "distributed"
)

// VAlign is the vertical alignment. Excel's default is bottom.
type VAlign string

const (
	VAlignDefault     VAlign = ""
	VAlignTop         VAlign = "top"
	VAlignCenter      VAlign = "center"
	VAlignBottom      VAlign = "bottom"
	VAlignJustify     VAlign = "justify"
	VAlignDistributed VAlign = "distributed"
)

// Alignment holds the cell alignment settings.
type Alignment struct {
	Horizontal      HAlign
	Vertical        VAlign
	WrapText        bool
	ShrinkToFit     bool
	JustifyLastLine bool
	Indent          int
	RelativeIndent  int
	TextRotation    int // 0..180; 255 = vertical text
	ReadingOrder    int // 0 context, 1 left-to-right, 2 right-to-left
}

// IsZero reports whether the alignment is the default.
func (a Alignment) IsZero() bool { return a == Alignment{} }

// Protection controls what sheet protection does to the cell. Cells are
// locked by default, so the zero value means locked and visible.
type Protection struct {
	Unlocked bool // the cell may be edited on a protected sheet
	Hidden   bool // the formula is hidden on a protected sheet
}

// IsZero reports whether the protection is the default.
func (p Protection) IsZero() bool { return p == Protection{} }

// Style is the full cell format. Every part defaults to the workbook default
// when zero, so literals only name what differs:
//
//	style.Style{Font: style.Font{Bold: true}, NumFmt: style.NumFmtPercent2}
type Style struct {
	Font       Font
	Fill       Fill
	Border     Border
	Alignment  Alignment
	Protection Protection
	// NumFmt is the number format code ("0.00", "yyyy-mm-dd", ...). Use the
	// NumFmt* constants for the built-in formats, or any custom code.
	NumFmt string
	// Named is the cell style this format derives from ("Normal", "Heading
	// 1", a custom name). Empty means Normal.
	Named string
	// QuotePrefix marks text that was typed with a leading apostrophe.
	QuotePrefix bool
	// PivotButton is set on pivot table header cells.
	PivotButton bool
}

// IsZero reports whether the style is the workbook default.
func (s Style) IsZero() bool {
	return s.Font.IsZero() && s.Fill.IsZero() && s.Border.IsZero() && s.Alignment.IsZero() &&
		s.Protection.IsZero() && s.NumFmt == "" && s.Named == "" && !s.QuotePrefix && !s.PivotButton
}

// Equal compares two styles, including gradient stops.
func (s Style) Equal(o Style) bool {
	if s.Font != o.Font || s.Border != o.Border || s.Alignment != o.Alignment || s.Protection != o.Protection ||
		s.NumFmt != o.NumFmt || s.Named != o.Named || s.QuotePrefix != o.QuotePrefix || s.PivotButton != o.PivotButton {
		return false
	}
	return s.Fill.Equal(o.Fill)
}

// Equal compares fills, including gradients.
func (f Fill) Equal(o Fill) bool {
	if f.Pattern != o.Pattern || f.Foreground != o.Foreground || f.Background != o.Background {
		return false
	}
	if (f.Gradient == nil) != (o.Gradient == nil) {
		return false
	}
	if f.Gradient == nil {
		return true
	}
	g, h := *f.Gradient, *o.Gradient
	if g.Type != h.Type || g.Degree != h.Degree || g.Left != h.Left || g.Right != h.Right || g.Top != h.Top ||
		g.Bottom != h.Bottom || len(g.Stops) != len(h.Stops) {
		return false
	}
	for i := range g.Stops {
		if g.Stops[i] != h.Stops[i] {
			return false
		}
	}
	return true
}

// Key returns a string that identifies the style's value, for maps.
func (s Style) Key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%+v|%+v|%+v|%+v|%+v|%s|%s|%t|%t", s.Font, s.Border, s.Alignment, s.Protection, s.Fill.Pattern, s.NumFmt, s.Named, s.QuotePrefix, s.PivotButton)
	fmt.Fprintf(&b, "|%+v|%+v", s.Fill.Foreground, s.Fill.Background)
	if s.Fill.Gradient != nil {
		fmt.Fprintf(&b, "|%+v", *s.Fill.Gradient)
	}
	return b.String()
}

// Merge returns s with every non-zero part of o applied on top (a
// differential style, as conditional formatting and table styles use).
func (s Style) Merge(o Style) Style {
	if !o.Font.IsZero() {
		s.Font = mergeFont(s.Font, o.Font)
	}
	if !o.Fill.IsZero() {
		s.Fill = o.Fill
	}
	if !o.Border.IsZero() {
		s.Border = mergeBorder(s.Border, o.Border)
	}
	if !o.Alignment.IsZero() {
		s.Alignment = o.Alignment
	}
	if !o.Protection.IsZero() {
		s.Protection = o.Protection
	}
	if o.NumFmt != "" {
		s.NumFmt = o.NumFmt
	}
	if o.Named != "" {
		s.Named = o.Named
	}
	s.QuotePrefix = s.QuotePrefix || o.QuotePrefix
	return s
}

func mergeFont(a, b Font) Font {
	if b.Name != "" {
		a.Name = b.Name
	}
	if b.Size != 0 {
		a.Size = b.Size
	}
	a.Bold = a.Bold || b.Bold
	a.Italic = a.Italic || b.Italic
	a.Strike = a.Strike || b.Strike
	a.Outline = a.Outline || b.Outline
	a.Shadow = a.Shadow || b.Shadow
	a.Condense = a.Condense || b.Condense
	a.Extend = a.Extend || b.Extend
	if b.Underline != "" {
		a.Underline = b.Underline
	}
	if !b.Color.IsZero() {
		a.Color = b.Color
	}
	if b.VertAlign != "" {
		a.VertAlign = b.VertAlign
	}
	if b.Family != 0 {
		a.Family = b.Family
	}
	if b.Charset != 0 {
		a.Charset = b.Charset
	}
	if b.Scheme != "" {
		a.Scheme = b.Scheme
	}
	return a
}

func mergeBorder(a, b Border) Border {
	if b.Left.Style != "" {
		a.Left = b.Left
	}
	if b.Right.Style != "" {
		a.Right = b.Right
	}
	if b.Top.Style != "" {
		a.Top = b.Top
	}
	if b.Bottom.Style != "" {
		a.Bottom = b.Bottom
	}
	if b.Diagonal.Style != "" {
		a.Diagonal = b.Diagonal
		a.DiagonalUp, a.DiagonalDown = b.DiagonalUp, b.DiagonalDown
	}
	return a
}
