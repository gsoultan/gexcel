// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"fmt"
	"strings"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

// chartXML builds a c:chartSpace part.
func chartXML(ch *chart.Chart, wb *model.Workbook) ([]byte, error) {
	if len(ch.Raw) > 0 {
		return ch.Raw, nil
	}
	if ch.Type.IsChartEx() {
		return nil, fmt.Errorf("%w: chart type %q (Excel 2016 chartEx) is not written yet", ErrUnsupported, ch.Type)
	}
	cw := &chartWriter{b: ooxml.NewBuilder(), theme: wb.Theme, axisID: 100000000}
	b := cw.b
	b.Start("c:chartSpace", "xmlns:c", ooxml.NSChart, "xmlns:a", ooxml.NSDrawing, "xmlns:r", ooxml.NSRel)
	b.Empty("c:date1904", "val", ooxml.Bool(wb.Date1904))
	b.Empty("c:lang", "val", "en-US")
	b.Empty("c:roundedCorners", "val", ooxml.Bool(ch.RoundedCorners))
	st := ch.Style
	if st == 0 {
		st = 2
	}
	b.Start("mc:AlternateContent", "xmlns:mc", ooxml.NSMC)
	b.Start("mc:Choice", "Requires", "c14", "xmlns:c14", "http://schemas.microsoft.com/office/drawing/2007/8/2/chart").Empty("c14:style", "val", ooxml.Int(100+st)).End()
	b.Start("mc:Fallback").Empty("c:style", "val", ooxml.Int(st)).End()
	b.End()
	b.Start("c:chart")
	if ch.Title != nil && !ch.Title.Hidden {
		cw.titleXML(ch.Title, 0)
		b.Empty("c:autoTitleDeleted", "val", "0")
	} else {
		b.Empty("c:autoTitleDeleted", "val", "1")
	}
	if ch.Type.Is3D() || ch.View3D != nil {
		v := ch.View3D
		if v == nil {
			v = &chart.View3D{RotX: 15, RotY: 20}
		}
		b.Start("c:view3D")
		b.Empty("c:rotX", "val", ooxml.Int(v.RotX))
		b.Empty("c:rotY", "val", ooxml.Int(v.RotY))
		if v.DepthPercent > 0 {
			b.Empty("c:depthPercent", "val", ooxml.Int(v.DepthPercent))
		}
		if v.RightAngleAxes != nil {
			b.Empty("c:rAngAx", "val", ooxml.Bool(*v.RightAngleAxes))
		} else {
			b.Empty("c:rAngAx", "val", "1")
		}
		if v.Perspective > 0 {
			b.Empty("c:perspective", "val", ooxml.Int(v.Perspective))
		}
		b.End()
	}
	b.Start("c:plotArea")
	b.Empty("c:layout")
	// Primary group.
	primary := []int{cw.newAxisID(), cw.newAxisID()}
	if ch.Type.Is3D() && ch.Type.Family() != "pie" {
		primary = append(primary, cw.newAxisID())
	}
	cw.groupXML(ch, ch.Type, ch.Series, ch.GapWidth, ch.Overlap, primary, 0)
	var secondary []int
	for i, ov := range ch.Overlays {
		axes := primary
		if ov.Secondary {
			if secondary == nil {
				secondary = []int{cw.newAxisID(), cw.newAxisID()}
			}
			axes = secondary
		}
		cw.groupXML(ch, ov.Type, ov.Series, ov.GapWidth, ov.Overlap, axes, i+1+len(ch.Series))
	}
	if ch.Type.HasCategoryAxis() {
		cw.axesXML(ch, ch.Type, primary, false)
		if secondary != nil {
			ovType := ch.Type
			for _, ov := range ch.Overlays {
				if ov.Secondary {
					ovType = ov.Type
					break
				}
			}
			cw.axesXML(ch, ovType, secondary, true)
		}
	}
	if dt := ch.DataTable; dt != nil {
		b.Start("c:dTable")
		b.Empty("c:showHorzBorder", "val", ooxml.Bool(dt.ShowHorzBorder))
		b.Empty("c:showVertBorder", "val", ooxml.Bool(dt.ShowVertBorder))
		b.Empty("c:showOutline", "val", ooxml.Bool(dt.ShowOutline))
		b.Empty("c:showKeys", "val", ooxml.Bool(dt.ShowKeys))
		b.End()
	}
	if !ch.PlotAreaFill.IsZero() {
		b.Start("c:spPr")
		cw.solidFill(ch.PlotAreaFill)
		b.End()
	}
	b.End() // plotArea
	if ch.Legend == nil || !ch.Legend.Hidden {
		lg := ch.Legend
		if lg == nil {
			lg = &chart.Legend{}
		}
		pos := lg.Position
		if pos == "" {
			pos = chart.LegendRight
			if ch.Type.Family() == "pie" || ch.Type.Family() == "doughnut" {
				pos = chart.LegendBottom
			}
		}
		b.Start("c:legend")
		b.Empty("c:legendPos", "val", string(pos))
		b.Empty("c:overlay", "val", ooxml.Bool(lg.Overlay))
		if !lg.Font.IsZero() {
			cw.txPr(lg.Font, 0)
		}
		b.End()
	}
	pv := true
	if ch.PlotVisibleOnly != nil {
		pv = *ch.PlotVisibleOnly
	}
	b.Empty("c:plotVisOnly", "val", ooxml.Bool(pv))
	b.Empty("c:dispBlanksAs", "val", orDefault(ch.DisplayBlanksAs, "gap"))
	b.End() // chart
	if !ch.Background.IsZero() || !ch.Font.IsZero() {
		if !ch.Background.IsZero() {
			b.Start("c:spPr")
			cw.solidFill(ch.Background)
			b.End()
		}
		if !ch.Font.IsZero() {
			cw.txPr(ch.Font, 0)
		}
	}
	b.Empty("c:printSettings")
	b.End()
	return b.Bytes(), nil
}

type chartWriter struct {
	b      *ooxml.Builder
	theme  *style.Theme
	axisID int
}

func (cw *chartWriter) newAxisID() int {
	cw.axisID++
	return cw.axisID
}

func (cw *chartWriter) rgb(c style.Color) string { return cw.theme.Resolve(c) }

func (cw *chartWriter) solidFill(c style.Color) {
	b := cw.b
	b.Start("a:solidFill")
	if c.Kind == style.ColorRGB && len(c.ARGB) == 8 && c.ARGB[:2] != "FF" {
		alpha := 100000 * int(hexByte(c.ARGB[:2])) / 255
		b.Start("a:srgbClr", "val", c.ARGB[2:]).Empty("a:alpha", "val", ooxml.Int(alpha)).End()
	} else {
		b.Empty("a:srgbClr", "val", cw.rgb(c))
	}
	b.End()
}

func hexByte(s string) byte {
	var v byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= c - '0'
		case c >= 'a' && c <= 'f':
			v |= c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v |= c - 'A' + 10
		}
	}
	return v
}

// spPr writes shape properties for a fill and line; empty when both are unset.
func (cw *chartWriter) spPr(fill style.Color, line chart.LineStyle, forceLine bool) {
	if fill.IsZero() && line == (chart.LineStyle{}) && !forceLine {
		return
	}
	b := cw.b
	b.Start("c:spPr")
	if !fill.IsZero() {
		cw.solidFill(fill)
	}
	cw.lineXML(line, forceLine)
	b.End()
}

func (cw *chartWriter) lineXML(line chart.LineStyle, force bool) {
	if line == (chart.LineStyle{}) && !force {
		return
	}
	b := cw.b
	attrs := []string{}
	if line.Width > 0 {
		attrs = append(attrs, "w", ooxml.Int(int(line.Width*model.EMUPerPoint)))
	}
	attrs = append(attrs, "cap", "rnd")
	b.Start("a:ln", attrs...)
	switch {
	case line.NoLine:
		b.Empty("a:noFill")
	case !line.Color.IsZero():
		cw.solidFill(line.Color)
	}
	if line.Dash != "" {
		b.Empty("a:prstDash", "val", line.Dash)
	}
	b.Empty("a:round")
	b.End()
}

func (cw *chartWriter) txPr(f style.Font, rot int) {
	b := cw.b
	b.Start("c:txPr")
	if rot != 0 {
		b.Empty("a:bodyPr", "rot", ooxml.Int(rot*60000))
	} else {
		b.Empty("a:bodyPr")
	}
	b.Empty("a:lstStyle")
	b.Start("a:p").Start("a:pPr")
	cw.rPr("a:defRPr", f)
	b.End().Empty("a:endParaRPr", "lang", "en-US").End()
	b.End()
}

func (cw *chartWriter) rPr(tag string, f style.Font) {
	b := cw.b
	attrs := []string{}
	if f.Size > 0 {
		attrs = append(attrs, "sz", ooxml.Int(int(f.Size*100)))
	}
	attrs = append(attrs, "b", map[bool]string{true: "1", false: "0"}[f.Bold], "i", map[bool]string{true: "1", false: ""}[f.Italic])
	if f.Underline != "" {
		attrs = append(attrs, "u", "sng")
	}
	if f.Strike {
		attrs = append(attrs, "strike", "sngStrike")
	}
	if f.Color.IsZero() && f.Name == "" {
		b.Empty(tag, attrs...)
		return
	}
	b.Start(tag, attrs...)
	if !f.Color.IsZero() {
		cw.solidFill(f.Color)
	}
	if f.Name != "" {
		b.Empty("a:latin", "typeface", f.Name)
		b.Empty("a:cs", "typeface", f.Name)
	}
	b.End()
}

func (cw *chartWriter) titleXML(t *chart.Title, rot int) {
	b := cw.b
	b.Start("c:title")
	if t.Formula != "" || t.Text != "" {
		b.Start("c:tx")
		if t.Formula != "" {
			b.Start("c:strRef").Elem("c:f", strings.TrimPrefix(t.Formula, "=")).End()
		} else {
			b.Start("c:rich")
			r := rot
			if t.Rotation != 0 {
				r = t.Rotation
			}
			if r != 0 {
				b.Empty("a:bodyPr", "rot", ooxml.Int(r*60000), "vert", "horz")
			} else {
				b.Empty("a:bodyPr")
			}
			b.Empty("a:lstStyle")
			for line := range strings.SplitSeq(t.Text, "\n") {
				b.Start("a:p").Start("a:pPr")
				cw.rPr("a:defRPr", t.Font)
				b.End()
				b.Start("a:r")
				cw.rPr("a:rPr", t.Font)
				b.Elem("a:t", line)
				b.End().End()
			}
			b.End()
		}
		b.End()
	}
	b.Empty("c:overlay", "val", ooxml.Bool(t.Overlay))
	b.End()
}

// groupXML writes one chart group element (c:barChart, c:lineChart, ...).
func (cw *chartWriter) groupXML(ch *chart.Chart, t chart.Type, series []chart.Series, gap, overlap int, axes []int, firstIdx int) {
	b := cw.b
	fam := t.Family()
	elem := "c:" + fam + "Chart"
	if t.Is3D() && fam != "surface" {
		elem = "c:" + fam + "3DChart"
	}
	if t == chart.Surface3D || t == chart.Surface {
		elem = "c:surface3DChart"
	}
	b.Start(elem)
	switch fam {
	case "bar":
		dir := "col"
		if t.IsBar() {
			dir = "bar"
		}
		b.Empty("c:barDir", "val", dir)
		b.Empty("c:grouping", "val", t.Grouping())
	case "line", "area":
		b.Empty("c:grouping", "val", t.Grouping())
	case "radar":
		rs := "marker"
		if t == chart.RadarFilled {
			rs = "filled"
		} else if t == chart.Radar {
			rs = "standard"
		}
		b.Empty("c:radarStyle", "val", rs)
	case "scatter":
		ss := "lineMarker"
		switch t {
		case chart.ScatterSmooth, chart.ScatterSmoothNM:
			ss = "smoothMarker"
		case chart.Scatter:
			ss = "lineMarker"
		case chart.ScatterLines, chart.ScatterLinesNM:
			ss = "lineMarker"
		}
		b.Empty("c:scatterStyle", "val", ss)
	case "ofPie":
		op := "pie"
		if t == chart.BarOfPie {
			op = "bar"
		}
		b.Empty("c:ofPieType", "val", op)
	case "surface":
		if t == chart.SurfaceWire {
			b.Empty("c:wireframe", "val", "1")
		}
	}
	if fam != "surface" && fam != "stock" {
		vc := fam == "pie" || fam == "doughnut" || fam == "ofPie"
		if ch.VaryColors != nil {
			vc = *ch.VaryColors
		}
		b.Empty("c:varyColors", "val", ooxml.Bool(vc))
	}
	for i := range series {
		cw.seriesXML(ch, t, &series[i], firstIdx+i)
	}
	if fam == "line" && (t == chart.Line || t == chart.LineStacked || t == chart.LinePercent) {
		b.Empty("c:marker", "val", "1")
	}
	if fam == "bar" {
		gw := gap
		if gw == 0 {
			gw = 150
			if t.Is3D() {
				gw = 150
			}
		}
		b.Empty("c:gapWidth", "val", ooxml.Int(gw))
		ov := overlap
		if ov == 0 && (t == chart.ColumnStacked || t == chart.ColumnPercent || t == chart.BarStacked || t == chart.BarPercent) {
			ov = 100
		}
		if ov != 0 && !t.Is3D() {
			b.Empty("c:overlap", "val", ooxml.Int(ov))
		}
		if t.Is3D() {
			b.Empty("c:shape", "val", "box")
		}
	}
	if fam == "ofPie" {
		if gap > 0 {
			b.Empty("c:gapWidth", "val", ooxml.Int(gap))
		}
		b.Empty("c:secondPieSize", "val", "75")
	}
	if fam == "pie" || fam == "doughnut" {
		b.Empty("c:firstSliceAng", "val", ooxml.Int(ch.FirstSliceAngle))
		if fam == "doughnut" {
			hs := ch.HoleSize
			if hs == 0 {
				hs = 75
			}
			b.Empty("c:holeSize", "val", ooxml.Int(hs))
		}
	}
	if fam == "bubble" {
		b.Empty("c:bubbleScale", "val", "100")
	}
	if t.HasCategoryAxis() {
		for _, id := range axes {
			b.Empty("c:axId", "val", ooxml.Int(id))
		}
	}
	b.End()
}

func (cw *chartWriter) seriesXML(ch *chart.Chart, t chart.Type, s *chart.Series, idx int) {
	b := cw.b
	fam := t.Family()
	b.Start("c:ser")
	b.Empty("c:idx", "val", ooxml.Int(idx))
	b.Empty("c:order", "val", ooxml.Int(idx))
	if s.NameRef != "" || s.Name != "" {
		b.Start("c:tx")
		if s.NameRef != "" {
			b.Start("c:strRef").Elem("c:f", strings.TrimPrefix(s.NameRef, "=")).End()
		} else {
			b.Elem("c:v", s.Name)
		}
		b.End()
	}
	line := s.Line
	if (fam == "line" || fam == "scatter" || fam == "radar") && !s.Fill.IsZero() && line.Color.IsZero() {
		line.Color = s.Fill
	}
	if t == chart.ScatterSmoothNM || t == chart.ScatterLinesNM || t == chart.Scatter && false {
		// handled by marker none
	}
	if fam == "line" || fam == "scatter" || fam == "radar" {
		if t == chart.Scatter {
			// Markers only: no line.
			line.NoLine = true
		}
		b.Start("c:spPr")
		cw.lineXML(line, true)
		b.End()
	} else {
		cw.spPr(s.Fill, s.Line, false)
	}
	if s.InvertIfNegative != nil {
		b.Empty("c:invertIfNegative", "val", ooxml.Bool(*s.InvertIfNegative))
	} else if fam == "bar" || fam == "bubble" {
		b.Empty("c:invertIfNegative", "val", "0")
	}
	if fam == "line" || fam == "scatter" || fam == "radar" {
		m := s.Marker
		if m == nil {
			switch t {
			case chart.Line, chart.LineStacked, chart.LinePercent, chart.Line3D, chart.ScatterSmoothNM, chart.ScatterLinesNM, chart.Radar:
				m = &chart.Marker{Symbol: "none"}
			}
		}
		if m != nil {
			b.Start("c:marker")
			b.Empty("c:symbol", "val", orDefault(m.Symbol, "circle"))
			if m.Size > 0 && m.Symbol != "none" {
				b.Empty("c:size", "val", ooxml.Int(m.Size))
			}
			if !m.Fill.IsZero() || !m.Line.IsZero() {
				b.Start("c:spPr")
				if !m.Fill.IsZero() {
					cw.solidFill(m.Fill)
				}
				if !m.Line.IsZero() {
					cw.lineXML(chart.LineStyle{Color: m.Line}, false)
				}
				b.End()
			}
			b.End()
		}
	}
	for _, dp := range s.DataPoints {
		b.Start("c:dPt")
		b.Empty("c:idx", "val", ooxml.Int(dp.Index))
		if dp.InvertIfNegative != nil {
			b.Empty("c:invertIfNegative", "val", ooxml.Bool(*dp.InvertIfNegative))
		}
		if dp.Marker != nil {
			b.Start("c:marker").Empty("c:symbol", "val", orDefault(dp.Marker.Symbol, "circle")).End()
		}
		b.Empty("c:bubble3D", "val", "0")
		if dp.Explosion > 0 {
			b.Empty("c:explosion", "val", ooxml.Int(dp.Explosion))
		}
		cw.spPr(dp.Fill, dp.Line, false)
		b.End()
	}
	if s.Explosion > 0 && (fam == "pie" || fam == "ofPie") {
		b.Empty("c:explosion", "val", ooxml.Int(s.Explosion))
	}
	if dl := s.DataLabels; dl != nil {
		cw.dataLabelsXML(dl)
	}
	for _, tl := range s.Trendlines {
		b.Start("c:trendline")
		if tl.Name != "" {
			b.Elem("c:name", tl.Name)
		}
		cw.spPr(style.Color{}, tl.Line, false)
		b.Empty("c:trendlineType", "val", orDefault(tl.Type, "linear"))
		if tl.Type == "poly" {
			b.Empty("c:order", "val", ooxml.Int(max(tl.Order, 2)))
		}
		if tl.Type == "movingAvg" {
			b.Empty("c:period", "val", ooxml.Int(max(tl.Period, 2)))
		}
		if tl.Forward != 0 {
			b.Empty("c:forward", "val", ooxml.Float(tl.Forward))
		}
		if tl.Backward != 0 {
			b.Empty("c:backward", "val", ooxml.Float(tl.Backward))
		}
		if tl.Intercept != nil {
			b.Empty("c:intercept", "val", ooxml.Float(*tl.Intercept))
		}
		b.Empty("c:dispRSqr", "val", ooxml.Bool(tl.DisplayRSquared))
		b.Empty("c:dispEq", "val", ooxml.Bool(tl.DisplayEquation))
		b.End()
	}
	for _, eb := range s.ErrorBars {
		b.Start("c:errBars")
		if eb.Direction != "" {
			b.Empty("c:errDir", "val", eb.Direction)
		}
		b.Empty("c:errBarType", "val", orDefault(eb.Type, "both"))
		b.Empty("c:errValType", "val", orDefault(eb.ValueType, "fixedVal"))
		b.Empty("c:noEndCap", "val", ooxml.Bool(eb.NoEndCap))
		if eb.ValueType == "cust" {
			if eb.Plus != "" {
				b.Start("c:plus").Start("c:numRef").Elem("c:f", eb.Plus).End().End()
			}
			if eb.Minus != "" {
				b.Start("c:minus").Start("c:numRef").Elem("c:f", eb.Minus).End().End()
			}
		} else if eb.ValueType != "stdErr" {
			b.Empty("c:val", "val", ooxml.Float(eb.Value))
		}
		cw.spPr(style.Color{}, eb.Line, false)
		b.End()
	}
	catTag, valTag := "c:cat", "c:val"
	if fam == "scatter" || fam == "bubble" {
		catTag, valTag = "c:xVal", "c:yVal"
	}
	if s.Categories != "" || len(s.CatLiterals) > 0 {
		b.Start(catTag)
		if s.Categories != "" {
			refTag := "c:strRef"
			if fam == "scatter" || fam == "bubble" {
				refTag = "c:numRef"
			}
			b.Start(refTag).Elem("c:f", strings.TrimPrefix(s.Categories, "=")).End()
		} else {
			b.Start("c:strLit")
			b.Empty("c:ptCount", "val", ooxml.Int(len(s.CatLiterals)))
			for i, v := range s.CatLiterals {
				b.Start("c:pt", "idx", ooxml.Int(i)).Elem("c:v", v).End()
			}
			b.End()
		}
		b.End()
	}
	b.Start(valTag)
	if s.Values != "" {
		b.Start("c:numRef").Elem("c:f", strings.TrimPrefix(s.Values, "=")).End()
	} else {
		b.Start("c:numLit")
		b.Empty("c:formatCode", "val", "General")
		b.Empty("c:ptCount", "val", ooxml.Int(len(s.ValLiterals)))
		for i, v := range s.ValLiterals {
			b.Start("c:pt", "idx", ooxml.Int(i)).Elem("c:v", ooxml.Float(v)).End()
		}
		b.End()
	}
	b.End()
	if fam == "bubble" {
		b.Start("c:bubbleSize")
		if s.Bubbles != "" {
			b.Start("c:numRef").Elem("c:f", strings.TrimPrefix(s.Bubbles, "=")).End()
		} else {
			b.Start("c:numLit").Empty("c:ptCount", "val", "0").End()
		}
		b.End()
		b.Empty("c:bubble3D", "val", ooxml.Bool(t == chart.Bubble3D))
	}
	if fam == "line" || fam == "scatter" {
		smooth := s.Smooth || t == chart.ScatterSmooth || t == chart.ScatterSmoothNM
		b.Empty("c:smooth", "val", ooxml.Bool(smooth))
	}
	if s.Shape != "" {
		b.Empty("c:shape", "val", s.Shape)
	}
	b.End()
}

func (cw *chartWriter) dataLabelsXML(dl *chart.DataLabels) {
	b := cw.b
	b.Start("c:dLbls")
	if dl.NumFmt != "" {
		b.Empty("c:numFmt", "formatCode", dl.NumFmt, "sourceLinked", "0")
	}
	if !dl.Font.IsZero() {
		cw.txPr(dl.Font, 0)
	}
	if dl.Position != "" {
		b.Empty("c:dLblPos", "val", dl.Position)
	}
	b.Empty("c:showLegendKey", "val", ooxml.Bool(dl.ShowLegendKey))
	b.Empty("c:showVal", "val", ooxml.Bool(dl.ShowValue))
	b.Empty("c:showCatName", "val", ooxml.Bool(dl.ShowCategory))
	b.Empty("c:showSerName", "val", ooxml.Bool(dl.ShowSeriesName))
	b.Empty("c:showPercent", "val", ooxml.Bool(dl.ShowPercent))
	b.Empty("c:showBubbleSize", "val", ooxml.Bool(dl.ShowBubbleSize))
	if dl.Separator != "" {
		b.Elem("c:separator", dl.Separator)
	}
	if dl.ShowLeaderLines {
		b.Empty("c:showLeaderLines", "val", "1")
	}
	b.End()
}

// axesXML writes the category/X axis and value/Y axis (and the series
// axis for 3-D charts) for an axis id pair.
func (cw *chartWriter) axesXML(ch *chart.Chart, t chart.Type, ids []int, secondary bool) {
	fam := t.Family()
	xa, ya := ch.XAxis, ch.YAxis
	catPos, valPos := "b", "l"
	if t.IsBar() {
		catPos, valPos = "l", "b"
	}
	if secondary {
		catPos, valPos = "b", "r"
		if t.IsBar() {
			catPos, valPos = "l", "t"
		}
	}
	if xa.Position != "" {
		catPos = xa.Position
	}
	if ya.Position != "" {
		valPos = ya.Position
	}
	deleted := xa.Deleted || xa.Hidden
	if secondary {
		deleted = true
	}
	switch {
	case fam == "scatter" || fam == "bubble":
		cw.valAxXML(&xa, ids[0], ids[1], catPos, deleted, "midCat", t)
	case xa.DateAxis:
		cw.dateAxXML(&xa, ids[0], ids[1], catPos, deleted)
	default:
		cw.catAxXML(&xa, ids[0], ids[1], catPos, deleted, fam)
	}
	cb := ya.CrossBetween
	if cb == "" {
		cb = "between"
		if fam == "line" || fam == "scatter" || fam == "radar" || fam == "bubble" {
			cb = "midCat"
		}
		if fam == "area" {
			cb = "midCat"
		}
	}
	cw.valAxXML(&ya, ids[1], ids[0], valPos, ya.Deleted || ya.Hidden, cb, t)
	if len(ids) > 2 {
		b := cw.b
		b.Start("c:serAx")
		b.Empty("c:axId", "val", ooxml.Int(ids[2]))
		b.Start("c:scaling").Empty("c:orientation", "val", "minMax").End()
		b.Empty("c:delete", "val", ooxml.Bool(ch.ZAxis.Deleted || ch.ZAxis.Hidden))
		b.Empty("c:axPos", "val", "b")
		if ch.ZAxis.Title != nil {
			cw.titleXML(ch.ZAxis.Title, 0)
		}
		b.Empty("c:majorTickMark", "val", "out")
		b.Empty("c:minorTickMark", "val", "none")
		b.Empty("c:tickLblPos", "val", "nextTo")
		b.Empty("c:crossAx", "val", ooxml.Int(ids[1]))
		b.Empty("c:crosses", "val", "autoZero")
		b.End()
	}
}

func (cw *chartWriter) axisCommon(a *chart.Axis, id, crossID int, pos string, deleted bool, rot int) {
	b := cw.b
	b.Empty("c:axId", "val", ooxml.Int(id))
	b.Start("c:scaling")
	if a.LogBase > 0 {
		b.Empty("c:logBase", "val", ooxml.Float(a.LogBase))
	}
	orient := orDefault(a.Orientation, "minMax")
	if a.Reverse {
		orient = "maxMin"
	}
	b.Empty("c:orientation", "val", orient)
	if a.Max != nil {
		b.Empty("c:max", "val", ooxml.Float(*a.Max))
	}
	if a.Min != nil {
		b.Empty("c:min", "val", ooxml.Float(*a.Min))
	}
	b.End()
	b.Empty("c:delete", "val", ooxml.Bool(deleted))
	b.Empty("c:axPos", "val", pos)
	if a.MajorGridlines {
		b.Empty("c:majorGridlines")
	}
	if a.MinorGridlines {
		b.Empty("c:minorGridlines")
	}
	if a.Title != nil && !a.Title.Hidden {
		cw.titleXML(a.Title, rot)
	}
	if a.NumFmt != "" {
		linked := "0"
		if a.NumFmtLinked != nil && *a.NumFmtLinked {
			linked = "1"
		}
		b.Empty("c:numFmt", "formatCode", a.NumFmt, "sourceLinked", linked)
	} else {
		b.Empty("c:numFmt", "formatCode", "General", "sourceLinked", "1")
	}
	b.Empty("c:majorTickMark", "val", orDefault(a.MajorTickMark, "out"))
	b.Empty("c:minorTickMark", "val", orDefault(a.MinorTickMark, "none"))
	b.Empty("c:tickLblPos", "val", orDefault(a.TickLabelPos, "nextTo"))
	if a.Line != (chart.LineStyle{}) {
		cw.spPr(style.Color{}, a.Line, false)
	}
	if !a.LabelFont.IsZero() || a.TextRotation != 0 {
		cw.txPr(a.LabelFont, a.TextRotation)
	}
	b.Empty("c:crossAx", "val", ooxml.Int(crossID))
	if a.CrossesAt != nil {
		b.Empty("c:crossesAt", "val", ooxml.Float(*a.CrossesAt))
	} else {
		b.Empty("c:crosses", "val", orDefault(a.Crosses, "autoZero"))
	}
}

func (cw *chartWriter) catAxXML(a *chart.Axis, id, crossID int, pos string, deleted bool, fam string) {
	b := cw.b
	b.Start("c:catAx")
	cw.axisCommon(a, id, crossID, pos, deleted, 0)
	b.Empty("c:auto", "val", "1")
	b.Empty("c:lblAlgn", "val", orDefault(a.LabelAlign, "ctr"))
	b.Empty("c:lblOffset", "val", ooxml.Int(orInt(a.LabelOffset, 100)))
	if a.TickLabelSkip > 0 {
		b.Empty("c:tickLblSkip", "val", ooxml.Int(a.TickLabelSkip))
	}
	if a.TickMarkSkip > 0 {
		b.Empty("c:tickMarkSkip", "val", ooxml.Int(a.TickMarkSkip))
	}
	b.Empty("c:noMultiLvlLbl", "val", ooxml.Bool(a.NoMultiLvlLbl))
	b.End()
}

func (cw *chartWriter) dateAxXML(a *chart.Axis, id, crossID int, pos string, deleted bool) {
	b := cw.b
	b.Start("c:dateAx")
	cw.axisCommon(a, id, crossID, pos, deleted, 0)
	b.Empty("c:auto", "val", "1")
	b.Empty("c:lblOffset", "val", ooxml.Int(orInt(a.LabelOffset, 100)))
	b.Empty("c:baseTimeUnit", "val", orDefault(a.BaseTimeUnit, "days"))
	if a.MajorUnit != nil {
		b.Empty("c:majorUnit", "val", ooxml.Float(*a.MajorUnit))
		b.Empty("c:majorTimeUnit", "val", orDefault(a.MajorTimeUnit, "days"))
	}
	if a.MinorUnit != nil {
		b.Empty("c:minorUnit", "val", ooxml.Float(*a.MinorUnit))
		b.Empty("c:minorTimeUnit", "val", orDefault(a.MinorTimeUnit, "days"))
	}
	b.End()
}

func (cw *chartWriter) valAxXML(a *chart.Axis, id, crossID int, pos string, deleted bool, crossBetween string, t chart.Type) {
	b := cw.b
	b.Start("c:valAx")
	rot := 0
	if pos == "l" || pos == "r" {
		rot = -90
	}
	if t.IsBar() && (pos == "l" || pos == "r") {
		rot = -90
	}
	cw.axisCommon(a, id, crossID, pos, deleted, rot)
	b.Empty("c:crossBetween", "val", crossBetween)
	if a.MajorUnit != nil {
		b.Empty("c:majorUnit", "val", ooxml.Float(*a.MajorUnit))
	}
	if a.MinorUnit != nil {
		b.Empty("c:minorUnit", "val", ooxml.Float(*a.MinorUnit))
	}
	if a.DisplayUnits != "" {
		b.Start("c:dispUnits").Empty("c:builtInUnit", "val", a.DisplayUnits).Start("c:dispUnitsLbl").End().End()
	}
	b.End()
}

func orInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
