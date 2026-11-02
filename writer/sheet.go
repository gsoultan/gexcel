// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

// sheetWriter writes one worksheet part and its sub-parts.
type sheetWriter struct {
	p     *packer
	sh    *model.Sheet
	part  string
	index int

	drawingRel   string
	vmlRel       string
	commentsPart string
	tableRels    []string
	linkRels     map[int]string // hyperlink index -> rel id
}

// prepare writes the sub-parts (drawing, comments, tables) and registers
// their relationships so the sheet XML can refer to them.
func (w *sheetWriter) prepare() error {
	sh := w.sh
	if len(sh.Images) > 0 || len(sh.Charts) > 0 || len(sh.Shapes) > 0 {
		relID, err := w.p.writeDrawing(w.part, sh)
		if err != nil {
			return err
		}
		w.drawingRel = relID
	}
	if len(sh.Comments) > 0 {
		vmlRel, err := w.p.writeComments(w.part, w.index, sh)
		if err != nil {
			return err
		}
		w.vmlRel = vmlRel
	}
	for _, t := range sh.Tables {
		relID, err := w.p.writeTable(w.part, sh, t)
		if err != nil {
			return err
		}
		w.tableRels = append(w.tableRels, relID)
	}
	w.linkRels = map[int]string{}
	for i, h := range sh.Hyperlinks {
		if h.Target != "" {
			w.linkRels[i] = w.p.pw.Rel(w.part, ooxml.RelHyperlink, h.Target, "External")
		}
	}
	for _, pt := range sh.PivotTables {
		if pt.Table == nil {
			continue
		}
		if !w.p.pw.Has(pt.Table.Name) {
			if err := w.p.pw.Add(pt.Table.Name, pt.Table.ContentType, pt.Table.Data); err != nil {
				return err
			}
			if err := w.p.writeRawRelated(pt.Table.Name, pt.Table); err != nil {
				return err
			}
		}
		w.p.pw.Rel(w.part, ooxml.RelPivotTable, ooxml.Relative(w.part, pt.Table.Name), "")
	}
	return nil
}

func (w *sheetWriter) writeHead(out io.Writer) error {
	sh := w.sh
	b := ooxml.NewBuilder()
	b.Start("worksheet", "xmlns", ooxml.NSMain, "xmlns:r", ooxml.NSRel, "xmlns:mc", ooxml.NSMC, "mc:Ignorable", "x14ac xr xr2 xr3",
		"xmlns:x14ac", ooxml.NSX14ac, "xmlns:xr", ooxml.NSXR, "xmlns:xr2", ooxml.NSXR2, "xmlns:xr3", ooxml.NSXR3)
	// sheetPr
	prAttrs := []string{"codeName", sh.CodeName, "filterMode", boolAttr(sh.FilterMode), "published", boolAttr(sh.Published),
		"transitionEvaluation", boolAttr(sh.TransitionEval)}
	if sh.EnableCondFmtCalc != nil {
		prAttrs = append(prAttrs, "enableFormatConditionsCalculation", ooxml.Bool(*sh.EnableCondFmtCalc))
	}
	hasPr := !sh.TabColor.IsZero() || sh.FitToPage || sh.SummaryBelow != nil || sh.SummaryRight != nil || sh.ApplyStyles
	if hasPr || hasAttr(prAttrs) {
		b.Start("sheetPr", prAttrs...)
		colorXML(b, "tabColor", sh.TabColor)
		if sh.SummaryBelow != nil || sh.SummaryRight != nil || sh.ApplyStyles {
			oa := []string{"applyStyles", boolAttr(sh.ApplyStyles)}
			if sh.SummaryBelow != nil {
				oa = append(oa, "summaryBelow", ooxml.Bool(*sh.SummaryBelow))
			}
			if sh.SummaryRight != nil {
				oa = append(oa, "summaryRight", ooxml.Bool(*sh.SummaryRight))
			}
			b.Empty("outlinePr", oa...)
		}
		if sh.FitToPage {
			b.Empty("pageSetUpPr", "fitToPage", "1")
		}
		b.End()
	}
	// dimension
	if r, ok := sh.UsedRange(); ok {
		b.Empty("dimension", "ref", r.String())
	} else if sh.Dimension.Valid() {
		b.Empty("dimension", "ref", sh.Dimension.String())
	}
	w.viewsXML(b)
	// sheetFormatPr
	fa := []string{}
	if sh.DefaultColWidth > 0 {
		fa = append(fa, "defaultColWidth", ooxml.Float(sh.DefaultColWidth))
	} else {
		fa = append(fa, "baseColWidth", "10")
	}
	if sh.DefaultRowHeight > 0 {
		fa = append(fa, "defaultRowHeight", ooxml.Float(sh.DefaultRowHeight))
	} else {
		fa = append(fa, "defaultRowHeight", "15")
	}
	fa = append(fa, "customHeight", boolAttr(sh.CustomRowHeight), "zeroHeight", boolAttr(sh.ZeroHeight), "thickTop", boolAttr(sh.ThickTop),
		"thickBottom", boolAttr(sh.ThickBottom), "outlineLevelRow", intAttr(outlineLevelRow(sh)), "outlineLevelCol", intAttr(outlineLevelCol(sh)), "x14ac:dyDescent", "0.25")
	b.Empty("sheetFormatPr", fa...)
	if len(sh.Cols) > 0 {
		b.Start("cols")
		for _, c := range sh.Cols {
			attrs := []string{"min", ooxml.Int(c.Min), "max", ooxml.Int(c.Max)}
			if c.Width > 0 {
				attrs = append(attrs, "width", ooxml.Float(c.Width))
			} else {
				attrs = append(attrs, "width", ooxml.Float(orWidth(sh.DefaultColWidth)))
			}
			attrs = append(attrs, "style", intAttr(c.StyleID), "hidden", boolAttr(c.Hidden), "bestFit", boolAttr(c.BestFit),
				"customWidth", boolAttr(c.CustomWidth || c.Width > 0), "phonetic", boolAttr(c.Phonetic), "outlineLevel", intAttr(c.OutlineLevel), "collapsed", boolAttr(c.Collapsed))
			b.Empty("col", attrs...)
		}
		b.End()
	}
	b.WriteString("<sheetData>")
	_, err := out.Write(b.Bytes())
	return err
}

func orWidth(w float64) float64 {
	if w > 0 {
		return w
	}
	return 9.140625
}

func hasAttr(attrs []string) bool {
	for i := 1; i < len(attrs); i += 2 {
		if attrs[i] != "" {
			return true
		}
	}
	return false
}

func outlineLevelRow(sh *model.Sheet) int {
	m := sh.OutlineLevelRow
	for _, r := range sh.Rows {
		if r.OutlineLevel > m {
			m = r.OutlineLevel
		}
	}
	return m
}

func outlineLevelCol(sh *model.Sheet) int {
	m := sh.OutlineLevelCol
	for _, c := range sh.Cols {
		if c.OutlineLevel > m {
			m = c.OutlineLevel
		}
	}
	return m
}

func (w *sheetWriter) viewsXML(b *ooxml.Builder) {
	sh := w.sh
	v := sh.View
	b.Start("sheetViews")
	attrs := []string{"showGridLines", invBool(v.HideGridLines), "showRowColHeaders", invBool(v.HideHeadings), "showZeros", invBool(v.HideZeros),
		"showFormulas", boolAttr(v.ShowFormulas), "showRuler", invBool(v.HideRuler), "showOutlineSymbols", invBool(v.HideOutline),
		"showWhiteSpace", invBool(v.HideWhiteSpace), "rightToLeft", boolAttr(v.RightToLeft), "tabSelected", boolAttr(v.TabSelected || w.index == w.p.wb.View.ActiveTab),
		"windowProtection", boolAttr(v.WindowProtection), "view", v.Type, "topLeftCell", v.TopLeftCell,
		"zoomScale", intAttr(v.ZoomScale), "zoomScaleNormal", intAttr(v.ZoomNormal), "zoomScalePageLayoutView", intAttr(v.ZoomPageLayout), "zoomScaleSheetLayoutView", intAttr(v.ZoomPageBreak),
		"colorId", intAttr(v.GridLineColor), "workbookViewId", "0"}
	b.Start("sheetView", attrs...)
	if f := v.Freeze; f != nil && (f.Rows > 0 || f.Cols > 0) {
		topLeft := f.TopLeftCell
		if topLeft == "" {
			topLeft = ref.NewCell(f.Rows+1, f.Cols+1).String()
		}
		active := f.ActivePane
		if active == "" {
			switch {
			case f.Rows > 0 && f.Cols > 0:
				active = "bottomRight"
			case f.Rows > 0:
				active = "bottomLeft"
			default:
				active = "topRight"
			}
		}
		b.Empty("pane", "xSplit", intAttr(f.Cols), "ySplit", intAttr(f.Rows), "topLeftCell", topLeft, "activePane", active, "state", "frozen")
		if len(v.Selections) == 0 {
			b.Empty("selection", "pane", active, "activeCell", topLeft, "sqref", topLeft)
		}
	} else if s := v.Split; s != nil {
		active := s.ActivePane
		if active == "" {
			active = "bottomRight"
		}
		b.Empty("pane", "xSplit", ooxml.Float(s.XSplit), "ySplit", ooxml.Float(s.YSplit), "topLeftCell", s.TopLeftCell, "activePane", active, "state", "split")
	}
	for _, sel := range v.Selections {
		b.Empty("selection", "pane", sel.Pane, "activeCell", sel.ActiveCell, "sqref", ref.FormatRanges(sel.Ranges))
	}
	b.End().End()
}

// writeRows writes the <row> elements from the model.
func (w *sheetWriter) writeRows(out io.Writer) error {
	bw := bufio.NewWriterSize(out, 64<<10)
	var b ooxml.Builder
	for _, r := range w.sh.SortedRows() {
		b.Reset()
		w.rowXML(&b, r, r.SortedCells())
		if _, err := bw.Write(b.Bytes()); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// rowXML writes one row with its cells in column order.
func (w *sheetWriter) rowXML(b *ooxml.Builder, r *model.Row, cells []*model.Cell) {
	if len(cells) == 0 && r.IsEmpty() {
		return
	}
	attrs := []string{"r", ooxml.Int(r.Index)}
	if len(cells) > 0 {
		attrs = append(attrs, "spans", fmt.Sprintf("%d:%d", cells[0].Col, cells[len(cells)-1].Col))
	}
	attrs = append(attrs, "s", intAttr(r.StyleID), "customFormat", boolAttr(r.StyleID != 0 || r.CustomFormat))
	if r.Height > 0 {
		attrs = append(attrs, "ht", ooxml.Float(r.Height), "customHeight", "1")
	}
	attrs = append(attrs, "hidden", boolAttr(r.Hidden), "outlineLevel", intAttr(r.OutlineLevel), "collapsed", boolAttr(r.Collapsed),
		"thickTop", boolAttr(r.ThickTop), "thickBot", boolAttr(r.ThickBottom), "ph", boolAttr(r.ShowPhonetic))
	if len(cells) == 0 {
		b.Empty("row", attrs...)
		return
	}
	b.Start("row", attrs...)
	for _, c := range cells {
		w.cellXML(b, c)
	}
	b.End()
}

// cellXML writes one <c> element.
func (w *sheetWriter) cellXML(b *ooxml.Builder, c *model.Cell) {
	if c.Value.IsEmpty() && c.Formula == nil && c.StyleID == 0 {
		return
	}
	attrs := []string{"r", ref.Cell{Row: c.Row, Col: c.Col}.String(), "s", intAttr(c.StyleID)}
	v := c.Value
	typ := ""
	switch v.Kind {
	case model.KindString:
		if c.Formula != nil {
			typ = "str"
		} else if v.Inline || w.p.opts.InlineStrings {
			typ = "inlineStr"
		} else {
			typ = "s"
		}
	case model.KindBool:
		typ = "b"
	case model.KindError:
		typ = "e"
	}
	attrs = append(attrs, "t", typ)
	if c.Formula != nil && c.Formula.Dynamic {
		attrs = append(attrs, "cm", "1")
		w.p.hasDynamic = true
	}
	if c.ValueMetadata > 0 {
		attrs = append(attrs, "vm", ooxml.Int(c.ValueMetadata))
	}
	attrs = append(attrs, "ph", boolAttr(c.ShowPhonetic))
	if v.IsEmpty() && c.Formula == nil {
		b.Empty("c", attrs...)
		return
	}
	b.Start("c", attrs...)
	if f := c.Formula; f != nil {
		fa := []string{}
		switch f.Kind {
		case model.FormulaArray:
			fa = append(fa, "t", "array", "ref", rangeAttr(f.Ref, c))
		case model.FormulaShared:
			fa = append(fa, "t", "shared", "si", ooxml.Int(f.SharedIndex))
			if f.Master {
				fa = append(fa, "ref", rangeAttr(f.Ref, c))
			}
		case model.FormulaDataTable:
			fa = append(fa, "t", "dataTable", "ref", rangeAttr(f.Ref, c), "dt2D", boolAttr(f.DataTable2D), "dtr", boolAttr(f.DataTableRow), "r1", f.DataTableRef1, "r2", f.DataTableRef2)
		}
		fa = append(fa, "ca", boolAttr(f.CalcAlways))
		text := formula.AddPrefixes(strings.TrimPrefix(f.Text, "="))
		if f.Kind == model.FormulaShared && !f.Master {
			b.Empty("f", fa...)
		} else {
			b.Elem("f", text, fa...)
		}
	}
	switch v.Kind {
	case model.KindString:
		switch typ {
		case "s":
			b.Elem("v", ooxml.Int(w.p.sst.add(v)))
		case "inlineStr":
			b.WriteString("<is>")
			b.WriteString(siXML(v))
			b.WriteString("</is>")
		default:
			b.Elem("v", ooxml.EscapeUnicode(v.Str))
		}
	case model.KindNumber:
		b.Elem("v", ooxml.Float(v.Num))
	case model.KindBool:
		b.Elem("v", ooxml.Bool(v.Bool))
	case model.KindError:
		b.Elem("v", v.Str)
	}
	b.End()
}

func rangeAttr(r ref.Range, c *model.Cell) string {
	if !r.Valid() {
		return ref.Cell{Row: c.Row, Col: c.Col}.String()
	}
	return r.String()
}

// writeTail writes everything after sheetData.
func (w *sheetWriter) writeTail(out io.Writer) error {
	sh := w.sh
	b := &ooxml.Builder{}
	b.WriteString("</sheetData>")
	if p := sh.Protection; p != nil {
		w.protectionXML(b, p)
	}
	if af := sh.AutoFilter; af != nil {
		autoFilterXML(b, af)
	}
	if len(sh.Merges) > 0 {
		b.Start("mergeCells", "count", ooxml.Int(len(sh.Merges)))
		for _, m := range sh.Merges {
			b.Empty("mergeCell", "ref", m.String())
		}
		b.End()
	}
	var x14cf []func(*ooxml.Builder)
	priority := 1
	for _, cf := range sh.ConditionalFormats {
		w.condFmtXML(b, cf, &priority, &x14cf)
	}
	if n := len(sh.DataValidations); n > 0 {
		b.Start("dataValidations", "count", ooxml.Int(n))
		for _, dv := range sh.DataValidations {
			dataValidationXML(b, dv)
		}
		b.End()
	}
	if len(sh.Hyperlinks) > 0 {
		b.Start("hyperlinks")
		for i, h := range sh.Hyperlinks {
			b.Empty("hyperlink", "ref", h.Range.String(), "r:id", w.linkRels[i], "location", h.Location, "tooltip", h.Tooltip, "display", h.Display)
		}
		b.End()
	}
	po := sh.PrintOptions
	if po.GridLines || po.Headings || po.HorizontalCentered || po.VerticalCentered {
		b.Empty("printOptions", "gridLines", boolAttr(po.GridLines), "gridLinesSet", boolAttr(po.GridLinesSet), "headings", boolAttr(po.Headings),
			"horizontalCentered", boolAttr(po.HorizontalCentered), "verticalCentered", boolAttr(po.VerticalCentered))
	}
	m := sh.PageMargins
	if m.IsZero() {
		m = model.DefaultPageMargins()
	}
	b.Empty("pageMargins", "left", ooxml.Float(m.Left), "right", ooxml.Float(m.Right), "top", ooxml.Float(m.Top), "bottom", ooxml.Float(m.Bottom), "header", ooxml.Float(m.Header), "footer", ooxml.Float(m.Footer))
	ps := sh.PageSetup
	if ps != (model.PageSetup{}) {
		attrs := []string{"paperSize", intAttr(ps.PaperSize), "scale", intAttr(ps.Scale), "firstPageNumber", intAttr(ps.FirstPageNumber),
			"fitToWidth", fitAttr(ps.FitToWidth), "fitToHeight", fitAttr(ps.FitToHeight), "pageOrder", ps.PageOrder, "orientation", string(ps.Orientation),
			"usePrinterDefaults", boolAttr(ps.UsePrinterDefaults), "blackAndWhite", boolAttr(ps.BlackAndWhite), "draft", boolAttr(ps.Draft),
			"cellComments", ps.CellComments, "useFirstPageNumber", boolAttr(ps.UseFirstPage), "errors", ps.Errors,
			"horizontalDpi", intAttr(ps.HorizontalDPI), "verticalDpi", intAttr(ps.VerticalDPI), "copies", intAttr(ps.Copies)}
		b.Empty("pageSetup", attrs...)
	}
	if hf := sh.HeaderFooter; !hf.IsZero() {
		b.Start("headerFooter", "differentOddEven", boolAttr(hf.DifferentOddEven), "differentFirst", boolAttr(hf.DifferentFirst),
			"scaleWithDoc", map[bool]string{true: "0", false: ""}[hf.NoScaleWithDoc], "alignWithMargins", map[bool]string{true: "0", false: ""}[hf.NoAlignWithMargins])
		for _, e := range [...]struct{ tag, v string }{{"oddHeader", hf.OddHeader}, {"oddFooter", hf.OddFooter}, {"evenHeader", hf.EvenHeader}, {"evenFooter", hf.EvenFooter}, {"firstHeader", hf.FirstHeader}, {"firstFooter", hf.FirstFooter}} {
			if e.v != "" {
				b.Elem(e.tag, e.v)
			}
		}
		b.End()
	}
	breaksXML(b, "rowBreaks", sh.RowBreaks, ref.MaxCols-1)
	breaksXML(b, "colBreaks", sh.ColBreaks, ref.MaxRows-1)
	if len(sh.IgnoredErrors) > 0 {
		b.Start("ignoredErrors")
		for _, ie := range sh.IgnoredErrors {
			b.Empty("ignoredError", "sqref", ref.FormatRanges(ie.Ranges), "evalError", boolAttr(ie.EvalError), "twoDigitTextYear", boolAttr(ie.TwoDigitTextYear),
				"numberStoredAsText", boolAttr(ie.NumberStoredAsTxt), "formula", boolAttr(ie.Formula), "formulaRange", boolAttr(ie.FormulaRange),
				"unlockedFormula", boolAttr(ie.UnlockedFormula), "emptyCellReference", boolAttr(ie.EmptyCellRef), "listDataValidation", boolAttr(ie.ListDataValidation), "calculatedColumn", boolAttr(ie.CalculatedColumn))
		}
		b.End()
	}
	if w.drawingRel != "" {
		b.Empty("drawing", "r:id", w.drawingRel)
	}
	if w.vmlRel != "" {
		b.Empty("legacyDrawing", "r:id", w.vmlRel)
	}
	if len(w.tableRels) > 0 {
		b.Start("tableParts", "count", ooxml.Int(len(w.tableRels)))
		for _, id := range w.tableRels {
			b.Empty("tablePart", "r:id", id)
		}
		b.End()
	}
	// Extensions: x14 conditional formatting, sparklines, preserved ones.
	if len(x14cf) > 0 || len(sh.Sparklines) > 0 || len(sh.Extensions) > 0 {
		b.Start("extLst")
		if len(x14cf) > 0 {
			b.Start("ext", "uri", "{78C0D931-6437-407d-A8EE-F0AAD7539E65}", "xmlns:x14", ooxml.NSX14)
			b.Start("x14:conditionalFormattings")
			for _, fn := range x14cf {
				fn(b)
			}
			b.End().End()
		}
		if len(sh.Sparklines) > 0 {
			sparklinesXML(b, sh.Sparklines)
		}
		for _, e := range sh.Extensions {
			b.Write(e.Data)
		}
		b.End()
	}
	b.WriteString("</worksheet>")
	_, err := out.Write(b.Bytes())
	return err
}

func fitAttr(v int) string {
	if v == 0 {
		return ""
	}
	return ooxml.Int(v)
}

func breaksXML(b *ooxml.Builder, tag string, breaks []model.PageBreak, maxDef int) {
	if len(breaks) == 0 {
		return
	}
	manual := 0
	for _, br := range breaks {
		if br.Manual {
			manual++
		}
	}
	b.Start(tag, "count", ooxml.Int(len(breaks)), "manualBreakCount", ooxml.Int(manual))
	for _, br := range breaks {
		mx := br.Max
		if mx == 0 {
			mx = maxDef
		}
		b.Empty("brk", "id", ooxml.Int(br.Index), "max", ooxml.Int(mx), "man", boolAttr(br.Manual))
	}
	b.End()
}

func (w *sheetWriter) protectionXML(b *ooxml.Builder, p *model.SheetProtection) {
	attrs := []string{}
	switch {
	case p.Password != "":
		alg, hash, salt, spin := hashPassword(p.Password)
		attrs = append(attrs, "algorithmName", alg, "hashValue", hash, "saltValue", salt, "spinCount", ooxml.Int(spin))
	case p.HashValue != "":
		attrs = append(attrs, "algorithmName", p.Algorithm, "hashValue", p.HashValue, "saltValue", p.SaltValue, "spinCount", ooxml.Int(p.SpinCount))
	case p.LegacyHash != "":
		attrs = append(attrs, "password", p.LegacyHash)
	}
	// Excel semantics: attribute "1" means the action is locked (forbidden).
	attrs = append(attrs, "sheet", "1")
	if !p.Objects {
		attrs = append(attrs, "objects", "1")
	}
	if !p.Scenarios {
		attrs = append(attrs, "scenarios", "1")
	}
	lock := func(name string, allowed bool) {
		if !allowed {
			attrs = append(attrs, name, "1")
		}
	}
	lock("formatCells", p.FormatCells)
	lock("formatColumns", p.FormatColumns)
	lock("formatRows", p.FormatRows)
	lock("insertColumns", p.InsertColumns)
	lock("insertRows", p.InsertRows)
	lock("insertHyperlinks", p.InsertHyperlinks)
	lock("deleteColumns", p.DeleteColumns)
	lock("deleteRows", p.DeleteRows)
	if p.NoSelectLocked {
		attrs = append(attrs, "selectLockedCells", "1")
	}
	lock("sort", p.Sort)
	lock("autoFilter", p.AutoFilter)
	lock("pivotTables", p.PivotTables)
	if p.NoSelectUnlocked {
		attrs = append(attrs, "selectUnlockedCells", "1")
	}
	b.Empty("sheetProtection", attrs...)
}

func autoFilterXML(b *ooxml.Builder, af *model.AutoFilter) {
	if len(af.Columns) == 0 && af.SortState == nil {
		b.Empty("autoFilter", "ref", af.Range.String())
		return
	}
	b.Start("autoFilter", "ref", af.Range.String())
	for _, fc := range af.Columns {
		filterColumnXML(b, fc)
	}
	if af.SortState != nil {
		sortStateXML(b, af.SortState)
	}
	b.End()
}

func filterColumnXML(b *ooxml.Builder, fc model.FilterColumn) {
	attrs := []string{"colId", ooxml.Int(fc.ColID), "hiddenButton", boolAttr(fc.HiddenButton)}
	if fc.ShowButton != nil {
		attrs = append(attrs, "showButton", ooxml.Bool(*fc.ShowButton))
	}
	b.Start("filterColumn", attrs...)
	switch {
	case len(fc.Values) > 0 || fc.Blank || len(fc.DateGroups) > 0:
		b.Start("filters", "blank", boolAttr(fc.Blank))
		for _, v := range fc.Values {
			b.Empty("filter", "val", v)
		}
		for _, d := range fc.DateGroups {
			b.Empty("dateGroupItem", "year", ooxml.Int(d.Year), "month", intAttr(d.Month), "day", intAttr(d.Day), "hour", intAttr(d.Hour), "minute", intAttr(d.Minute), "second", intAttr(d.Second), "dateTimeGrouping", d.Grouping)
		}
		b.End()
	case len(fc.Custom) > 0:
		b.Start("customFilters", "and", boolAttr(fc.And))
		for _, c := range fc.Custom {
			b.Empty("customFilter", "operator", c.Operator, "val", c.Value)
		}
		b.End()
	case fc.Top10 != nil:
		t := fc.Top10
		b.Empty("top10", "top", map[bool]string{true: "", false: "0"}[t.Top], "percent", boolAttr(t.Percent), "val", ooxml.Float(t.Val), "filterVal", ooxml.Float(t.FilterVal))
	case fc.Dynamic != nil:
		d := fc.Dynamic
		attrs := []string{"type", d.Type}
		if d.Val != 0 {
			attrs = append(attrs, "val", ooxml.Float(d.Val))
		}
		if d.MaxVal != 0 {
			attrs = append(attrs, "maxVal", ooxml.Float(d.MaxVal))
		}
		b.Empty("dynamicFilter", attrs...)
	case fc.ColorFilter != nil:
		b.Empty("colorFilter", "dxfId", ooxml.Int(fc.ColorFilter.DxfID), "cellColor", map[bool]string{true: "", false: "0"}[fc.ColorFilter.CellColor])
	case fc.IconFilter != nil:
		b.Empty("iconFilter", "iconSet", fc.IconFilter.IconSet, "iconId", ooxml.Int(fc.IconFilter.IconID))
	}
	b.End()
}

func sortStateXML(b *ooxml.Builder, ss *model.SortState) {
	b.Start("sortState", "ref", ss.Range.String(), "caseSensitive", boolAttr(ss.CaseSensitive), "columnSort", boolAttr(ss.ColumnSort))
	for _, c := range ss.Conditions {
		attrs := []string{"ref", c.Range.String(), "descending", boolAttr(c.Descending), "sortBy", c.SortBy, "customList", c.CustomList}
		if c.SortBy == "cellColor" || c.SortBy == "fontColor" {
			attrs = append(attrs, "dxfId", ooxml.Int(c.DxfID))
		}
		if c.SortBy == "icon" {
			attrs = append(attrs, "iconSet", c.IconSet, "iconId", ooxml.Int(c.IconID))
		}
		b.Empty("sortCondition", attrs...)
	}
	b.End()
}

func dataValidationXML(b *ooxml.Builder, dv model.DataValidation) {
	attrs := []string{"type", string(dv.Type), "errorStyle", dv.ErrorStyle, "imeMode", dv.IMEMode, "operator", string(dv.Operator),
		"allowBlank", boolAttr(dv.AllowBlank), "showDropDown", boolAttr(dv.ShowDropDown), "showInputMessage", boolAttr(dv.ShowInputMessage || dv.Prompt != "" || dv.PromptTitle != ""),
		"showErrorMessage", boolAttr(dv.ShowErrorMessage || dv.Error != "" || dv.ErrorTitle != ""), "errorTitle", dv.ErrorTitle, "error", dv.Error,
		"promptTitle", dv.PromptTitle, "prompt", dv.Prompt, "sqref", ref.FormatRanges(dv.Ranges)}
	b.Start("dataValidation", attrs...)
	if dv.Formula1 != "" {
		b.Elem("formula1", strings.TrimPrefix(dv.Formula1, "="))
	}
	if dv.Formula2 != "" {
		b.Elem("formula2", strings.TrimPrefix(dv.Formula2, "="))
	}
	b.End()
}

// condFmtXML writes a conditionalFormatting block; rules needing x14 extras
// append a closure that writes the extension part.
func (w *sheetWriter) condFmtXML(b *ooxml.Builder, cf model.ConditionalFormat, priority *int, x14 *[]func(*ooxml.Builder)) {
	b.Start("conditionalFormatting", "sqref", ref.FormatRanges(cf.Ranges), "pivot", boolAttr(cf.Pivot))
	for _, r := range cf.Rules {
		pr := r.Priority
		if pr == 0 {
			pr = *priority
			*priority++
		} else if pr >= *priority {
			*priority = pr + 1
		}
		attrs := []string{"type", string(r.Type)}
		if r.HasStyle || !r.Style.IsZero() {
			attrs = append(attrs, "dxfId", ooxml.Int(w.p.wb.Styles.AddDxf(r.Style)))
		} else if r.DxfID > 0 || r.Type == model.CfCellIs || r.Type == model.CfExpression {
			if r.DxfID > 0 || len(w.p.wb.Styles.Dxfs) > 0 {
				attrs = append(attrs, "dxfId", ooxml.Int(r.DxfID))
			}
		}
		attrs = append(attrs, "priority", ooxml.Int(pr), "stopIfTrue", boolAttr(r.StopIfTrue), "operator", string(r.Operator), "text", r.Text,
			"timePeriod", r.TimePeriod, "rank", intAttr(r.Rank), "bottom", boolAttr(r.Bottom), "percent", boolAttr(r.Percent), "equalAverage", boolAttr(r.EqualAverage), "stdDev", intAttr(r.StdDev))
		if r.AboveAverage != nil && !*r.AboveAverage {
			attrs = append(attrs, "aboveAverage", "0")
		}
		b.Start("cfRule", attrs...)
		formulas := r.Formulas
		if len(formulas) == 0 {
			switch r.Type {
			case model.CfContainsText:
				formulas = []string{fmt.Sprintf(`NOT(ISERROR(SEARCH("%s",%s)))`, escapeQuotes(r.Text), cf.Ranges[0].From.String())}
			case model.CfNotContainsText:
				formulas = []string{fmt.Sprintf(`ISERROR(SEARCH("%s",%s))`, escapeQuotes(r.Text), cf.Ranges[0].From.String())}
			case model.CfBeginsWith:
				formulas = []string{fmt.Sprintf(`LEFT(%s,LEN("%s"))="%s"`, cf.Ranges[0].From.String(), escapeQuotes(r.Text), escapeQuotes(r.Text))}
			case model.CfEndsWith:
				formulas = []string{fmt.Sprintf(`RIGHT(%s,LEN("%s"))="%s"`, cf.Ranges[0].From.String(), escapeQuotes(r.Text), escapeQuotes(r.Text))}
			case model.CfContainsBlanks:
				formulas = []string{fmt.Sprintf(`LEN(TRIM(%s))=0`, cf.Ranges[0].From.String())}
			case model.CfNotContainsBlanks:
				formulas = []string{fmt.Sprintf(`LEN(TRIM(%s))>0`, cf.Ranges[0].From.String())}
			case model.CfContainsErrors:
				formulas = []string{fmt.Sprintf(`ISERROR(%s)`, cf.Ranges[0].From.String())}
			case model.CfNotContainsErrors:
				formulas = []string{fmt.Sprintf(`NOT(ISERROR(%s))`, cf.Ranges[0].From.String())}
			}
		}
		for _, f := range formulas {
			b.Elem("formula", strings.TrimPrefix(f, "="))
		}
		if cs := r.ColorScale; cs != nil {
			b.Start("colorScale")
			for _, v := range cs.Values {
				cfvoXML(b, v)
			}
			for _, c := range cs.Colors {
				colorXML(b, "color", c)
			}
			b.End()
		}
		if db := r.DataBar; db != nil {
			attrs := []string{"minLength", intAttr(db.MinLength), "maxLength", intAttr(db.MaxLength)}
			if db.ShowValue != nil && !*db.ShowValue {
				attrs = append(attrs, "showValue", "0")
			}
			b.Start("dataBar", attrs...)
			cfvoXML(b, db.Min)
			cfvoXML(b, db.Max)
			colorXML(b, "color", db.Color)
			b.End()
			if dataBarNeedsExt(db) {
				id := fmt.Sprintf("{DA7ABA51-0000-4000-8000-%012X}", pr)
				b.Start("extLst").Start("ext", "uri", "{B025F937-C7B1-47D3-B67F-A62EFF666E3E}", "xmlns:x14", ooxml.NSX14).Elem("x14:id", id).End().End()
				dbc := *db
				ranges := ref.FormatRanges(cf.Ranges)
				*x14 = append(*x14, func(b *ooxml.Builder) {
					b.Start("x14:conditionalFormatting", "xmlns:xm", "http://schemas.microsoft.com/office/excel/2006/main")
					b.Start("x14:cfRule", "type", "dataBar", "id", id)
					attrs := []string{"minLength", intAttr(dbc.MinLength), "maxLength", intAttr(dbc.MaxLength), "border", boolAttr(!dbc.BorderColor.IsZero()),
						"negativeBarColorSameAsPositive", boolAttr(dbc.NegativeColor.IsZero()), "negativeBarBorderColorSameAsPositive", boolAttr(dbc.NegativeBorder.IsZero()),
						"axisPosition", dbc.AxisPosition, "direction", dbc.Direction}
					if dbc.Gradient != nil {
						attrs = append(attrs, "gradient", ooxml.Bool(*dbc.Gradient))
					}
					b.Start("x14:dataBar", attrs...)
					x14cfvo(b, dbc.Min, true)
					x14cfvo(b, dbc.Max, false)
					colorXML(b, "x14:borderColor", dbc.BorderColor)
					colorXML(b, "x14:negativeFillColor", dbc.NegativeColor)
					colorXML(b, "x14:negativeBorderColor", dbc.NegativeBorder)
					colorXML(b, "x14:axisColor", dbc.AxisColor)
					b.End().End()
					b.Elem("xm:sqref", ranges)
					b.End()
				})
			}
		}
		if is := r.IconSet; is != nil {
			attrs := []string{"iconSet", orDefault(is.Set, "3TrafficLights1"), "reverse", boolAttr(is.Reverse)}
			if is.ShowValue != nil && !*is.ShowValue {
				attrs = append(attrs, "showValue", "0")
			}
			if is.Percent != nil && !*is.Percent {
				attrs = append(attrs, "percent", "0")
			}
			b.Start("iconSet", attrs...)
			for _, v := range is.Values {
				cfvoXML(b, v)
			}
			b.End()
		}
		b.End()
	}
	b.End()
}

func dataBarNeedsExt(db *model.DataBar) bool {
	return db.Gradient != nil || !db.BorderColor.IsZero() || !db.NegativeColor.IsZero() || !db.NegativeBorder.IsZero() || !db.AxisColor.IsZero() || db.AxisPosition != "" || db.Direction != ""
}

func x14cfvo(b *ooxml.Builder, v model.CfValue, isMin bool) {
	t := string(v.Type)
	if t == "" {
		if isMin {
			t = "autoMin"
		} else {
			t = "autoMax"
		}
	}
	if v.Value != "" {
		b.Start("x14:cfvo", "type", t).Elem("xm:f", v.Value).End()
	} else {
		b.Empty("x14:cfvo", "type", t)
	}
}

func cfvoXML(b *ooxml.Builder, v model.CfValue) {
	t := string(v.Type)
	if t == "" {
		t = "num"
	}
	attrs := []string{"type", t, "val", v.Value}
	if v.GTE != nil && !*v.GTE {
		attrs = append(attrs, "gte", "0")
	}
	b.Empty("cfvo", attrs...)
}

func escapeQuotes(s string) string { return strings.ReplaceAll(s, `"`, `""`) }

func sparklinesXML(b *ooxml.Builder, groups []model.SparklineGroup) {
	b.Start("ext", "uri", "{05C60535-1F16-4fd2-B633-F4F36F0B64E0}", "xmlns:x14", ooxml.NSX14)
	b.Start("x14:sparklineGroups", "xmlns:xm", "http://schemas.microsoft.com/office/excel/2006/main")
	for _, g := range groups {
		attrs := []string{"type", g.Type, "displayEmptyCellsAs", orDefault(g.DisplayEmptyCellsAs, "gap"), "markers", boolAttr(g.ShowMarkers), "high", boolAttr(g.ShowHigh),
			"low", boolAttr(g.ShowLow), "first", boolAttr(g.ShowFirst), "last", boolAttr(g.ShowLast), "negative", boolAttr(g.ShowNegative), "displayXAxis", boolAttr(g.DisplayXAxis),
			"displayHidden", boolAttr(g.DisplayHidden), "minAxisType", g.MinAxisType, "maxAxisType", g.MaxAxisType, "rightToLeft", boolAttr(g.RightToLeft)}
		if g.LineWeight != 0 {
			attrs = append(attrs, "lineWeight", ooxml.Float(g.LineWeight))
		}
		if g.MinAxisType == "custom" {
			attrs = append(attrs, "manualMin", ooxml.Float(g.ManualMin))
		}
		if g.MaxAxisType == "custom" {
			attrs = append(attrs, "manualMax", ooxml.Float(g.ManualMax))
		}
		b.Start("x14:sparklineGroup", attrs...)
		colors := [...]struct {
			tag string
			c   style.Color
		}{{"x14:colorSeries", g.ColorSeries}, {"x14:colorNegative", g.ColorNegative}, {"x14:colorAxis", g.ColorAxis}, {"x14:colorMarkers", g.ColorMarkers},
			{"x14:colorFirst", g.ColorFirst}, {"x14:colorLast", g.ColorLast}, {"x14:colorHigh", g.ColorHigh}, {"x14:colorLow", g.ColorLow}}
		for _, c := range colors {
			col := c.c
			if col.IsZero() {
				col = style.ThemeColor(style.ThemeAccent1, 0)
			}
			colorXML(b, c.tag, col)
		}
		if g.DateAxis != "" {
			b.Elem("xm:f", g.DateAxis)
		}
		b.Start("x14:sparklines")
		for _, s := range g.Sparklines {
			b.Start("x14:sparkline").Elem("xm:f", s.Formula).Elem("xm:sqref", s.Ref.String()).End()
		}
		b.End().End()
	}
	b.End().End()
}
