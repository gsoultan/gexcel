// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package gexcel_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gsoultan/gexcel"
	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/gexcel/writer/wtest"
)

func build(t *testing.T) gexcel.Workbook {
	t.Helper()
	wb := gexcel.New()
	sh := wb.SheetAt(0)
	if err := sh.SetName("Report"); err != nil {
		t.Fatal(err)
	}
	header := style.Style{Font: style.Font{Bold: true, Color: style.White}, Fill: style.Solid(style.RGB("4472C4")), Alignment: style.Alignment{Horizontal: style.HAlignCenter}}
	if err := sh.SetRow("A1", "Region", "Sales", "When", "Share"); err != nil {
		t.Fatal(err)
	}
	if err := sh.SetStyle("A1:D1", header); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC)
	rows := [][]any{{"North", 1200.5, when, 0.25}, {"South", 980, when.AddDate(0, 0, 1), 0.2}, {"East", 1510.25, when.AddDate(0, 0, 2), 0.31}, {"West", 1160, when.AddDate(0, 0, 3), 0.24}}
	if err := sh.SetRows("A2", rows); err != nil {
		t.Fatal(err)
	}
	for c := range sh.Range("B2:B5") {
		c.SetNumFmt("#,##0.00")
	}
	for c := range sh.Range("D2:D5") {
		c.SetNumFmt("0%")
	}
	sh.Cell("A6").Set("Total").SetBold(true)
	sh.Cell("B6").SetFormula("SUM(B2:B5)").SetNumFmt("#,##0.00")
	sh.Cell("E2").SetFormulaResult("B2*2", 2401)
	sh.Cell("F1").SetHyperlink("https://example.com", "Example")
	sh.Cell("F2").SetHyperlink("#Report!A1", "top")
	sh.Cell("A2").SetComment("Gembit", "Best region")
	sh.SetColWidth(1, 1, 18)
	sh.SetRowHeight(1, 24)
	sh.Freeze(1, 0)
	if err := sh.Merge("H1:I2"); err != nil {
		t.Fatal(err)
	}
	if _, err := sh.AddTable("A1:D5", "Sales", gexcel.TableStyle("TableStyleMedium9")); err != nil {
		t.Fatal(err)
	}
	if err := sh.AddDataValidation("D2:D5", model.DataValidation{Type: model.ValidationDecimal, Operator: model.OpBetween, Formula1: "0", Formula2: "1", ShowErrorMessage: true, Error: "0..1"}); err != nil {
		t.Fatal(err)
	}
	if err := sh.AddConditionalFormat("B2:B5", model.CondFmtRule{Type: model.CfCellIs, Operator: model.CfOpGreater, Formulas: []string{"1000"}, Style: style.Style{Font: style.Font{Bold: true, Color: style.RGB("9C0006")}}, HasStyle: true}); err != nil {
		t.Fatal(err)
	}
	ch := chart.New(chart.Column)
	ch.Title = &chart.Title{Text: "Sales"}
	ch.Series = []chart.Series{{Name: "Sales", Categories: "Report!$A$2:$A$5", Values: "Report!$B$2:$B$5"}}
	if err := sh.AddChart("H4", "N20", ch); err != nil {
		t.Fatal(err)
	}
	if err := sh.AddImage("H22", model.Image{Data: wtest.PNGDot()}); err != nil {
		t.Fatal(err)
	}
	sh.Protect("secret", gexcel.AllowSort, gexcel.AllowAutoFilter)
	sh.PageSetup().Orientation = model.OrientationLandscape
	sh.HeaderFooter().OddHeader = "&CReport"
	sh.SetPrintTitleRows(1, 1)
	wb.SetDefinedName("Total", "Report!$B$6")
	wb.Properties().Title = "API sample"

	second, err := wb.AddSheet("Notes")
	if err != nil {
		t.Fatal(err)
	}
	second.AppendRow("a", 1, true)
	second.AppendRow(nil, time.Duration(90*time.Minute), errors.New("#N/A"))
	second.Cell("A3").SetRich(model.RichRun{Text: "bold ", Font: &style.Font{Bold: true}}, model.RichRun{Text: "plain"})
	second.SetHidden(true)
	return wb
}

func TestAPIRoundTrip(t *testing.T) {
	wb := build(t)
	if err := wb.Validate(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := wb.Save(&buf); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GEXCEL_OUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "api.xlsx"), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := gexcel.OpenBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	sh, ok := got.Sheet("report")
	if !ok {
		t.Fatal("sheet Report missing")
	}
	if v := sh.Cell("A1").String(); v != "Region" {
		t.Errorf("A1 = %q", v)
	}
	if !sh.Cell("A1").Style().Font.Bold {
		t.Error("A1 not bold")
	}
	if f, _ := sh.Cell("B2").Float(); f != 1200.5 {
		t.Errorf("B2 = %v", f)
	}
	if s := sh.Cell("B2").Formatted(); s != "1,200.50" {
		t.Errorf("B2 formatted %q", s)
	}
	if tm, ok := sh.Cell("C2").Time(); !ok || !tm.Equal(time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("C2 time %v %v", tm, ok)
	}
	if _, isTime := sh.Cell("C2").Any().(time.Time); !isTime {
		t.Errorf("C2 Any = %T", sh.Cell("C2").Any())
	}
	if s := sh.Cell("D2").Formatted(); s != "25%" {
		t.Errorf("D2 formatted %q", s)
	}
	if f := sh.Cell("B6").Formula(); f != "SUM(B2:B5)" {
		t.Errorf("B6 formula %q", f)
	}
	if err := got.Calculate(); err != nil {
		t.Fatal(err)
	}
	if v, _ := sh.Cell("B6").Float(); v != 4850.75 {
		t.Errorf("B6 calculated %v", v)
	}
	if s := sh.Cell("B6").Formatted(); s != "4,850.75" {
		t.Errorf("B6 formatted %q", s)
	}
	if v, _ := sh.Cell("E2").Float(); v != 2401 {
		t.Errorf("E2 cached %v", v)
	}
	if h, ok := sh.Cell("F1").Hyperlink(); !ok || h.Target != "https://example.com" {
		t.Errorf("F1 hyperlink %+v", h)
	}
	if h, ok := sh.Cell("F2").Hyperlink(); !ok || h.Location != "Report!A1" {
		t.Errorf("F2 hyperlink %+v", h)
	}
	if c, ok := sh.Cell("A2").Comment(); !ok || c.Text != "Best region" || c.Author != "Gembit" {
		t.Errorf("A2 comment %+v", c)
	}
	if m := sh.Merges(); len(m) != 1 || m[0].String() != "H1:I2" {
		t.Errorf("merges %v", m)
	}
	if sh.MaxRow() != 6 || sh.MaxCol() != 6 {
		t.Errorf("used %d x %d", sh.MaxRow(), sh.MaxCol())
	}
	msh := sh.Model()
	if len(msh.Tables) != 1 || len(msh.Charts) != 1 || len(msh.Images) != 1 || len(msh.DataValidations) != 1 || len(msh.ConditionalFormats) != 1 || msh.Protection == nil || msh.View.Freeze == nil || msh.PrintTitleRows == nil {
		t.Errorf("features lost: %+v", msh)
	}
	if msh.PageSetup.Orientation != model.OrientationLandscape || msh.HeaderFooter.OddHeader != "&CReport" {
		t.Errorf("print setup lost")
	}
	names := got.DefinedNames()
	if len(names) != 1 || names[0].Name != "Total" || names[0].RefersTo != "Report!$B$6" {
		t.Errorf("names %+v", names)
	}
	notes, _ := got.Sheet("Notes")
	if !notes.Hidden() {
		t.Error("Notes should be hidden")
	}
	if b, ok := notes.Cell("C1").Bool(); !ok || !b {
		t.Error("C1 bool")
	}
	if v := notes.Cell("C2").Value(); v.Kind != model.KindError || v.Str != "#N/A" {
		t.Errorf("C2 = %+v", v)
	}
	if s := notes.Cell("B2").Formatted(); s != "1:30:00" {
		t.Errorf("duration %q", s)
	}
	if v := notes.Cell("A3").Value(); len(v.Rich) != 2 || v.Str != "bold plain" {
		t.Errorf("rich %+v", v)
	}
	n := 0
	for range notes.Cells() {
		n++
	}
	if n != 6 {
		t.Errorf("cells %d", n)
	}
}

func TestSheetManagement(t *testing.T) {
	wb := gexcel.New()
	if _, err := wb.AddSheet("Sheet1"); !errors.Is(err, gexcel.ErrSheetExists) {
		t.Errorf("dup: %v", err)
	}
	if _, err := wb.AddSheet("bad/name"); !errors.Is(err, gexcel.ErrInvalidName) {
		t.Errorf("invalid: %v", err)
	}
	a, _ := wb.AddSheet("A")
	b, _ := wb.AddSheet("B")
	a.Cell("A1").Set(1)
	a.Cell("B1").SetFormula("A!A1+1")
	wb.SetSheetDefinedName("B", "Local", "B!$A$1")
	if err := wb.MoveSheet("B", 0); err != nil {
		t.Fatal(err)
	}
	if wb.SheetAt(0).Name() != "B" || b.Index() != 0 {
		t.Errorf("order %v", wb.Sheets())
	}
	if dn := wb.DefinedNames(); dn[0].SheetIndex != 0 {
		t.Errorf("local name index %+v", dn)
	}
	cp, err := wb.CopySheet("A", "A2")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := cp.Cell("A1").Float(); v != 1 {
		t.Error("copy lost value")
	}
	cp.Cell("A1").Set(2)
	if v, _ := a.Cell("A1").Float(); v != 1 {
		t.Error("copy shares cells")
	}
	if err := a.SetName("Alpha"); err != nil {
		t.Fatal(err)
	}
	if f := a.Cell("B1").Formula(); f != "Alpha!A1+1" {
		t.Errorf("formula not renamed: %q", f)
	}
	if err := wb.RemoveSheet("Sheet1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := wb.Sheet("Sheet1"); ok {
		t.Error("still present")
	}
	for len(wb.Sheets()) > 1 {
		wb.RemoveSheet(wb.SheetAt(0).Name())
	}
	if err := wb.RemoveSheet(wb.SheetAt(0).Name()); !errors.Is(err, gexcel.ErrLastSheet) {
		t.Errorf("last: %v", err)
	}
	bad := wb.SheetAt(0).Cell("nope")
	if bad.Err() == nil {
		t.Error("expected ref error")
	}
	bad.Set(1) // must not panic
}

func TestOpenSample(t *testing.T) {
	var buf bytes.Buffer
	if err := gexcel.Wrap(wtest.SampleWorkbook()).Save(&buf); err != nil {
		t.Fatal(err)
	}
	wb, err := gexcel.OpenStream(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(wb.Sheets()) == 0 {
		t.Fatal("no sheets")
	}
	for _, sh := range wb.Sheets() {
		for r := range sh.Rows() {
			_ = r.Values()
		}
	}
}
