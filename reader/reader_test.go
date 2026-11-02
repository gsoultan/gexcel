// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader_test

import (
	"bytes"
	"testing"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/reader"
	"github.com/gsoultan/gexcel/writer"
	"github.com/gsoultan/gexcel/writer/wtest"
)

func roundTrip(t *testing.T, wb *model.Workbook, opts ...writer.Options) *model.Workbook {
	t.Helper()
	var buf bytes.Buffer
	if err := writer.Write(wb, &buf, opts...); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := reader.ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return got
}

func TestRoundTripSample(t *testing.T) {
	src := wtest.SampleWorkbook()
	got := roundTrip(t, src)
	if len(got.Sheets) != len(src.Sheets) {
		t.Fatalf("sheets: got %d want %d", len(got.Sheets), len(src.Sheets))
	}
	for i, ss := range src.Sheets {
		gs := got.Sheets[i]
		if gs.Name != ss.Name {
			t.Errorf("sheet %d name %q want %q", i, gs.Name, ss.Name)
		}
		for _, row := range ss.SortedRows() {
			for _, c := range row.SortedCells() {
				gc := gs.Lookup(c.Row, c.Col)
				if gc == nil {
					if c.Value.Kind == model.KindEmpty && c.Formula == nil {
						continue
					}
					t.Errorf("%s!R%dC%d missing", ss.Name, c.Row, c.Col)
					continue
				}
				if !gc.Value.Equal(c.Value) && !(c.Formula != nil) {
					t.Errorf("%s!R%dC%d value %+v want %+v", ss.Name, c.Row, c.Col, gc.Value, c.Value)
				}
				if c.Formula != nil && (gc.Formula == nil || gc.Formula.Text != c.Formula.Text) {
					t.Errorf("%s!R%dC%d formula %+v want %+v", ss.Name, c.Row, c.Col, gc.Formula, c.Formula)
				}
				if c.StyleID != 0 {
					ws, gsx := src.Styles.Get(c.StyleID), got.Styles.Get(gc.StyleID)
					if !ws.Equal(gsx) {
						t.Errorf("%s!R%dC%d style\n got %+v\nwant %+v", ss.Name, c.Row, c.Col, gsx, ws)
					}
				}
			}
		}
		if len(gs.Merges) != len(ss.Merges) {
			t.Errorf("%s merges %d want %d", ss.Name, len(gs.Merges), len(ss.Merges))
		}
		if len(gs.Hyperlinks) != len(ss.Hyperlinks) {
			t.Errorf("%s hyperlinks %d want %d", ss.Name, len(gs.Hyperlinks), len(ss.Hyperlinks))
		}
		if len(gs.DataValidations) != len(ss.DataValidations) {
			t.Errorf("%s data validations %d want %d", ss.Name, len(gs.DataValidations), len(ss.DataValidations))
		}
		if len(gs.ConditionalFormats) != len(ss.ConditionalFormats) {
			t.Errorf("%s conditional formats %d want %d", ss.Name, len(gs.ConditionalFormats), len(ss.ConditionalFormats))
		}
		if len(gs.Comments) != len(ss.Comments) {
			t.Errorf("%s comments %d want %d", ss.Name, len(gs.Comments), len(ss.Comments))
		}
		if len(gs.Tables) != len(ss.Tables) {
			t.Errorf("%s tables %d want %d", ss.Name, len(gs.Tables), len(ss.Tables))
		}
		if len(gs.Charts) != len(ss.Charts) {
			t.Errorf("%s charts %d want %d", ss.Name, len(gs.Charts), len(ss.Charts))
		}
		if len(gs.Images) != len(ss.Images) {
			t.Errorf("%s images %d want %d", ss.Name, len(gs.Images), len(ss.Images))
		}
		if len(gs.Sparklines) != len(ss.Sparklines) {
			t.Errorf("%s sparkline groups %d want %d", ss.Name, len(gs.Sparklines), len(ss.Sparklines))
		}
		if (gs.View.Freeze == nil) != (ss.View.Freeze == nil) {
			t.Errorf("%s freeze mismatch", ss.Name)
		}
		if (gs.Protection == nil) != (ss.Protection == nil) {
			t.Errorf("%s protection mismatch", ss.Name)
		}
		if (gs.AutoFilter == nil) != (ss.AutoFilter == nil) {
			t.Errorf("%s autofilter mismatch", ss.Name)
		}
		if gs.PageSetup.Orientation != ss.PageSetup.Orientation {
			t.Errorf("%s orientation %q want %q", ss.Name, gs.PageSetup.Orientation, ss.PageSetup.Orientation)
		}
		if gs.HeaderFooter.OddHeader != ss.HeaderFooter.OddHeader {
			t.Errorf("%s header %q want %q", ss.Name, gs.HeaderFooter.OddHeader, ss.HeaderFooter.OddHeader)
		}
		if (gs.PrintTitleRows == nil) != (ss.PrintTitleRows == nil) {
			t.Errorf("%s print titles mismatch", ss.Name)
		}
	}
	if len(got.DefinedNames) != len(src.DefinedNames) {
		t.Errorf("defined names %d want %d: %+v", len(got.DefinedNames), len(src.DefinedNames), got.DefinedNames)
	}
	if got.Properties.Title != src.Properties.Title || got.Properties.Creator != src.Properties.Creator {
		t.Errorf("properties %+v", got.Properties)
	}
	// Second trip must be stable.
	again := roundTrip(t, got)
	if len(again.Sheets) != len(got.Sheets) {
		t.Fatalf("second trip sheets %d", len(again.Sheets))
	}
}

func TestReadInlineStrings(t *testing.T) {
	src := wtest.SampleWorkbook()
	got := roundTrip(t, src, writer.Options{InlineStrings: true})
	c := got.Sheets[0].Lookup(1, 1)
	if c == nil || c.Value.Kind != model.KindString || !c.Value.Inline {
		t.Fatalf("A1 = %+v", c)
	}
}

func TestReadLimits(t *testing.T) {
	src := wtest.SampleWorkbook()
	var buf bytes.Buffer
	if err := writer.Write(src, &buf); err != nil {
		t.Fatal(err)
	}
	got, err := reader.ReadBytes(buf.Bytes(), reader.Options{MaxCells: 3})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range got.Sheets[0].Rows {
		n += len(r.Cells)
	}
	if n > 3 {
		t.Errorf("cells %d > limit", n)
	}
	if _, err := reader.ReadBytes([]byte("not a zip")); err == nil {
		t.Error("expected error for garbage")
	}
	if _, err := reader.ReadBytes([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0}); err != reader.ErrEncrypted {
		t.Errorf("expected ErrEncrypted, got %v", err)
	}
}

func FuzzRead(f *testing.F) {
	var buf bytes.Buffer
	if err := writer.Write(wtest.SampleWorkbook(), &buf); err != nil {
		f.Fatal(err)
	}
	f.Add(buf.Bytes())
	f.Fuzz(func(t *testing.T, data []byte) {
		wb, err := reader.ReadBytes(data, reader.Options{MaxPartSize: 1 << 22, MaxCells: 100000})
		if err != nil || wb == nil {
			return
		}
		var out bytes.Buffer
		_ = writer.Write(wb, &out)
	})
}

func TestStream(t *testing.T) {
	src := wtest.SampleWorkbook()
	var buf bytes.Buffer
	if err := writer.Write(src, &buf); err != nil {
		t.Fatal(err)
	}
	st, err := reader.OpenStreamBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if len(st.SheetNames()) != len(src.Sheets) {
		t.Fatalf("sheets %v", st.SheetNames())
	}
	want := src.Sheets[0]
	n := 0
	for row, err := range st.Rows(want.Name) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		for _, c := range row.SortedCells() {
			wc := want.Lookup(c.Row, c.Col)
			if wc == nil {
				t.Errorf("unexpected cell %d,%d", c.Row, c.Col)
				continue
			}
			if wc.Formula == nil && !wc.Value.Equal(c.Value) {
				t.Errorf("R%dC%d = %+v want %+v", c.Row, c.Col, c.Value, wc.Value)
			}
		}
	}
	if n != len(want.SortedRows()) {
		t.Errorf("rows %d want %d", n, len(want.SortedRows()))
	}
	// Early stop must not error.
	for _, err := range st.Rows(want.Name) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	for _, err := range st.Rows("nope") {
		if err != reader.ErrNoSheet {
			t.Errorf("expected ErrNoSheet, got %v", err)
		}
	}
}
