// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ref

import "testing"

func TestColNames(t *testing.T) {
	cases := map[int]string{1: "A", 26: "Z", 27: "AA", 52: "AZ", 53: "BA", 702: "ZZ", 703: "AAA", 16384: "XFD"}
	for n, want := range cases {
		if got := ColName(n); got != want {
			t.Errorf("ColName(%d)=%q want %q", n, got, want)
		}
		if got := ColNumber(want); got != n {
			t.Errorf("ColNumber(%q)=%d want %d", want, got, n)
		}
	}
	if ColNumber("XFE") != 0 || ColNumber("") != 0 || ColNumber("A1") != 0 {
		t.Error("invalid column names accepted")
	}
}

func TestCells(t *testing.T) {
	c, err := ParseCell("$B$7")
	if err != nil || c.Row != 7 || c.Col != 2 || !c.AbsRow || !c.AbsCol {
		t.Fatalf("ParseCell: %+v %v", c, err)
	}
	if c.String() != "$B$7" {
		t.Error(c.String())
	}
	if NewCell(3, 4).String() != "D3" {
		t.Error(NewCell(3, 4).String())
	}
	for _, bad := range []string{"", "A", "1", "A0", "XFE1", "A1048577", "A1B", "$$A1", "a-1"} {
		if _, err := ParseCell(bad); err == nil {
			t.Errorf("ParseCell(%q) accepted", bad)
		}
	}
	if c, err := ParseCell("xfd1048576"); err != nil || c.Col != MaxCols || c.Row != MaxRows {
		t.Error("lower-case max cell", err)
	}
	if got := NewCell(5, 5).R1C1(NewCell(2, 7)); got != "R[3]C[-2]" {
		t.Error(got)
	}
	if got := NewCell(5, 5).Abs().R1C1(NewCell(2, 7)); got != "R5C5" {
		t.Error(got)
	}
	rc, err := ParseR1C1("R[3]C[-2]", NewCell(2, 7))
	if err != nil || rc != NewCell(5, 5) {
		t.Error(rc, err)
	}
	rc, err = ParseR1C1("r2c", NewCell(10, 10))
	if err != nil || rc.Row != 2 || !rc.AbsRow || rc.Col != 10 || rc.AbsCol {
		t.Error(rc, err)
	}
}

func TestRanges(t *testing.T) {
	r := MustRange("C3:A1")
	if r.String() != "A1:C3" || r.Rows() != 3 || r.Cols() != 3 || r.Size() != 9 {
		t.Error(r)
	}
	if MustRange("B2").String() != "B2" || !MustRange("B2").IsCell() {
		t.Error("single cell")
	}
	wc := MustRange("$A:C")
	if !wc.WholeCols || wc.From.Row != 1 || wc.To.Row != MaxRows || wc.String() != "$A:C" {
		t.Errorf("%+v %s", wc, wc)
	}
	wr := MustRange("2:5")
	if !wr.WholeRows || wr.From.Col != 1 || wr.To.Col != MaxCols || wr.String() != "2:5" {
		t.Errorf("%+v", wr)
	}
	if !r.Contains(2, 2) || r.Contains(4, 1) {
		t.Error("contains")
	}
	i, ok := r.Intersect(MustRange("B2:D4"))
	if !ok || i.String() != "B2:C3" {
		t.Error(i, ok)
	}
	if _, ok := r.Intersect(MustRange("D4:E5")); ok {
		t.Error("should not intersect")
	}
	if r.Union(MustRange("E5")).String() != "A1:E5" {
		t.Error("union")
	}
	n := 0
	r.Each(func(row, col int) bool { n++; return n < 4 })
	if n != 4 {
		t.Error("Each stop", n)
	}
	for _, bad := range []string{"A1:", ":B2", "A:1", "1:A", "A1:B2:C3", "A1 B2"} {
		if _, err := ParseRange(bad); err == nil {
			t.Errorf("ParseRange(%q) accepted", bad)
		}
	}
	rs, err := ParseRanges("A1:B2 D4")
	if err != nil || len(rs) != 2 || FormatRanges(rs) != "A1:B2 D4" {
		t.Error(rs, err)
	}
}

func TestAreas(t *testing.T) {
	a, err := ParseArea("'My ''Sheet'''!$A$1:$B$2")
	if err != nil || a.Sheet != "My 'Sheet'" || a.Range.String() != "$A$1:$B$2" {
		t.Fatal(a, err)
	}
	if a.String() != "'My ''Sheet'''!$A$1:$B$2" {
		t.Error(a.String())
	}
	a, err = ParseArea("Sheet1!A1")
	if err != nil || a.Sheet != "Sheet1" || a.Range.String() != "A1" {
		t.Fatal(a, err)
	}
	if QuoteSheet("Sheet1") != "Sheet1" || QuoteSheet("1st") != "'1st'" || QuoteSheet("A1") != "'A1'" ||
		QuoteSheet("R1") != "'R1'" || QuoteSheet("Data 2024") != "'Data 2024'" || QuoteSheet("Ünïcode") != "Ünïcode" {
		t.Error("QuoteSheet")
	}
	as, err := ParseAreas("Sheet1!$A$1:$B$2,Sheet1!$D$4")
	if err != nil || len(as) != 2 || as[1].Range.String() != "$D$4" {
		t.Error(as, err)
	}
	if _, err := ParseArea("'Unterminated!A1"); err == nil {
		t.Error("unterminated quote accepted")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"A1", "$A$1:B2", "A:C", "2:5", "'x''y'!A1", "R[1]C[-1]"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if r, err := ParseRange(s); err == nil {
			if !r.Valid() {
				t.Fatalf("invalid range parsed from %q: %+v", s, r)
			}
			if r2, err := ParseRange(r.String()); err != nil || r2 != r {
				t.Fatalf("round trip %q -> %q: %+v %v", s, r.String(), r2, err)
			}
		}
		if a, err := ParseArea(s); err == nil {
			if a2, err := ParseArea(a.String()); err != nil || a2 != a {
				t.Fatalf("area round trip %q -> %q", s, a.String())
			}
		}
		ParseR1C1(s, NewCell(5, 5))
	})
}
