// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package format

import (
	"testing"
	"time"
)

func TestSerial(t *testing.T) {
	cases := []struct {
		t time.Time
		s float64
	}{
		{time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), 1},
		{time.Date(1900, 2, 28, 0, 0, 0, 0, time.UTC), 59},
		{time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC), 61},
		{time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), 45306.5},
		{time.Date(2026, 11, 2, 6, 30, 0, 0, time.UTC), 46328.270833333336},
		{time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC), 0},
	}
	for _, c := range cases {
		if got := ToSerial(c.t, false); got != c.s {
			t.Errorf("ToSerial(%v)=%v want %v", c.t, got, c.s)
		}
		if got := FromSerial(c.s, false); !got.Equal(c.t) {
			t.Errorf("FromSerial(%v)=%v want %v", c.s, got, c.t)
		}
	}
	if got := ToSerial(time.Date(1904, 1, 2, 0, 0, 0, 0, time.UTC), true); got != 1 {
		t.Error("1904 system", got)
	}
	loc := time.FixedZone("x", 3600)
	if ToSerial(time.Date(2024, 1, 15, 12, 0, 0, 0, loc), false) != 45306.5 {
		t.Error("wall clock should be used")
	}
	if SerialToDuration(1.5) != 36*time.Hour || DurationToSerial(12*time.Hour) != 0.5 {
		t.Error("durations")
	}
}

func TestNumbers(t *testing.T) {
	cases := []struct {
		v    float64
		code string
		want string
	}{
		{1234.5678, "General", "1234.5678"},
		{0, "General", "0"},
		{1e15, "General", "1E+15"},
		{0.00001234, "General", "1.234E-05"},
		{1234.5678, "0", "1235"},
		{1234.5678, "0.00", "1234.57"},
		{1234.5678, "#,##0", "1,235"},
		{1234.5678, "#,##0.00", "1,234.57"},
		{-1234.5678, "#,##0.00", "-1,234.57"},
		{1234567.891, "#,##0.00", "1,234,567.89"},
		{0.256, "0%", "26%"},
		{0.256, "0.0%", "25.6%"},
		{5, "0.00", "5.00"},
		{5, "0.##", "5."},
		{5.5, "0.##", "5.5"},
		{0.5, "#.##", ".5"},
		{0.5, "0.00", "0.50"},
		{12, "000", "012"},
		{12, "0000.0", "0012.0"},
		{1234.5678, "0.00E+00", "1.23E+03"},
		{0.000123, "0.00E+00", "1.23E-04"},
		{1234567, "##0.0E+0", "1.2E+6"},
		{1234567, `#,##0,`, "1,235"},
		{1234567, `#,##0,,`, "1"},
		{1234567, `0.0,,"M"`, "1.2M"},
		{-5, "0;(0)", "(5)"},
		{-5, "0;[Red]-0;\"zero\"", "-5"},
		{0, "0;-0;\"zero\"", "zero"},
		{0, "0;-0", "0"},
		{5, "0;-0;\"zero\";@", "5"},
		{42, `"Total: "0`, "Total: 42"},
		{42, `0" units"`, "42 units"},
		{42, `0\ \k\g`, "42 kg"},
		{1234.5, `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`, " $1,234.50 "},
		{-1234.5, `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`, " $(1,234.50)"},
		{0, `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`, " $-   "},
		{1234.5, `[$€-407]#,##0.00`, "€1,234.50"},
		{1234.5, `#,##0.00 [$€-407]`, "1,234.50 €"},
		{150, "[>=100]\"big\";\"small\"", "big"},
		{50, "[>=100]\"big\";\"small\"", "small"},
		{5.75, "# ?/?", "5 3/4"},
		{0.5, "?/?", "1/2"},
		{5.333, "# ??/??", "5  1/3"},
		{3.125, "# ?/8", "3 1/8"},
		{5, "# ?/?", "5    "},
		{0.75, "# ?/?", "  3/4"},
		{2.5, "0.0", "2.5"},
		{2.55, "0.0", "2.6"},
		{1.005, "0.00", "1.01"},
		{1e11, "General", "1E+11"},
		{99999999999, "General", "99999999999"},
		{123456789012, "General", "1.23457E+11"},
		{3.14159265358979, "General", "3.141592654"},
		{-0.5, "0.00", "-0.50"},
		{1, `"Yes";"No";"Maybe"`, "Yes"},
		{-7, "0.0;;;", ""},
	}
	for _, c := range cases {
		if got := Number(c.v, c.code, false); got != c.want {
			t.Errorf("Number(%v, %q)=%q want %q", c.v, c.code, got, c.want)
		}
	}
}

func TestDates(t *testing.T) {
	d := time.Date(2024, 3, 5, 14, 7, 9, 0, time.UTC)
	cases := []struct{ code, want string }{
		{"yyyy-mm-dd", "2024-03-05"},
		{"m/d/yyyy", "3/5/2024"},
		{"mm-dd-yy", "03-05-24"},
		{"d-mmm-yy", "5-Mar-24"},
		{"mmmm d, yyyy", "March 5, 2024"},
		{"dddd", "Tuesday"},
		{"ddd d mmm", "Tue 5 Mar"},
		{"h:mm", "14:07"},
		{"hh:mm:ss", "14:07:09"},
		{"h:mm AM/PM", "2:07 PM"},
		{"h:mm:ss A/P", "2:07:09 P"},
		{"yyyy-mm-dd hh:mm:ss", "2024-03-05 14:07:09"},
		{"m/d/yy h:mm", "3/5/24 14:07"},
		{"mm:ss", "07:09"},
		{`yyyy"年"m"月"d"日"`, "2024年3月5日"},
		{"mmmmm", "M"},
		{"dd/mm/yyyy;@", "05/03/2024"},
		{"yy", "24"},
	}
	for _, c := range cases {
		if got := Time(d, c.code); got != c.want {
			t.Errorf("Time(%q)=%q want %q", c.code, got, c.want)
		}
	}
	// Elapsed time beyond 24 hours and fractional seconds.
	if got := Number(1.5+1.0/24, "[h]:mm:ss", false); got != "37:00:00" {
		t.Error("elapsed", got)
	}
	if got := Number(90.0/1440, "[mm]:ss", false); got != "90:00" {
		t.Error("elapsed minutes", got)
	}
	if got := Number(0.5+1.2345/86400, "ss.00", false); got != "01.23" {
		t.Error("fractional seconds", got)
	}
	if got := Number(0.5+1.2345/86400, "mm:ss.0", false); got != "00:01.2" {
		t.Error("fractional seconds", got)
	}
	// 59.7 seconds rounds up to the next minute in minute-precision formats.
	if got := Number(1.0/24+59.7/86400, "h:mm", false); got != "1:01" {
		t.Error("rounding", got)
	}
	if got := Number(-1, "yyyy-mm-dd", false); got != "#######" {
		t.Error("negative date", got)
	}
	if got := Number(0.5, "h:mm AM/PM", false); got != "12:00 PM" {
		t.Error("noon", got)
	}
	if got := Number(0, "h:mm AM/PM", false); got != "12:00 AM" {
		t.Error("midnight", got)
	}
}

func TestText(t *testing.T) {
	if got := Text("abc", "@"); got != "abc" {
		t.Error(got)
	}
	if got := Text("abc", `"Name: "@`); got != "Name: abc" {
		t.Error(got)
	}
	if got := Text("abc", "0.00"); got != "abc" {
		t.Error(got)
	}
	if got := Text("abc", `0;-0;0;"<"@">"`); got != "<abc>" {
		t.Error(got)
	}
	if got := Number(5, `0;-0;0;"<"@">"`, false); got != "5" {
		t.Error(got)
	}
	if Bool(true) != "TRUE" || Bool(false) != "FALSE" {
		t.Error("bool")
	}
	if !Parse("yyyy-mm-dd").IsDate() || Parse("0.00").IsDate() || !Parse("@;yyyy").IsDate() {
		t.Error("IsDate")
	}
}

func FuzzFormat(f *testing.F) {
	for _, s := range []string{"0.00", "#,##0", "yyyy-mm-dd h:mm:ss", "# ?/?", "[>5]0;[<0]-0;0", `"a"@`, "0.00E+00", "[h]:mm", "General"} {
		f.Add(s, 1234.5678)
	}
	f.Fuzz(func(t *testing.T, code string, v float64) {
		Number(v, code, false)
		Number(v, code, true)
		Text("x", code)
	})
}
