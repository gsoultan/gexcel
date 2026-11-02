// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
)

func (d *docReader) readComments(sh *model.Sheet, part string) {
	root, err := d.dom(part)
	if err != nil || root == nil {
		return
	}
	d.use(part)
	var authors []string
	for _, a := range root.Path("authors").Each("author") {
		authors = append(authors, a.Text)
	}
	for _, c := range root.Path("commentList").Each("comment") {
		cell, err := ref.ParseCell(c.Attr("ref"))
		if err != nil {
			continue
		}
		cm := model.Comment{Ref: cell}
		if i := ooxml.ParseInt(c.Attr("authorId"), -1); i >= 0 && i < len(authors) {
			cm.Author = authors[i]
		}
		if strings.HasPrefix(cm.Author, "tc=") {
			cm.Threaded = true
			cm.ID = strings.TrimPrefix(cm.Author, "tc=")
			cm.Author = ""
		}
		if t := c.Child("text"); t != nil {
			v := parseRichText(t)
			cm.Text = v.Str
			cm.Rich = v.Rich
			// Strip the conventional "Author:\n" prefix Excel adds.
			if !cm.Threaded && cm.Author != "" && strings.HasPrefix(cm.Text, cm.Author+":\n") {
				cm.Text = strings.TrimPrefix(cm.Text, cm.Author+":\n")
				cm.Rich = nil
			}
		}
		sh.Comments = append(sh.Comments, cm)
	}
}

func (d *docReader) readThreadedComments(sh *model.Sheet, part string) {
	root, err := d.dom(part)
	if err != nil || root == nil {
		return
	}
	d.use(part)
	persons := d.persons()
	byID := map[string]*model.Comment{}
	for i := range sh.Comments {
		if sh.Comments[i].Threaded {
			byID[sh.Comments[i].ID] = &sh.Comments[i]
		}
	}
	for _, tc := range root.Each("threadedComment") {
		id := tc.Attr("id")
		text := ""
		if t := tc.Child("text"); t != nil {
			text = t.Text
		}
		author := persons[tc.Attr("personId")]
		if parent := tc.Attr("parentId"); parent != "" {
			if c := byID[parent]; c != nil {
				c.Replies = append(c.Replies, model.CommentReply{Author: author, Text: text, ID: id, Created: tc.Attr("dT")})
			}
			continue
		}
		c := byID[id]
		if c == nil {
			cell, err := ref.ParseCell(tc.Attr("ref"))
			if err != nil {
				continue
			}
			sh.Comments = append(sh.Comments, model.Comment{Ref: cell, Threaded: true, ID: id})
			c = &sh.Comments[len(sh.Comments)-1]
			byID[id] = c
		}
		c.Text = text
		c.Rich = nil
		c.Author = author
		c.Created = tc.Attr("dT")
		c.Done = ooxml.ParseBool(tc.Attr("done"))
	}
}

func (d *docReader) persons() map[string]string {
	out := map[string]string{}
	wbRels, _ := d.pr.Rels(d.wbPart)
	rel, ok := ooxml.FindRel(wbRels, ooxml.RelPersons)
	if !ok {
		return out
	}
	root, err := d.dom(ooxml.Resolve(d.wbPart, rel.Target))
	if err != nil || root == nil {
		return out
	}
	for _, p := range root.Each("person") {
		out[p.Attr("id")] = p.Attr("displayName")
	}
	return out
}

var vmlShapeRe = regexp.MustCompile(`(?s)<v:shape\b[^>]*>.*?</v:shape>`)
var vmlRowRe = regexp.MustCompile(`<x:Row>\s*(\d+)\s*</x:Row>`)
var vmlColRe = regexp.MustCompile(`<x:Column>\s*(\d+)\s*</x:Column>`)
var vmlVisibleRe = regexp.MustCompile(`visibility:\s*visible`)
var vmlSizeRe = regexp.MustCompile(`width:\s*([\d.]+)pt;height:\s*([\d.]+)pt`)

// readVML picks the comment box visibility and size out of the legacy drawing.
func (d *docReader) readVML(sh *model.Sheet, part string) {
	data, err := d.pr.Part(part)
	if err != nil {
		return
	}
	d.use(part)
	d.use(ooxml.RelsName(part))
	for _, shape := range vmlShapeRe.FindAll(data, -1) {
		if !strings.Contains(string(shape), `ObjectType="Note"`) {
			continue
		}
		rm := vmlRowRe.FindSubmatch(shape)
		cm := vmlColRe.FindSubmatch(shape)
		if rm == nil || cm == nil {
			continue
		}
		row, _ := strconv.Atoi(string(rm[1]))
		col, _ := strconv.Atoi(string(cm[1]))
		for i := range sh.Comments {
			c := &sh.Comments[i]
			if c.Ref.Row == row+1 && c.Ref.Col == col+1 {
				c.Visible = vmlVisibleRe.Match(shape) || strings.Contains(string(shape), "<x:Visible/>")
				if m := vmlSizeRe.FindSubmatch(shape); m != nil {
					w, _ := strconv.ParseFloat(string(m[1]), 64)
					h, _ := strconv.ParseFloat(string(m[2]), 64)
					c.Width, c.Height = w/0.75, h/0.75
				}
			}
		}
	}
}

func (d *docReader) readTable(sh *model.Sheet, part string) {
	data, err := d.pr.Part(part)
	if err != nil {
		return
	}
	d.use(part)
	d.use(ooxml.RelsName(part))
	root, err := ooxml.ParseDOM(data)
	if err != nil || root == nil {
		return
	}
	t := &model.Table{ID: ooxml.ParseInt(root.Attr("id"), 0), Name: root.Attr("name"), DisplayName: root.Attr("displayName"), Comment: root.Attr("comment")}
	t.Range, _ = ref.ParseRange(root.Attr("ref"))
	if root.Has("headerRowCount") {
		b := ooxml.ParseInt(root.Attr("headerRowCount"), 1) > 0
		t.HeaderRow = &b
	}
	if n := ooxml.ParseInt(root.Attr("totalsRowCount"), 0); n > 0 {
		t.TotalsRow = true
		t.TotalsRowCount = n
	}
	if root.Attr("tableType") != "" {
		t.Raw = data // query/xml tables: keep verbatim
	}
	if af := root.Child("autoFilter"); af != nil {
		t.AutoFilter = parseAutoFilter(af)
	} else {
		t.NoAutoFilter = true
	}
	for _, c := range root.Path("tableColumns").Each("tableColumn") {
		col := model.TableColumn{ID: ooxml.ParseInt(c.Attr("id"), 0), Name: c.Attr("name"), UniqueName: c.Attr("uniqueName"), TotalsRowFunction: c.Attr("totalsRowFunction"),
			TotalsRowLabel: c.Attr("totalsRowLabel"), QueryTableFieldID: ooxml.ParseInt(c.Attr("queryTableFieldId"), 0), DataDxfID: ooxml.ParseInt(c.Attr("dataDxfId"), 0),
			HeaderDxfID: ooxml.ParseInt(c.Attr("headerRowDxfId"), 0), TotalsDxfID: ooxml.ParseInt(c.Attr("totalsRowDxfId"), 0)}
		if f := c.Child("calculatedColumnFormula"); f != nil {
			col.CalculatedColumnFormula = f.Text
		}
		if f := c.Child("totalsRowFormula"); f != nil {
			col.TotalsRowFormula = f.Text
		}
		t.Columns = append(t.Columns, col)
	}
	if si := root.Child("tableStyleInfo"); si != nil {
		t.Style = model.TableStyle{Name: si.Attr("name"), ShowFirstColumn: ooxml.ParseBool(si.Attr("showFirstColumn")), ShowLastColumn: ooxml.ParseBool(si.Attr("showLastColumn")),
			ShowRowStripes: ooxml.ParseBool(si.Attr("showRowStripes")), ShowColumnStripes: ooxml.ParseBool(si.Attr("showColumnStripes"))}
	}
	sh.Tables = append(sh.Tables, t)
}

// readChartsheet parses the chart of a chart sheet when possible; the raw
// part stays for round trips.
func (d *docReader) readChartsheet(sh *model.Sheet) {
	if sh.Raw == nil || d.opts.SkipDrawings {
		return
	}
	for _, r := range sh.Raw.Rels {
		if r.Type != ooxml.RelDrawing {
			continue
		}
		drawing := sh.Raw.Related[ooxml.Resolve(sh.Raw.Name, r.Target)]
		if drawing == nil {
			continue
		}
		for _, dr := range drawing.Rels {
			if dr.Type == ooxml.RelChart {
				if cp := drawing.Related[ooxml.Resolve(drawing.Name, dr.Target)]; cp != nil {
					if ch := parseChart(cp.Data); ch != nil {
						sh.Charts = append(sh.Charts, ch)
					}
				}
			}
		}
	}
}
