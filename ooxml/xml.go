// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package ooxml holds the Open Packaging Conventions plumbing shared by the
// reader and the writer: part names, content types, relationship types, a
// zip package reader/writer, and small XML helpers.
package ooxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Header is the XML declaration every part starts with.
const Header = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// XML namespaces.
const (
	NSMain          = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	NSRel           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	NSPkgRel        = "http://schemas.openxmlformats.org/package/2006/relationships"
	NSContentTypes  = "http://schemas.openxmlformats.org/package/2006/content-types"
	NSDrawing       = "http://schemas.openxmlformats.org/drawingml/2006/main"
	NSSpreadsheetDr = "http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing"
	NSChart         = "http://schemas.openxmlformats.org/drawingml/2006/chart"
	NSChartEx       = "http://schemas.microsoft.com/office/drawing/2014/chartex"
	NSMC            = "http://schemas.openxmlformats.org/markup-compatibility/2006"
	NSX14ac         = "http://schemas.microsoft.com/office/spreadsheetml/2009/9/ac"
	NSX14           = "http://schemas.microsoft.com/office/spreadsheetml/2009/9/main"
	NSX15           = "http://schemas.microsoft.com/office/spreadsheetml/2010/11/main"
	NSXR            = "http://schemas.microsoft.com/office/spreadsheetml/2014/revision"
	NSXR2           = "http://schemas.microsoft.com/office/spreadsheetml/2015/revision2"
	NSXR3           = "http://schemas.microsoft.com/office/spreadsheetml/2016/revision3"
	NSXR6           = "http://schemas.microsoft.com/office/spreadsheetml/2016/revision6"
	NSXR10          = "http://schemas.microsoft.com/office/spreadsheetml/2016/revision10"
	NSXDA           = "http://schemas.microsoft.com/office/spreadsheetml/2017/dynamicarray"
	NSThreaded      = "http://schemas.microsoft.com/office/spreadsheetml/2018/threadedcomments"
	NSVML           = "urn:schemas-microsoft-com:vml"
	NSOffice        = "urn:schemas-microsoft-com:office:office"
	NSExcel         = "urn:schemas-microsoft-com:office:excel"
	NSCoreProps     = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
	NSDC            = "http://purl.org/dc/elements/1.1/"
	NSDCTerms       = "http://purl.org/dc/terms/"
	NSDCMIType      = "http://purl.org/dc/dcmitype/"
	NSXSI           = "http://www.w3.org/2001/XMLSchema-instance"
	NSExtProps      = "http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
	NSDocPropsVT    = "http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"
	NSCustomProps   = "http://schemas.openxmlformats.org/officeDocument/2006/custom-properties"
)

// Builder writes XML with a strings.Builder, escaping text and attributes.
type Builder struct {
	bytes.Buffer
	open []string
}

// NewBuilder returns a builder that starts with the XML header.
func NewBuilder() *Builder {
	b := &Builder{}
	b.WriteString(Header)
	return b
}

// Start opens an element with attributes given as name, value pairs.
// Attributes with an empty value are skipped unless the name ends with "!".
func (b *Builder) Start(tag string, attrs ...string) *Builder {
	b.WriteByte('<')
	b.WriteString(tag)
	b.attrs(attrs)
	b.WriteByte('>')
	b.open = append(b.open, tag)
	return b
}

// Empty writes a self-closing element.
func (b *Builder) Empty(tag string, attrs ...string) *Builder {
	b.WriteByte('<')
	b.WriteString(tag)
	b.attrs(attrs)
	b.WriteString("/>")
	return b
}

// Elem writes an element with text content.
func (b *Builder) Elem(tag, text string, attrs ...string) *Builder {
	b.WriteByte('<')
	b.WriteString(tag)
	b.attrs(attrs)
	b.WriteByte('>')
	b.Text(text)
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteByte('>')
	return b
}

func (b *Builder) attrs(attrs []string) {
	for i := 0; i+1 < len(attrs); i += 2 {
		name, val := attrs[i], attrs[i+1]
		if val == "" && !strings.HasSuffix(name, "!") {
			continue
		}
		name = strings.TrimSuffix(name, "!")
		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteString(`="`)
		b.WriteString(EscapeAttr(val))
		b.WriteByte('"')
	}
}

// End closes the innermost open element.
func (b *Builder) End() *Builder {
	n := len(b.open) - 1
	b.WriteString("</")
	b.WriteString(b.open[n])
	b.WriteByte('>')
	b.open = b.open[:n]
	return b
}

// Text writes escaped character data.
func (b *Builder) Text(s string) *Builder {
	b.WriteString(EscapeText(s))
	return b
}

// Raw writes bytes as they are.
func (b *Builder) Raw(s string) *Builder {
	b.WriteString(s)
	return b
}

// EscapeText escapes text content and drops characters XML 1.0 forbids.
func EscapeText(s string) string {
	s = Clean(s)
	if !strings.ContainsAny(s, "<>&\r") {
		return s
	}
	var out bytes.Buffer
	xml.EscapeText(&out, []byte(s))
	return out.String()
}

// EscapeAttr escapes an attribute value (double-quoted).
func EscapeAttr(s string) string {
	s = Clean(s)
	if !strings.ContainsAny(s, "<>&\"\n\t\r") {
		return s
	}
	var out bytes.Buffer
	xml.EscapeText(&out, []byte(s))
	return out.String()
}

// Clean removes characters that are not legal in XML 1.0 documents.
func Clean(s string) string {
	bad := func(r rune) bool {
		return r < 0x20 && r != '\n' && r != '\t' && r != '\r' || r == 0xFFFE || r == 0xFFFF || r == utf8.RuneError || r >= 0xD800 && r <= 0xDFFF
	}
	if !strings.ContainsFunc(s, bad) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if bad(r) {
			return -1
		}
		return r
	}, s)
}

// EscapeUnicode writes characters that Excel cannot store in XML as the
// _xHHHH_ escapes the shared string table uses, and escapes a literal
// "_xHHHH_" sequence itself.
func EscapeUnicode(s string) string {
	need := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 && c != '\n' && c != '\t' && c != '\r' {
			need = true
			break
		}
		if c == '_' && i+6 < len(s) && s[i+1] == 'x' && s[i+6] == '_' && isHex4(s[i+2:i+6]) {
			need = true
			break
		}
	}
	if !need {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c < 0x20 && c != '\n' && c != '\t' && c != '\r':
			b.WriteString("_x")
			b.WriteString(strings.ToUpper(leftPad(strconv.FormatInt(int64(c), 16), 4)))
			b.WriteByte('_')
		case c == '_' && i+6 < len(s) && s[i+1] == 'x' && s[i+6] == '_' && isHex4(s[i+2:i+6]):
			b.WriteString("_x005F_")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// UnescapeUnicode reverses EscapeUnicode.
func UnescapeUnicode(s string) string {
	if !strings.Contains(s, "_x") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '_' && i+6 < len(s) && s[i+1] == 'x' && s[i+6] == '_' && isHex4(s[i+2:i+6]) {
			v, _ := strconv.ParseUint(s[i+2:i+6], 16, 16)
			b.WriteRune(rune(v))
			i += 6
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isHex4(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func leftPad(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// Bool formats a boolean attribute as "1" or "0".
func Bool(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// Int formats an integer.
func Int(v int) string { return strconv.Itoa(v) }

// Float formats a float the way Excel writes cell values and attributes:
// shortest round-trip decimal, exponent form only for very large or small
// magnitudes.
func Float(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return "0"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	av := math.Abs(v)
	if av >= 1e-4 && av < 1e21 {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strconv.FormatFloat(v, 'E', -1, 64)
}

// ParseFloat parses a float attribute, accepting both Go and Excel forms.
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// ParseBool parses a boolean attribute ("1", "0", "true", "false").
func ParseBool(s string) bool {
	switch strings.TrimSpace(s) {
	case "1", "true", "on":
		return true
	}
	return false
}

// ParseInt parses an integer attribute, returning def on error.
func ParseInt(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

// Node is a parsed XML element, for the smaller parts that are easier to
// walk as a tree than to stream.
type Node struct {
	Name     string // local name
	Space    string // namespace URI
	Attrs    []xml.Attr
	Children []*Node
	Text     string // concatenated character data
}

// ParseDOM parses an XML document into a Node tree. Processing
// instructions and comments are dropped.
func ParseDOM(data []byte) (*Node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var stack []*Node
	var root *Node
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &Node{Name: t.Name.Local, Space: t.Name.Space, Attrs: t.Attr}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.Children = append(p.Children, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("ooxml: empty document")
	}
	return root, nil
}

// Attr returns an attribute by local name ("" when absent).
func (n *Node) Attr(name string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// Has reports whether the attribute is present.
func (n *Node) Has(name string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return true
		}
	}
	return false
}

// Child returns the first child with the local name, or nil.
func (n *Node) Child(name string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Path follows a chain of child names.
func (n *Node) Path(names ...string) *Node {
	for _, name := range names {
		n = n.Child(name)
		if n == nil {
			return nil
		}
	}
	return n
}

// Each returns the children with the local name.
func (n *Node) Each(name string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, c := range n.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

// Val returns the "val" attribute, the common DrawingML/chart shape.
func (n *Node) Val() string { return n.Attr("val") }

// ValFloat returns the "val" attribute as a float, or def.
func (n *Node) ValFloat(def float64) float64 {
	if n == nil {
		return def
	}
	f, err := ParseFloat(n.Attr("val"))
	if err != nil {
		return def
	}
	return f
}

// ValInt returns the "val" attribute as an int, or def.
func (n *Node) ValInt(def int) int {
	if n == nil {
		return def
	}
	return ParseInt(n.Attr("val"), def)
}

// ValBool returns the "val" attribute as a bool; a missing attribute on a
// present element means true (as in <b/>).
func (n *Node) ValBool() bool {
	if n == nil {
		return false
	}
	if !n.Has("val") {
		return true
	}
	return ParseBool(n.Attr("val"))
}

// ValBoolDefault is ValBool on a possibly nil node: a missing element is
// false, a present element without val is true.
func (n *Node) ValBoolDefault() bool {
	if n == nil {
		return false
	}
	return n.ValBool()
}
