// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package wtest builds sample workbooks shared by the writer, reader and
// black-box tests.
package wtest

import (
	"bytes"
	"image"
	"image/png"
	"time"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

// SampleWorkbook builds a workbook that touches most features.
func SampleWorkbook() *model.Workbook {
	wb := model.New()
	wb.Properties.Title = "Sample"
	wb.Properties.Creator = "gexcel"
	wb.CustomProps = []model.CustomProperty{{Name: "Project", Value: "gexcel"}, {Name: "Count", Value: 3}}
	sh := wb.AddSheet("Data")
	bold := wb.Styles.Add(style.Style{Font: style.Font{Bold: true}, Fill: style.Solid(style.RGB("DDEBF7")), Border: style.Thin(style.Black), Alignment: style.Alignment{Horizontal: style.HAlignCenter}})
	money := wb.Styles.Add(style.Style{NumFmt: style.NumFmtThousands2})
	date := wb.Styles.Add(style.Style{NumFmt: style.NumFmtISODate})
	for i, h := range []string{"Item", "Qty", "Price", "Total", "Date"} {
		c := sh.Cell(1, i+1)
		c.Value = model.String(h)
		c.StyleID = bold
	}
	items := []struct {
		name  string
		qty   float64
		price float64
	}{{"Widget", 3, 9.99}, {"Gadget", 10, 24.5}, {"Gizmo", 1, 199}}
	for i, it := range items {
		r := i + 2
		sh.Cell(r, 1).Value = model.String(it.name)
		sh.Cell(r, 2).Value = model.Number(it.qty)
		sh.Cell(r, 3).Value = model.Number(it.price)
		sh.Cell(r, 3).StyleID = money
		t := sh.Cell(r, 4)
		t.Formula = &model.Formula{Text: "B" + ooxml.Int(r) + "*C" + ooxml.Int(r)}
		t.StyleID = money
		d := sh.Cell(r, 5)
		d.Value = model.Date(time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC), false)
		d.StyleID = date
	}
	sh.Cell(5, 1).Value = model.String("Total")
	sh.Cell(5, 4).Formula = &model.Formula{Text: "SUM(D2:D4)"}
	sh.Cell(5, 4).StyleID = money
	sh.Cell(6, 1).Value = model.Rich(model.RichRun{Text: "Rich "}, model.RichRun{Text: "bold", Font: &style.Font{Bold: true, Color: style.Red}})
	sh.Cell(6, 2).Value = model.Boolean(true)
	sh.Cell(6, 3).Value = model.Error(model.ErrNA)
	sh.Cell(7, 1).Value = model.String("  spaced  ")
	sh.Cell(7, 2).Value = model.Value{Kind: model.KindString, Str: "inline", Inline: true}
	sh.Merges = append(sh.Merges, ref.MustRange("A8:C8"))
	sh.Cell(8, 1).Value = model.String("merged")
	sh.SetCol(1, func(c *model.Col) { c.Width = 20 })
	sh.Row(1).Height = 24
	sh.View.Freeze = &model.Pane{Rows: 1}
	sh.Hyperlinks = append(sh.Hyperlinks, model.Hyperlink{Range: ref.MustRange("A2"), Target: "https://example.com", Tooltip: "Example"},
		model.Hyperlink{Range: ref.MustRange("A3"), Location: "'Data'!A1", Display: "top"})
	sh.AutoFilter = &model.AutoFilter{Range: ref.MustRange("A1:E4")}
	sh.DataValidations = append(sh.DataValidations, model.DataValidation{Ranges: []ref.Range{ref.MustRange("B2:B4")}, Type: model.ValidationWhole, Operator: OpBetween(), Formula1: "0", Formula2: "100", ShowErrorMessage: true, Error: "0..100"},
		model.DataValidation{Ranges: []ref.Range{ref.MustRange("A2:A4")}, Type: model.ValidationList, Formula1: `"Widget,Gadget,Gizmo"`})
	sh.ConditionalFormats = append(sh.ConditionalFormats, model.ConditionalFormat{Ranges: []ref.Range{ref.MustRange("D2:D4")}, Rules: []model.CondFmtRule{
		{Type: model.CfCellIs, Operator: model.CfOpGreater, Formulas: []string{"100"}, Style: style.Style{Font: style.Font{Color: style.RGB("9C0006")}, Fill: style.Solid(style.RGB("FFC7CE"))}},
		{Type: model.CfDataBar, DataBar: &model.DataBar{Min: model.CfValue{Type: model.CfvoMin}, Max: model.CfValue{Type: model.CfvoMax}, Color: style.RGB("638EC6")}},
		{Type: model.CfColorScale, ColorScale: &model.ColorScale{Values: []model.CfValue{{Type: model.CfvoMin}, {Type: model.CfvoMax}}, Colors: []style.Color{style.RGB("F8696B"), style.RGB("63BE7B")}}},
	}})
	sh.Comments = append(sh.Comments, model.Comment{Ref: ref.MustCell("B2"), Author: "Tester", Text: "A note"})
	sh.PageSetup = model.PageSetup{Orientation: model.OrientationLandscape, PaperSize: 9}
	sh.HeaderFooter = model.HeaderFooter{OddHeader: "&CSample", OddFooter: "&RPage &P of &N"}
	sh.PrintTitleRows = &[]ref.Range{ref.MustRange("1:1")}[0]
	sh.Protection = &model.SheetProtection{Password: "secret", FormatCells: true}
	// Second sheet: table, chart, image.
	sh2 := wb.AddSheet("Report")
	sh2.TabColor = style.Red
	for i, h := range []string{"Region", "Sales"} {
		sh2.Cell(1, i+1).Value = model.String(h)
	}
	for i, r := range []struct {
		n string
		v float64
	}{{"North", 120}, {"South", 80}, {"East", 95}, {"West", 60}} {
		sh2.Cell(i+2, 1).Value = model.String(r.n)
		sh2.Cell(i+2, 2).Value = model.Number(r.v)
	}
	sh2.Tables = append(sh2.Tables, &model.Table{Name: "Sales", Range: ref.MustRange("A1:B6"), TotalsRow: true, Columns: []model.TableColumn{{Name: "Region", TotalsRowLabel: "Total"}, {Name: "Sales", TotalsRowFunction: "sum"}}})
	sh2.Cell(6, 1).Value = model.String("Total")
	sh2.Cell(6, 2).Formula = &model.Formula{Text: "SUBTOTAL(109,Sales[Sales])"}
	ch := chart.New(chart.Column, chart.Series{NameRef: "Report!$B$1", Categories: "Report!$A$2:$A$5", Values: "Report!$B$2:$B$5"})
	ch.Title = &chart.Title{Text: "Sales by region"}
	ch.XAxis.Title = &chart.Title{Text: "Region"}
	ch.YAxis.Title = &chart.Title{Text: "Sales"}
	ch.YAxis.MajorGridlines = true
	ch.Anchor = chart.AnchorRef{FromRow: 8, FromCol: 1, ToRow: 24, ToCol: 8}
	sh2.Charts = append(sh2.Charts, ch)
	pie := chart.New(chart.Pie, chart.Series{Categories: "Report!$A$2:$A$5", Values: "Report!$B$2:$B$5", DataLabels: &chart.DataLabels{ShowPercent: true}})
	pie.Anchor = chart.AnchorRef{FromRow: 8, FromCol: 10, ToRow: 24, ToCol: 16}
	sh2.Charts = append(sh2.Charts, pie)
	sh2.Images = append(sh2.Images, &model.Image{Name: "dot", Format: "png", Data: PNGDot(), Anchor: model.Anchor{Kind: model.AnchorOneCell, From: model.CellOffset{Row: 1, Col: 5}, Width: 64 * model.EMUPerPixel, Height: 64 * model.EMUPerPixel}})
	sh2.Sparklines = append(sh2.Sparklines, model.SparklineGroup{Type: "line", Sparklines: []model.Sparkline{{Formula: "Report!B2:B5", Ref: ref.MustCell("D2")}}})
	wb.DefinedNames = append(wb.DefinedNames, model.DefinedName{Name: "TaxRate", RefersTo: "0.2"}, model.DefinedName{Name: "Prices", RefersTo: "Data!$C$2:$C$4", Local: true})
	// Dynamic array formula.
	sh2.Cell(1, 8).Formula = &model.Formula{Text: "SORT(B2:B5)", Kind: model.FormulaArray, Ref: ref.MustRange("H1:H4"), Dynamic: true}
	return wb
}

func OpBetween() model.DataValidationOperator { return model.OpBetween }

// pngDot is a small red PNG.
func PNGDot() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = 0xff
		if i%4 == 1 || i%4 == 2 {
			img.Pix[i] = 0
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
