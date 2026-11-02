// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

// parseChart converts a c:chartSpace part into a chart.Chart. The raw
// bytes are kept too, so unsupported details survive a round trip when the
// chart is not modified.
func parseChart(data []byte) *chart.Chart {
	root, err := ooxml.ParseDOM(data)
	if err != nil || root == nil {
		return nil
	}
	c := root.Child("chart")
	if c == nil {
		return nil
	}
	ch := &chart.Chart{Raw: data}
	if st := root.Child("style"); st != nil {
		ch.Style = st.ValInt(0)
	}
	if rc := root.Child("roundedCorners"); rc != nil {
		ch.RoundedCorners = rc.ValBool()
	}
	if t := c.Child("title"); t != nil {
		ch.Title = parseTitle(t)
	}
	if ad := c.Child("autoTitleDeleted"); ad != nil && ad.ValBool() && ch.Title == nil {
		ch.Title = nil
	}
	if v := c.Child("view3D"); v != nil {
		ch.View3D = &chart.View3D{RotX: v.Child("rotX").ValInt(0), RotY: v.Child("rotY").ValInt(0), Perspective: v.Child("perspective").ValInt(0), DepthPercent: v.Child("depthPercent").ValInt(0)}
		if r := v.Child("rAngAx"); r != nil {
			b := r.ValBool()
			ch.View3D.RightAngleAxes = &b
		}
	}
	pa := c.Child("plotArea")
	if pa == nil {
		return ch
	}
	type group struct {
		node   *ooxml.Node
		typ    chart.Type
		axisID []int
	}
	var groups []group
	for _, n := range pa.Children {
		if !strings.HasSuffix(n.Name, "Chart") {
			continue
		}
		g := group{node: n, typ: chartTypeOf(n)}
		for _, a := range n.Each("axId") {
			g.axisID = append(g.axisID, a.ValInt(0))
		}
		groups = append(groups, g)
	}
	if len(groups) == 0 {
		return ch
	}
	ch.Type = groups[0].typ
	ch.Series = parseSeriesList(groups[0].node, groups[0].typ)
	ch.GapWidth = groups[0].node.Child("gapWidth").ValInt(0)
	ch.Overlap = groups[0].node.Child("overlap").ValInt(0)
	ch.FirstSliceAngle = groups[0].node.Child("firstSliceAng").ValInt(0)
	ch.HoleSize = groups[0].node.Child("holeSize").ValInt(0)
	if vc := groups[0].node.Child("varyColors"); vc != nil {
		b := vc.ValBool()
		ch.VaryColors = &b
	}
	primaryAxes := groups[0].axisID
	for _, g := range groups[1:] {
		ov := chart.Overlay{Type: g.typ, Series: parseSeriesList(g.node, g.typ), GapWidth: g.node.Child("gapWidth").ValInt(0), Overlap: g.node.Child("overlap").ValInt(0)}
		if len(g.axisID) > 0 && len(primaryAxes) > 0 && g.axisID[0] != primaryAxes[0] {
			ov.Secondary = true
		}
		ch.Overlays = append(ch.Overlays, ov)
	}
	// Axes: match by id to the primary group.
	axisByID := map[int]*ooxml.Node{}
	for _, n := range pa.Children {
		switch n.Name {
		case "catAx", "valAx", "dateAx", "serAx":
			axisByID[n.Child("axId").ValInt(0)] = n
		}
	}
	if len(primaryAxes) > 0 {
		if n := axisByID[primaryAxes[0]]; n != nil {
			ch.XAxis = parseAxis(n)
		}
	}
	if len(primaryAxes) > 1 {
		if n := axisByID[primaryAxes[1]]; n != nil {
			ch.YAxis = parseAxis(n)
		}
	}
	if len(primaryAxes) > 2 {
		if n := axisByID[primaryAxes[2]]; n != nil {
			ch.ZAxis = parseAxis(n)
		}
	}
	if dt := pa.Child("dTable"); dt != nil {
		ch.DataTable = &chart.DataTable{ShowHorzBorder: dt.Child("showHorzBorder").ValBool(), ShowVertBorder: dt.Child("showVertBorder").ValBool(), ShowOutline: dt.Child("showOutline").ValBool(), ShowKeys: dt.Child("showKeys").ValBool()}
	}
	if sp := pa.Child("spPr"); sp != nil {
		ch.PlotAreaFill = parseSolidFill(sp)
	}
	if lg := c.Child("legend"); lg != nil {
		ch.Legend = &chart.Legend{Position: chart.LegendPosition(lg.Child("legendPos").Val()), Overlay: lg.Child("overlay").ValBool()}
		if tx := lg.Child("txPr"); tx != nil {
			ch.Legend.Font = parseTxPr(tx)
		}
	} else {
		ch.Legend = &chart.Legend{Hidden: true}
	}
	if pv := c.Child("plotVisOnly"); pv != nil {
		b := pv.ValBool()
		ch.PlotVisibleOnly = &b
	}
	if db := c.Child("dispBlanksAs"); db != nil {
		ch.DisplayBlanksAs = db.Val()
	}
	if sp := root.Child("spPr"); sp != nil {
		ch.Background = parseSolidFill(sp)
	}
	if tx := root.Child("txPr"); tx != nil {
		ch.Font = parseTxPr(tx)
	}
	return ch
}

func chartTypeOf(n *ooxml.Node) chart.Type {
	grouping := n.Child("grouping").Val()
	switch n.Name {
	case "barChart", "bar3DChart":
		bar := n.Child("barDir").Val() == "bar"
		three := n.Name == "bar3DChart"
		switch {
		case three && bar:
			return chart.Bar3D
		case three:
			return chart.Column3D
		case bar && grouping == "stacked":
			return chart.BarStacked
		case bar && grouping == "percentStacked":
			return chart.BarPercent
		case bar:
			return chart.Bar
		case grouping == "stacked":
			return chart.ColumnStacked
		case grouping == "percentStacked":
			return chart.ColumnPercent
		}
		return chart.Column
	case "lineChart", "line3DChart":
		if n.Name == "line3DChart" {
			return chart.Line3D
		}
		switch grouping {
		case "stacked":
			return chart.LineStacked
		case "percentStacked":
			return chart.LinePercent
		}
		if m := n.Child("marker"); m != nil && m.ValBool() {
			// Series markers decide; keep plain line unless series have markers.
			for _, s := range n.Each("ser") {
				if mk := s.Path("marker", "symbol"); mk != nil && mk.Val() != "none" {
					return chart.LineMarkers
				}
			}
		}
		return chart.Line
	case "pieChart":
		return chart.Pie
	case "pie3DChart":
		return chart.Pie3D
	case "ofPieChart":
		if n.Child("ofPieType").Val() == "bar" {
			return chart.BarOfPie
		}
		return chart.PieOfPie
	case "doughnutChart":
		return chart.Doughnut
	case "areaChart", "area3DChart":
		if n.Name == "area3DChart" {
			return chart.Area3D
		}
		switch grouping {
		case "stacked":
			return chart.AreaStacked
		case "percentStacked":
			return chart.AreaPercent
		}
		return chart.Area
	case "scatterChart":
		st := n.Child("scatterStyle").Val()
		noMarkers := false
		noLine := false
		for _, s := range n.Each("ser") {
			if mk := s.Path("marker", "symbol"); mk != nil && mk.Val() == "none" {
				noMarkers = true
			}
			if s.Path("spPr", "ln", "noFill") != nil {
				noLine = true
			}
		}
		switch {
		case st == "smoothMarker" && noMarkers:
			return chart.ScatterSmoothNM
		case st == "smoothMarker":
			return chart.ScatterSmooth
		case noLine:
			return chart.Scatter
		case noMarkers:
			return chart.ScatterLinesNM
		}
		return chart.ScatterLines
	case "bubbleChart":
		for _, s := range n.Each("ser") {
			if b := s.Child("bubble3D"); b != nil && b.ValBool() {
				return chart.Bubble3D
			}
		}
		return chart.Bubble
	case "radarChart":
		switch n.Child("radarStyle").Val() {
		case "filled":
			return chart.RadarFilled
		case "marker":
			return chart.RadarMarkers
		}
		return chart.Radar
	case "stockChart":
		return chart.Stock
	case "surfaceChart":
		return chart.Surface
	case "surface3DChart":
		if n.Child("wireframe") != nil && n.Child("wireframe").ValBool() {
			return chart.SurfaceWire
		}
		return chart.Surface3D
	}
	return chart.Column
}

func parseSeriesList(n *ooxml.Node, t chart.Type) []chart.Series {
	var out []chart.Series
	for _, s := range n.Each("ser") {
		out = append(out, parseSeries(s, t))
	}
	return out
}

func refOf(n *ooxml.Node) string {
	if n == nil {
		return ""
	}
	for _, kind := range []string{"strRef", "numRef", "multiLvlStrRef"} {
		if r := n.Child(kind); r != nil {
			if f := r.Child("f"); f != nil {
				return strings.TrimSpace(f.Text)
			}
		}
	}
	return ""
}

func parseSeries(s *ooxml.Node, t chart.Type) chart.Series {
	ser := chart.Series{Index: s.Child("idx").ValInt(0), Order: s.Child("order").ValInt(0)}
	if tx := s.Child("tx"); tx != nil {
		ser.NameRef = refOf(tx)
		if v := tx.Child("v"); v != nil {
			ser.Name = v.Text
		}
		if sr := tx.Path("strRef", "strCache", "pt", "v"); sr != nil && ser.Name == "" {
			ser.Name = sr.Text
		}
	}
	fam := t.Family()
	catTag, valTag := "cat", "val"
	if fam == "scatter" || fam == "bubble" {
		catTag, valTag = "xVal", "yVal"
	}
	if cat := s.Child(catTag); cat != nil {
		ser.Categories = refOf(cat)
		if lit := cat.Child("strLit"); lit != nil {
			for _, pt := range lit.Each("pt") {
				ser.CatLiterals = append(ser.CatLiterals, nodeText(pt.Child("v")))
			}
		}
	}
	if val := s.Child(valTag); val != nil {
		ser.Values = refOf(val)
		if lit := val.Child("numLit"); lit != nil {
			for _, pt := range lit.Each("pt") {
				f, _ := strconv.ParseFloat(strings.TrimSpace(nodeText(pt.Child("v"))), 64)
				ser.ValLiterals = append(ser.ValLiterals, f)
			}
		}
	}
	if bs := s.Child("bubbleSize"); bs != nil {
		ser.Bubbles = refOf(bs)
	}
	if sp := s.Child("spPr"); sp != nil {
		ser.Fill = parseSolidFill(sp)
		ser.Line = parseLineStyle(sp.Child("ln"))
		if (fam == "line" || fam == "scatter" || fam == "radar") && ser.Fill.IsZero() {
			ser.Fill = ser.Line.Color
		}
	}
	if m := s.Child("marker"); m != nil {
		mk := &chart.Marker{Symbol: m.Child("symbol").Val(), Size: m.Child("size").ValInt(0)}
		if sp := m.Child("spPr"); sp != nil {
			mk.Fill = parseSolidFill(sp)
			mk.Line = parseLineStyle(sp.Child("ln")).Color
		}
		ser.Marker = mk
	}
	if sm := s.Child("smooth"); sm != nil {
		ser.Smooth = sm.ValBool()
	}
	if ex := s.Child("explosion"); ex != nil {
		ser.Explosion = ex.ValInt(0)
	}
	if inv := s.Child("invertIfNegative"); inv != nil {
		b := inv.ValBool()
		ser.InvertIfNegative = &b
	}
	if sh := s.Child("shape"); sh != nil {
		ser.Shape = sh.Val()
	}
	if dl := s.Child("dLbls"); dl != nil {
		ser.DataLabels = &chart.DataLabels{ShowValue: dl.Child("showVal").ValBoolDefault(), ShowCategory: dl.Child("showCatName").ValBoolDefault(), ShowSeriesName: dl.Child("showSerName").ValBoolDefault(),
			ShowPercent: dl.Child("showPercent").ValBoolDefault(), ShowLegendKey: dl.Child("showLegendKey").ValBoolDefault(), ShowBubbleSize: dl.Child("showBubbleSize").ValBoolDefault(),
			ShowLeaderLines: dl.Child("showLeaderLines").ValBoolDefault(), Position: dl.Child("dLblPos").Val(), Separator: nodeText(dl.Child("separator"))}
		if nf := dl.Child("numFmt"); nf != nil {
			ser.DataLabels.NumFmt = nf.Attr("formatCode")
		}
		if tx := dl.Child("txPr"); tx != nil {
			ser.DataLabels.Font = parseTxPr(tx)
		}
	}
	for _, tl := range s.Each("trendline") {
		tr := chart.Trendline{Type: tl.Child("trendlineType").Val(), Order: tl.Child("order").ValInt(0), Period: tl.Child("period").ValInt(0), Name: nodeText(tl.Child("name")),
			Forward: tl.Child("forward").ValFloat(0), Backward: tl.Child("backward").ValFloat(0), DisplayEquation: tl.Child("dispEq").ValBoolDefault(), DisplayRSquared: tl.Child("dispRSqr").ValBoolDefault()}
		if ic := tl.Child("intercept"); ic != nil {
			v := ic.ValFloat(0)
			tr.Intercept = &v
		}
		if sp := tl.Child("spPr"); sp != nil {
			tr.Line = parseLineStyle(sp.Child("ln"))
		}
		ser.Trendlines = append(ser.Trendlines, tr)
	}
	for _, eb := range s.Each("errBars") {
		e := chart.ErrorBars{Direction: eb.Child("errDir").Val(), Type: eb.Child("errBarType").Val(), ValueType: eb.Child("errValType").Val(), Value: eb.Child("val").ValFloat(0), NoEndCap: eb.Child("noEndCap").ValBoolDefault()}
		e.Plus = refOf(eb.Child("plus"))
		e.Minus = refOf(eb.Child("minus"))
		if sp := eb.Child("spPr"); sp != nil {
			e.Line = parseLineStyle(sp.Child("ln"))
		}
		ser.ErrorBars = append(ser.ErrorBars, e)
	}
	for _, dp := range s.Each("dPt") {
		p := chart.DataPoint{Index: dp.Child("idx").ValInt(0), Explosion: dp.Child("explosion").ValInt(0)}
		if sp := dp.Child("spPr"); sp != nil {
			p.Fill = parseSolidFill(sp)
			p.Line = parseLineStyle(sp.Child("ln"))
		}
		ser.DataPoints = append(ser.DataPoints, p)
	}
	return ser
}

func parseSolidFill(sp *ooxml.Node) style.Color {
	if sp == nil {
		return style.Color{}
	}
	if sf := sp.Child("solidFill"); sf != nil {
		if c := sf.Child("srgbClr"); c != nil {
			col := style.RGB(c.Attr("val"))
			if a := c.Child("alpha"); a != nil {
				alpha := a.ValInt(100000) * 255 / 100000
				col = style.RGB(strings.ToUpper(strconv.FormatInt(int64(alpha), 16)) + col.ARGB[2:])
				if len(col.ARGB) != 8 {
					col = style.RGB(c.Attr("val"))
				}
			}
			return col
		}
		if c := sf.Child("schemeClr"); c != nil {
			return schemeColor(c.Attr("val"))
		}
	}
	return style.Color{}
}

func schemeColor(name string) style.Color {
	idx := map[string]int{"bg1": style.ThemeLight1, "tx1": style.ThemeDark1, "bg2": style.ThemeLight2, "tx2": style.ThemeDark2, "lt1": style.ThemeLight1, "dk1": style.ThemeDark1, "lt2": style.ThemeLight2, "dk2": style.ThemeDark2,
		"accent1": style.ThemeAccent1, "accent2": style.ThemeAccent2, "accent3": style.ThemeAccent3, "accent4": style.ThemeAccent4, "accent5": style.ThemeAccent5, "accent6": style.ThemeAccent6, "hlink": style.ThemeHyperlink, "folHlink": style.ThemeFollowedHyperlink}
	if i, ok := idx[name]; ok {
		return style.ThemeColor(i, 0)
	}
	return style.Color{}
}

func parseLineStyle(ln *ooxml.Node) chart.LineStyle {
	if ln == nil {
		return chart.LineStyle{}
	}
	l := chart.LineStyle{Color: parseSolidFill(ln), Dash: ln.Child("prstDash").Val(), NoLine: ln.Child("noFill") != nil}
	if w := ln.Attr("w"); w != "" {
		emu, _ := strconv.Atoi(w)
		l.Width = float64(emu) / 12700
	}
	return l
}

func parseTitle(t *ooxml.Node) *chart.Title {
	title := &chart.Title{Overlay: t.Child("overlay").ValBoolDefault()}
	if tx := t.Child("tx"); tx != nil {
		title.Formula = refOf(tx)
		if rich := tx.Child("rich"); rich != nil {
			var lines []string
			for _, p := range rich.Each("p") {
				var sb strings.Builder
				for _, r := range p.Each("r") {
					if rp := r.Child("rPr"); rp != nil && title.Font.IsZero() {
						title.Font = parseRPr(rp)
					}
					sb.WriteString(nodeText(r.Child("t")))
				}
				lines = append(lines, sb.String())
			}
			title.Text = strings.Join(lines, "\n")
			if bp := rich.Child("bodyPr"); bp != nil {
				if rot := bp.Attr("rot"); rot != "" {
					r, _ := strconv.Atoi(rot)
					title.Rotation = r / 60000
				}
			}
		}
	}
	return title
}

func parseTxPr(tx *ooxml.Node) style.Font {
	if d := tx.Path("p", "pPr", "defRPr"); d != nil {
		return parseRPr(d)
	}
	return style.Font{}
}

func parseRPr(rp *ooxml.Node) style.Font {
	f := style.Font{Bold: ooxml.ParseBool(rp.Attr("b")), Italic: ooxml.ParseBool(rp.Attr("i"))}
	if sz := rp.Attr("sz"); sz != "" {
		n, _ := strconv.Atoi(sz)
		f.Size = float64(n) / 100
	}
	if rp.Attr("u") != "" && rp.Attr("u") != "none" {
		f.Underline = style.UnderlineSingle
	}
	if rp.Attr("strike") != "" && rp.Attr("strike") != "noStrike" {
		f.Strike = true
	}
	f.Color = parseSolidFill(rp)
	if l := rp.Child("latin"); l != nil {
		f.Name = l.Attr("typeface")
	}
	return f
}

func parseAxis(n *ooxml.Node) chart.Axis {
	a := chart.Axis{Deleted: n.Child("delete").ValBoolDefault(), Position: n.Child("axPos").Val(), MajorGridlines: n.Child("majorGridlines") != nil, MinorGridlines: n.Child("minorGridlines") != nil,
		MajorTickMark: n.Child("majorTickMark").Val(), MinorTickMark: n.Child("minorTickMark").Val(), TickLabelPos: n.Child("tickLblPos").Val(), Crosses: n.Child("crosses").Val(),
		CrossBetween: n.Child("crossBetween").Val(), LabelAlign: n.Child("lblAlgn").Val(), LabelOffset: n.Child("lblOffset").ValInt(0), TickLabelSkip: n.Child("tickLblSkip").ValInt(0),
		TickMarkSkip: n.Child("tickMarkSkip").ValInt(0), NoMultiLvlLbl: n.Child("noMultiLvlLbl").ValBoolDefault(), DateAxis: n.Name == "dateAx", BaseTimeUnit: n.Child("baseTimeUnit").Val(),
		MajorTimeUnit: n.Child("majorTimeUnit").Val(), MinorTimeUnit: n.Child("minorTimeUnit").Val()}
	a.Hidden = a.Deleted
	if t := n.Child("title"); t != nil {
		a.Title = parseTitle(t)
	}
	if sc := n.Child("scaling"); sc != nil {
		a.Orientation = sc.Child("orientation").Val()
		a.Reverse = a.Orientation == "maxMin"
		if mx := sc.Child("max"); mx != nil {
			v := mx.ValFloat(0)
			a.Max = &v
		}
		if mn := sc.Child("min"); mn != nil {
			v := mn.ValFloat(0)
			a.Min = &v
		}
		a.LogBase = sc.Child("logBase").ValFloat(0)
	}
	if nf := n.Child("numFmt"); nf != nil {
		a.NumFmt = nf.Attr("formatCode")
		b := ooxml.ParseBool(nf.Attr("sourceLinked"))
		a.NumFmtLinked = &b
		if a.NumFmt == "General" && b {
			a.NumFmt = ""
		}
	}
	if mu := n.Child("majorUnit"); mu != nil {
		v := mu.ValFloat(0)
		a.MajorUnit = &v
	}
	if mu := n.Child("minorUnit"); mu != nil {
		v := mu.ValFloat(0)
		a.MinorUnit = &v
	}
	if ca := n.Child("crossesAt"); ca != nil {
		v := ca.ValFloat(0)
		a.CrossesAt = &v
	}
	if du := n.Path("dispUnits", "builtInUnit"); du != nil {
		a.DisplayUnits = du.Val()
	}
	if sp := n.Child("spPr"); sp != nil {
		a.Line = parseLineStyle(sp.Child("ln"))
	}
	if tx := n.Child("txPr"); tx != nil {
		a.LabelFont = parseTxPr(tx)
		if bp := tx.Child("bodyPr"); bp != nil {
			if rot := bp.Attr("rot"); rot != "" {
				r, _ := strconv.Atoi(rot)
				a.TextRotation = r / 60000
			}
		}
	}
	return a
}
