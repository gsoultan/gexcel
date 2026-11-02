// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/ref"
)

// Node is a parsed formula expression.
type Node interface{ node() }

// Literal nodes.
type (
	// NumberNode is a numeric literal.
	NumberNode struct{ Value float64 }
	// StringNode is a text literal (unquoted).
	StringNode struct{ Value string }
	// BoolNode is TRUE or FALSE.
	BoolNode struct{ Value bool }
	// ErrorNode is an error literal such as #N/A.
	ErrorNode struct{ Code string }
	// RefNode is a cell or range reference, optionally on another sheet.
	// External refers to another workbook ("[1]") when non-empty.
	RefNode struct {
		Sheet    string // "" = current sheet; "Sheet1:Sheet3" for 3-D references
		Range    ref.Range
		External string
	}
	// NameNode is a defined name or structured reference head.
	NameNode struct{ Name string }
	// StructuredNode is a table reference such as Table1[Col].
	StructuredNode struct{ Text string }
	// UnaryNode is +x, -x, x%, @x or x# (spill).
	UnaryNode struct {
		Op   string
		X    Node
		Post bool
	}
	// BinaryNode is x op y.
	BinaryNode struct {
		Op   string
		L, R Node
	}
	// CallNode is a function call.
	CallNode struct {
		Name string
		Args []Node // nil entries are omitted arguments
	}
	// ArrayNode is an array constant {1,2;3,4}.
	ArrayNode struct{ Rows [][]Node }
	// EmptyNode stands for an omitted argument.
	EmptyNode struct{}
)

func (NumberNode) node()     {}
func (StringNode) node()     {}
func (BoolNode) node()       {}
func (ErrorNode) node()      {}
func (RefNode) node()        {}
func (NameNode) node()       {}
func (StructuredNode) node() {}
func (UnaryNode) node()      {}
func (BinaryNode) node()     {}
func (CallNode) node()       {}
func (ArrayNode) node()      {}
func (EmptyNode) node()      {}

// ErrSyntax is returned for formulas that do not parse.
var ErrSyntax = errors.New("formula: syntax error")

type parser struct {
	toks []Token
	pos  int
}

// Parse builds the expression tree of a formula (leading "=" optional).
func Parse(text string) (Node, error) {
	toks := Tokenize(text)
	// Drop whitespace except where it acts as the intersection operator
	// (between two references), which we do not support; keep it simple.
	clean := toks[:0:0]
	for _, t := range toks {
		if t.Kind != TokSpace {
			clean = append(clean, t)
		}
	}
	p := &parser{toks: clean}
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: empty formula", ErrSyntax)
	}
	n, err := p.expr(0)
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("%w: unexpected %q", ErrSyntax, p.toks[p.pos].Text)
	}
	return n, nil
}

func (p *parser) peek() *Token {
	if p.pos < len(p.toks) {
		return &p.toks[p.pos]
	}
	return nil
}

func (p *parser) next() *Token {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

// Precedence climbing. Higher binds tighter.
func binPrec(op string) int {
	switch op {
	case "=", "<>", "<", ">", "<=", ">=":
		return 1
	case "&":
		return 2
	case "+", "-":
		return 3
	case "*", "/":
		return 4
	case "^":
		return 5
	case ":":
		return 8
	}
	return 0
}

func (p *parser) expr(minPrec int) (Node, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t == nil || t.Kind != TokOp {
			return left, nil
		}
		prec := binPrec(t.Text)
		if prec == 0 || prec < minPrec {
			return left, nil
		}
		p.next()
		// ^ is left-associative in Excel (2^3^2 = 64); all others too.
		right, err := p.expr(prec + 1)
		if err != nil {
			return nil, err
		}
		if t.Text == ":" {
			// Sheet1:'Sheet 3'!A1 tokenizes as name : ref; fold it into a 3-D reference.
			if ln, ok := left.(NameNode); ok {
				if rn, ok := right.(RefNode); ok && rn.Sheet != "" && !strings.Contains(rn.Sheet, ":") {
					rn.Sheet = ln.Name + ":" + rn.Sheet
					left = rn
					continue
				}
			}
		}
		left = BinaryNode{Op: t.Text, L: left, R: right}
	}
}

func (p *parser) unary() (Node, error) {
	t := p.peek()
	if t == nil {
		return nil, fmt.Errorf("%w: unexpected end", ErrSyntax)
	}
	if t.Kind == TokOp && (t.Text == "-" || t.Text == "+" || t.Text == "@") {
		p.next()
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		if t.Text == "+" {
			return x, nil
		}
		return UnaryNode{Op: t.Text, X: x}, nil
	}
	x, err := p.primary()
	if err != nil {
		return nil, err
	}
	// Postfix operators.
	for {
		t := p.peek()
		if t == nil || t.Kind != TokOp || (t.Text != "%" && t.Text != "#") {
			return x, nil
		}
		p.next()
		x = UnaryNode{Op: t.Text, X: x, Post: true}
	}
}

func (p *parser) primary() (Node, error) {
	t := p.next()
	if t == nil {
		return nil, fmt.Errorf("%w: unexpected end", ErrSyntax)
	}
	switch t.Kind {
	case TokNumber:
		f, err := strconv.ParseFloat(t.Text, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: bad number %q", ErrSyntax, t.Text)
		}
		return NumberNode{f}, nil
	case TokString:
		s := t.Text
		if len(s) >= 2 {
			s = s[1 : len(s)-1]
		}
		return StringNode{strings.ReplaceAll(s, `""`, `"`)}, nil
	case TokBool:
		return BoolNode{strings.EqualFold(t.Text, "TRUE")}, nil
	case TokError:
		return ErrorNode{strings.ToUpper(t.Text)}, nil
	case TokRef:
		return parseRefNode(t.Text)
	case TokName:
		return NameNode{t.Text}, nil
	case TokStructured:
		return StructuredNode{t.Text}, nil
	case TokFunc:
		return p.call(t.Text)
	case TokOpen:
		x, err := p.expr(0)
		if err != nil {
			return nil, err
		}
		if c := p.next(); c == nil || c.Kind != TokClose {
			return nil, fmt.Errorf("%w: missing )", ErrSyntax)
		}
		return x, nil
	case TokArrayOpen:
		return p.array()
	}
	return nil, fmt.Errorf("%w: unexpected %q", ErrSyntax, t.Text)
}

func (p *parser) call(name string) (Node, error) {
	if o := p.next(); o == nil || o.Kind != TokOpen {
		return nil, fmt.Errorf("%w: expected ( after %s", ErrSyntax, name)
	}
	name = strings.ToUpper(name)
	name = strings.TrimPrefix(name, "_XLFN.")
	name = strings.TrimPrefix(name, "_XLWS.")
	call := CallNode{Name: name}
	if c := p.peek(); c != nil && c.Kind == TokClose {
		p.next()
		return call, nil
	}
	for {
		t := p.peek()
		if t == nil {
			return nil, fmt.Errorf("%w: unterminated call %s", ErrSyntax, name)
		}
		if t.Kind == TokSep {
			p.next()
			call.Args = append(call.Args, EmptyNode{})
			continue
		}
		if t.Kind == TokClose {
			p.next()
			call.Args = append(call.Args, EmptyNode{})
			return call, nil
		}
		arg, err := p.expr(0)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, arg)
		t = p.next()
		if t == nil {
			return nil, fmt.Errorf("%w: unterminated call %s", ErrSyntax, name)
		}
		if t.Kind == TokClose {
			return call, nil
		}
		if t.Kind != TokSep {
			return nil, fmt.Errorf("%w: unexpected %q in %s", ErrSyntax, t.Text, name)
		}
	}
}

func (p *parser) array() (Node, error) {
	var rows [][]Node
	var cur []Node
	for {
		t := p.peek()
		if t == nil {
			return nil, fmt.Errorf("%w: unterminated array", ErrSyntax)
		}
		switch {
		case t.Kind == TokArrayClose:
			p.next()
			rows = append(rows, cur)
			return ArrayNode{rows}, nil
		case t.Kind == TokSep && t.Text == ";":
			p.next()
			rows = append(rows, cur)
			cur = nil
		case t.Kind == TokSep:
			p.next()
		default:
			x, err := p.unary()
			if err != nil {
				return nil, err
			}
			cur = append(cur, x)
		}
	}
}

func parseRefNode(text string) (Node, error) {
	n := RefNode{}
	rest := text
	if strings.HasPrefix(rest, "[") {
		if i := strings.IndexByte(rest, ']'); i > 0 {
			n.External = rest[1:i]
			rest = rest[i+1:]
		}
	}
	if i := strings.LastIndexByte(rest, '!'); i >= 0 {
		sheet := rest[:i]
		rest = rest[i+1:]
		if strings.HasPrefix(sheet, "'") && strings.HasSuffix(sheet, "'") && len(sheet) >= 2 {
			sheet = strings.ReplaceAll(sheet[1:len(sheet)-1], "''", "'")
		}
		if strings.HasPrefix(sheet, "[") {
			if j := strings.IndexByte(sheet, ']'); j > 0 {
				n.External = sheet[1:j]
				sheet = sheet[j+1:]
			}
		}
		n.Sheet = sheet
	}
	r, err := ref.ParseRange(rest)
	if err != nil {
		return nil, fmt.Errorf("%w: bad reference %q", ErrSyntax, text)
	}
	n.Range = r
	return n, nil
}

// String renders a node back to formula text.
func String(n Node) string {
	var b strings.Builder
	writeNode(&b, n)
	return b.String()
}

func writeNode(b *strings.Builder, n Node) {
	switch x := n.(type) {
	case NumberNode:
		b.WriteString(strconv.FormatFloat(x.Value, 'g', -1, 64))
	case StringNode:
		b.WriteString(`"` + strings.ReplaceAll(x.Value, `"`, `""`) + `"`)
	case BoolNode:
		if x.Value {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	case ErrorNode:
		b.WriteString(x.Code)
	case RefNode:
		if x.External != "" {
			b.WriteString("[" + x.External + "]")
		}
		if x.Sheet != "" {
			b.WriteString(ref.QuoteSheet(x.Sheet) + "!")
		}
		b.WriteString(x.Range.String())
	case NameNode:
		b.WriteString(x.Name)
	case StructuredNode:
		b.WriteString(x.Text)
	case UnaryNode:
		if x.Post {
			writeNode(b, x.X)
			b.WriteString(x.Op)
		} else {
			b.WriteString(x.Op)
			writeNode(b, x.X)
		}
	case BinaryNode:
		if l, ok := x.L.(BinaryNode); ok && binPrec(l.Op) < binPrec(x.Op) {
			b.WriteByte('(')
			writeNode(b, l)
			b.WriteByte(')')
		} else {
			writeNode(b, x.L)
		}
		b.WriteString(x.Op)
		if r, ok := x.R.(BinaryNode); ok && binPrec(r.Op) <= binPrec(x.Op) {
			b.WriteByte('(')
			writeNode(b, r)
			b.WriteByte(')')
		} else {
			writeNode(b, x.R)
		}
	case CallNode:
		b.WriteString(x.Name)
		b.WriteByte('(')
		for i, a := range x.Args {
			if i > 0 {
				b.WriteByte(',')
			}
			writeNode(b, a)
		}
		b.WriteByte(')')
	case ArrayNode:
		b.WriteByte('{')
		for i, row := range x.Rows {
			if i > 0 {
				b.WriteByte(';')
			}
			for j, c := range row {
				if j > 0 {
					b.WriteByte(',')
				}
				writeNode(b, c)
			}
		}
		b.WriteByte('}')
	case EmptyNode:
	}
}

// References lists every reference in a formula (used for dependency
// tracking). Names are not resolved.
func References(n Node) []RefNode {
	var out []RefNode
	walk(n, func(x Node) {
		if r, ok := x.(RefNode); ok {
			out = append(out, r)
		}
	})
	return out
}

func walk(n Node, fn func(Node)) {
	if n == nil {
		return
	}
	fn(n)
	switch x := n.(type) {
	case UnaryNode:
		walk(x.X, fn)
	case BinaryNode:
		walk(x.L, fn)
		walk(x.R, fn)
	case CallNode:
		for _, a := range x.Args {
			walk(a, fn)
		}
	case ArrayNode:
		for _, row := range x.Rows {
			for _, c := range row {
				walk(c, fn)
			}
		}
	}
}
