// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Command gexcel-dump prints the structure and cells of a workbook; handy
// for checking what gexcel reads from a file.
package main

import (
	"fmt"
	"os"

	"github.com/gsoultan/gexcel/reader"
)

func main() {
	wb, err := reader.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}
	fmt.Println("props:", wb.Properties.Creator, "names:", wb.DefinedNames, "date1904:", wb.Date1904, "theme:", wb.Theme.Colors[4])
	for _, sh := range wb.Sheets {
		fmt.Printf("== %s kind=%v state=%q rows=%d cols=%v merges=%v freeze=%v prot=%v hl=%d dv=%d cf=%d tables=%d charts=%d images=%d comments=%d titles=%v orient=%s header=%q\n", sh.Name, sh.Kind, sh.State, len(sh.Rows), sh.Cols, sh.Merges, sh.View.Freeze, sh.Protection != nil, len(sh.Hyperlinks), len(sh.DataValidations), len(sh.ConditionalFormats), len(sh.Tables), len(sh.Charts), len(sh.Images), len(sh.Comments), sh.PrintTitleRows, sh.PageSetup.Orientation, sh.HeaderFooter.OddHeader)
		for _, row := range sh.SortedRows() {
			for _, c := range row.SortedCells() {
				st := wb.Styles.Get(c.StyleID)
				f := ""
				if c.Formula != nil {
					f = " f=" + c.Formula.Text
				}
				fmt.Printf("  R%dC%d %v %s fmt=%q bold=%v fill=%v%s\n", c.Row, c.Col, c.Value.Kind, c.Value.Text(), st.NumFmt, st.Font.Bold, st.Fill.Foreground, f)
			}
		}
		for _, cf := range sh.ConditionalFormats {
			fmt.Printf("  CF %v %+v\n", cf.Ranges, cf.Rules[0].Type)
		}
		for _, ch := range sh.Charts {
			fmt.Printf("  chart %v title=%v series=%d anchor=%+v\n", ch.Type, ch.Title, len(ch.Series), ch.Anchor)
		}
		for _, t := range sh.Tables {
			fmt.Printf("  table %s %v cols=%d style=%s\n", t.DisplayName, t.Range, len(t.Columns), t.Style.Name)
		}
		for _, c := range sh.Comments {
			fmt.Printf("  comment %v %q by %q\n", c.Ref, c.Text, c.Author)
		}
		for _, d := range sh.DataValidations {
			fmt.Printf("  dv %v %s %s\n", d.Ranges, d.Type, d.Formula1)
		}
	}
	fmt.Println("preserved:", len(wb.Preserved))
}
