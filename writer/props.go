// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

func corePropsXML(p model.Properties, now time.Time) []byte {
	b := ooxml.NewBuilder()
	b.Start("cp:coreProperties", "xmlns:cp", ooxml.NSCoreProps, "xmlns:dc", ooxml.NSDC, "xmlns:dcterms", ooxml.NSDCTerms, "xmlns:dcmitype", ooxml.NSDCMIType, "xmlns:xsi", ooxml.NSXSI)
	elem := func(tag, v string) {
		if v != "" {
			b.Elem(tag, v)
		}
	}
	elem("dc:title", p.Title)
	elem("dc:subject", p.Subject)
	elem("dc:creator", p.Creator)
	elem("cp:keywords", p.Keywords)
	elem("dc:description", p.Description)
	elem("cp:lastModifiedBy", p.LastModifiedBy)
	elem("cp:revision", p.Revision)
	if !p.LastPrinted.IsZero() {
		b.Elem("cp:lastPrinted", p.LastPrinted.UTC().Format("2006-01-02T15:04:05Z"))
	}
	created := p.Created
	if created.IsZero() {
		created = now
	}
	modified := p.Modified
	if modified.IsZero() {
		modified = now
	}
	b.Elem("dcterms:created", created.UTC().Format("2006-01-02T15:04:05Z"), "xsi:type", "dcterms:W3CDTF")
	b.Elem("dcterms:modified", modified.UTC().Format("2006-01-02T15:04:05Z"), "xsi:type", "dcterms:W3CDTF")
	elem("cp:category", p.Category)
	elem("cp:contentStatus", p.ContentStatus)
	elem("dc:language", p.Language)
	elem("dc:identifier", p.Identifier)
	elem("cp:version", p.Version)
	b.End()
	return b.Bytes()
}

func appPropsXML(wb *model.Workbook) []byte {
	p := wb.Properties
	b := ooxml.NewBuilder()
	b.Start("Properties", "xmlns", ooxml.NSExtProps, "xmlns:vt", ooxml.NSDocPropsVT)
	b.Elem("Application", orDefault(p.Application, "Microsoft Excel"))
	b.Elem("DocSecurity", ooxml.Int(p.DocSecurity))
	b.Elem("ScaleCrop", boolWord(p.ScaleCrop))
	// Titles of parts: the sheet names.
	b.Start("HeadingPairs").Start("vt:vector", "size", "2", "baseType", "variant")
	b.Start("vt:variant").Elem("vt:lpstr", "Worksheets").End()
	b.Start("vt:variant").Elem("vt:i4", ooxml.Int(len(wb.Sheets))).End()
	b.End().End()
	b.Start("TitlesOfParts").Start("vt:vector", "size", ooxml.Int(len(wb.Sheets)), "baseType", "lpstr")
	for _, s := range wb.Sheets {
		b.Elem("vt:lpstr", s.Name)
	}
	b.End().End()
	if p.Manager != "" {
		b.Elem("Manager", p.Manager)
	}
	if p.Company != "" {
		b.Elem("Company", p.Company)
	}
	if p.HyperlinkBase != "" {
		b.Elem("HyperlinkBase", p.HyperlinkBase)
	}
	if p.Template != "" {
		b.Elem("Template", p.Template)
	}
	b.Elem("LinksUpToDate", boolWord(p.LinksUpToDate))
	b.Elem("SharedDoc", boolWord(p.SharedDoc))
	b.Elem("HyperlinksChanged", boolWord(p.HyperlinksChanged))
	b.Elem("AppVersion", orDefault(p.AppVersion, "16.0300"))
	b.End()
	return b.Bytes()
}

func boolWord(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func customPropsXML(props []model.CustomProperty) []byte {
	b := ooxml.NewBuilder()
	b.Start("Properties", "xmlns", ooxml.NSCustomProps, "xmlns:vt", ooxml.NSDocPropsVT)
	for i, p := range props {
		b.Start("property", "fmtid", "{D5CDD505-2E9C-101B-9397-08002B2CF9AE}", "pid", ooxml.Int(i+2), "name", p.Name)
		switch v := p.Value.(type) {
		case string:
			b.Elem("vt:lpwstr", v)
		case bool:
			b.Elem("vt:bool", boolWord(v))
		case int:
			b.Elem("vt:i4", ooxml.Int(v))
		case int64:
			b.Elem("vt:i8", strconv.FormatInt(v, 10))
		case float64:
			b.Elem("vt:r8", ooxml.Float(v))
		case time.Time:
			b.Elem("vt:filetime", v.UTC().Format("2006-01-02T15:04:05Z"))
		default:
			b.Elem("vt:lpwstr", fmt.Sprint(v))
		}
		b.End()
	}
	b.End()
	return b.Bytes()
}

// themeXML builds a minimal but complete theme1.xml from the theme colors
// and fonts.
func themeXML(t *style.Theme) []byte {
	if t == nil {
		t = style.DefaultTheme()
	}
	b := ooxml.NewBuilder()
	b.Start("a:theme", "xmlns:a", ooxml.NSDrawing, "name", orDefault(t.Name, "Office Theme"))
	b.Start("a:themeElements")
	b.Start("a:clrScheme", "name", "Office")
	names := [...]string{"dk1", "lt1", "dk2", "lt2", "accent1", "accent2", "accent3", "accent4", "accent5", "accent6", "hlink", "folHlink"}
	for i, n := range names {
		b.Start("a:" + n)
		switch n {
		case "dk1":
			b.Empty("a:sysClr", "val", "windowText", "lastClr", t.Colors[i])
		case "lt1":
			b.Empty("a:sysClr", "val", "window", "lastClr", t.Colors[i])
		default:
			b.Empty("a:srgbClr", "val", t.Colors[i])
		}
		b.End()
	}
	b.End()
	b.Start("a:fontScheme", "name", "Office")
	for _, f := range [...]struct{ tag, name string }{{"a:majorFont", orDefault(t.MajorFont, "Calibri Light")}, {"a:minorFont", orDefault(t.MinorFont, "Calibri")}} {
		b.Start(f.tag)
		b.Empty("a:latin", "typeface", f.name, "panose", "020F0302020204030204")
		b.Empty("a:ea", "typeface", "")
		b.Empty("a:cs", "typeface", "")
		for _, s := range [...][2]string{{"Jpan", "游ゴシック"}, {"Hang", "맑은 고딕"}, {"Hans", "等线"}, {"Hant", "新細明體"}, {"Arab", "Arial"}, {"Hebr", "Arial"}, {"Thai", "Tahoma"}, {"Viet", "Arial"}, {"Cyrl", f.name}, {"Grek", f.name}} {
			b.Empty("a:font", "script", s[0], "typeface", s[1])
		}
		b.End()
	}
	b.End()
	b.Start("a:fmtScheme", "name", "Office")
	b.Start("a:fillStyleLst")
	b.Start("a:solidFill").Empty("a:schemeClr", "val", "phClr").End()
	for _, lum := range [...]string{"110000", "105000"} {
		b.Start("a:gradFill", "rotWithShape", "1").Start("a:gsLst")
		b.Start("a:gs", "pos", "0").Start("a:schemeClr", "val", "phClr").Empty("a:satMod", "val", "103000").Empty("a:lumMod", "val", "102000").Empty("a:tint", "val", "94000").End().End()
		b.Start("a:gs", "pos", "100000").Start("a:schemeClr", "val", "phClr").Empty("a:satMod", "val", lum).Empty("a:lumMod", "val", "99000").End().End()
		b.End().Empty("a:lin", "ang", "5400000", "scaled", "0").End()
	}
	b.End()
	b.Start("a:lnStyleLst")
	for _, w := range [...]string{"6350", "12700", "19050"} {
		b.Start("a:ln", "w", w, "cap", "flat", "cmpd", "sng", "algn", "ctr").Start("a:solidFill").Empty("a:schemeClr", "val", "phClr").End().Empty("a:prstDash", "val", "solid").Empty("a:miter", "lim", "800000").End()
	}
	b.End()
	b.Start("a:effectStyleLst")
	b.Start("a:effectStyle").Empty("a:effectLst").End()
	b.Start("a:effectStyle").Empty("a:effectLst").End()
	b.Start("a:effectStyle").Start("a:effectLst").Start("a:outerShdw", "blurRad", "57150", "dist", "19050", "dir", "5400000", "algn", "ctr", "rotWithShape", "0").Start("a:srgbClr", "val", "000000").Empty("a:alpha", "val", "63000").End().End().End().End()
	b.End()
	b.Start("a:bgFillStyleLst")
	b.Start("a:solidFill").Empty("a:schemeClr", "val", "phClr").End()
	b.Start("a:solidFill").Start("a:schemeClr", "val", "phClr").Empty("a:tint", "val", "95000").Empty("a:satMod", "val", "170000").End().End()
	b.Start("a:gradFill", "rotWithShape", "1").Start("a:gsLst")
	b.Start("a:gs", "pos", "0").Start("a:schemeClr", "val", "phClr").Empty("a:tint", "val", "93000").Empty("a:satMod", "val", "150000").Empty("a:shade", "val", "98000").Empty("a:lumMod", "val", "102000").End().End()
	b.Start("a:gs", "pos", "100000").Start("a:schemeClr", "val", "phClr").Empty("a:shade", "val", "63000").Empty("a:satMod", "val", "120000").End().End()
	b.End().Empty("a:lin", "ang", "5400000", "scaled", "0").End()
	b.End()
	b.End() // fmtScheme
	b.End() // themeElements
	b.Empty("a:objectDefaults")
	b.Empty("a:extraClrSchemeLst")
	b.End()
	return b.Bytes()
}
