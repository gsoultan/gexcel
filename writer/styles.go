// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"sort"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

// stylesXML builds xl/styles.xml.
func stylesXML(s *model.Styles) []byte {
	b := ooxml.NewBuilder()
	b.Start("styleSheet", "xmlns", ooxml.NSMain, "xmlns:mc", ooxml.NSMC, "mc:Ignorable", "x14ac x16r2 xr",
		"xmlns:x14ac", ooxml.NSX14ac, "xmlns:x16r2", "http://schemas.microsoft.com/office/spreadsheetml/2015/02/main", "xmlns:xr", ooxml.NSXR)
	if len(s.NumFmts) > 0 {
		ids := make([]int, 0, len(s.NumFmts))
		for id := range s.NumFmts {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		b.Start("numFmts", "count", ooxml.Int(len(ids)))
		for _, id := range ids {
			b.Empty("numFmt", "numFmtId", ooxml.Int(id), "formatCode", s.NumFmts[id])
		}
		b.End()
	}
	b.Start("fonts", "count", ooxml.Int(len(s.Fonts)), "x14ac:knownFonts", "1")
	for _, f := range s.Fonts {
		fontXML(b, f, "font", "name")
	}
	b.End()
	b.Start("fills", "count", ooxml.Int(len(s.Fills)))
	for _, f := range s.Fills {
		fillXML(b, f, false)
	}
	b.End()
	b.Start("borders", "count", ooxml.Int(len(s.Borders)))
	for _, bd := range s.Borders {
		borderXML(b, bd)
	}
	b.End()
	b.Start("cellStyleXfs", "count", ooxml.Int(len(s.CellStyleXfs)))
	for _, x := range s.CellStyleXfs {
		xfXML(b, x, false)
	}
	b.End()
	b.Start("cellXfs", "count", ooxml.Int(len(s.CellXfs)))
	for _, x := range s.CellXfs {
		xfXML(b, x, true)
	}
	b.End()
	b.Start("cellStyles", "count", ooxml.Int(len(s.CellStyles)))
	for _, cs := range s.CellStyles {
		attrs := []string{"name", cs.Name, "xfId", ooxml.Int(cs.XfID)}
		if cs.BuiltinID >= 0 {
			attrs = append(attrs, "builtinId", ooxml.Int(cs.BuiltinID))
		}
		attrs = append(attrs, "customBuiltin", boolAttr(cs.CustomBuiltin), "hidden", boolAttr(cs.Hidden), "iLevel", intAttr(cs.ILevel))
		b.Empty("cellStyle", attrs...)
	}
	b.End()
	b.Start("dxfs", "count", ooxml.Int(len(s.Dxfs)))
	for _, d := range s.Dxfs {
		dxfXML(b, d)
	}
	b.End()
	tsAttrs := []string{"count", ooxml.Int(len(s.TableStyles)), "defaultTableStyle", orDefault(s.DefaultTableStyle, "TableStyleMedium2"), "defaultPivotStyle", orDefault(s.DefaultPivotStyle, "PivotStyleLight16")}
	if len(s.TableStyles) == 0 {
		b.Empty("tableStyles", tsAttrs...)
	} else {
		b.Start("tableStyles", tsAttrs...)
		for _, ts := range s.TableStyles {
			b.Write(ts.Data)
		}
		b.End()
	}
	if len(s.IndexedColors) > 0 || len(s.MRUColors) > 0 {
		b.Start("colors")
		if len(s.IndexedColors) > 0 {
			b.Start("indexedColors")
			for _, c := range s.IndexedColors {
				b.Empty("rgbColor", "rgb", c)
			}
			b.End()
		}
		if len(s.MRUColors) > 0 {
			b.Start("mruColors")
			for _, c := range s.MRUColors {
				b.Empty("color", "rgb", c)
			}
			b.End()
		}
		b.End()
	}
	if len(s.Extensions) > 0 {
		b.Start("extLst")
		for _, e := range s.Extensions {
			b.Write(e.Data)
		}
		b.End()
	}
	b.End()
	return b.Bytes()
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// colorAttrs returns the attributes of a color element.
func colorAttrs(c style.Color) []string {
	switch c.Kind {
	case style.ColorRGB:
		return []string{"rgb", c.ARGB}
	case style.ColorTheme:
		a := []string{"theme", ooxml.Int(c.Theme)}
		if c.Tint != 0 {
			a = append(a, "tint", ooxml.Float(c.Tint))
		}
		return a
	case style.ColorIndexed:
		return []string{"indexed", ooxml.Int(c.Indexed)}
	case style.ColorAuto:
		return []string{"auto", "1"}
	}
	return nil
}

func colorXML(b *ooxml.Builder, tag string, c style.Color) {
	if c.IsZero() {
		return
	}
	b.Empty(tag, colorAttrs(c)...)
}

// fontXML writes a font (tag "font", name tag "name") or run properties
// (tag "rPr", name tag "rFont").
func fontXML(b *ooxml.Builder, f style.Font, tag, nameTag string) {
	b.Start(tag)
	if f.Bold {
		b.Empty("b")
	}
	if f.Italic {
		b.Empty("i")
	}
	if f.Strike {
		b.Empty("strike")
	}
	if f.Condense {
		b.Empty("condense")
	}
	if f.Extend {
		b.Empty("extend")
	}
	if f.Outline {
		b.Empty("outline")
	}
	if f.Shadow {
		b.Empty("shadow")
	}
	if f.Underline != "" {
		if f.Underline == style.UnderlineSingle {
			b.Empty("u")
		} else {
			b.Empty("u", "val", string(f.Underline))
		}
	}
	if f.VertAlign != "" {
		b.Empty("vertAlign", "val", string(f.VertAlign))
	}
	if f.Size > 0 {
		b.Empty("sz", "val", ooxml.Float(f.Size))
	}
	colorXML(b, "color", f.Color)
	if f.Name != "" {
		b.Empty(nameTag, "val", f.Name)
	}
	if f.Family != 0 {
		b.Empty("family", "val", ooxml.Int(f.Family))
	}
	if f.Charset != 0 {
		b.Empty("charset", "val", ooxml.Int(f.Charset))
	}
	if f.Scheme != "" {
		b.Empty("scheme", "val", f.Scheme)
	}
	b.End()
}

// fillXML writes a fill. In differential formats Excel stores the solid
// color in bgColor, so dxf fills write both.
func fillXML(b *ooxml.Builder, f style.Fill, dxf bool) {
	b.Start("fill")
	if g := f.Gradient; g != nil {
		attrs := []string{"type", string(g.Type)}
		if g.Degree != 0 {
			attrs = append(attrs, "degree", ooxml.Float(g.Degree))
		}
		if g.Type == style.GradientPath {
			attrs = append(attrs, "left", ooxml.Float(g.Left), "right", ooxml.Float(g.Right), "top", ooxml.Float(g.Top), "bottom", ooxml.Float(g.Bottom))
		}
		b.Start("gradientFill", attrs...)
		for _, st := range g.Stops {
			b.Start("stop", "position", ooxml.Float(st.Position))
			colorXML(b, "color", st.Color)
			b.End()
		}
		b.End()
	} else {
		pattern := f.Pattern
		if pattern == "" && !f.Foreground.IsZero() {
			pattern = style.PatternSolid
		}
		if pattern == "" && f.Background.IsZero() {
			b.Empty("patternFill", "patternType", "none")
		} else {
			if dxf && pattern == style.PatternSolid {
				b.Start("patternFill", "patternType", "solid")
				colorXML(b, "fgColor", f.Foreground)
				bg := f.Background
				if bg.IsZero() {
					bg = f.Foreground
				}
				colorXML(b, "bgColor", bg)
				b.End()
			} else {
				b.Start("patternFill", "patternType", string(pattern))
				colorXML(b, "fgColor", f.Foreground)
				colorXML(b, "bgColor", f.Background)
				b.End()
			}
		}
	}
	b.End()
}

func borderXML(b *ooxml.Builder, bd style.Border) {
	b.Start("border", "diagonalUp", boolAttr(bd.DiagonalUp), "diagonalDown", boolAttr(bd.DiagonalDown))
	lineXML(b, "left", bd.Left)
	lineXML(b, "right", bd.Right)
	lineXML(b, "top", bd.Top)
	lineXML(b, "bottom", bd.Bottom)
	lineXML(b, "diagonal", bd.Diagonal)
	b.End()
}

func lineXML(b *ooxml.Builder, tag string, l style.Line) {
	if l.Style == "" {
		b.Empty(tag)
		return
	}
	b.Start(tag, "style", string(l.Style))
	colorXML(b, "color", l.Color)
	b.End()
}

func alignmentAttrs(a style.Alignment) []string {
	attrs := []string{"horizontal", string(a.Horizontal), "vertical", string(a.Vertical), "wrapText", boolAttr(a.WrapText),
		"shrinkToFit", boolAttr(a.ShrinkToFit), "justifyLastLine", boolAttr(a.JustifyLastLine), "indent", intAttr(a.Indent),
		"relativeIndent", intAttr(a.RelativeIndent), "textRotation", intAttr(a.TextRotation), "readingOrder", intAttr(a.ReadingOrder)}
	return attrs
}

func xfXML(b *ooxml.Builder, x model.Xf, cell bool) {
	attrs := []string{"numFmtId", ooxml.Int(x.NumFmtID), "fontId", ooxml.Int(x.FontID), "fillId", ooxml.Int(x.FillID), "borderId", ooxml.Int(x.BorderID)}
	if cell {
		attrs = append(attrs, "xfId", ooxml.Int(x.XfID))
	}
	attrs = append(attrs, "quotePrefix", boolAttr(x.QuotePrefix), "pivotButton", boolAttr(x.PivotButton),
		"applyNumberFormat", boolAttr(x.ApplyNumFmt), "applyFont", boolAttr(x.ApplyFont), "applyFill", boolAttr(x.ApplyFill),
		"applyBorder", boolAttr(x.ApplyBorder), "applyAlignment", boolAttr(x.ApplyAlignment), "applyProtection", boolAttr(x.ApplyProtection))
	hasAlign := !x.Alignment.IsZero()
	hasProt := !x.Protection.IsZero()
	if !hasAlign && !hasProt {
		b.Empty("xf", attrs...)
		return
	}
	b.Start("xf", attrs...)
	if hasAlign {
		b.Empty("alignment", alignmentAttrs(x.Alignment)...)
	}
	if hasProt {
		b.Empty("protection", "locked", map[bool]string{true: "0", false: ""}[x.Protection.Unlocked], "hidden", boolAttr(x.Protection.Hidden))
	}
	b.End()
}

// dxfXML writes a differential format.
func dxfXML(b *ooxml.Builder, d style.Style) {
	b.Start("dxf")
	if !d.Font.IsZero() {
		fontXML(b, d.Font, "font", "name")
	}
	if d.NumFmt != "" {
		id, ok := style.BuiltinNumFmtID(d.NumFmt)
		if !ok {
			id = 164
		}
		b.Empty("numFmt", "numFmtId", ooxml.Int(id), "formatCode", d.NumFmt)
	}
	if !d.Fill.IsZero() {
		fillXML(b, d.Fill, true)
	}
	if !d.Alignment.IsZero() {
		b.Empty("alignment", alignmentAttrs(d.Alignment)...)
	}
	if !d.Border.IsZero() {
		borderXML(b, d.Border)
	}
	if !d.Protection.IsZero() {
		b.Empty("protection", "locked", map[bool]string{true: "0", false: ""}[d.Protection.Unlocked], "hidden", boolAttr(d.Protection.Hidden))
	}
	b.End()
}
