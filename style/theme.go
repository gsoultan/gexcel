// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package style

import (
	"fmt"
	"math"
	"strconv"
)

// Theme is the workbook theme: the twelve color scheme entries and the
// heading/body fonts. Raw keeps the theme1.xml part for round trips.
type Theme struct {
	Name      string
	Colors    [12]string // RRGGBB in Theme* order
	MajorFont string     // headings
	MinorFont string     // body (the default cell font)
	Raw       []byte     // theme part as read, or nil for the default theme
}

// DefaultTheme is the Office 2013+ theme Excel uses for new workbooks.
func DefaultTheme() *Theme {
	return &Theme{
		Name:      "Office Theme",
		Colors:    [12]string{"000000", "FFFFFF", "44546A", "E7E6E6", "4472C4", "ED7D31", "A5A5A5", "FFC000", "5B9BD5", "70AD47", "0563C1", "954F72"},
		MajorFont: "Calibri Light",
		MinorFont: "Calibri",
	}
}

// Indexed legacy palette (ECMA-376 18.8.27), entries 0..63; 64 and 65 are
// the system foreground and background.
var indexedPalette = [...]string{
	"000000", "FFFFFF", "FF0000", "00FF00", "0000FF", "FFFF00", "FF00FF", "00FFFF",
	"000000", "FFFFFF", "FF0000", "00FF00", "0000FF", "FFFF00", "FF00FF", "00FFFF",
	"800000", "008000", "000080", "808000", "800080", "008080", "C0C0C0", "808080",
	"9999FF", "993366", "FFFFCC", "CCFFFF", "660066", "FF8080", "0066CC", "CCCCFF",
	"000080", "FF00FF", "FFFF00", "00FFFF", "800080", "800000", "008080", "0000FF",
	"00CCFF", "CCFFFF", "CCFFCC", "FFFF99", "99CCFF", "FF99CC", "CC99FF", "FFCC99",
	"3366FF", "33CCCC", "99CC00", "FFCC00", "FF9900", "FF6600", "666699", "969696",
	"003366", "339966", "003300", "333300", "993300", "993366", "333399", "333333",
	"000000", "FFFFFF",
}

// IndexedRGB returns the RRGGBB value of a legacy palette index.
func IndexedRGB(i int) (string, bool) {
	if i < 0 || i >= len(indexedPalette) {
		return "", false
	}
	return indexedPalette[i], true
}

// Resolve converts any color to an "RRGGBB" string using the theme (nil
// means the default theme). Automatic and unset colors resolve to black.
func (t *Theme) Resolve(c Color) string {
	if t == nil {
		t = DefaultTheme()
	}
	switch c.Kind {
	case ColorRGB:
		if len(c.ARGB) == 8 {
			return c.ARGB[2:]
		}
		return c.ARGB
	case ColorTheme:
		if c.Theme < 0 || c.Theme >= len(t.Colors) {
			return "000000"
		}
		// Excel swaps dk1/lt1 and dk2/lt2 when mapping theme indexes to the
		// clrScheme order: index 0 is lt1 (window), 1 is dk1 (window text),
		// 2 is lt2, 3 is dk2.
		idx := c.Theme
		switch idx {
		case 0:
			idx = 1
		case 1:
			idx = 0
		case 2:
			idx = 3
		case 3:
			idx = 2
		}
		return ApplyTint(t.Colors[idx], c.Tint)
	case ColorIndexed:
		if rgb, ok := IndexedRGB(c.Indexed); ok {
			return rgb
		}
	}
	return "000000"
}

// ApplyTint lightens (tint > 0) or darkens (tint < 0) an RRGGBB color the
// way Excel does, in HLS space on the luminance channel.
func ApplyTint(rgb string, tint float64) string {
	if tint == 0 || len(rgb) != 6 {
		return rgb
	}
	r, g, b := hexByte(rgb[0:2]), hexByte(rgb[2:4]), hexByte(rgb[4:6])
	h, l, s := rgbToHLS(float64(r)/255, float64(g)/255, float64(b)/255)
	if tint < 0 {
		l = l * (1 + tint)
	} else {
		l = l*(1-tint) + tint
	}
	rr, gg, bb := hlsToRGB(h, l, s)
	return fmt.Sprintf("%02X%02X%02X", round8(rr*255), round8(gg*255), round8(bb*255))
}

func round8(v float64) uint8 {
	v = math.Round(v)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func hexByte(s string) uint8 {
	v, _ := strconv.ParseUint(s, 16, 8)
	return uint8(v)
}

func rgbToHLS(r, g, b float64) (h, l, s float64) {
	mx := max(r, g, b)
	mn := min(r, g, b)
	l = (mx + mn) / 2
	if mx == mn {
		return 0, l, 0
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, l, s
}

func hlsToRGB(h, l, s float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return hue(p, q, h+1.0/3), hue(p, q, h), hue(p, q, h-1.0/3)
}

func hue(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}
