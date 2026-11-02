// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package style

import "testing"

func TestColors(t *testing.T) {
	if RGB("#ff0000").ARGB != "FFFF0000" || RGB("80FF0000").ARGB != "80FF0000" || !RGB("xyz").IsZero() || !RGB("12345").IsZero() {
		t.Error("RGB")
	}
	if RGBA(1, 2, 3, 255).ARGB != "FF010203" {
		t.Error("RGBA")
	}
	th := DefaultTheme()
	if th.Resolve(ThemeColor(ThemeAccent1, 0)) != "4472C4" || th.Resolve(ThemeColor(ThemeDark1, 0)) != "000000" ||
		th.Resolve(ThemeColor(ThemeLight1, 0)) != "FFFFFF" || th.Resolve(Indexed(2)) != "FF0000" || th.Resolve(RGB("ABCDEF")) != "ABCDEF" {
		t.Error("Resolve")
	}
	var nilTheme *Theme
	if nilTheme.Resolve(ThemeColor(ThemeAccent2, 0)) != "ED7D31" {
		t.Error("nil theme")
	}
	// Excel: accent1 "lighter 60%" (tint 0.5999) is B4C7E7, "lighter 80%" is D9E2F3.
	if got := ApplyTint("4472C4", 0.5999); got != "B4C7E7" && got != "B5C7E7" {
		t.Error("tint", got)
	}
	if got := ApplyTint("4472C4", 0.7999); got != "D9E2F3" && got != "DAE3F3" {
		t.Error("tint 80", got)
	}
	if got := ApplyTint("4472C4", -0.25); got != "2F5597" && got != "305496" && got != "2F5496" {
		t.Error("shade", got)
	}
	if ApplyTint("FFFFFF", -0.5) != "808080" && ApplyTint("FFFFFF", -0.5) != "7F7F7F" {
		t.Error("white shade", ApplyTint("FFFFFF", -0.5))
	}
}

func TestNumFmt(t *testing.T) {
	if id, ok := BuiltinNumFmtID(NumFmtPercent2); !ok || id != 10 {
		t.Error("builtin id")
	}
	if code, ok := BuiltinNumFmt(22); !ok || code != "m/d/yy h:mm" {
		t.Error("builtin code")
	}
	if _, ok := BuiltinNumFmt(100); ok {
		t.Error("unknown builtin")
	}
	dates := []string{"yyyy-mm-dd", "d-mmm", "h:mm AM/PM", "[h]:mm:ss", `"Date: "yyyy`, "m/d/yy h:mm", "mmss.0", "dd/mm/yyyy;@"}
	for _, d := range dates {
		if !IsDateFormat(d) {
			t.Errorf("IsDateFormat(%q)=false", d)
		}
	}
	notDates := []string{"General", "0.00", "#,##0", `"m"0`, `0 "months"`, "@", "[Red]0.0", `0\d`, "0%", `_("$"* #,##0.00_)`}
	for _, d := range notDates {
		if IsDateFormat(d) {
			t.Errorf("IsDateFormat(%q)=true", d)
		}
	}
}

func TestStyleMerge(t *testing.T) {
	base := Style{Font: Font{Name: "Arial", Size: 10}, NumFmt: "0"}
	over := Style{Font: Font{Bold: true, Color: Red}, Fill: Solid(Yellow), Border: Border{Bottom: Line{Style: LineThin}}}
	m := base.Merge(over)
	if m.Font.Name != "Arial" || m.Font.Size != 10 || !m.Font.Bold || m.Font.Color != Red || m.NumFmt != "0" ||
		m.Fill.Foreground != Yellow || m.Border.Bottom.Style != LineThin {
		t.Errorf("%+v", m)
	}
	if !(Style{}).IsZero() || m.IsZero() {
		t.Error("IsZero")
	}
	g := Style{Fill: Fill{Gradient: &Gradient{Degree: 90, Stops: []GradientStop{{0, Red}, {1, Blue}}}}}
	g2 := Style{Fill: Fill{Gradient: &Gradient{Degree: 90, Stops: []GradientStop{{0, Red}, {1, Blue}}}}}
	if !g.Equal(g2) || g.Key() != g2.Key() || g.Equal(m) {
		t.Error("gradient equality")
	}
}
