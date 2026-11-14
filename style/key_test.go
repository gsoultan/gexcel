// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package style

import "testing"

// TestKeyDistinguishesFields checks that every field of a Style changes its key.
func TestKeyDistinguishesFields(t *testing.T) {
	base := Style{}
	variants := []Style{
		{Font: Font{Name: "Arial"}}, {Font: Font{Size: 11}}, {Font: Font{Bold: true}}, {Font: Font{Italic: true}},
		{Font: Font{Underline: "single"}}, {Font: Font{Strike: true}}, {Font: Font{Color: Color{Kind: ColorRGB, ARGB: "FF112233"}}},
		{Font: Font{Color: Color{Kind: ColorTheme, Theme: 3, Tint: 0.5}}}, {Font: Font{VertAlign: "superscript"}}, {Font: Font{Family: 2}},
		{Font: Font{Charset: 1}}, {Font: Font{Scheme: "minor"}},
		{Fill: Fill{Pattern: "solid"}}, {Fill: Fill{Foreground: Color{Kind: ColorRGB, ARGB: "FFFFFFFF"}}}, {Fill: Fill{Background: Color{Kind: ColorIndexed, Indexed: 64}}},
		{Fill: Fill{Gradient: &Gradient{Type: "linear", Degree: 90}}}, {Fill: Fill{Gradient: &Gradient{Stops: []GradientStop{{Position: 1}}}}},
		{Border: Border{Left: Line{Style: "thin"}}}, {Border: Border{Right: Line{Color: Color{Kind: ColorRGB, ARGB: "FF000000"}}}}, {Border: Border{Top: Line{Style: "thin"}}},
		{Border: Border{Bottom: Line{Style: "thin"}}}, {Border: Border{Diagonal: Line{Style: "thin"}}}, {Border: Border{DiagonalUp: true}}, {Border: Border{DiagonalDown: true}}, {Border: Border{Outline: true}},
		{Alignment: Alignment{Horizontal: "center"}}, {Alignment: Alignment{Vertical: "top"}}, {Alignment: Alignment{WrapText: true}}, {Alignment: Alignment{ShrinkToFit: true}},
		{Alignment: Alignment{JustifyLastLine: true}}, {Alignment: Alignment{Indent: 1}}, {Alignment: Alignment{RelativeIndent: 1}}, {Alignment: Alignment{TextRotation: 90}}, {Alignment: Alignment{ReadingOrder: 2}},
		{Protection: Protection{Unlocked: true}}, {Protection: Protection{Hidden: true}}, {NumFmt: "0.00"}, {Named: "Heading 1"}, {QuotePrefix: true}, {PivotButton: true},
	}
	seen := map[string]int{base.Key(): -1}
	for i, v := range variants {
		k := v.Key()
		if j, dup := seen[k]; dup {
			t.Errorf("variant %d has the same key as variant %d: %q", i, j, k)
		}
		seen[k] = i
		if v.Key() != k {
			t.Errorf("variant %d: key not stable", i)
		}
	}
}
