// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"math"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
)

// Context supplies cell values and names to the evaluator.
type Context interface {
	// CellValue returns the value of a cell; sheet "" means the current
	// sheet. ok is false for unknown sheets.
	CellValue(sheet string, row, col int) (Value, bool)
	// Name resolves a defined name to its formula text for the sheet in
	// scope (sheet-local first, then workbook).
	Name(name, sheet string) (string, bool)
	// Table resolves a structured reference to a range on a sheet.
	Table(text, sheet string) (Value, bool)
	// Date1904 reports the workbook date system.
	Date1904() bool
	// SheetNames lists sheets in order (for 3-D references).
	SheetNames() []string
}

// Evaluator evaluates parsed formulas against a Context.
type Evaluator struct {
	ctx   Context
	sheet string
	cell  ref.Cell
	depth int
	// Funcs can override or add functions; keys are upper-case.
	Funcs map[string]Func
}

// Func implements a worksheet function. Arguments are unevaluated nodes so
// that lazy functions (IF, IFERROR) can skip branches; use e.Arg to
// evaluate one.
type Func func(e *Evaluator, args []Node) Value

// NewEvaluator creates an evaluator for the given context.
func NewEvaluator(ctx Context) *Evaluator {
	return &Evaluator{ctx: ctx}
}

// Eval evaluates formula text in the scope of a cell on a sheet.
func (e *Evaluator) Eval(text, sheet string, cell ref.Cell) Value {
	n, err := Parse(text)
	if err != nil {
		return errName
	}
	return e.EvalNode(n, sheet, cell)
}

// EvalNode evaluates a parsed expression in scope.
func (e *Evaluator) EvalNode(n Node, sheet string, cell ref.Cell) Value {
	prevSheet, prevCell := e.sheet, e.cell
	e.sheet, e.cell = sheet, cell
	v := e.deref(e.eval(n))
	e.sheet, e.cell = prevSheet, prevCell
	return v
}

// Arg evaluates argument i (scalar-dereferenced). Missing arguments yield Empty.
func (e *Evaluator) Arg(args []Node, i int) Value {
	if i >= len(args) {
		return Value{}
	}
	if _, ok := args[i].(EmptyNode); ok {
		return Value{}
	}
	return e.deref(e.eval(args[i]))
}

// ArgRaw evaluates argument i keeping references intact.
func (e *Evaluator) ArgRaw(args []Node, i int) Value {
	if i >= len(args) {
		return Value{}
	}
	if _, ok := args[i].(EmptyNode); ok {
		return Value{}
	}
	return e.eval(args[i])
}

// deref turns a single-cell reference into its value and a multi-cell
// reference into an array.
func (e *Evaluator) deref(v Value) Value {
	if v.Kind != Reference {
		return v
	}
	if v.Range.IsCell() {
		return e.cellValue(v.Sheet, v.Range.From.Row, v.Range.From.Col)
	}
	return e.toArray(v)
}

// toArray materialises a reference as an array value.
func (e *Evaluator) toArray(v Value) Value {
	if v.Kind != Reference {
		return v
	}
	r := v.Range
	rows := r.Rows()
	cols := r.Cols()
	if rows*cols > 10_000_000 {
		return errNum
	}
	out := make([][]Value, rows)
	for i := 0; i < rows; i++ {
		out[i] = make([]Value, cols)
		for j := 0; j < cols; j++ {
			out[i][j] = e.cellValue(v.Sheet, r.From.Row+i, r.From.Col+j)
		}
	}
	return Arr(out)
}

func (e *Evaluator) cellValue(sheet string, row, col int) Value {
	if sheet == "" {
		sheet = e.sheet
	}
	if row < 1 || col < 1 || row > ref.MaxRows || col > ref.MaxCols {
		return errRef
	}
	v, ok := e.ctx.CellValue(sheet, row, col)
	if !ok {
		return errRef
	}
	return v
}

const maxDepth = 256

func (e *Evaluator) eval(n Node) Value {
	e.depth++
	defer func() { e.depth-- }()
	if e.depth > maxDepth {
		return errCalc
	}
	switch x := n.(type) {
	case NumberNode:
		return Num(x.Value)
	case StringNode:
		return Str(x.Value)
	case BoolNode:
		return Boolean(x.Value)
	case ErrorNode:
		return Err(x.Code)
	case EmptyNode:
		return Value{}
	case RefNode:
		if x.External != "" {
			return errRef
		}
		sheet := x.Sheet
		if i := strings.IndexByte(sheet, ':'); i > 0 {
			// 3-D reference: evaluate as a stacked array of each sheet's range.
			return e.threeD(sheet[:i], sheet[i+1:], x.Range)
		}
		r := x.Range
		if r.WholeCols {
			r.From.Row, r.To.Row = 1, ref.MaxRows
		}
		if r.WholeRows {
			r.From.Col, r.To.Col = 1, ref.MaxCols
		}
		return Ref(sheet, r)
	case NameNode:
		text, ok := e.ctx.Name(x.Name, e.sheet)
		if !ok {
			return errName
		}
		sub, err := Parse(text)
		if err != nil {
			return errName
		}
		return e.eval(sub)
	case StructuredNode:
		if v, ok := e.ctx.Table(x.Text, e.sheet); ok {
			return v
		}
		return errRef
	case ArrayNode:
		rows := make([][]Value, len(x.Rows))
		for i, row := range x.Rows {
			rows[i] = make([]Value, len(row))
			for j, c := range row {
				rows[i][j] = e.deref(e.eval(c))
			}
		}
		return Arr(rows)
	case UnaryNode:
		return e.unary(x)
	case BinaryNode:
		return e.binary(x)
	case CallNode:
		return e.call(x)
	case valueNode:
		return x.v
	}
	return errValue
}

func (e *Evaluator) threeD(from, to string, r ref.Range) Value {
	names := e.ctx.SheetNames()
	start, end := -1, -1
	for i, n := range names {
		if strings.EqualFold(n, from) {
			start = i
		}
		if strings.EqualFold(n, to) {
			end = i
		}
	}
	if start < 0 || end < 0 {
		return errRef
	}
	if start > end {
		start, end = end, start
	}
	var rows [][]Value
	for i := start; i <= end; i++ {
		a := e.toArray(Ref(names[i], r))
		rows = append(rows, a.Rows...)
	}
	return Arr(rows)
}

func (e *Evaluator) unary(x UnaryNode) Value {
	switch x.Op {
	case "-":
		return e.mapScalar(e.deref(e.eval(x.X)), func(v Value) Value {
			f, err := toNumber(v)
			if err.Kind == Error {
				return err
			}
			return Num(-f)
		})
	case "%":
		return e.mapScalar(e.deref(e.eval(x.X)), func(v Value) Value {
			f, err := toNumber(v)
			if err.Kind == Error {
				return err
			}
			return Num(f / 100)
		})
	case "@":
		// Implicit intersection: a single value from a range in the
		// formula's row or column.
		v := e.eval(x.X)
		if v.Kind == Reference {
			return e.implicitIntersect(v)
		}
		return e.deref(v).At(0, 0)
	case "#":
		// Spill reference: without a spill tracker, use the anchor's value.
		return e.deref(e.eval(x.X))
	}
	return errValue
}

func (e *Evaluator) implicitIntersect(v Value) Value {
	r := v.Range
	switch {
	case r.IsCell():
		return e.cellValue(v.Sheet, r.From.Row, r.From.Col)
	case r.From.Col == r.To.Col && e.cell.Row >= r.From.Row && e.cell.Row <= r.To.Row:
		return e.cellValue(v.Sheet, e.cell.Row, r.From.Col)
	case r.From.Row == r.To.Row && e.cell.Col >= r.From.Col && e.cell.Col <= r.To.Col:
		return e.cellValue(v.Sheet, r.From.Row, e.cell.Col)
	}
	return errValue
}

// mapScalar applies fn to a scalar or element-wise to an array.
func (e *Evaluator) mapScalar(v Value, fn func(Value) Value) Value {
	if v.Kind == Reference {
		v = e.deref(v)
	}
	if v.Kind != Array {
		return fn(v)
	}
	out := make([][]Value, len(v.Rows))
	for i, row := range v.Rows {
		out[i] = make([]Value, len(row))
		for j, c := range row {
			out[i][j] = fn(c)
		}
	}
	return Arr(out)
}

// zip applies fn element-wise with broadcasting of 1xN, Nx1 and scalars.
func (e *Evaluator) zip(a, b Value, fn func(a, b Value) Value) Value {
	a, b = e.deref(a), e.deref(b)
	if a.Kind != Array && b.Kind != Array {
		return fn(a, b)
	}
	ar, ac := a.Dims()
	br, bc := b.Dims()
	rows, cols := max(ar, br), max(ac, bc)
	if (ar != 1 && br != 1 && ar != br) || (ac != 1 && bc != 1 && ac != bc) {
		// Mismatched: Excel fills with #N/A outside the overlap.
	}
	out := make([][]Value, rows)
	for i := 0; i < rows; i++ {
		out[i] = make([]Value, cols)
		for j := 0; j < cols; j++ {
			ai, aj := i, j
			if ar == 1 {
				ai = 0
			}
			if ac == 1 {
				aj = 0
			}
			bi, bj := i, j
			if br == 1 {
				bi = 0
			}
			if bc == 1 {
				bj = 0
			}
			if ai >= ar || aj >= ac || bi >= br || bj >= bc {
				out[i][j] = errNA
				continue
			}
			out[i][j] = fn(a.At(ai, aj), b.At(bi, bj))
		}
	}
	return Arr(out)
}

func (e *Evaluator) binary(x BinaryNode) Value {
	if x.Op == ":" {
		l, r := e.eval(x.L), e.eval(x.R)
		if l.Kind != Reference || r.Kind != Reference {
			return errRef
		}
		return Ref(l.Sheet, l.Range.Union(r.Range))
	}
	l, r := e.eval(x.L), e.eval(x.R)
	// Single-cell references in arithmetic resolve to scalars; ranges use
	// implicit intersection in legacy formulas but we treat them as arrays.
	if l.Kind == Reference && l.Range.IsCell() {
		l = e.deref(l)
	}
	if r.Kind == Reference && r.Range.IsCell() {
		r = e.deref(r)
	}
	return e.zip(l, r, func(a, b Value) Value { return binaryScalar(x.Op, a, b) })
}

func binaryScalar(op string, a, b Value) Value {
	if a.Kind == Error {
		return a
	}
	if b.Kind == Error {
		return b
	}
	switch op {
	case "&":
		return Str(toText(a) + toText(b))
	case "=", "<>", "<", ">", "<=", ">=":
		c := compare(a, b)
		var res bool
		switch op {
		case "=":
			res = c == 0
		case "<>":
			res = c != 0
		case "<":
			res = c < 0
		case ">":
			res = c > 0
		case "<=":
			res = c <= 0
		case ">=":
			res = c >= 0
		}
		return Boolean(res)
	}
	x, err := toNumber(a)
	if err.Kind == Error {
		return err
	}
	y, err := toNumber(b)
	if err.Kind == Error {
		return err
	}
	switch op {
	case "+":
		return Num(x + y)
	case "-":
		return Num(x - y)
	case "*":
		return Num(x * y)
	case "/":
		if y == 0 {
			return errDiv0
		}
		return Num(x / y)
	case "^":
		if x == 0 && y < 0 {
			return errDiv0
		}
		r := math.Pow(x, y)
		if math.IsNaN(r) {
			return errNum
		}
		return Num(r)
	}
	return errValue
}

func (e *Evaluator) call(x CallNode) (result Value) {
	// A malformed argument list must never take the caller down; surface
	// it as #VALUE! the way Excel reports an unusable argument.
	defer func() {
		if r := recover(); r != nil {
			result = errValue
		}
	}()
	if e.Funcs != nil {
		if f, ok := e.Funcs[x.Name]; ok {
			return f(e, x.Args)
		}
	}
	f, ok := builtins[x.Name]
	if !ok {
		return errName
	}
	return f(e, x.Args)
}

// Functions lists the names of all built-in functions.
func Functions() []string {
	out := make([]string, 0, len(builtins))
	for k := range builtins {
		out = append(out, k)
	}
	return out
}

// numbersOf flattens arguments into numbers the way SUM does: numbers in
// references are used, text and booleans in references are skipped, direct
// text arguments are converted, errors propagate.
func (e *Evaluator) numbersOf(args []Node) ([]float64, Value) {
	var out []float64
	for _, a := range args {
		if _, ok := a.(EmptyNode); ok {
			continue
		}
		v := e.eval(a)
		fromRef := v.Kind == Reference || v.Kind == Array
		v = e.deref(v)
		var err Value
		each(v, func(x Value) bool {
			switch x.Kind {
			case Number:
				out = append(out, x.Num)
			case Bool:
				if !fromRef {
					if x.Bool {
						out = append(out, 1)
					} else {
						out = append(out, 0)
					}
				}
			case Text:
				if !fromRef {
					f, ok := parseNumber(x.Str)
					if !ok {
						err = errValue
						return false
					}
					out = append(out, f)
				}
			case Error:
				err = x
				return false
			}
			return true
		})
		if err.Kind == Error {
			return nil, err
		}
	}
	return out, Value{}
}

// each visits scalars of a value (arrays element-wise).
func each(v Value, fn func(Value) bool) {
	if v.Kind != Array {
		fn(v)
		return
	}
	for _, row := range v.Rows {
		for _, c := range row {
			if !fn(c) {
				return
			}
		}
	}
}

// num evaluates a scalar numeric argument.
func (e *Evaluator) num(args []Node, i int) (float64, Value) {
	v := e.Arg(args, i)
	if v.Kind == Array {
		v = v.At(0, 0)
	}
	return toNumber(v)
}

func (e *Evaluator) numDefault(args []Node, i int, def float64) (float64, Value) {
	if i >= len(args) {
		return def, Value{}
	}
	if _, ok := args[i].(EmptyNode); ok {
		return def, Value{}
	}
	return e.num(args, i)
}

func (e *Evaluator) text(args []Node, i int) (string, Value) {
	v := e.Arg(args, i)
	if v.Kind == Error {
		return "", v
	}
	return toText(v), Value{}
}

func (e *Evaluator) boolean(args []Node, i int) (bool, Value) {
	return toBool(e.Arg(args, i))
}

// rangeOf evaluates an argument that must be a reference or array and
// returns it as an array plus the reference (if any).
func (e *Evaluator) rangeOf(args []Node, i int) (Value, Value) {
	raw := e.ArgRaw(args, i)
	if raw.Kind == Error {
		return raw, raw
	}
	if raw.Kind == Reference {
		return e.toArray(raw), raw
	}
	if raw.Kind != Array {
		return Arr([][]Value{{raw}}), Value{}
	}
	return raw, Value{}
}

// modelDate1904 exposes the date system for helpers.
func (e *Evaluator) date1904() bool { return e.ctx.Date1904() }

// Compile-time check that model errors are the codes we emit.
var _ = model.ErrNA
