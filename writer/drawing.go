// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"fmt"
	"strings"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// imageContentTypes maps image formats to content types and extensions.
var imageContentTypes = map[string][2]string{
	"png":  {"image/png", "png"},
	"jpeg": {"image/jpeg", "jpeg"},
	"jpg":  {"image/jpeg", "jpeg"},
	"gif":  {"image/gif", "gif"},
	"bmp":  {"image/bmp", "bmp"},
	"tiff": {"image/tiff", "tiff"},
	"tif":  {"image/tiff", "tiff"},
	"svg":  {"image/svg+xml", "svg"},
	"emf":  {"image/x-emf", "emf"},
	"wmf":  {"image/x-wmf", "wmf"},
	"webp": {"image/webp", "webp"},
	"ico":  {"image/x-icon", "ico"},
}

// writeDrawing writes the drawing part of a sheet (pictures, charts,
// shapes) and returns the relationship id the sheet uses.
func (p *packer) writeDrawing(sheetPart string, sh *model.Sheet) (string, error) {
	p.nDrawing++
	name := fmt.Sprintf("xl/drawings/drawing%d.xml", p.nDrawing)
	b := ooxml.NewBuilder()
	b.Start("xdr:wsDr", "xmlns:xdr", ooxml.NSSpreadsheetDr, "xmlns:a", ooxml.NSDrawing, "xmlns:r", ooxml.NSRel)
	id := 1
	for _, img := range sh.Images {
		id++
		img.ID = id
		ct, ok := imageContentTypes[strings.ToLower(img.Format)]
		if !ok {
			return "", fmt.Errorf("%w: image format %q", ErrUnsupported, img.Format)
		}
		p.nImage++
		media := fmt.Sprintf("xl/media/image%d.%s", p.nImage, ct[1])
		p.pw.Default(ct[1], ct[0])
		if err := p.pw.Add(media, "", img.Data); err != nil {
			return "", err
		}
		rel := p.pw.Rel(name, ooxml.RelImage, ooxml.Relative(name, media), "")
		linkRel := ""
		if img.Hyperlink != "" {
			linkRel = p.pw.Rel(name, ooxml.RelHyperlink, img.Hyperlink, "External")
		}
		anchorStart(b, img.Anchor)
		b.Start("xdr:pic")
		b.Start("xdr:nvPicPr")
		b.Start("xdr:cNvPr", "id", ooxml.Int(id), "name", orDefault(img.Name, fmt.Sprintf("Picture %d", id-1)), "descr", img.Description)
		if linkRel != "" {
			b.Empty("a:hlinkClick", "xmlns:r", ooxml.NSRel, "r:id", linkRel)
		}
		b.End()
		b.Start("xdr:cNvPicPr")
		b.Empty("a:picLocks", "noChangeAspect", boolAttr(img.LockAspect || true))
		b.End().End()
		b.Start("xdr:blipFill")
		b.Empty("a:blip", "xmlns:r", ooxml.NSRel, "r:embed", rel)
		b.Start("a:stretch").Empty("a:fillRect").End()
		b.End()
		b.Start("xdr:spPr")
		xfrmXML(b, img.Anchor)
		b.Start("a:prstGeom", "prst", "rect").Empty("a:avLst").End()
		b.End()
		b.End() // pic
		b.Empty("xdr:clientData")
		b.End() // anchor
	}
	for _, ch := range sh.Charts {
		id++
		p.nChart++
		chartPart := fmt.Sprintf("xl/charts/chart%d.xml", p.nChart)
		data, err := chartXML(ch, p.wb)
		if err != nil {
			return "", err
		}
		if err := p.pw.Add(chartPart, ooxml.CTChart, data); err != nil {
			return "", err
		}
		rel := p.pw.Rel(name, ooxml.RelChart, ooxml.Relative(name, chartPart), "")
		anchorStart(b, chartAnchor(ch))
		b.Start("xdr:graphicFrame", "macro", "")
		b.Start("xdr:nvGraphicFramePr")
		b.Empty("xdr:cNvPr", "id", ooxml.Int(id), "name", orDefault(ch.Name, fmt.Sprintf("Chart %d", p.nChart)))
		b.Empty("xdr:cNvGraphicFramePr")
		b.End()
		b.Start("xdr:xfrm").Empty("a:off", "x", "0", "y", "0").Empty("a:ext", "cx", "0", "cy", "0").End()
		b.Start("a:graphic").Start("a:graphicData", "uri", ooxml.NSChart)
		b.Empty("c:chart", "xmlns:c", ooxml.NSChart, "xmlns:r", ooxml.NSRel, "r:id", rel)
		b.End().End()
		b.End() // graphicFrame
		b.Empty("xdr:clientData")
		b.End()
	}
	for _, shp := range sh.Shapes {
		id++
		if len(shp.Raw) > 0 {
			b.Write(shp.Raw)
			continue
		}
		anchorStart(b, shp.Anchor)
		b.Start("xdr:sp", "macro", "", "textlink", "")
		b.Start("xdr:nvSpPr").Empty("xdr:cNvPr", "id", ooxml.Int(id), "name", orDefault(shp.Name, fmt.Sprintf("Shape %d", id))).Empty("xdr:cNvSpPr").End()
		b.Start("xdr:spPr")
		xfrmXML(b, shp.Anchor)
		b.Start("a:prstGeom", "prst", orDefault(shp.Preset, "rect")).Empty("a:avLst").End()
		if !shp.Fill.IsZero() {
			b.Start("a:solidFill").Empty("a:srgbClr", "val", p.wb.Theme.Resolve(shp.Fill)).End()
		}
		if !shp.Line.IsZero() {
			b.Start("a:ln").Start("a:solidFill").Empty("a:srgbClr", "val", p.wb.Theme.Resolve(shp.Line)).End().End()
		}
		b.End()
		if shp.Text != "" {
			b.Start("xdr:txBody").Empty("a:bodyPr", "wrap", "square", "rtlCol", "0", "anchor", "ctr").Empty("a:lstStyle")
			for line := range strings.SplitSeq(shp.Text, "\n") {
				b.Start("a:p").Empty("a:pPr", "algn", "ctr").Start("a:r").Empty("a:rPr", "lang", "en-US").Elem("a:t", line).End().End()
			}
			b.End()
		}
		b.End()
		b.Empty("xdr:clientData")
		b.End()
	}
	b.End()
	if err := p.pw.Add(name, ooxml.CTDrawing, b.Bytes()); err != nil {
		return "", err
	}
	return p.pw.Rel(sheetPart, ooxml.RelDrawing, ooxml.Relative(sheetPart, name), ""), nil
}

func chartAnchor(ch *chart.Chart) model.Anchor {
	a := ch.Anchor
	an := model.Anchor{
		From: model.CellOffset{Row: a.FromRow, Col: a.FromCol, RowOff: a.FromRowOff, ColOff: a.FromColOff},
		To:   model.CellOffset{Row: a.ToRow, Col: a.ToCol, RowOff: a.ToRowOff, ColOff: a.ToColOff},
	}
	if a.OneCell || a.ToRow == 0 {
		an.Kind = model.AnchorOneCell
		an.Width, an.Height = ch.Width, ch.Height
		if an.Width == 0 {
			an.Width = 5 * model.EMUPerInch
		}
		if an.Height == 0 {
			an.Height = 3 * model.EMUPerInch
		}
	}
	if an.From.Row == 0 {
		an.From.Row = 1
	}
	if an.From.Col == 0 {
		an.From.Col = 1
	}
	return an
}

func anchorStart(b *ooxml.Builder, a model.Anchor) {
	switch a.Kind {
	case model.AnchorOneCell:
		b.Start("xdr:oneCellAnchor")
		markerXML(b, "xdr:from", a.From)
		b.Empty("xdr:ext", "cx", fmt.Sprint(a.Width), "cy", fmt.Sprint(a.Height))
	case model.AnchorAbsolute:
		b.Start("xdr:absoluteAnchor")
		b.Empty("xdr:pos", "x", fmt.Sprint(a.X), "y", fmt.Sprint(a.Y))
		b.Empty("xdr:ext", "cx", fmt.Sprint(a.Width), "cy", fmt.Sprint(a.Height))
	default:
		b.Start("xdr:twoCellAnchor", "editAs", a.EditAs)
		markerXML(b, "xdr:from", a.From)
		markerXML(b, "xdr:to", a.To)
	}
}

func markerXML(b *ooxml.Builder, tag string, c model.CellOffset) {
	b.Start(tag)
	b.Elem("xdr:col", ooxml.Int(max(c.Col-1, 0)))
	b.Elem("xdr:colOff", fmt.Sprint(c.ColOff))
	b.Elem("xdr:row", ooxml.Int(max(c.Row-1, 0)))
	b.Elem("xdr:rowOff", fmt.Sprint(c.RowOff))
	b.End()
}

func xfrmXML(b *ooxml.Builder, a model.Anchor) {
	b.Start("a:xfrm")
	b.Empty("a:off", "x", fmt.Sprint(a.X), "y", fmt.Sprint(a.Y))
	b.Empty("a:ext", "cx", fmt.Sprint(a.Width), "cy", fmt.Sprint(a.Height))
	b.End()
}

// writeChartsheet writes a chart sheet part with its single chart.
func (p *packer) writeChartsheet(i int, name string, sh *model.Sheet) error {
	if len(sh.Charts) == 0 {
		return fmt.Errorf("%w: chartsheet %q has no chart", ErrUnsupported, sh.Name)
	}
	p.nDrawing++
	drawing := fmt.Sprintf("xl/drawings/drawing%d.xml", p.nDrawing)
	p.nChart++
	chartPart := fmt.Sprintf("xl/charts/chart%d.xml", p.nChart)
	data, err := chartXML(sh.Charts[0], p.wb)
	if err != nil {
		return err
	}
	if err := p.pw.Add(chartPart, ooxml.CTChart, data); err != nil {
		return err
	}
	crel := p.pw.Rel(drawing, ooxml.RelChart, ooxml.Relative(drawing, chartPart), "")
	b := ooxml.NewBuilder()
	b.Start("xdr:wsDr", "xmlns:xdr", ooxml.NSSpreadsheetDr, "xmlns:a", ooxml.NSDrawing, "xmlns:r", ooxml.NSRel)
	b.Start("xdr:absoluteAnchor")
	b.Empty("xdr:pos", "x", "0", "y", "0")
	b.Empty("xdr:ext", "cx", "9307200", "cy", "6080760")
	b.Start("xdr:graphicFrame", "macro", "")
	b.Start("xdr:nvGraphicFramePr").Empty("xdr:cNvPr", "id", "2", "name", orDefault(sh.Charts[0].Name, "Chart 1")).Empty("xdr:cNvGraphicFramePr").End()
	b.Start("xdr:xfrm").Empty("a:off", "x", "0", "y", "0").Empty("a:ext", "cx", "0", "cy", "0").End()
	b.Start("a:graphic").Start("a:graphicData", "uri", ooxml.NSChart)
	b.Empty("c:chart", "xmlns:c", ooxml.NSChart, "xmlns:r", ooxml.NSRel, "r:id", crel)
	b.End().End().End()
	b.Empty("xdr:clientData")
	b.End().End()
	if err := p.pw.Add(drawing, ooxml.CTDrawing, b.Bytes()); err != nil {
		return err
	}
	drel := p.pw.Rel(name, ooxml.RelDrawing, ooxml.Relative(name, drawing), "")
	cs := ooxml.NewBuilder()
	cs.Start("chartsheet", "xmlns", ooxml.NSMain, "xmlns:r", ooxml.NSRel)
	cs.Start("sheetPr", "codeName", sh.CodeName)
	colorXML(cs, "tabColor", sh.TabColor)
	cs.End()
	cs.Start("sheetViews").Empty("sheetView", "tabSelected", boolAttr(i == p.wb.View.ActiveTab), "zoomScale", intAttr(sh.View.ZoomScale), "workbookViewId", "0").End()
	m := sh.PageMargins
	if m.IsZero() {
		m = model.DefaultPageMargins()
	}
	cs.Empty("pageMargins", "left", ooxml.Float(m.Left), "right", ooxml.Float(m.Right), "top", ooxml.Float(m.Top), "bottom", ooxml.Float(m.Bottom), "header", ooxml.Float(m.Header), "footer", ooxml.Float(m.Footer))
	cs.Empty("drawing", "r:id", drel)
	cs.End()
	return p.pw.Add(name, ooxml.CTChartsheet, cs.Bytes())
}
