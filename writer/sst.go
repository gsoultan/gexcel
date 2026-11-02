// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// sharedStrings deduplicates strings into the shared string table. Items
// are stored as serialised <si> bodies so rich text dedups by content too.
type sharedStrings struct {
	items []string
	index map[string]int
	count int // total references
}

func newSharedStrings() *sharedStrings { return &sharedStrings{index: map[string]int{}} }

func (s *sharedStrings) add(v model.Value) int {
	si := siXML(v)
	s.count++
	if i, ok := s.index[si]; ok {
		return i
	}
	s.items = append(s.items, si)
	s.index[si] = len(s.items) - 1
	return len(s.items) - 1
}

func (s *sharedStrings) xml() []byte {
	b := ooxml.NewBuilder()
	b.Start("sst", "xmlns", ooxml.NSMain, "count", ooxml.Int(s.count), "uniqueCount", ooxml.Int(len(s.items)))
	for _, si := range s.items {
		b.WriteString("<si>")
		b.WriteString(si)
		b.WriteString("</si>")
	}
	b.End()
	return b.Bytes()
}

// siXML serialises the inside of an <si> or <is> element.
func siXML(v model.Value) string {
	if len(v.Rich) == 0 {
		return textXML(v.Str)
	}
	var b ooxml.Builder
	for _, r := range v.Rich {
		b.WriteString("<r>")
		if r.Font != nil {
			fontXML(&b, *r.Font, "rPr", "rFont")
		}
		b.WriteString(textXML(r.Text))
		b.WriteString("</r>")
	}
	return b.String()
}

// textXML writes a <t> element, preserving leading/trailing whitespace.
func textXML(s string) string {
	s = ooxml.EscapeUnicode(s)
	if len(s) > model.MaxCellChars {
		s = truncateRunes(s, model.MaxCellChars)
	}
	if s != "" && (s[0] == ' ' || s[len(s)-1] == ' ' || strings.ContainsAny(s, "\n\t\r")) {
		return `<t xml:space="preserve">` + ooxml.EscapeText(s) + "</t>"
	}
	return "<t>" + ooxml.EscapeText(s) + "</t>"
}

func truncateRunes(s string, n int) string {
	i := 0
	for pos := range s {
		if i == n {
			return s[:pos]
		}
		i++
	}
	return s
}
