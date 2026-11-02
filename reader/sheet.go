// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
)

// readSheet parses a worksheet part. sheetData is streamed with the token
// decoder; the rest of the sheet is small and parsed as a DOM.
func (d *docReader) readSheet(sh *model.Sheet, part string) error {
	d.use(part)
	rels, _ := d.pr.Rels(part)
	d.use(ooxml.RelsName(part))
	rc, err := d.pr.Open(part)
	if err != nil {
		return err
	}
	defer rc.Close()
	dec := xml.NewDecoder(io.LimitReader(rc, d.opts.MaxPartSize))
	dec.Strict = false
	sp := &sheetParser{d: d, sh: sh, part: part, rels: rels, dec: dec}
	if err := sp.parse(); err != nil {
		return err
	}
	sp.expandSharedFormulas()
	// Sub-parts.
	for _, r := range rels {
		target := ooxml.Resolve(part, r.Target)
		switch r.Type {
		case ooxml.RelComments:
			d.readComments(sh, target)
		case ooxml.RelThreadedComments:
			d.readThreadedComments(sh, target)
		case ooxml.RelVMLDrawing:
			d.readVML(sh, target)
		case ooxml.RelTable:
			d.readTable(sh, target)
		case ooxml.RelDrawing:
			if !d.opts.SkipDrawings {
				d.readDrawing(sh, target)
			} else {
				d.use(target)
			}
		case ooxml.RelPivotTable:
			if raw := d.rawPart(target); raw != nil {
				sh.PivotTables = append(sh.PivotTables, &model.PivotTable{Name: target, Table: raw})
			}
		case ooxml.RelHyperlink:
		default:
			if r.TargetMode != "External" {
				d.rawPart(target)
			}
		}
	}
	return nil
}

type sheetParser struct {
	d    *docReader
	sh   *model.Sheet
	part string
	rels []ooxml.Relationship
	dec  *xml.Decoder
	// shared formula masters by si
	shared map[int]*model.Cell
	// pending x14 conditional formatting ids -> rule
	cfByID map[string]*model.CondFmtRule
	// onRow, when set, receives each finished row, which is then dropped
	// from the sheet (streaming mode). Returning false stops parsing.
	onRow func(*model.Row) bool
}

// errStop ends a streamed parse early.
var errStop = errors.New("stop")

func (p *sheetParser) parse() error {
	dec := p.dec
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if p.d.opts.Strict {
				return fmt.Errorf("reader: %s: %w", p.part, err)
			}
			return nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				continue
			}
			if t.Name.Local == "sheetData" && depth == 2 {
				if err := p.parseSheetData(); err != nil {
					return err
				}
				depth--
				continue
			}
			// Everything else: parse as a subtree DOM.
			n, err := p.subtree(t)
			if err != nil {
				if p.d.opts.Strict {
					return err
				}
				return nil
			}
			depth--
			p.element(n)
		case xml.EndElement:
			depth--
		}
	}
}

// subtree reads the rest of the element started by se into a Node.
func (p *sheetParser) subtree(se xml.StartElement) (*ooxml.Node, error) {
	root := &ooxml.Node{Name: se.Name.Local, Space: se.Name.Space, Attrs: se.Attr}
	stack := []*ooxml.Node{root}
	for len(stack) > 0 {
		tok, err := p.dec.Token()
		if err != nil {
			return root, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &ooxml.Node{Name: t.Name.Local, Space: t.Name.Space, Attrs: t.Attr}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].Text += string(t)
		}
	}
	return root, nil
}

// parseSheetData streams rows and cells.
func (p *sheetParser) parseSheetData() error {
	dec := p.dec
	sh := p.sh
	var row *model.Row
	var cell *model.Cell
	var cellType, formulaType string
	var inV, inF, inIs, inT, inRPr bool
	var text strings.Builder
	var isNode *ooxml.Node
	var isStack []*ooxml.Node
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if p.d.opts.Strict {
				return err
			}
			return nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = p.rowStart(t)
			case "c":
				cell, cellType = p.cellStart(t, row)
				formulaType = ""
			case "f":
				inF = true
				text.Reset()
				f := &model.Formula{}
				formulaType = ooxml.Attr(t, "t")
				switch formulaType {
				case "array":
					f.Kind = model.FormulaArray
					f.Master = true
				case "shared":
					f.Kind = model.FormulaShared
					f.SharedIndex = ooxml.ParseInt(ooxml.Attr(t, "si"), 0)
				case "dataTable":
					f.Kind = model.FormulaDataTable
					f.DataTable2D = ooxml.ParseBool(ooxml.Attr(t, "dt2D"))
					f.DataTableRow = ooxml.ParseBool(ooxml.Attr(t, "dtr"))
					f.DataTableRef1 = ooxml.Attr(t, "r1")
					f.DataTableRef2 = ooxml.Attr(t, "r2")
				}
				if rs := ooxml.Attr(t, "ref"); rs != "" {
					if r, err := ref.ParseRange(rs); err == nil {
						f.Ref = r
						f.Master = true
					}
				}
				f.CalcAlways = ooxml.ParseBool(ooxml.Attr(t, "ca"))
				if cell != nil {
					cell.Formula = f
				}
			case "v":
				inV = true
				text.Reset()
			case "is":
				inIs = true
				isNode = &ooxml.Node{Name: "is"}
				isStack = []*ooxml.Node{isNode}
			default:
				if inIs {
					n := &ooxml.Node{Name: t.Name.Local, Attrs: t.Attr}
					parent := isStack[len(isStack)-1]
					parent.Children = append(parent.Children, n)
					isStack = append(isStack, n)
				}
			}
			_ = inT
			_ = inRPr
		case xml.CharData:
			switch {
			case inV || inF:
				text.Write(t)
			case inIs && len(isStack) > 0:
				isStack[len(isStack)-1].Text += string(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "sheetData":
				return nil
			case "row":
				if row != nil && p.onRow != nil {
					p.expandSharedFormulas()
					keep := p.onRow(row)
					delete(sh.Rows, row.Index)
					if !keep {
						return errStop
					}
				} else if row != nil && row.IsEmpty() && len(row.Cells) == 0 {
					delete(sh.Rows, row.Index)
				}
				row = nil
			case "c":
				cell = nil
			case "f":
				inF = false
				if cell != nil && cell.Formula != nil {
					cell.Formula.Text = formula.StripPrefixes(text.String())
					if cell.Formula.Kind == model.FormulaShared {
						if p.shared == nil {
							p.shared = map[int]*model.Cell{}
						}
						if cell.Formula.Master || cell.Formula.Text != "" {
							cell.Formula.Master = true
							p.shared[cell.Formula.SharedIndex] = cell
						}
					}
					if cell.Formula.Kind == model.FormulaArray && cell.Formula.Dynamic {
						// marked by cm attribute
					}
				}
			case "v":
				inV = false
				if cell != nil {
					p.setValue(cell, cellType, text.String())
				}
			case "is":
				inIs = false
				if cell != nil && isNode != nil {
					v := parseRichText(isNode)
					v.Inline = true
					cell.Value = v
				}
				isNode = nil
			default:
				if inIs && len(isStack) > 1 {
					isStack = isStack[:len(isStack)-1]
				}
			}
		}
	}
}

func (p *sheetParser) rowStart(t xml.StartElement) *model.Row {
	idx := ooxml.ParseInt(ooxml.Attr(t, "r"), 0)
	if idx <= 0 {
		idx = len(p.sh.Rows) + 1
	}
	row := p.sh.Row(idx)
	for _, a := range t.Attr {
		switch a.Name.Local {
		case "ht":
			row.Height, _ = ooxml.ParseFloat(a.Value)
		case "customHeight":
			row.CustomHeight = ooxml.ParseBool(a.Value)
		case "hidden":
			row.Hidden = ooxml.ParseBool(a.Value)
		case "s":
			row.StyleID = ooxml.ParseInt(a.Value, 0)
		case "customFormat":
			row.CustomFormat = ooxml.ParseBool(a.Value)
		case "outlineLevel":
			row.OutlineLevel = ooxml.ParseInt(a.Value, 0)
		case "collapsed":
			row.Collapsed = ooxml.ParseBool(a.Value)
		case "thickTop":
			row.ThickTop = ooxml.ParseBool(a.Value)
		case "thickBot":
			row.ThickBottom = ooxml.ParseBool(a.Value)
		case "ph":
			row.ShowPhonetic = ooxml.ParseBool(a.Value)
		}
	}
	if !row.CustomFormat {
		row.StyleID = 0
	}
	return row
}

func (p *sheetParser) cellStart(t xml.StartElement, row *model.Row) (*model.Cell, string) {
	if row == nil {
		row = p.sh.Row(len(p.sh.Rows) + 1)
	}
	var r, c int
	if rs := ooxml.Attr(t, "r"); rs != "" {
		if cr, err := ref.ParseCell(rs); err == nil {
			r, c = cr.Row, cr.Col
		}
	}
	if c == 0 {
		c = row.MaxCol() + 1
	}
	if r == 0 {
		r = row.Index
	}
	if p.d.opts.MaxCells > 0 {
		p.d.cells++
		if p.d.cells > p.d.opts.MaxCells {
			return nil, ""
		}
	}
	cell := row.Cell(c)
	cell.Row = r
	typ := ooxml.Attr(t, "t")
	cell.StyleID = ooxml.ParseInt(ooxml.Attr(t, "s"), 0)
	cell.ShowPhonetic = ooxml.ParseBool(ooxml.Attr(t, "ph"))
	cell.ValueMetadata = ooxml.ParseInt(ooxml.Attr(t, "vm"), 0)
	if cm := ooxml.Attr(t, "cm"); cm != "" {
		cell.Formula = &model.Formula{Dynamic: true}
	}
	return cell, typ
}

func (p *sheetParser) setValue(cell *model.Cell, typ, text string) {
	switch typ {
	case "s":
		i, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && i >= 0 && i < len(p.d.sst) {
			cell.Value = p.d.sst[i]
		} else {
			cell.Value = model.String("")
		}
	case "str":
		cell.Value = model.String(ooxml.UnescapeUnicode(text))
	case "b":
		cell.Value = model.Boolean(ooxml.ParseBool(text))
	case "e":
		cell.Value = model.Error(strings.TrimSpace(text))
	case "d":
		// ISO 8601 date (rare); keep as string.
		cell.Value = model.String(strings.TrimSpace(text))
	default:
		if strings.TrimSpace(text) == "" {
			cell.Value = model.Value{}
			break
		}
		f, err := ooxml.ParseFloat(text)
		if err != nil {
			cell.Value = model.String(strings.TrimSpace(text))
		} else {
			cell.Value = model.Number(f)
		}
	}
	if cell.Formula != nil && cell.Formula.Dynamic && cell.Formula.Text == "" && cell.Formula.Kind == model.FormulaNormal {
		// cm="1" without a formula element: a spilled value cell.
		cell.Formula = nil
	}
}

// expandSharedFormulas gives every cell of a shared formula group its own
// translated formula text, so users see the real formula per cell.
func (p *sheetParser) expandSharedFormulas() {
	if len(p.shared) == 0 {
		return
	}
	for _, row := range p.sh.Rows {
		for _, c := range row.Cells {
			f := c.Formula
			if f == nil || f.Kind != model.FormulaShared || f.Master {
				continue
			}
			master, ok := p.shared[f.SharedIndex]
			if !ok || master.Formula == nil {
				continue
			}
			f.Text = formula.Shift(master.Formula.Text, c.Row-master.Row, c.Col-master.Col)
			f.Ref = master.Formula.Ref
		}
	}
}

// element handles a non-sheetData child of <worksheet>.
func (p *sheetParser) element(n *ooxml.Node) {
	sh := p.sh
	switch n.Name {
	case "sheetPr":
		sh.CodeName = n.Attr("codeName")
		sh.FilterMode = ooxml.ParseBool(n.Attr("filterMode"))
		sh.Published = ooxml.ParseBool(n.Attr("published"))
		sh.TransitionEval = ooxml.ParseBool(n.Attr("transitionEvaluation"))
		if n.Has("enableFormatConditionsCalculation") {
			b := ooxml.ParseBool(n.Attr("enableFormatConditionsCalculation"))
			sh.EnableCondFmtCalc = &b
		}
		sh.TabColor = parseColor(n.Child("tabColor"))
		if o := n.Child("outlinePr"); o != nil {
			sh.ApplyStyles = ooxml.ParseBool(o.Attr("applyStyles"))
			if o.Has("summaryBelow") {
				b := ooxml.ParseBool(o.Attr("summaryBelow"))
				sh.SummaryBelow = &b
			}
			if o.Has("summaryRight") {
				b := ooxml.ParseBool(o.Attr("summaryRight"))
				sh.SummaryRight = &b
			}
		}
		if ps := n.Child("pageSetUpPr"); ps != nil {
			sh.FitToPage = ooxml.ParseBool(ps.Attr("fitToPage"))
		}
	case "dimension":
		if r, err := ref.ParseRange(n.Attr("ref")); err == nil {
			sh.Dimension = r
		}
	case "sheetViews":
		if v := n.Child("sheetView"); v != nil {
			p.parseView(v)
		}
	case "sheetFormatPr":
		sh.DefaultRowHeight, _ = ooxml.ParseFloat(n.Attr("defaultRowHeight"))
		sh.DefaultColWidth, _ = ooxml.ParseFloat(n.Attr("defaultColWidth"))
		sh.CustomRowHeight = ooxml.ParseBool(n.Attr("customHeight"))
		sh.ZeroHeight = ooxml.ParseBool(n.Attr("zeroHeight"))
		sh.ThickTop = ooxml.ParseBool(n.Attr("thickTop"))
		sh.ThickBottom = ooxml.ParseBool(n.Attr("thickBottom"))
		sh.OutlineLevelRow = ooxml.ParseInt(n.Attr("outlineLevelRow"), 0)
		sh.OutlineLevelCol = ooxml.ParseInt(n.Attr("outlineLevelCol"), 0)
	case "cols":
		for _, c := range n.Each("col") {
			col := model.Col{Min: ooxml.ParseInt(c.Attr("min"), 1), Max: ooxml.ParseInt(c.Attr("max"), 1),
				CustomWidth: ooxml.ParseBool(c.Attr("customWidth")), BestFit: ooxml.ParseBool(c.Attr("bestFit")), Hidden: ooxml.ParseBool(c.Attr("hidden")),
				StyleID: ooxml.ParseInt(c.Attr("style"), 0), OutlineLevel: ooxml.ParseInt(c.Attr("outlineLevel"), 0), Collapsed: ooxml.ParseBool(c.Attr("collapsed")), Phonetic: ooxml.ParseBool(c.Attr("phonetic"))}
			col.Width, _ = ooxml.ParseFloat(c.Attr("width"))
			if col.Max < col.Min {
				col.Max = col.Min
			}
			sh.Cols = append(sh.Cols, col)
		}
	case "sheetProtection":
		p.parseProtection(n)
	case "autoFilter":
		sh.AutoFilter = parseAutoFilter(n)
	case "sortState":
		if sh.AutoFilter != nil {
			sh.AutoFilter.SortState = parseSortState(n)
		}
	case "mergeCells":
		for _, m := range n.Each("mergeCell") {
			if r, err := ref.ParseRange(m.Attr("ref")); err == nil {
				sh.Merges = append(sh.Merges, r)
			}
		}
	case "conditionalFormatting":
		p.parseCondFmt(n)
	case "dataValidations":
		for _, dv := range n.Each("dataValidation") {
			sh.DataValidations = append(sh.DataValidations, parseDataValidation(dv))
		}
	case "hyperlinks":
		for _, h := range n.Each("hyperlink") {
			r, err := ref.ParseRange(h.Attr("ref"))
			if err != nil {
				continue
			}
			hl := model.Hyperlink{Range: r, Location: h.Attr("location"), Tooltip: h.Attr("tooltip"), Display: h.Attr("display")}
			for _, a := range h.Attrs {
				if a.Name.Local == "id" {
					if rel, ok := ooxml.RelByID(p.rels, a.Value); ok {
						hl.Target = rel.Target
					}
				}
			}
			sh.Hyperlinks = append(sh.Hyperlinks, hl)
		}
	case "printOptions":
		sh.PrintOptions = model.PrintOptions{GridLines: ooxml.ParseBool(n.Attr("gridLines")), GridLinesSet: ooxml.ParseBool(n.Attr("gridLinesSet")), Headings: ooxml.ParseBool(n.Attr("headings")),
			HorizontalCentered: ooxml.ParseBool(n.Attr("horizontalCentered")), VerticalCentered: ooxml.ParseBool(n.Attr("verticalCentered"))}
	case "pageMargins":
		m := &sh.PageMargins
		m.Left, _ = ooxml.ParseFloat(n.Attr("left"))
		m.Right, _ = ooxml.ParseFloat(n.Attr("right"))
		m.Top, _ = ooxml.ParseFloat(n.Attr("top"))
		m.Bottom, _ = ooxml.ParseFloat(n.Attr("bottom"))
		m.Header, _ = ooxml.ParseFloat(n.Attr("header"))
		m.Footer, _ = ooxml.ParseFloat(n.Attr("footer"))
	case "pageSetup":
		sh.PageSetup = model.PageSetup{Orientation: model.Orientation(n.Attr("orientation")), PaperSize: ooxml.ParseInt(n.Attr("paperSize"), 0), Scale: ooxml.ParseInt(n.Attr("scale"), 0),
			FitToWidth: ooxml.ParseInt(n.Attr("fitToWidth"), 0), FitToHeight: ooxml.ParseInt(n.Attr("fitToHeight"), 0), FirstPageNumber: ooxml.ParseInt(n.Attr("firstPageNumber"), 0),
			UseFirstPage: ooxml.ParseBool(n.Attr("useFirstPageNumber")), BlackAndWhite: ooxml.ParseBool(n.Attr("blackAndWhite")), Draft: ooxml.ParseBool(n.Attr("draft")),
			CellComments: n.Attr("cellComments"), Errors: n.Attr("errors"), PageOrder: n.Attr("pageOrder"), HorizontalDPI: ooxml.ParseInt(n.Attr("horizontalDpi"), 0),
			VerticalDPI: ooxml.ParseInt(n.Attr("verticalDpi"), 0), Copies: ooxml.ParseInt(n.Attr("copies"), 0), UsePrinterDefaults: ooxml.ParseBool(n.Attr("usePrinterDefaults"))}
	case "headerFooter":
		hf := &sh.HeaderFooter
		hf.DifferentOddEven = ooxml.ParseBool(n.Attr("differentOddEven"))
		hf.DifferentFirst = ooxml.ParseBool(n.Attr("differentFirst"))
		hf.NoScaleWithDoc = n.Has("scaleWithDoc") && !ooxml.ParseBool(n.Attr("scaleWithDoc"))
		hf.NoAlignWithMargins = n.Has("alignWithMargins") && !ooxml.ParseBool(n.Attr("alignWithMargins"))
		for _, c := range n.Children {
			switch c.Name {
			case "oddHeader":
				hf.OddHeader = c.Text
			case "oddFooter":
				hf.OddFooter = c.Text
			case "evenHeader":
				hf.EvenHeader = c.Text
			case "evenFooter":
				hf.EvenFooter = c.Text
			case "firstHeader":
				hf.FirstHeader = c.Text
			case "firstFooter":
				hf.FirstFooter = c.Text
			}
		}
	case "rowBreaks", "colBreaks":
		var out []model.PageBreak
		for _, b := range n.Each("brk") {
			out = append(out, model.PageBreak{Index: ooxml.ParseInt(b.Attr("id"), 0), Max: ooxml.ParseInt(b.Attr("max"), 0), Manual: ooxml.ParseBool(b.Attr("man"))})
		}
		if n.Name == "rowBreaks" {
			sh.RowBreaks = out
		} else {
			sh.ColBreaks = out
		}
	case "ignoredErrors":
		for _, ie := range n.Each("ignoredError") {
			rs, _ := ref.ParseRanges(ie.Attr("sqref"))
			sh.IgnoredErrors = append(sh.IgnoredErrors, model.IgnoredError{Ranges: rs, EvalError: ooxml.ParseBool(ie.Attr("evalError")), TwoDigitTextYear: ooxml.ParseBool(ie.Attr("twoDigitTextYear")),
				NumberStoredAsTxt: ooxml.ParseBool(ie.Attr("numberStoredAsText")), Formula: ooxml.ParseBool(ie.Attr("formula")), FormulaRange: ooxml.ParseBool(ie.Attr("formulaRange")),
				UnlockedFormula: ooxml.ParseBool(ie.Attr("unlockedFormula")), EmptyCellRef: ooxml.ParseBool(ie.Attr("emptyCellReference")), ListDataValidation: ooxml.ParseBool(ie.Attr("listDataValidation")), CalculatedColumn: ooxml.ParseBool(ie.Attr("calculatedColumn"))})
		}
	case "drawing", "legacyDrawing", "tableParts", "legacyDrawingHF", "picture", "sheetCalcPr", "phoneticPr", "customSheetViews", "protectedRanges", "scenarios", "dataConsolidate", "cellWatches", "smartTags", "customProperties", "oleObjects", "controls", "webPublishItems":
		// Handled through relationships, or not modelled.
	case "extLst":
		for _, e := range n.Children {
			if !p.parseExt(e) {
				sh.Extensions = append(sh.Extensions, model.RawXML{URI: e.Attr("uri"), Data: rawXML(p.part, p.d, "extLst", e)})
			}
		}
	}
}

func (p *sheetParser) parseView(v *ooxml.Node) {
	sv := &p.sh.View
	sv.HideGridLines = v.Has("showGridLines") && !ooxml.ParseBool(v.Attr("showGridLines"))
	sv.HideHeadings = v.Has("showRowColHeaders") && !ooxml.ParseBool(v.Attr("showRowColHeaders"))
	sv.HideZeros = v.Has("showZeros") && !ooxml.ParseBool(v.Attr("showZeros"))
	sv.ShowFormulas = ooxml.ParseBool(v.Attr("showFormulas"))
	sv.HideRuler = v.Has("showRuler") && !ooxml.ParseBool(v.Attr("showRuler"))
	sv.HideOutline = v.Has("showOutlineSymbols") && !ooxml.ParseBool(v.Attr("showOutlineSymbols"))
	sv.HideWhiteSpace = v.Has("showWhiteSpace") && !ooxml.ParseBool(v.Attr("showWhiteSpace"))
	sv.RightToLeft = ooxml.ParseBool(v.Attr("rightToLeft"))
	sv.TabSelected = ooxml.ParseBool(v.Attr("tabSelected"))
	sv.WindowProtection = ooxml.ParseBool(v.Attr("windowProtection"))
	sv.Type = v.Attr("view")
	sv.TopLeftCell = v.Attr("topLeftCell")
	sv.ZoomScale = ooxml.ParseInt(v.Attr("zoomScale"), 0)
	sv.ZoomNormal = ooxml.ParseInt(v.Attr("zoomScaleNormal"), 0)
	sv.ZoomPageLayout = ooxml.ParseInt(v.Attr("zoomScalePageLayoutView"), 0)
	sv.ZoomPageBreak = ooxml.ParseInt(v.Attr("zoomScaleSheetLayoutView"), 0)
	sv.GridLineColor = ooxml.ParseInt(v.Attr("colorId"), 0)
	if pn := v.Child("pane"); pn != nil {
		pane := &model.Pane{TopLeftCell: pn.Attr("topLeftCell"), ActivePane: pn.Attr("activePane")}
		xs, _ := ooxml.ParseFloat(pn.Attr("xSplit"))
		ys, _ := ooxml.ParseFloat(pn.Attr("ySplit"))
		if pn.Attr("state") == "frozen" || pn.Attr("state") == "frozenSplit" {
			pane.Cols, pane.Rows = int(xs), int(ys)
			sv.Freeze = pane
		} else {
			pane.XSplit, pane.YSplit = xs, ys
			sv.Split = pane
		}
	}
	for _, s := range v.Each("selection") {
		rs, _ := ref.ParseRanges(s.Attr("sqref"))
		sv.Selections = append(sv.Selections, model.Selection{Pane: s.Attr("pane"), ActiveCell: s.Attr("activeCell"), Ranges: rs})
	}
}

func (p *sheetParser) parseProtection(n *ooxml.Node) {
	locked := func(name string) bool { return ooxml.ParseBool(n.Attr(name)) }
	pr := &model.SheetProtection{
		Algorithm: n.Attr("algorithmName"), HashValue: n.Attr("hashValue"), SaltValue: n.Attr("saltValue"), SpinCount: ooxml.ParseInt(n.Attr("spinCount"), 0), LegacyHash: n.Attr("password"),
		Sheet: locked("sheet"), Objects: !locked("objects"), Scenarios: !locked("scenarios"),
		FormatCells: !locked("formatCells"), FormatColumns: !locked("formatColumns"), FormatRows: !locked("formatRows"),
		InsertColumns: !locked("insertColumns"), InsertRows: !locked("insertRows"), InsertHyperlinks: !locked("insertHyperlinks"),
		DeleteColumns: !locked("deleteColumns"), DeleteRows: !locked("deleteRows"), Sort: !locked("sort"), AutoFilter: !locked("autoFilter"), PivotTables: !locked("pivotTables"),
		NoSelectLocked: locked("selectLockedCells"), NoSelectUnlocked: locked("selectUnlockedCells"),
	}
	p.sh.Protection = pr
}

func parseAutoFilter(n *ooxml.Node) *model.AutoFilter {
	af := &model.AutoFilter{}
	af.Range, _ = ref.ParseRange(n.Attr("ref"))
	for _, fc := range n.Each("filterColumn") {
		c := model.FilterColumn{ColID: ooxml.ParseInt(fc.Attr("colId"), 0), HiddenButton: ooxml.ParseBool(fc.Attr("hiddenButton"))}
		if fc.Has("showButton") {
			b := ooxml.ParseBool(fc.Attr("showButton"))
			c.ShowButton = &b
		}
		if f := fc.Child("filters"); f != nil {
			c.Blank = ooxml.ParseBool(f.Attr("blank"))
			for _, v := range f.Each("filter") {
				c.Values = append(c.Values, v.Attr("val"))
			}
			for _, dg := range f.Each("dateGroupItem") {
				c.DateGroups = append(c.DateGroups, model.DateGroupItem{Grouping: dg.Attr("dateTimeGrouping"), Year: ooxml.ParseInt(dg.Attr("year"), 0), Month: ooxml.ParseInt(dg.Attr("month"), 0),
					Day: ooxml.ParseInt(dg.Attr("day"), 0), Hour: ooxml.ParseInt(dg.Attr("hour"), 0), Minute: ooxml.ParseInt(dg.Attr("minute"), 0), Second: ooxml.ParseInt(dg.Attr("second"), 0)})
			}
		}
		if cf := fc.Child("customFilters"); cf != nil {
			c.And = ooxml.ParseBool(cf.Attr("and"))
			for _, v := range cf.Each("customFilter") {
				c.Custom = append(c.Custom, model.CustomFilter{Operator: v.Attr("operator"), Value: v.Attr("val")})
			}
		}
		if t := fc.Child("top10"); t != nil {
			top := !t.Has("top") || ooxml.ParseBool(t.Attr("top"))
			val, _ := ooxml.ParseFloat(t.Attr("val"))
			fv, _ := ooxml.ParseFloat(t.Attr("filterVal"))
			c.Top10 = &model.Top10Filter{Top: top, Percent: ooxml.ParseBool(t.Attr("percent")), Val: val, FilterVal: fv}
		}
		if dy := fc.Child("dynamicFilter"); dy != nil {
			val, _ := ooxml.ParseFloat(dy.Attr("val"))
			mv, _ := ooxml.ParseFloat(dy.Attr("maxVal"))
			c.Dynamic = &model.DynamicFilter{Type: dy.Attr("type"), Val: val, MaxVal: mv}
		}
		if cl := fc.Child("colorFilter"); cl != nil {
			c.ColorFilter = &model.ColorFilter{DxfID: ooxml.ParseInt(cl.Attr("dxfId"), 0), CellColor: !cl.Has("cellColor") || ooxml.ParseBool(cl.Attr("cellColor"))}
		}
		if ic := fc.Child("iconFilter"); ic != nil {
			c.IconFilter = &model.IconFilter{IconSet: ic.Attr("iconSet"), IconID: ooxml.ParseInt(ic.Attr("iconId"), 0)}
		}
		af.Columns = append(af.Columns, c)
	}
	if ss := n.Child("sortState"); ss != nil {
		af.SortState = parseSortState(ss)
	}
	return af
}

func parseSortState(n *ooxml.Node) *model.SortState {
	ss := &model.SortState{CaseSensitive: ooxml.ParseBool(n.Attr("caseSensitive")), ColumnSort: ooxml.ParseBool(n.Attr("columnSort"))}
	ss.Range, _ = ref.ParseRange(n.Attr("ref"))
	for _, c := range n.Each("sortCondition") {
		sc := model.SortCondition{Descending: ooxml.ParseBool(c.Attr("descending")), SortBy: c.Attr("sortBy"), DxfID: ooxml.ParseInt(c.Attr("dxfId"), 0),
			IconSet: c.Attr("iconSet"), IconID: ooxml.ParseInt(c.Attr("iconId"), 0), CustomList: c.Attr("customList")}
		sc.Range, _ = ref.ParseRange(c.Attr("ref"))
		ss.Conditions = append(ss.Conditions, sc)
	}
	return ss
}

func parseDataValidation(n *ooxml.Node) model.DataValidation {
	dv := model.DataValidation{Type: model.DataValidationType(n.Attr("type")), Operator: model.DataValidationOperator(n.Attr("operator")),
		AllowBlank: ooxml.ParseBool(n.Attr("allowBlank")), ShowDropDown: ooxml.ParseBool(n.Attr("showDropDown")), ShowInputMessage: ooxml.ParseBool(n.Attr("showInputMessage")),
		ShowErrorMessage: ooxml.ParseBool(n.Attr("showErrorMessage")), ErrorStyle: n.Attr("errorStyle"), ErrorTitle: n.Attr("errorTitle"), Error: n.Attr("error"),
		PromptTitle: n.Attr("promptTitle"), Prompt: n.Attr("prompt"), IMEMode: n.Attr("imeMode")}
	dv.Ranges, _ = ref.ParseRanges(n.Attr("sqref"))
	if dv.Operator == "between" {
		dv.Operator = model.OpBetween
	}
	if f := n.Child("formula1"); f != nil {
		dv.Formula1 = f.Text
	}
	if f := n.Child("formula2"); f != nil {
		dv.Formula2 = f.Text
	}
	return dv
}

func parseCfvo(n *ooxml.Node) model.CfValue {
	v := model.CfValue{Type: model.CfValueType(n.Attr("type")), Value: n.Attr("val")}
	if n.Has("gte") {
		b := ooxml.ParseBool(n.Attr("gte"))
		v.GTE = &b
	}
	if f := n.Child("f"); f != nil && v.Value == "" {
		v.Value = f.Text
	}
	return v
}

func (p *sheetParser) parseCondFmt(n *ooxml.Node) {
	cf := model.ConditionalFormat{Pivot: ooxml.ParseBool(n.Attr("pivot"))}
	cf.Ranges, _ = ref.ParseRanges(n.Attr("sqref"))
	for _, r := range n.Each("cfRule") {
		rule := model.CondFmtRule{Type: model.CondFmtType(r.Attr("type")), Priority: ooxml.ParseInt(r.Attr("priority"), 0), StopIfTrue: ooxml.ParseBool(r.Attr("stopIfTrue")),
			Operator: model.CondFmtOperator(r.Attr("operator")), Text: r.Attr("text"), TimePeriod: r.Attr("timePeriod"), Rank: ooxml.ParseInt(r.Attr("rank"), 0),
			Bottom: ooxml.ParseBool(r.Attr("bottom")), Percent: ooxml.ParseBool(r.Attr("percent")), EqualAverage: ooxml.ParseBool(r.Attr("equalAverage")), StdDev: ooxml.ParseInt(r.Attr("stdDev"), 0)}
		if r.Has("dxfId") {
			rule.DxfID = ooxml.ParseInt(r.Attr("dxfId"), 0)
			rule.Style = p.d.wb.Styles.Dxf(rule.DxfID)
			rule.HasStyle = true
		}
		if r.Has("aboveAverage") {
			b := ooxml.ParseBool(r.Attr("aboveAverage"))
			rule.AboveAverage = &b
		}
		for _, f := range r.Each("formula") {
			rule.Formulas = append(rule.Formulas, f.Text)
		}
		if cs := r.Child("colorScale"); cs != nil {
			sc := &model.ColorScale{}
			for _, v := range cs.Each("cfvo") {
				sc.Values = append(sc.Values, parseCfvo(v))
			}
			for _, c := range cs.Each("color") {
				sc.Colors = append(sc.Colors, parseColor(c))
			}
			rule.ColorScale = sc
		}
		if db := r.Child("dataBar"); db != nil {
			d := &model.DataBar{MinLength: ooxml.ParseInt(db.Attr("minLength"), 0), MaxLength: ooxml.ParseInt(db.Attr("maxLength"), 0)}
			if db.Has("showValue") {
				b := ooxml.ParseBool(db.Attr("showValue"))
				d.ShowValue = &b
			}
			vals := db.Each("cfvo")
			if len(vals) > 0 {
				d.Min = parseCfvo(vals[0])
			}
			if len(vals) > 1 {
				d.Max = parseCfvo(vals[1])
			}
			d.Color = parseColor(db.Child("color"))
			rule.DataBar = d
		}
		if is := r.Child("iconSet"); is != nil {
			set := &model.IconSet{Set: is.Attr("iconSet"), Reverse: ooxml.ParseBool(is.Attr("reverse"))}
			if is.Has("showValue") {
				b := ooxml.ParseBool(is.Attr("showValue"))
				set.ShowValue = &b
			}
			if is.Has("percent") {
				b := ooxml.ParseBool(is.Attr("percent"))
				set.Percent = &b
			}
			for _, v := range is.Each("cfvo") {
				set.Values = append(set.Values, parseCfvo(v))
			}
			rule.IconSet = set
		}
		if ext := r.Path("extLst"); ext != nil {
			for _, e := range ext.Children {
				if id := e.Child("id"); id != nil {
					if p.cfByID == nil {
						p.cfByID = map[string]*model.CondFmtRule{}
					}
					p.cfByID[strings.TrimSpace(id.Text)] = &rule
				}
			}
		}
		cf.Rules = append(cf.Rules, rule)
	}
	// Register rule pointers after append (slice elements are stable now).
	for i := range cf.Rules {
		for id, rp := range p.cfByID {
			if rp == &cf.Rules[i] {
				continue
			}
			if rp.Priority == cf.Rules[i].Priority && rp.Type == cf.Rules[i].Type && rp != &cf.Rules[i] {
				p.cfByID[id] = &cf.Rules[i]
			}
		}
	}
	p.sh.ConditionalFormats = append(p.sh.ConditionalFormats, cf)
}

// parseExt handles known worksheet extensions; it returns false when the
// extension should be preserved raw.
func (p *sheetParser) parseExt(e *ooxml.Node) bool {
	switch strings.ToUpper(e.Attr("uri")) {
	case "{05C60535-1F16-4FD2-B633-F4F36F0B64E0}": // sparklines
		for _, g := range e.Path("sparklineGroups").Each("sparklineGroup") {
			grp := model.SparklineGroup{Type: g.Attr("type"), DisplayEmptyCellsAs: g.Attr("displayEmptyCellsAs"), ShowMarkers: ooxml.ParseBool(g.Attr("markers")),
				ShowHigh: ooxml.ParseBool(g.Attr("high")), ShowLow: ooxml.ParseBool(g.Attr("low")), ShowFirst: ooxml.ParseBool(g.Attr("first")), ShowLast: ooxml.ParseBool(g.Attr("last")),
				ShowNegative: ooxml.ParseBool(g.Attr("negative")), DisplayXAxis: ooxml.ParseBool(g.Attr("displayXAxis")), DisplayHidden: ooxml.ParseBool(g.Attr("displayHidden")),
				MinAxisType: g.Attr("minAxisType"), MaxAxisType: g.Attr("maxAxisType"), RightToLeft: ooxml.ParseBool(g.Attr("rightToLeft"))}
			grp.LineWeight, _ = ooxml.ParseFloat(g.Attr("lineWeight"))
			grp.ManualMin, _ = ooxml.ParseFloat(g.Attr("manualMin"))
			grp.ManualMax, _ = ooxml.ParseFloat(g.Attr("manualMax"))
			grp.ColorSeries = parseColor(g.Child("colorSeries"))
			grp.ColorNegative = parseColor(g.Child("colorNegative"))
			grp.ColorAxis = parseColor(g.Child("colorAxis"))
			grp.ColorMarkers = parseColor(g.Child("colorMarkers"))
			grp.ColorFirst = parseColor(g.Child("colorFirst"))
			grp.ColorLast = parseColor(g.Child("colorLast"))
			grp.ColorHigh = parseColor(g.Child("colorHigh"))
			grp.ColorLow = parseColor(g.Child("colorLow"))
			if f := g.Child("f"); f != nil {
				grp.DateAxis = f.Text
			}
			for _, s := range g.Path("sparklines").Each("sparkline") {
				sl := model.Sparkline{}
				if f := s.Child("f"); f != nil {
					sl.Formula = f.Text
				}
				if sq := s.Child("sqref"); sq != nil {
					sl.Ref, _ = ref.ParseCell(strings.TrimSpace(sq.Text))
				}
				grp.Sparklines = append(grp.Sparklines, sl)
			}
			p.sh.Sparklines = append(p.sh.Sparklines, grp)
		}
		return true
	case "{78C0D931-6437-407D-A8EE-F0AAD7539E65}": // x14 conditional formatting
		for _, cf := range e.Path("conditionalFormattings").Each("conditionalFormatting") {
			for _, r := range cf.Each("cfRule") {
				id := strings.TrimSpace(r.Attr("id"))
				rule := p.cfByID[id]
				if rule == nil || rule.DataBar == nil {
					if rule == nil {
						return false
					}
					continue
				}
				if db := r.Child("dataBar"); db != nil {
					d := rule.DataBar
					if db.Has("gradient") {
						b := ooxml.ParseBool(db.Attr("gradient"))
						d.Gradient = &b
					}
					d.AxisPosition = db.Attr("axisPosition")
					d.Direction = db.Attr("direction")
					d.BorderColor = parseColor(db.Child("borderColor"))
					d.NegativeColor = parseColor(db.Child("negativeFillColor"))
					d.NegativeBorder = parseColor(db.Child("negativeBorderColor"))
					d.AxisColor = parseColor(db.Child("axisColor"))
				}
			}
		}
		return true
	}
	return false
}
