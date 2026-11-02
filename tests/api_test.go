// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package tests

import (
	"bytes"
	"testing"
	"time"

	"github.com/gsoultan/gexcel"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/style"
)

// TestQuickStart is the README example, kept compiling and green.
func TestQuickStart(t *testing.T) {
	wb := gexcel.New()
	sh := wb.SheetAt(0)
	sh.SetName("Report")
	sh.SetRow("A1", "Item", "Qty", "Price", "Total")
	sh.SetStyle("A1:D1", style.Style{Font: style.Font{Bold: true}, Fill: style.Solid(style.RGB("DDEBF7"))})
	sh.SetRows("A2", [][]any{{"Widget", 4, 2.5}, {"Gadget", 2, 10}})
	for r := 2; r <= 3; r++ {
		sh.CellAt(r, 4).SetFormula("B" + itoa(r) + "*C" + itoa(r)).SetNumFmt(style.NumFmtCurrency)
	}
	sh.Cell("D4").SetFormula("SUM(D2:D3)").SetNumFmt(style.NumFmtCurrency).SetBold(true)
	sh.Cell("F1").Set(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	sh.SetColWidth(1, 1, 16)
	sh.Freeze(1, 0)
	if _, err := sh.AddTable("A1:D3", "Orders"); err != nil {
		t.Fatal(err)
	}
	if err := wb.Calculate(); err != nil {
		t.Fatal(err)
	}
	if v, _ := sh.Cell("D4").Float(); v != 30 {
		t.Fatalf("D4 = %v", v)
	}
	if s := sh.Cell("D4").Formatted(); s != "$30.00" {
		t.Fatalf("D4 formatted %q", s)
	}
	var buf bytes.Buffer
	if err := wb.Save(&buf); err != nil {
		t.Fatal(err)
	}
	back, err := gexcel.OpenBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	rs, _ := back.Sheet("Report")
	if rs.Cell("D4").Formula() != "SUM(D2:D3)" || rs.Cell("A2").String() != "Widget" {
		t.Fatal("round trip lost data")
	}
	if tm, ok := rs.Cell("F1").Time(); !ok || tm.Month() != time.May {
		t.Fatalf("F1 = %v", tm)
	}
	if rs.Cell("F1").Any().(time.Time).Day() != 1 {
		t.Fatal("F1 Any")
	}
	n := 0
	for row := range rs.Rows() {
		for range row.Cells() {
			n++
		}
	}
	if n != 14 {
		t.Fatalf("cells %d", n)
	}
	if v := rs.Value("A3"); v.Kind != model.KindString {
		t.Fatalf("A3 %+v", v)
	}
}

func itoa(i int) string { return string(rune('0' + i)) }

func TestStreamLargeSheet(t *testing.T) {
	wb := gexcel.New()
	sh := wb.SheetAt(0)
	const rows = 5000
	for r := 1; r <= rows; r++ {
		sh.CellAt(r, 1).Set(r)
		sh.CellAt(r, 2).Set("row " + itoa(r%10))
	}
	var buf bytes.Buffer
	if err := wb.Save(&buf); err != nil {
		t.Fatal(err)
	}
	st, err := gexcel.StreamReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	sum := 0.0
	n := 0
	for row, err := range st.Rows("Sheet1") {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if c := row.Lookup(1); c != nil {
			sum += c.Value.Num
		}
	}
	if n != rows || sum != float64(rows*(rows+1)/2) {
		t.Fatalf("rows %d sum %v", n, sum)
	}
}
