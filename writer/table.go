// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"fmt"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// writeTable writes a table part and returns its relationship id.
func (p *packer) writeTable(sheetPart string, sh *model.Sheet, t *model.Table) (string, error) {
	p.nTable++
	id := t.ID
	if id == 0 {
		id = p.nTable
	}
	name := fmt.Sprintf("xl/tables/table%d.xml", p.nTable)
	if len(t.Raw) > 0 {
		if err := p.pw.Add(name, ooxml.CTTable, t.Raw); err != nil {
			return "", err
		}
		return p.pw.Rel(sheetPart, ooxml.RelTable, ooxml.Relative(sheetPart, name), ""), nil
	}
	cols := t.Columns
	if len(cols) == 0 {
		// Derive the columns from the header row cells.
		for c := t.Range.From.Col; c <= t.Range.To.Col; c++ {
			nm := ""
			if cell := sh.Lookup(t.Range.From.Row, c); cell != nil {
				nm = cell.Value.Text()
			}
			if nm == "" {
				nm = fmt.Sprintf("Column%d", c-t.Range.From.Col+1)
			}
			cols = append(cols, model.TableColumn{Name: nm})
		}
	}
	seen := map[string]bool{}
	for i := range cols {
		if cols[i].ID == 0 {
			cols[i].ID = i + 1
		}
		k := strings.ToLower(cols[i].Name)
		if seen[k] {
			cols[i].Name = fmt.Sprintf("%s%d", cols[i].Name, i+1)
			k = strings.ToLower(cols[i].Name)
		}
		seen[k] = true
	}
	tname := t.Name
	if tname == "" {
		tname = fmt.Sprintf("Table%d", id)
	}
	dname := orDefault(t.DisplayName, tname)
	b := ooxml.NewBuilder()
	attrs := []string{"xmlns", ooxml.NSMain, "xmlns:mc", ooxml.NSMC, "mc:Ignorable", "xr xr3", "xmlns:xr", ooxml.NSXR, "xmlns:xr3", ooxml.NSXR3,
		"id", ooxml.Int(id), "name", tname, "displayName", dname, "ref", t.Range.String(), "comment", t.Comment}
	if t.HeaderRow != nil && !*t.HeaderRow {
		attrs = append(attrs, "headerRowCount", "0")
	}
	if t.TotalsRow {
		attrs = append(attrs, "totalsRowCount", ooxml.Int(max(t.TotalsRowCount, 1)))
	} else {
		attrs = append(attrs, "totalsRowShown", "0")
	}
	b.Start("table", attrs...)
	if !t.NoAutoFilter && (t.HeaderRow == nil || *t.HeaderRow) {
		af := t.AutoFilter
		if af == nil {
			r := t.Range
			if t.TotalsRow {
				r.To.Row--
			}
			af = &model.AutoFilter{Range: r}
		}
		autoFilterXML(b, af)
	}
	b.Start("tableColumns", "count", ooxml.Int(len(cols)))
	for _, c := range cols {
		ca := []string{"id", ooxml.Int(c.ID), "uniqueName", c.UniqueName, "name", c.Name, "totalsRowFunction", c.TotalsRowFunction, "totalsRowLabel", c.TotalsRowLabel,
			"queryTableFieldId", intAttr(c.QueryTableFieldID), "dataDxfId", intAttr(c.DataDxfID), "headerRowDxfId", intAttr(c.HeaderDxfID), "totalsRowDxfId", intAttr(c.TotalsDxfID)}
		if c.CalculatedColumnFormula == "" && c.TotalsRowFormula == "" {
			b.Empty("tableColumn", ca...)
			continue
		}
		b.Start("tableColumn", ca...)
		if c.CalculatedColumnFormula != "" {
			b.Elem("calculatedColumnFormula", strings.TrimPrefix(c.CalculatedColumnFormula, "="))
		}
		if c.TotalsRowFormula != "" {
			b.Elem("totalsRowFormula", strings.TrimPrefix(c.TotalsRowFormula, "="))
		}
		b.End()
	}
	b.End()
	st := t.Style
	if st.Name == "" {
		st = model.DefaultTableStyle()
	}
	b.Empty("tableStyleInfo", "name", st.Name, "showFirstColumn", ooxml.Bool(st.ShowFirstColumn), "showLastColumn", ooxml.Bool(st.ShowLastColumn),
		"showRowStripes", ooxml.Bool(st.ShowRowStripes), "showColumnStripes", ooxml.Bool(st.ShowColumnStripes))
	b.End()
	if err := p.pw.Add(name, ooxml.CTTable, b.Bytes()); err != nil {
		return "", err
	}
	return p.pw.Rel(sheetPart, ooxml.RelTable, ooxml.Relative(sheetPart, name), ""), nil
}
