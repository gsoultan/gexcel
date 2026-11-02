// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

func (d *docReader) readWorkbook(wbRels []ooxml.Relationship) error {
	root, err := d.dom(d.wbPart)
	if err != nil {
		return err
	}
	if root == nil {
		return ErrNotSpreadsheet
	}
	wb := d.wb
	if pr := root.Child("workbookPr"); pr != nil {
		wb.Date1904 = ooxml.ParseBool(pr.Attr("date1904"))
		wb.CodeName = pr.Attr("codeName")
		wb.FilterPrivacy = ooxml.ParseBool(pr.Attr("filterPrivacy"))
		wb.HidePivotFieldList = ooxml.ParseBool(pr.Attr("hidePivotFieldList"))
		wb.UpdateLinks = pr.Attr("updateLinks")
		wb.DefaultThemeVersion = ooxml.ParseInt(pr.Attr("defaultThemeVersion"), 0)
		if pr.Has("showInkAnnotation") {
			v := ooxml.ParseBool(pr.Attr("showInkAnnotation"))
			wb.ShowInkAnnotation = &v
		}
	}
	if wp := root.Child("workbookProtection"); wp != nil {
		wb.Protection = &model.WorkbookProtection{
			LockStructure: ooxml.ParseBool(wp.Attr("lockStructure")), LockWindows: ooxml.ParseBool(wp.Attr("lockWindows")), LockRevision: ooxml.ParseBool(wp.Attr("lockRevision")),
			Algorithm: wp.Attr("workbookAlgorithmName"), HashValue: wp.Attr("workbookHashValue"), SaltValue: wp.Attr("workbookSaltValue"), SpinCount: ooxml.ParseInt(wp.Attr("workbookSpinCount"), 0),
			LegacyHash: wp.Attr("workbookPassword"), RevisionsAlgorithm: wp.Attr("revisionsAlgorithmName"), RevisionsHash: wp.Attr("revisionsHashValue"), RevisionsSalt: wp.Attr("revisionsSaltValue"), RevisionsSpin: ooxml.ParseInt(wp.Attr("revisionsSpinCount"), 0),
		}
	}
	if bv := root.Path("bookViews", "workbookView"); bv != nil {
		v := &wb.View
		v.ActiveTab = ooxml.ParseInt(bv.Attr("activeTab"), 0)
		v.FirstSheet = ooxml.ParseInt(bv.Attr("firstSheet"), 0)
		v.Visibility = bv.Attr("visibility")
		v.Minimized = ooxml.ParseBool(bv.Attr("minimized"))
		v.HideHorizontalScroll = bv.Has("showHorizontalScroll") && !ooxml.ParseBool(bv.Attr("showHorizontalScroll"))
		v.HideVerticalScroll = bv.Has("showVerticalScroll") && !ooxml.ParseBool(bv.Attr("showVerticalScroll"))
		v.HideSheetTabs = bv.Has("showSheetTabs") && !ooxml.ParseBool(bv.Attr("showSheetTabs"))
		v.XWindow = ooxml.ParseInt(bv.Attr("xWindow"), 0)
		v.YWindow = ooxml.ParseInt(bv.Attr("yWindow"), 0)
		v.WindowWidth = ooxml.ParseInt(bv.Attr("windowWidth"), 0)
		v.WindowHeight = ooxml.ParseInt(bv.Attr("windowHeight"), 0)
		v.TabRatio = ooxml.ParseInt(bv.Attr("tabRatio"), 0)
		if bv.Has("autoFilterDateGrouping") {
			b := ooxml.ParseBool(bv.Attr("autoFilterDateGrouping"))
			v.AutoFilterDateGrouping = &b
		}
	}
	if cp := root.Child("calcPr"); cp != nil {
		c := &wb.Calc
		c.CalcID = ooxml.ParseInt(cp.Attr("calcId"), 0)
		c.CalcMode = cp.Attr("calcMode")
		c.FullCalcOnLoad = ooxml.ParseBool(cp.Attr("fullCalcOnLoad"))
		c.RefModeR1C1 = cp.Attr("refMode") == "R1C1"
		c.Iterate = ooxml.ParseBool(cp.Attr("iterate"))
		c.IterateCount = ooxml.ParseInt(cp.Attr("iterateCount"), 0)
		c.IterateDelta, _ = ooxml.ParseFloat(cp.Attr("iterateDelta"))
		c.NoFullPrecision = cp.Has("fullPrecision") && !ooxml.ParseBool(cp.Attr("fullPrecision"))
		c.ForceFullCalc = ooxml.ParseBool(cp.Attr("forceFullCalc"))
		for _, a := range [...]struct {
			name string
			dst  **bool
		}{{"calcCompleted", &c.CalcCompleted}, {"calcOnSave", &c.CalcOnSave}, {"concurrentCalc", &c.ConcurrentCalc}} {
			if cp.Has(a.name) {
				b := ooxml.ParseBool(cp.Attr(a.name))
				*a.dst = &b
			}
		}
	}
	// External references map index -> rel id for formulas like [1]Sheet!A1;
	// the order of externalReference elements defines the index.
	// Sheets.
	for _, s := range root.Path("sheets").Each("sheet") {
		name := s.Attr("name")
		rid := s.Attr("id")
		for _, a := range s.Attrs {
			if a.Name.Local == "id" && a.Name.Space == ooxml.NSRel {
				rid = a.Value
			}
		}
		sh := model.NewSheet(name)
		sh.State = model.SheetState(s.Attr("state"))
		wb.Sheets = append(wb.Sheets, sh)
		rel, ok := ooxml.RelByID(wbRels, rid)
		if !ok {
			continue
		}
		part := ooxml.Resolve(d.wbPart, rel.Target)
		switch rel.Type {
		case ooxml.RelWorksheet:
			if d.skipSheets {
				if d.sheetParts == nil {
					d.sheetParts = map[string]string{}
				}
				d.sheetParts[name] = part
				d.use(part)
				continue
			}
			if !d.wantSheet(name) {
				d.use(part)
				continue
			}
			if err := d.readSheet(sh, part); err != nil {
				return err
			}
		case ooxml.RelChartsheet:
			sh.Kind = model.Chartsheet
			sh.Raw = d.rawPart(part)
			d.readChartsheet(sh)
		case ooxml.RelDialogsheet:
			sh.Kind = model.DialogSheet
			sh.Raw = d.rawPart(part)
		case ooxml.RelMacrosheet:
			sh.Kind = model.MacroSheet
			sh.Raw = d.rawPart(part)
		}
	}
	for _, dn := range root.Path("definedNames").Each("definedName") {
		n := model.DefinedName{Name: dn.Attr("name"), RefersTo: formula.StripPrefixes(strings.TrimSpace(dn.Text)), Comment: dn.Attr("comment"),
			Hidden: ooxml.ParseBool(dn.Attr("hidden")), Function: ooxml.ParseBool(dn.Attr("function")), VBProc: ooxml.ParseBool(dn.Attr("vbProcedure"))}
		if dn.Has("localSheetId") {
			n.Local = true
			n.SheetIndex = ooxml.ParseInt(dn.Attr("localSheetId"), 0)
		}
		if n.Local && n.SheetIndex < len(wb.Sheets) && d.decodeBuiltinName(wb.Sheets[n.SheetIndex], n) {
			continue
		}
		wb.DefinedNames = append(wb.DefinedNames, n)
	}
	for _, pc := range root.Path("pivotCaches").Each("pivotCache") {
		rid := ""
		for _, a := range pc.Attrs {
			if a.Name.Local == "id" {
				rid = a.Value
			}
		}
		if rel, ok := ooxml.RelByID(wbRels, rid); ok {
			if raw := d.rawPart(ooxml.Resolve(d.wbPart, rel.Target)); raw != nil {
				if wb.PivotCaches == nil {
					wb.PivotCaches = map[int]*model.RawPart{}
				}
				wb.PivotCaches[ooxml.ParseInt(pc.Attr("cacheId"), 0)] = raw
			}
		}
	}
	if ext := root.Child("extLst"); ext != nil {
		for _, e := range ext.Children {
			wb.Extensions = append(wb.Extensions, model.RawXML{URI: e.Attr("uri"), Data: rawXML(d.wbPart, d, "extLst", e)})
		}
	}
	return nil
}

// decodeBuiltinName fills the sheet's print settings from the built-in
// names; it reports whether the name was consumed.
func (d *docReader) decodeBuiltinName(sh *model.Sheet, n model.DefinedName) bool {
	switch n.Name {
	case model.NamePrintArea:
		areas, err := ref.ParseAreas(n.RefersTo)
		if err != nil {
			return false
		}
		for _, a := range areas {
			sh.PrintArea = append(sh.PrintArea, a.Range)
		}
		return true
	case model.NamePrintTitles:
		areas, err := ref.ParseAreas(n.RefersTo)
		if err != nil {
			return false
		}
		for _, a := range areas {
			r := a.Range
			switch {
			case r.WholeRows:
				sh.PrintTitleRows = &r
			case r.WholeCols:
				sh.PrintTitleCols = &r
			}
		}
		return true
	case model.NameFilterDatabase:
		return true // regenerated from the autoFilter element
	}
	return false
}

// rawXML re-serialises a DOM node (used to preserve extension elements).
func rawXML(part string, d *docReader, parent string, n *ooxml.Node) []byte {
	var b ooxml.Builder
	writeNode(&b, n)
	return b.Bytes()
}

func writeNode(b *ooxml.Builder, n *ooxml.Node) {
	b.WriteByte('<')
	name := qname(n)
	b.WriteString(name)
	for _, a := range n.Attrs {
		b.WriteByte(' ')
		b.WriteString(attrName(a.Name))
		b.WriteString(`="`)
		b.WriteString(ooxml.EscapeAttr(a.Value))
		b.WriteByte('"')
	}
	if len(n.Children) == 0 && n.Text == "" {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	if len(n.Children) == 0 {
		b.Text(n.Text)
	}
	for _, c := range n.Children {
		writeNode(b, c)
	}
	b.WriteString("</")
	b.WriteString(name)
	b.WriteByte('>')
}

// qname writes the element with its namespace declared inline, so the
// fragment is self-contained wherever it is pasted.
func qname(n *ooxml.Node) string {
	if n.Space == "" || n.Space == ooxml.NSMain {
		return n.Name
	}
	prefix := prefixFor(n.Space)
	// Declare the namespace on this element if not already there.
	found := false
	for _, a := range n.Attrs {
		if a.Name.Space == "xmlns" && a.Name.Local == prefix || a.Name.Local == "xmlns" && a.Value == n.Space {
			found = true
		}
	}
	if !found {
		n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Space: "xmlns", Local: prefix}, Value: n.Space})
	}
	return prefix + ":" + n.Name
}

func attrName(x xml.Name) string {
	switch x.Space {
	case "":
		return x.Local
	case "xmlns":
		return "xmlns:" + x.Local
	case "http://www.w3.org/XML/1998/namespace":
		return "xml:" + x.Local
	}
	return prefixFor(x.Space) + ":" + x.Local
}

var nsPrefixes = map[string]string{
	ooxml.NSX14: "x14", ooxml.NSX15: "x15", ooxml.NSXR: "xr", ooxml.NSXR2: "xr2", ooxml.NSXR3: "xr3", ooxml.NSXR6: "xr6", ooxml.NSXR10: "xr10",
	ooxml.NSRel: "r", ooxml.NSX14ac: "x14ac", "http://schemas.microsoft.com/office/excel/2006/main": "xm",
	ooxml.NSDrawing: "a", ooxml.NSSpreadsheetDr: "xdr", ooxml.NSChart: "c", ooxml.NSMC: "mc", ooxml.NSXDA: "xda",
	"http://schemas.microsoft.com/office/spreadsheetml/2015/02/main": "x16r2",
	"http://schemas.microsoft.com/office/spreadsheetml/2010/11/ac":   "x15ac",
	"http://schemas.microsoft.com/office/spreadsheetml/2014/11/main": "x16",
	"http://schemas.microsoft.com/office/drawing/2014/chartex":       "cx",
	"http://schemas.microsoft.com/office/drawing/2007/8/2/chart":     "c14",
}

func prefixFor(ns string) string {
	if p, ok := nsPrefixes[ns]; ok {
		return p
	}
	return "ns" + strconv.Itoa(len(ns)%97)
}

func (d *docReader) readCoreProps(name string) {
	root, err := d.dom(name)
	if err != nil || root == nil {
		return
	}
	d.use(name)
	p := &d.wb.Properties
	for _, c := range root.Children {
		t := strings.TrimSpace(c.Text)
		switch c.Name {
		case "title":
			p.Title = t
		case "subject":
			p.Subject = t
		case "creator":
			p.Creator = t
		case "keywords":
			p.Keywords = t
		case "description":
			p.Description = t
		case "lastModifiedBy":
			p.LastModifiedBy = t
		case "revision":
			p.Revision = t
		case "category":
			p.Category = t
		case "contentStatus":
			p.ContentStatus = t
		case "language":
			p.Language = t
		case "identifier":
			p.Identifier = t
		case "version":
			p.Version = t
		case "created":
			p.Created = parseW3CDTF(t)
		case "modified":
			p.Modified = parseW3CDTF(t)
		case "lastPrinted":
			p.LastPrinted = parseW3CDTF(t)
		}
	}
}

func parseW3CDTF(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (d *docReader) readAppProps(name string) {
	root, err := d.dom(name)
	if err != nil || root == nil {
		return
	}
	d.use(name)
	p := &d.wb.Properties
	for _, c := range root.Children {
		t := strings.TrimSpace(c.Text)
		switch c.Name {
		case "Application":
			p.Application = t
		case "AppVersion":
			p.AppVersion = t
		case "Company":
			p.Company = t
		case "Manager":
			p.Manager = t
		case "DocSecurity":
			p.DocSecurity = ooxml.ParseInt(t, 0)
		case "HyperlinkBase":
			p.HyperlinkBase = t
		case "Template":
			p.Template = t
		case "ScaleCrop":
			p.ScaleCrop = t == "true"
		case "LinksUpToDate":
			p.LinksUpToDate = t == "true"
		case "SharedDoc":
			p.SharedDoc = t == "true"
		case "HyperlinksChanged":
			p.HyperlinksChanged = t == "true"
		}
	}
}

func (d *docReader) readCustomProps(name string) {
	root, err := d.dom(name)
	if err != nil || root == nil {
		return
	}
	d.use(name)
	for _, p := range root.Each("property") {
		if len(p.Children) == 0 {
			continue
		}
		v := p.Children[0]
		t := strings.TrimSpace(v.Text)
		var val any = t
		switch v.Name {
		case "bool":
			val = t == "true"
		case "i4", "i2", "i8", "int", "ui4":
			if n, err := strconv.ParseInt(t, 10, 0); err == nil {
				val = int(n)
			}
		case "r8", "r4", "decimal":
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				val = f
			}
		case "filetime", "date":
			val = parseW3CDTF(t)
		}
		d.wb.CustomProps = append(d.wb.CustomProps, model.CustomProperty{Name: p.Attr("name"), Value: val})
	}
}

func (d *docReader) readTheme(name string) {
	data, err := d.pr.Part(name)
	if err != nil {
		return
	}
	d.use(name)
	th := style.DefaultTheme()
	th.Raw = data
	root, err := ooxml.ParseDOM(data)
	if err == nil && root != nil {
		th.Name = root.Attr("name")
		if cs := root.Path("themeElements", "clrScheme"); cs != nil {
			names := [...]string{"dk1", "lt1", "dk2", "lt2", "accent1", "accent2", "accent3", "accent4", "accent5", "accent6", "hlink", "folHlink"}
			for i, n := range names {
				c := cs.Child(n)
				if c == nil || len(c.Children) == 0 {
					continue
				}
				v := c.Children[0]
				switch v.Name {
				case "srgbClr":
					th.Colors[i] = strings.ToUpper(v.Attr("val"))
				case "sysClr":
					if last := v.Attr("lastClr"); last != "" {
						th.Colors[i] = strings.ToUpper(last)
					}
				}
			}
		}
		if fs := root.Path("themeElements", "fontScheme"); fs != nil {
			if l := fs.Path("majorFont", "latin"); l != nil {
				th.MajorFont = l.Attr("typeface")
			}
			if l := fs.Path("minorFont", "latin"); l != nil {
				th.MinorFont = l.Attr("typeface")
			}
		}
	}
	d.wb.Theme = th
}
