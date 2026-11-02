// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"errors"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
)

// ErrCycle is reported when formulas depend on each other.
var ErrCycle = errors.New("formula: circular reference")

// Calculator evaluates the formulas of a workbook and stores the results
// as cached cell values, so that files are shown correctly by viewers that
// do not recalculate.
type Calculator struct {
	wb    *model.Workbook
	eval  *Evaluator
	state map[cellKey]uint8 // 1 = in progress, 2 = done
	cycle bool
	// MaxCells caps the number of cells a single range may expand to.
	MaxCells int
}

type cellKey struct {
	sheet    int
	row, col int
}

// NewCalculator prepares a calculator for the workbook.
func NewCalculator(wb *model.Workbook) *Calculator {
	c := &Calculator{wb: wb, state: map[cellKey]uint8{}}
	c.eval = NewEvaluator(c)
	return c
}

// Calculate evaluates every formula cell in the workbook. It returns
// ErrCycle (after storing #CALC! in the affected cells) if a circular
// dependency is found; other errors stay in the cells as Excel error
// codes.
func Calculate(wb *model.Workbook) error {
	return NewCalculator(wb).Run()
}

// Run calculates every formula cell.
func (c *Calculator) Run() error {
	c.state = map[cellKey]uint8{}
	c.cycle = false
	for si, sh := range c.wb.Sheets {
		if sh.Kind != model.Worksheet {
			continue
		}
		for _, row := range sh.SortedRows() {
			for _, cell := range row.SortedCells() {
				if cell.Formula != nil {
					c.cellValue(si, cell.Row, cell.Col)
				}
			}
		}
	}
	if c.cycle {
		return ErrCycle
	}
	return nil
}

// Cell evaluates one cell's formula (if any) and returns its value.
func (c *Calculator) Cell(sheet string, a1 string) (Value, error) {
	si := c.wb.SheetIndex(sheet)
	if si < 0 {
		return errRef, nil
	}
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return errRef, err
	}
	v := c.cellValue(si, cr.Row, cr.Col)
	if c.cycle {
		return v, ErrCycle
	}
	return v, nil
}

// Evaluate computes an ad-hoc formula in the scope of a sheet without
// storing anything.
func (c *Calculator) Evaluate(sheet, formula string) Value {
	return c.eval.Eval(formula, sheet, ref.Cell{Row: 1, Col: 1})
}

func (c *Calculator) cellValue(si, row, col int) Value {
	sh := c.wb.Sheets[si]
	cell := sh.Lookup(row, col)
	if cell == nil {
		return Value{}
	}
	if cell.Formula == nil || cell.Formula.Text == "" {
		return FromModel(cell.Value)
	}
	key := cellKey{si, row, col}
	switch c.state[key] {
	case 2:
		return FromModel(cell.Value)
	case 1:
		c.cycle = true
		return errCalc
	}
	c.state[key] = 1
	f := cell.Formula
	text := f.Text
	if f.Kind == model.FormulaShared && !f.Master && text == "" {
		// Unexpanded shared formula without master text: nothing to do.
		c.state[key] = 2
		return FromModel(cell.Value)
	}
	v := c.eval.Eval(text, sh.Name, ref.Cell{Row: row, Col: col})
	legacyArray := f.Kind == model.FormulaArray && f.Master && !f.Dynamic && f.Ref.Valid() && !f.Ref.IsCell()
	if v.Kind == Array && legacyArray {
		c.spill(sh, f.Ref, v)
	} else if v.Kind == Array {
		// Dynamic arrays spill into empty neighbours.
		rows, cols := v.Dims()
		r := ref.Range{From: ref.Cell{Row: row, Col: col}, To: ref.Cell{Row: row + rows - 1, Col: col + cols - 1}}
		if r.Valid() && c.canSpill(sh, r, row, col) {
			c.spill(sh, r, v)
			if f.Dynamic || f.Kind == model.FormulaArray {
				f.Ref = r
			}
		} else if f.Dynamic {
			v = Err(model.ErrSpill)
		} else {
			v = v.At(0, 0)
		}
	}
	cell.Value = v.ToModel()
	c.state[key] = 2
	return v
}

func (c *Calculator) canSpill(sh *model.Sheet, r ref.Range, row, col int) bool {
	ok := true
	r.Each(func(rr, cc int) bool {
		if rr == row && cc == col {
			return true
		}
		if x := sh.Lookup(rr, cc); x != nil && (x.Formula != nil && !(x.Formula.Kind == model.FormulaArray && !x.Formula.Master) || !x.Value.IsEmpty() && x.Formula == nil) {
			ok = false
			return false
		}
		return true
	})
	return ok
}

func (c *Calculator) spill(sh *model.Sheet, r ref.Range, v Value) {
	rows, cols := v.Dims()
	for i := 0; i < r.Rows(); i++ {
		for j := 0; j < r.Cols(); j++ {
			rr, cc := r.From.Row+i, r.From.Col+j
			if rr == r.From.Row && cc == r.From.Col {
				continue
			}
			cell := sh.Cell(rr, cc)
			var ev Value
			switch {
			case i < rows && j < cols:
				ev = v.At(i, j)
			case rows == 1 && j < cols:
				ev = v.At(0, j)
			case cols == 1 && i < rows:
				ev = v.At(i, 0)
			default:
				ev = errNA
			}
			cell.Value = ev.ToModel()
			c.state[cellKey{c.wb.SheetIndex(sh.Name), rr, cc}] = 2
		}
	}
}

// Context implementation.

// CellValue implements Context.
func (c *Calculator) CellValue(sheet string, row, col int) (Value, bool) {
	si := c.wb.SheetIndex(sheet)
	if si < 0 {
		return errRef, false
	}
	return c.cellValue(si, row, col), true
}

// Name implements Context.
func (c *Calculator) Name(name, sheet string) (string, bool) {
	si := c.wb.SheetIndex(sheet)
	if si >= 0 {
		if dn := c.wb.LocalDefinedName(name, si); dn != nil {
			return dn.RefersTo, true
		}
	}
	if dn := c.wb.DefinedName(name); dn != nil {
		return dn.RefersTo, true
	}
	return "", false
}

// Table implements Context: Table[Col], Table[[#Headers],[Col]],
// Table[#All], Table[#Data], Table[#Totals], Table[@Col], Table[]
func (c *Calculator) Table(text, sheet string) (Value, bool) {
	i := strings.IndexByte(text, '[')
	if i < 0 {
		return Value{}, false
	}
	name := text[:i]
	spec := text[i:]
	var t *model.Table
	var tsheet *model.Sheet
	for _, sh := range c.wb.Sheets {
		for _, tb := range sh.Tables {
			if strings.EqualFold(tb.DisplayName, name) || strings.EqualFold(tb.Name, name) {
				t, tsheet = tb, sh
			}
		}
	}
	if t == nil {
		return Value{}, false
	}
	headerRows := 1
	if t.HeaderRow != nil && !*t.HeaderRow {
		headerRows = 0
	}
	totals := 0
	if t.TotalsRow {
		totals = max(1, t.TotalsRowCount)
	}
	data := t.Range
	data.From.Row += headerRows
	data.To.Row -= totals
	rows := data
	thisRow := false
	var cols []string
	// Parse specifiers.
	spec = strings.TrimSuffix(strings.TrimPrefix(spec, "["), "]")
	for _, part := range splitSpec(spec) {
		part = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(part, "["), "]"))
		switch strings.ToLower(part) {
		case "#all":
			rows = t.Range
		case "#data", "":
			rows = data
		case "#headers":
			rows = t.Range
			rows.To.Row = rows.From.Row + headerRows - 1
		case "#totals":
			rows = t.Range
			rows.From.Row = rows.To.Row - totals + 1
		case "#this row", "@":
			thisRow = true
		default:
			if strings.HasPrefix(part, "@") {
				thisRow = true
				part = strings.TrimPrefix(part, "@")
			}
			cols = append(cols, strings.ReplaceAll(part, "'", ""))
		}
	}
	colIdx := func(name string) int {
		for k, col := range t.Columns {
			if strings.EqualFold(col.Name, name) {
				return t.Range.From.Col + k
			}
		}
		return -1
	}
	r := rows
	if len(cols) > 0 {
		c1 := colIdx(cols[0])
		c2 := colIdx(cols[len(cols)-1])
		if c1 < 0 || c2 < 0 {
			return errRef, true
		}
		r.From.Col, r.To.Col = min(c1, c2), max(c1, c2)
	}
	if thisRow {
		row := c.eval.cell.Row
		if row < data.From.Row || row > data.To.Row {
			return errValue, true
		}
		r.From.Row, r.To.Row = row, row
	}
	return Ref(tsheet.Name, r), true
}

func splitSpec(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// Date1904 implements Context.
func (c *Calculator) Date1904() bool { return c.wb.Date1904 }

// SheetNames implements Context.
func (c *Calculator) SheetNames() []string {
	out := make([]string, len(c.wb.Sheets))
	for i, s := range c.wb.Sheets {
		out[i] = s.Name
	}
	return out
}
