// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/style"
)

// writeComments writes the legacy comments part, its VML drawing and, for
// threaded comments, the threadedComments part. It returns the VML
// relationship id for <legacyDrawing>.
func (p *packer) writeComments(sheetPart string, sheetIndex int, sh *model.Sheet) (string, error) {
	p.nComments++
	commentsPart := fmt.Sprintf("xl/comments%d.xml", p.nComments)
	p.nVML++
	vmlPart := fmt.Sprintf("xl/drawings/vmlDrawing%d.vml", p.nVML)

	authors := []string{}
	authorIdx := map[string]int{}
	authorID := func(a string) int {
		if id, ok := authorIdx[a]; ok {
			return id
		}
		authors = append(authors, a)
		authorIdx[a] = len(authors) - 1
		return len(authors) - 1
	}
	hasThreaded := false
	for _, c := range sh.Comments {
		if c.Threaded {
			hasThreaded = true
			authorID("tc=" + threadedID(c, sheetIndex))
		} else {
			authorID(orDefault(c.Author, "Author"))
		}
	}
	b := ooxml.NewBuilder()
	b.Start("comments", "xmlns", ooxml.NSMain, "xmlns:mc", ooxml.NSMC, "mc:Ignorable", "xr", "xmlns:xr", ooxml.NSXR)
	b.Start("authors")
	for _, a := range authors {
		b.Elem("author", a)
	}
	b.End()
	b.Start("commentList")
	for _, c := range sh.Comments {
		var aid int
		if c.Threaded {
			aid = authorIdx["tc="+threadedID(c, sheetIndex)]
		} else {
			aid = authorIdx[orDefault(c.Author, "Author")]
		}
		b.Start("comment", "ref", c.Ref.String(), "authorId", ooxml.Int(aid), "shapeId", "0")
		b.Start("text")
		switch {
		case c.Threaded:
			text := "[Threaded comment]\n\nYour version of Excel allows you to read this threaded comment; however, any edits to it will get removed if the file is opened in a newer version of Excel. Learn more: https://go.microsoft.com/fwlink/?linkid=870924\n\nComment:\n    " + c.Text
			for _, r := range c.Replies {
				text += "\nReply:\n    " + r.Text
			}
			b.WriteString(textXML(text))
		case len(c.Rich) > 0:
			b.WriteString(siXML(model.Value{Rich: c.Rich}))
		default:
			// Excel's convention: bold author name, then the text.
			b.WriteString("<r>")
			fontXML(b, modelFontBold(), "rPr", "rFont")
			b.WriteString(textXML(orDefault(c.Author, "Author") + ":"))
			b.WriteString("</r><r>")
			fontXML(b, modelFontPlain(), "rPr", "rFont")
			b.WriteString(textXML("\n" + c.Text))
			b.WriteString("</r>")
		}
		b.End().End()
	}
	b.End().End()
	if err := p.pw.Add(commentsPart, ooxml.CTComments, b.Bytes()); err != nil {
		return "", err
	}
	p.pw.Rel(sheetPart, ooxml.RelComments, ooxml.Relative(sheetPart, commentsPart), "")
	if err := p.pw.Add(vmlPart, ooxml.CTVML, vmlXML(sh, p.nVML)); err != nil {
		return "", err
	}
	p.pw.Default("vml", ooxml.CTVML)
	vmlRel := p.pw.Rel(sheetPart, ooxml.RelVMLDrawing, ooxml.Relative(sheetPart, vmlPart), "")
	if hasThreaded {
		p.nThreaded++
		tcPart := fmt.Sprintf("xl/threadedComments/threadedComment%d.xml", p.nThreaded)
		if err := p.pw.Add(tcPart, ooxml.CTThreadedComments, p.threadedCommentsXML(sh, sheetIndex)); err != nil {
			return "", err
		}
		p.pw.Rel(sheetPart, ooxml.RelThreadedComments, ooxml.Relative(sheetPart, tcPart), "")
	}
	return vmlRel, nil
}

func modelFontBold() style.Font {
	return style.Font{Bold: true, Size: 9, Name: "Tahoma", Family: 2, Charset: 1}
}

func modelFontPlain() style.Font { return style.Font{Size: 9, Name: "Tahoma", Family: 2, Charset: 1} }

// threadedID returns the stable GUID of a threaded comment.
func threadedID(c model.Comment, sheetIndex int) string {
	if c.ID != "" {
		return c.ID
	}
	return guidOf(fmt.Sprintf("comment/%d/%s/%s", sheetIndex, c.Ref.String(), c.Text))
}

// guidOf derives a deterministic GUID-shaped string from a key.
func guidOf(key string) string {
	h := sha1.Sum([]byte(key))
	return strings.ToUpper(fmt.Sprintf("{%x-%x-%x-%x-%x}", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16]))
}

func (p *packer) personID(author string) string {
	if id, ok := p.persons[author]; ok {
		return id
	}
	id := guidOf("person/" + author)
	p.persons[author] = id
	p.personOrder = append(p.personOrder, author)
	return id
}

func (p *packer) threadedCommentsXML(sh *model.Sheet, sheetIndex int) []byte {
	b := ooxml.NewBuilder()
	b.Start("ThreadedComments", "xmlns", ooxml.NSThreaded, "xmlns:x", ooxml.NSMain)
	for _, c := range sh.Comments {
		if !c.Threaded {
			continue
		}
		id := threadedID(c, sheetIndex)
		created := orDefault(c.Created, "2026-01-01T00:00:00.00")
		b.Start("threadedComment", "ref", c.Ref.String(), "dT", created, "personId", p.personID(orDefault(c.Author, "Author")), "id", id, "done", boolAttr(c.Done))
		b.Elem("text", c.Text)
		b.End()
		for i, r := range c.Replies {
			rid := r.ID
			if rid == "" {
				rid = guidOf(fmt.Sprintf("reply/%s/%d", id, i))
			}
			b.Start("threadedComment", "ref", c.Ref.String(), "dT", orDefault(r.Created, created), "personId", p.personID(orDefault(r.Author, "Author")), "id", rid, "parentId", id)
			b.Elem("text", r.Text)
			b.End()
		}
	}
	b.End()
	return b.Bytes()
}

func (p *packer) personsXML() []byte {
	b := ooxml.NewBuilder()
	b.Start("personList", "xmlns", "http://schemas.microsoft.com/office/spreadsheetml/2018/threadedcomments", "xmlns:x", ooxml.NSMain)
	for _, a := range p.personOrder {
		b.Empty("person", "displayName", a, "id", p.persons[a], "userId", a, "providerId", "None")
	}
	b.End()
	return b.Bytes()
}

// vmlXML builds the legacy drawing that positions comment boxes.
func vmlXML(sh *model.Sheet, n int) []byte {
	var b strings.Builder
	b.WriteString(`<xml xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">`)
	b.WriteString(`<o:shapelayout v:ext="edit"><o:idmap v:ext="edit" data="` + ooxml.Int(n) + `"/></o:shapelayout>`)
	b.WriteString(`<v:shapetype id="_x0000_t202" coordsize="21600,21600" o:spt="202" path="m,l,21600r21600,l21600,xe"><v:stroke joinstyle="miter"/><v:path gradientshapeok="t" o:connecttype="rect"/></v:shapetype>`)
	for i, c := range sh.Comments {
		w, h := c.Width, c.Height
		if w == 0 {
			w = 144
		}
		if h == 0 {
			h = 79
		}
		col, row := c.Ref.Col-1, c.Ref.Row-1
		// Place the box to the right of the cell, one row above.
		leftCol := col + 1
		topRow := max(row-1, 0)
		vis := "hidden"
		if c.Visible {
			vis = "visible"
		}
		fmt.Fprintf(&b, `<v:shape id="_x0000_s%d" type="#_x0000_t202" style="position:absolute;margin-left:%gpt;margin-top:%gpt;width:%gpt;height:%gpt;z-index:%d;visibility:%s" fillcolor="#ffffe1" o:insetmode="auto">`,
			n*1024+i+1, float64(leftCol)*48+15, float64(topRow)*15, w*0.75, h*0.75, i+1, vis)
		b.WriteString(`<v:fill color2="#ffffe1"/><v:shadow on="t" color="black" obscured="t"/><v:path o:connecttype="none"/>`)
		b.WriteString(`<v:textbox style="mso-direction-alt:auto"><div style="text-align:left"></div></v:textbox>`)
		fmt.Fprintf(&b, `<x:ClientData ObjectType="Note"><x:MoveWithCells/><x:SizeWithCells/><x:Anchor>%d, 15, %d, 2, %d, 31, %d, 1</x:Anchor><x:AutoFill>False</x:AutoFill><x:Row>%d</x:Row><x:Column>%d</x:Column>`,
			leftCol, topRow, leftCol+2, topRow+4, row, col)
		if c.Visible {
			b.WriteString(`<x:Visible/>`)
		}
		b.WriteString(`</x:ClientData></v:shape>`)
	}
	b.WriteString(`</xml>`)
	return []byte(b.String())
}
