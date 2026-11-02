// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

func (d *docReader) readStyles(name string) error {
	root, err := d.dom(name)
	if err != nil {
		return err
	}
	d.use(name)
	if root == nil {
		return nil
	}
	s := &model.Styles{NumFmts: map[int]string{}}
	for _, nf := range root.Path("numFmts").Each("numFmt") {
		id, code := ooxml.ParseInt(nf.Attr("numFmtId"), 0), nf.Attr("formatCode")
		if b, ok := style.BuiltinNumFmt(id); id < style.FirstCustomNumFmtID && (code == "General" || ok && code == b) {
			continue // builtin restated by LibreOffice; keep the table canonical
		}
		s.NumFmts[id] = code
	}
	for _, f := range root.Path("fonts").Each("font") {
		s.Fonts = append(s.Fonts, parseFont(f))
	}
	for _, f := range root.Path("fills").Each("fill") {
		s.Fills = append(s.Fills, parseFill(f))
	}
	for _, b := range root.Path("borders").Each("border") {
		s.Borders = append(s.Borders, parseBorder(b))
	}
	for _, x := range root.Path("cellStyleXfs").Each("xf") {
		s.CellStyleXfs = append(s.CellStyleXfs, parseXf(x))
	}
	for _, x := range root.Path("cellXfs").Each("xf") {
		s.CellXfs = append(s.CellXfs, parseXf(x))
	}
	for _, cs := range root.Path("cellStyles").Each("cellStyle") {
		c := model.CellStyle{Name: cs.Attr("name"), XfID: ooxml.ParseInt(cs.Attr("xfId"), 0), BuiltinID: -1,
			CustomBuiltin: ooxml.ParseBool(cs.Attr("customBuiltin")), Hidden: ooxml.ParseBool(cs.Attr("hidden")), ILevel: ooxml.ParseInt(cs.Attr("iLevel"), 0)}
		if cs.Has("builtinId") {
			c.BuiltinID = ooxml.ParseInt(cs.Attr("builtinId"), -1)
		}
		s.CellStyles = append(s.CellStyles, c)
	}
	for _, dx := range root.Path("dxfs").Each("dxf") {
		s.Dxfs = append(s.Dxfs, parseDxf(dx, s))
	}
	if ts := root.Child("tableStyles"); ts != nil {
		s.DefaultTableStyle = ts.Attr("defaultTableStyle")
		s.DefaultPivotStyle = ts.Attr("defaultPivotStyle")
		for _, t := range ts.Each("tableStyle") {
			s.TableStyles = append(s.TableStyles, model.RawXML{Data: rawXML(name, d, "tableStyles", t)})
		}
	}
	if cs := root.Child("colors"); cs != nil {
		for _, c := range cs.Path("indexedColors").Each("rgbColor") {
			s.IndexedColors = append(s.IndexedColors, c.Attr("rgb"))
		}
		for _, c := range cs.Path("mruColors").Each("color") {
			s.MRUColors = append(s.MRUColors, c.Attr("rgb"))
		}
	}
	if ext := root.Child("extLst"); ext != nil {
		for _, e := range ext.Children {
			s.Extensions = append(s.Extensions, model.RawXML{URI: e.Attr("uri"), Data: rawXML(name, d, "extLst", e)})
		}
	}
	// Guarantee the mandatory entries so Add() works on read workbooks.
	if len(s.Fonts) == 0 {
		s.Fonts = []style.Font{{Name: style.DefaultFontName, Size: style.DefaultFontSize}}
	}
	if len(s.Fills) == 0 {
		s.Fills = []style.Fill{{}, {Pattern: style.PatternGray125}}
	}
	if len(s.Borders) == 0 {
		s.Borders = []style.Border{{}}
	}
	if len(s.CellStyleXfs) == 0 {
		s.CellStyleXfs = []model.Xf{{}}
	}
	if len(s.CellXfs) == 0 {
		s.CellXfs = []model.Xf{{}}
	}
	if len(s.CellStyles) == 0 {
		s.CellStyles = []model.CellStyle{{Name: "Normal", BuiltinID: 0}}
	}
	s.Reindex()
	d.wb.Styles = s
	return nil
}

func parseColor(n *ooxml.Node) style.Color {
	if n == nil {
		return style.Color{}
	}
	switch {
	case n.Has("rgb"):
		c := style.RGB(n.Attr("rgb"))
		if c.IsZero() && len(n.Attr("rgb")) == 6 {
			c = style.RGB("FF" + n.Attr("rgb"))
		}
		return c
	case n.Has("theme"):
		tint, _ := ooxml.ParseFloat(n.Attr("tint"))
		return style.ThemeColor(ooxml.ParseInt(n.Attr("theme"), 0), tint)
	case n.Has("indexed"):
		return style.Indexed(ooxml.ParseInt(n.Attr("indexed"), 0))
	case n.Has("auto"):
		return style.Auto()
	}
	return style.Color{}
}

// parseFont parses <font> and <rPr> alike.
func parseFont(n *ooxml.Node) style.Font {
	var f style.Font
	for _, c := range n.Children {
		switch c.Name {
		case "b":
			f.Bold = c.ValBool()
		case "i":
			f.Italic = c.ValBool()
		case "strike":
			f.Strike = c.ValBool()
		case "condense":
			f.Condense = c.ValBool()
		case "extend":
			f.Extend = c.ValBool()
		case "outline":
			f.Outline = c.ValBool()
		case "shadow":
			f.Shadow = c.ValBool()
		case "u":
			if c.Has("val") {
				if c.Attr("val") != "none" {
					f.Underline = style.Underline(c.Attr("val"))
				}
			} else {
				f.Underline = style.UnderlineSingle
			}
		case "vertAlign":
			f.VertAlign = style.VertAlign(c.Attr("val"))
		case "sz":
			f.Size = c.ValFloat(0)
		case "color":
			f.Color = parseColor(c)
		case "name", "rFont":
			f.Name = c.Attr("val")
		case "family":
			f.Family = c.ValInt(0)
		case "charset":
			f.Charset = c.ValInt(0)
		case "scheme":
			f.Scheme = c.Attr("val")
		}
	}
	return f
}

func parseFill(n *ooxml.Node) style.Fill {
	var f style.Fill
	if p := n.Child("patternFill"); p != nil {
		f.Pattern = style.PatternType(p.Attr("patternType"))
		if f.Pattern == "none" {
			f.Pattern = style.PatternNone
		}
		f.Foreground = parseColor(p.Child("fgColor"))
		f.Background = parseColor(p.Child("bgColor"))
	}
	if g := n.Child("gradientFill"); g != nil {
		gr := &style.Gradient{Type: style.GradientType(g.Attr("type"))}
		gr.Degree, _ = ooxml.ParseFloat(g.Attr("degree"))
		gr.Left, _ = ooxml.ParseFloat(g.Attr("left"))
		gr.Right, _ = ooxml.ParseFloat(g.Attr("right"))
		gr.Top, _ = ooxml.ParseFloat(g.Attr("top"))
		gr.Bottom, _ = ooxml.ParseFloat(g.Attr("bottom"))
		for _, s := range g.Each("stop") {
			pos, _ := ooxml.ParseFloat(s.Attr("position"))
			gr.Stops = append(gr.Stops, style.GradientStop{Position: pos, Color: parseColor(s.Child("color"))})
		}
		f.Gradient = gr
	}
	return f
}

func parseLine(n *ooxml.Node) style.Line {
	if n == nil {
		return style.Line{}
	}
	return style.Line{Style: style.LineStyle(n.Attr("style")), Color: parseColor(n.Child("color"))}
}

func parseBorder(n *ooxml.Node) style.Border {
	b := style.Border{DiagonalUp: ooxml.ParseBool(n.Attr("diagonalUp")), DiagonalDown: ooxml.ParseBool(n.Attr("diagonalDown"))}
	b.Left = parseLine(n.Child("left"))
	if b.Left.Style == "" {
		b.Left = parseLine(n.Child("start"))
	}
	b.Right = parseLine(n.Child("right"))
	if b.Right.Style == "" {
		b.Right = parseLine(n.Child("end"))
	}
	b.Top = parseLine(n.Child("top"))
	b.Bottom = parseLine(n.Child("bottom"))
	b.Diagonal = parseLine(n.Child("diagonal"))
	return b
}

func parseAlignment(a *ooxml.Node) style.Alignment {
	if a == nil {
		return style.Alignment{}
	}
	return style.Alignment{
		Horizontal: style.HAlign(a.Attr("horizontal")), Vertical: style.VAlign(a.Attr("vertical")),
		WrapText: ooxml.ParseBool(a.Attr("wrapText")), ShrinkToFit: ooxml.ParseBool(a.Attr("shrinkToFit")), JustifyLastLine: ooxml.ParseBool(a.Attr("justifyLastLine")),
		Indent: ooxml.ParseInt(a.Attr("indent"), 0), RelativeIndent: ooxml.ParseInt(a.Attr("relativeIndent"), 0),
		TextRotation: ooxml.ParseInt(a.Attr("textRotation"), 0), ReadingOrder: ooxml.ParseInt(a.Attr("readingOrder"), 0),
	}
}

func parseProtection(p *ooxml.Node) style.Protection {
	if p == nil {
		return style.Protection{}
	}
	return style.Protection{Unlocked: p.Has("locked") && !ooxml.ParseBool(p.Attr("locked")), Hidden: ooxml.ParseBool(p.Attr("hidden"))}
}

func parseXf(x *ooxml.Node) model.Xf {
	xf := model.Xf{
		NumFmtID: ooxml.ParseInt(x.Attr("numFmtId"), 0), FontID: ooxml.ParseInt(x.Attr("fontId"), 0), FillID: ooxml.ParseInt(x.Attr("fillId"), 0),
		BorderID: ooxml.ParseInt(x.Attr("borderId"), 0), XfID: ooxml.ParseInt(x.Attr("xfId"), 0),
		QuotePrefix: ooxml.ParseBool(x.Attr("quotePrefix")), PivotButton: ooxml.ParseBool(x.Attr("pivotButton")),
		ApplyFont: ooxml.ParseBool(x.Attr("applyFont")), ApplyFill: ooxml.ParseBool(x.Attr("applyFill")), ApplyBorder: ooxml.ParseBool(x.Attr("applyBorder")),
		ApplyNumFmt: ooxml.ParseBool(x.Attr("applyNumberFormat")), ApplyAlignment: ooxml.ParseBool(x.Attr("applyAlignment")), ApplyProtection: ooxml.ParseBool(x.Attr("applyProtection")),
	}
	xf.Alignment = parseAlignment(x.Child("alignment"))
	xf.Protection = parseProtection(x.Child("protection"))
	return xf
}

func parseDxf(n *ooxml.Node, s *model.Styles) style.Style {
	var st style.Style
	if f := n.Child("font"); f != nil {
		st.Font = parseFont(f)
	}
	if nf := n.Child("numFmt"); nf != nil {
		st.NumFmt = nf.Attr("formatCode")
		if st.NumFmt == "" {
			st.NumFmt = s.NumFmtCode(ooxml.ParseInt(nf.Attr("numFmtId"), 0))
		}
	}
	if f := n.Child("fill"); f != nil {
		st.Fill = parseFill(f)
		// Differential solid fills carry the color in bgColor.
		if st.Fill.Gradient == nil && st.Fill.Foreground.IsZero() && !st.Fill.Background.IsZero() {
			st.Fill.Foreground = st.Fill.Background
			st.Fill.Background = style.Color{}
			if st.Fill.Pattern == "" {
				st.Fill.Pattern = style.PatternSolid
			}
		}
	}
	st.Alignment = parseAlignment(n.Child("alignment"))
	if b := n.Child("border"); b != nil {
		st.Border = parseBorder(b)
	}
	st.Protection = parseProtection(n.Child("protection"))
	return st
}

func (d *docReader) readSharedStrings(name string) error {
	data, err := d.pr.Part(name)
	if err != nil {
		return err
	}
	d.use(name)
	root, err := ooxml.ParseDOM(data)
	if err != nil {
		if d.opts.Strict {
			return err
		}
		if root == nil {
			return nil
		}
	}
	for _, si := range root.Each("si") {
		d.sst = append(d.sst, parseRichText(si))
	}
	return nil
}

// parseRichText parses an <si>, <is> or comment <text> element.
func parseRichText(si *ooxml.Node) model.Value {
	if t := si.Child("t"); t != nil && si.Child("r") == nil {
		return model.String(ooxml.UnescapeUnicode(t.Text))
	}
	var runs []model.RichRun
	var sb strings.Builder
	for _, r := range si.Children {
		switch r.Name {
		case "t":
			txt := ooxml.UnescapeUnicode(r.Text)
			sb.WriteString(txt)
			runs = append(runs, model.RichRun{Text: txt})
		case "r":
			run := model.RichRun{}
			if t := r.Child("t"); t != nil {
				run.Text = ooxml.UnescapeUnicode(t.Text)
			}
			if rp := r.Child("rPr"); rp != nil {
				f := parseFont(rp)
				run.Font = &f
			}
			sb.WriteString(run.Text)
			runs = append(runs, run)
		}
	}
	v := model.Value{Kind: model.KindString, Str: sb.String()}
	if len(runs) > 1 || len(runs) == 1 && runs[0].Font != nil {
		v.Rich = runs
	}
	return v
}
