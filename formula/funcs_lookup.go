// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"sort"
	"strings"

	"github.com/gsoultan/gexcel/ref"
)

func init() {
	register(map[string]Func{
		"VLOOKUP":    fnVLookup,
		"HLOOKUP":    fnHLookup,
		"XLOOKUP":    fnXLookup,
		"LOOKUP":     fnLookup,
		"INDEX":      fnIndex,
		"MATCH":      fnMatch,
		"XMATCH":     fnXMatch,
		"OFFSET":     fnOffset,
		"INDIRECT":   fnIndirect,
		"ROW":        fnRowCol(true),
		"COLUMN":     fnRowCol(false),
		"ROWS":       fnRowsCols(true),
		"COLUMNS":    fnRowsCols(false),
		"TRANSPOSE":  fnTranspose,
		"UNIQUE":     fnUnique,
		"SORT":       fnSort,
		"SORTBY":     fnSortBy,
		"FILTER":     fnFilter,
		"SEQUENCE":   fnSequence,
		"ADDRESS":    fnAddress,
		"CHOOSECOLS": fnChooseRC(true),
		"CHOOSEROWS": fnChooseRC(false),
		"TAKE":       fnTake,
		"DROP":       fnDrop,
		"HSTACK":     fnStack(true),
		"VSTACK":     fnStack(false),
		"TOCOL":      fnToColRow(true),
		"TOROW":      fnToColRow(false),
	})
}

// matchIn finds a value in a 1-D list. mode: 0 exact, 1 ascending
// (largest <= key), -1 descending (smallest >= key). wildcard allows
// * and ? in text keys for exact matches.
func matchIn(key Value, list []Value, mode int, wildcard bool) int {
	switch mode {
	case 0:
		for i, v := range list {
			if key.Kind == Text && v.Kind == Text {
				if wildcard && strings.ContainsAny(key.Str, "*?~") {
					if wildcardMatch(strings.ToLower(key.Str), strings.ToLower(v.Str)) {
						return i
					}
					continue
				}
				if strings.EqualFold(key.Str, v.Str) {
					return i
				}
				continue
			}
			if key.Kind == v.Kind && compare(key, v) == 0 {
				return i
			}
		}
		return -1
	case 1:
		// Binary search for largest value <= key among same-type entries.
		best := -1
		lo, hi := 0, len(list)-1
		for lo <= hi {
			mid := (lo + hi) / 2
			v := list[mid]
			if v.Kind != key.Kind && !(v.Kind == Empty) {
				// Different types: treat by rank order via compare.
			}
			c := compare(v, key)
			if c <= 0 {
				if v.Kind == key.Kind || v.Kind == Number && key.Kind == Number {
					best = mid
				}
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
		if best == -1 {
			// Linear fallback for unsorted data (Excel's result is undefined; be helpful).
			for i, v := range list {
				if v.Kind == key.Kind && compare(v, key) <= 0 {
					best = i
				}
			}
		}
		return best
	case -1:
		best := -1
		for i, v := range list {
			if v.Kind == key.Kind && compare(v, key) >= 0 {
				best = i
			} else if v.Kind == key.Kind && compare(v, key) < 0 {
				break
			}
		}
		return best
	}
	return -1
}

func column(a Value, c int) []Value {
	r, _ := a.Dims()
	out := make([]Value, r)
	for i := 0; i < r; i++ {
		out[i] = a.At(i, c)
	}
	return out
}

func rowOf(a Value, r int) []Value {
	_, c := a.Dims()
	out := make([]Value, c)
	for j := 0; j < c; j++ {
		out[j] = a.At(r, j)
	}
	return out
}

func fnVLookup(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	table, _ := e.rangeOf(args, 1)
	if table.Kind == Error {
		return table
	}
	col, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	approx := true
	if len(args) > 3 {
		approx, err = e.boolean(args, 3)
		if err.Kind == Error {
			return err
		}
	}
	_, cols := table.Dims()
	if int(col) < 1 || int(col) > cols {
		return errRef
	}
	mode := 0
	if approx {
		mode = 1
	}
	i := matchIn(key, column(table, 0), mode, true)
	if i < 0 {
		return errNA
	}
	return table.At(i, int(col)-1)
}

func fnHLookup(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	table, _ := e.rangeOf(args, 1)
	if table.Kind == Error {
		return table
	}
	row, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	approx := true
	if len(args) > 3 {
		approx, err = e.boolean(args, 3)
		if err.Kind == Error {
			return err
		}
	}
	rows, _ := table.Dims()
	if int(row) < 1 || int(row) > rows {
		return errRef
	}
	mode := 0
	if approx {
		mode = 1
	}
	i := matchIn(key, rowOf(table, 0), mode, true)
	if i < 0 {
		return errNA
	}
	return table.At(int(row)-1, i)
}

func flatten(a Value) []Value {
	var out []Value
	each(a, func(v Value) bool { out = append(out, v); return true })
	return out
}

func fnXLookup(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	lookup, _ := e.rangeOf(args, 1)
	if lookup.Kind == Error {
		return lookup
	}
	ret, _ := e.rangeOf(args, 2)
	if ret.Kind == Error {
		return ret
	}
	matchMode, _ := e.numDefault(args, 4, 0)
	searchMode, _ := e.numDefault(args, 5, 1)
	list := flatten(lookup)
	if searchMode < 0 {
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}
	var i int
	switch int(matchMode) {
	case 0:
		i = matchIn(key, list, 0, false)
	case 2:
		i = matchIn(key, list, 0, true)
	case -1, 1:
		i = matchIn(key, list, 0, false)
		if i < 0 {
			// next smaller (-1) or next larger (1): scan all
			best := -1
			for k, v := range list {
				if v.Kind != key.Kind {
					continue
				}
				c := compare(v, key)
				if matchMode == -1 && c < 0 && (best < 0 || compare(v, list[best]) > 0) {
					best = k
				}
				if matchMode == 1 && c > 0 && (best < 0 || compare(v, list[best]) < 0) {
					best = k
				}
			}
			i = best
		}
	}
	if i < 0 {
		if len(args) > 3 {
			if v := e.Arg(args, 3); v.Kind != Empty {
				return v
			}
		}
		return errNA
	}
	if searchMode < 0 {
		i = len(list) - 1 - i
	}
	lr, _ := lookup.Dims()
	rr, rc := ret.Dims()
	if lr > 1 && lr == rr {
		// vertical lookup: return the whole row of ret
		if rc == 1 {
			return ret.At(i, 0)
		}
		return Arr([][]Value{rowOf(ret, i)})
	}
	if rr == 1 {
		return ret.At(0, i)
	}
	// horizontal lookup returning a column
	col := column(ret, i)
	rows := make([][]Value, len(col))
	for k, v := range col {
		rows[k] = []Value{v}
	}
	return Arr(rows)
}

func fnLookup(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	vec, _ := e.rangeOf(args, 1)
	if vec.Kind == Error {
		return vec
	}
	res := vec
	if len(args) > 2 {
		res, _ = e.rangeOf(args, 2)
		if res.Kind == Error {
			return res
		}
	} else {
		r, c := vec.Dims()
		if c > r {
			// array form: last row
			res = Arr([][]Value{rowOf(vec, r-1)})
			vec = Arr([][]Value{rowOf(vec, 0)})
		} else if c > 1 {
			res = Arr(transposeRows([][]Value{column(vec, c-1)}))
			vec = Arr(transposeRows([][]Value{column(vec, 0)}))
		}
	}
	i := matchIn(key, flatten(vec), 1, false)
	if i < 0 {
		return errNA
	}
	list := flatten(res)
	if i >= len(list) {
		return errNA
	}
	return list[i]
}

func transposeRows(rows [][]Value) [][]Value {
	if len(rows) == 0 {
		return nil
	}
	out := make([][]Value, len(rows[0]))
	for j := range rows[0] {
		out[j] = make([]Value, len(rows))
		for i := range rows {
			out[j][i] = rows[i][j]
		}
	}
	return out
}

func fnIndex(e *Evaluator, args []Node) Value {
	arr, rf := e.rangeOf(args, 0)
	if arr.Kind == Error {
		return arr
	}
	r, err := e.numDefault(args, 1, 0)
	if err.Kind == Error {
		return err
	}
	c, err := e.numDefault(args, 2, 0)
	if err.Kind == Error {
		return err
	}
	rows, cols := arr.Dims()
	ri, ci := int(r), int(c)
	if rows == 1 && len(args) == 2 {
		// single row: INDEX(row, n) picks a column
		ri, ci = 1, int(r)
	}
	if ri < 0 || ci < 0 || ri > rows || ci > cols {
		return errRef
	}
	// Return a reference when possible so INDEX(...):INDEX(...) works.
	mkRef := func(r1, c1, r2, c2 int) Value {
		if rf.Kind == Reference {
			base := rf.Range.From
			return Ref(rf.Sheet, ref.Range{From: ref.Cell{Row: base.Row + r1, Col: base.Col + c1}, To: ref.Cell{Row: base.Row + r2, Col: base.Col + c2}})
		}
		return Value{}
	}
	switch {
	case ri == 0 && ci == 0:
		return arr
	case ri == 0:
		if v := mkRef(0, ci-1, rows-1, ci-1); v.Kind == Reference {
			return v
		}
		return Arr(transposeRows([][]Value{column(arr, ci-1)}))
	case ci == 0:
		if cols == 1 {
			if v := mkRef(ri-1, 0, ri-1, 0); v.Kind == Reference {
				return v
			}
			return arr.At(ri-1, 0)
		}
		if v := mkRef(ri-1, 0, ri-1, cols-1); v.Kind == Reference {
			return v
		}
		return Arr([][]Value{rowOf(arr, ri-1)})
	}
	if v := mkRef(ri-1, ci-1, ri-1, ci-1); v.Kind == Reference {
		return v
	}
	return arr.At(ri-1, ci-1)
}

func fnMatch(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	arr, _ := e.rangeOf(args, 1)
	if arr.Kind == Error {
		return arr
	}
	mode, err := e.numDefault(args, 2, 1)
	if err.Kind == Error {
		return err
	}
	i := matchIn(key, flatten(arr), int(mode), true)
	if i < 0 {
		return errNA
	}
	return Num(float64(i + 1))
}

func fnXMatch(e *Evaluator, args []Node) Value {
	key := e.Arg(args, 0)
	if key.Kind == Error {
		return key
	}
	arr, _ := e.rangeOf(args, 1)
	if arr.Kind == Error {
		return arr
	}
	matchMode, _ := e.numDefault(args, 2, 0)
	list := flatten(arr)
	var i int
	switch int(matchMode) {
	case 2:
		i = matchIn(key, list, 0, true)
	case -1:
		i = matchIn(key, list, 0, false)
		if i < 0 {
			i = matchIn(key, list, 1, false)
		}
	case 1:
		i = matchIn(key, list, 0, false)
		if i < 0 {
			i = matchIn(key, list, -1, false)
		}
	default:
		i = matchIn(key, list, 0, false)
	}
	if i < 0 {
		return errNA
	}
	return Num(float64(i + 1))
}

func fnOffset(e *Evaluator, args []Node) Value {
	base := e.ArgRaw(args, 0)
	if base.Kind != Reference {
		return errValue
	}
	rows, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	cols, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	h, err := e.numDefault(args, 3, float64(base.Range.Rows()))
	if err.Kind == Error {
		return err
	}
	w, err := e.numDefault(args, 4, float64(base.Range.Cols()))
	if err.Kind == Error {
		return err
	}
	if h <= 0 || w <= 0 {
		return errRef
	}
	from := base.Range.From.Offset(int(rows), int(cols))
	to := from.Offset(int(h)-1, int(w)-1)
	r := ref.Range{From: from, To: to}
	if !r.Valid() {
		return errRef
	}
	return Ref(base.Sheet, r)
}

func fnIndirect(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	a1 := true
	if len(args) > 1 {
		a1, _ = e.boolean(args, 1)
	}
	if !a1 {
		cr, perr := ref.ParseR1C1(s, e.cell)
		if perr != nil {
			return errRef
		}
		return Ref("", ref.CellRange(cr))
	}
	n, perr := parseRefNode(s)
	if perr != nil {
		// Maybe a defined name.
		if text, ok := e.ctx.Name(s, e.sheet); ok {
			return e.Eval(text, e.sheet, e.cell)
		}
		return errRef
	}
	rn := n.(RefNode)
	r := rn.Range
	if r.WholeCols {
		r.From.Row, r.To.Row = 1, ref.MaxRows
	}
	if r.WholeRows {
		r.From.Col, r.To.Col = 1, ref.MaxCols
	}
	return Ref(rn.Sheet, r)
}

func fnRowCol(isRow bool) Func {
	return func(e *Evaluator, args []Node) Value {
		if len(args) == 0 || isEmptyNode(args[0]) {
			if isRow {
				return Num(float64(e.cell.Row))
			}
			return Num(float64(e.cell.Col))
		}
		v := e.ArgRaw(args, 0)
		if v.Kind != Reference {
			return errValue
		}
		r := v.Range
		if r.IsCell() {
			if isRow {
				return Num(float64(r.From.Row))
			}
			return Num(float64(r.From.Col))
		}
		if isRow {
			rows := make([][]Value, r.Rows())
			for i := range rows {
				rows[i] = []Value{Num(float64(r.From.Row + i))}
			}
			return Arr(rows)
		}
		row := make([]Value, r.Cols())
		for j := range row {
			row[j] = Num(float64(r.From.Col + j))
		}
		return Arr([][]Value{row})
	}
}

func isEmptyNode(n Node) bool {
	_, ok := n.(EmptyNode)
	return ok
}

func fnRowsCols(isRows bool) Func {
	return func(e *Evaluator, args []Node) Value {
		v := e.ArgRaw(args, 0)
		if v.Kind == Error {
			return v
		}
		r, c := v.Dims()
		if isRows {
			return Num(float64(r))
		}
		return Num(float64(c))
	}
}

func fnTranspose(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	return Arr(transposeRows(a.Rows))
}

func valueKey(v Value) string {
	switch v.Kind {
	case Text:
		return "s:" + strings.ToLower(v.Str)
	case Number:
		return "n:" + formatNum(v.Num)
	case Bool:
		return "b:" + toText(v)
	case Error:
		return "e:" + v.Str
	}
	return ""
}

func fnUnique(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	byCol := false
	if len(args) > 1 {
		byCol, _ = e.boolean(args, 1)
	}
	once := false
	if len(args) > 2 {
		once, _ = e.boolean(args, 2)
	}
	rows := a.Rows
	if byCol {
		rows = transposeRows(rows)
	}
	counts := map[string]int{}
	keys := make([]string, len(rows))
	for i, r := range rows {
		var b strings.Builder
		for _, c := range r {
			b.WriteString(valueKey(c))
			b.WriteByte(0)
		}
		keys[i] = b.String()
		counts[keys[i]]++
	}
	seen := map[string]bool{}
	var out [][]Value
	for i, r := range rows {
		if once && counts[keys[i]] != 1 {
			continue
		}
		if seen[keys[i]] {
			continue
		}
		seen[keys[i]] = true
		out = append(out, r)
	}
	if byCol {
		out = transposeRows(out)
	}
	if len(out) == 0 {
		return errCalc
	}
	return Arr(out)
}

func sortRows(rows [][]Value, keys []int, desc []bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		for k, col := range keys {
			if col >= len(rows[i]) || col >= len(rows[j]) {
				continue
			}
			c := compare(rows[i][col], rows[j][col])
			// Empty cells sort last in Excel.
			if rows[i][col].Kind == Empty && rows[j][col].Kind != Empty {
				return false
			}
			if rows[j][col].Kind == Empty && rows[i][col].Kind != Empty {
				return true
			}
			if c == 0 {
				continue
			}
			if desc[k] {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

func fnSort(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	idx, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	order, err := e.numDefault(args, 2, 1)
	if err.Kind == Error {
		return err
	}
	byCol := false
	if len(args) > 3 {
		byCol, _ = e.boolean(args, 3)
	}
	rows := make([][]Value, len(a.Rows))
	copy(rows, a.Rows)
	if byCol {
		rows = transposeRows(rows)
	}
	if int(idx) < 1 || (len(rows) > 0 && int(idx) > len(rows[0])) {
		return errValue
	}
	sortRows(rows, []int{int(idx) - 1}, []bool{order < 0})
	if byCol {
		rows = transposeRows(rows)
	}
	return Arr(rows)
}

func fnSortBy(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	n := len(a.Rows)
	type rec struct {
		row  []Value
		keys []Value
	}
	recs := make([]rec, n)
	for i := range recs {
		recs[i].row = a.Rows[i]
	}
	var desc []bool
	for i := 1; i < len(args); i += 2 {
		by, _ := e.rangeOf(args, i)
		if by.Kind == Error {
			return by
		}
		list := flatten(by)
		if len(list) != n {
			return errValue
		}
		for k := range recs {
			recs[k].keys = append(recs[k].keys, list[k])
		}
		order, _ := e.numDefault(args, i+1, 1)
		desc = append(desc, order < 0)
	}
	sort.SliceStable(recs, func(i, j int) bool {
		for k := range recs[i].keys {
			c := compare(recs[i].keys[k], recs[j].keys[k])
			if c == 0 {
				continue
			}
			if desc[k] {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	out := make([][]Value, n)
	for i := range recs {
		out[i] = recs[i].row
	}
	return Arr(out)
}

func fnFilter(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	inc, _ := e.rangeOf(args, 1)
	if inc.Kind == Error {
		return inc
	}
	ir, ic := inc.Dims()
	ar, ac := a.Dims()
	var out [][]Value
	switch {
	case ic == 1 && ir == ar:
		for i := 0; i < ar; i++ {
			b, err := toBool(inc.At(i, 0))
			if err.Kind == Error {
				return err
			}
			if b {
				out = append(out, a.Rows[i])
			}
		}
	case ir == 1 && ic == ac:
		var cols []int
		for j := 0; j < ac; j++ {
			b, err := toBool(inc.At(0, j))
			if err.Kind == Error {
				return err
			}
			if b {
				cols = append(cols, j)
			}
		}
		if len(cols) > 0 {
			for i := 0; i < ar; i++ {
				row := make([]Value, len(cols))
				for k, j := range cols {
					row[k] = a.At(i, j)
				}
				out = append(out, row)
			}
		}
	default:
		return errValue
	}
	if len(out) == 0 {
		if len(args) > 2 {
			return e.Arg(args, 2)
		}
		return errCalc
	}
	return Arr(out)
}

func fnSequence(e *Evaluator, args []Node) Value {
	rows, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	cols, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	start, err := e.numDefault(args, 2, 1)
	if err.Kind == Error {
		return err
	}
	step, err := e.numDefault(args, 3, 1)
	if err.Kind == Error {
		return err
	}
	if rows < 1 || cols < 1 || rows*cols > 1_000_000 {
		return errValue
	}
	out := make([][]Value, int(rows))
	v := start
	for i := range out {
		out[i] = make([]Value, int(cols))
		for j := range out[i] {
			out[i][j] = Num(v)
			v += step
		}
	}
	return Arr(out)
}

func fnAddress(e *Evaluator, args []Node) Value {
	r, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	c, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	abs, err := e.numDefault(args, 2, 1)
	if err.Kind == Error {
		return err
	}
	a1 := true
	if len(args) > 3 {
		a1, _ = e.boolean(args, 3)
	}
	sheet := ""
	if len(args) > 4 {
		sheet, _ = e.text(args, 4)
	}
	if r < 1 || c < 1 {
		return errValue
	}
	cell := ref.Cell{Row: int(r), Col: int(c), AbsRow: abs == 1 || abs == 2, AbsCol: abs == 1 || abs == 3}
	var s string
	if a1 {
		s = cell.String()
	} else {
		s = "R" + itoa(int(r)) + "C" + itoa(int(c))
		if !cell.AbsRow {
			s = "R[" + itoa(int(r)) + "]C"
			if cell.AbsCol {
				s += itoa(int(c))
			} else {
				s += "[" + itoa(int(c)) + "]"
			}
		}
	}
	if sheet != "" {
		s = ref.QuoteSheet(sheet) + "!" + s
	}
	return Str(s)
}

func itoa(i int) string { return formatNum(float64(i)) }

func fnChooseRC(cols bool) Func {
	return func(e *Evaluator, args []Node) Value {
		a, _ := e.rangeOf(args, 0)
		if a.Kind == Error {
			return a
		}
		rows := a.Rows
		if cols {
			rows = transposeRows(rows)
		}
		var out [][]Value
		for i := 1; i < len(args); i++ {
			each(e.Arg(args, i), func(v Value) bool {
				k, _ := toNumber(v)
				idx := int(k)
				if idx < 0 {
					idx = len(rows) + idx + 1
				}
				if idx >= 1 && idx <= len(rows) {
					out = append(out, rows[idx-1])
				}
				return true
			})
		}
		if len(out) == 0 {
			return errValue
		}
		if cols {
			out = transposeRows(out)
		}
		return Arr(out)
	}
}

func fnTake(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	r, c := a.Dims()
	nr, err := e.numDefault(args, 1, float64(r))
	if err.Kind == Error {
		return err
	}
	nc, err := e.numDefault(args, 2, float64(c))
	if err.Kind == Error {
		return err
	}
	return Arr(slice2D(a.Rows, int(nr), int(nc), true))
}

func fnDrop(e *Evaluator, args []Node) Value {
	a, _ := e.rangeOf(args, 0)
	if a.Kind == Error {
		return a
	}
	nr, err := e.numDefault(args, 1, 0)
	if err.Kind == Error {
		return err
	}
	nc, err := e.numDefault(args, 2, 0)
	if err.Kind == Error {
		return err
	}
	return Arr(slice2D(a.Rows, int(nr), int(nc), false))
}

// slice2D takes (keep) or drops n rows/cols from the start (positive) or end (negative).
func slice2D(rows [][]Value, nr, nc int, take bool) [][]Value {
	r := len(rows)
	rowIdx := func() (int, int) {
		switch {
		case take && nr >= 0:
			return 0, min(nr, r)
		case take:
			return max(0, r+nr), r
		case nr >= 0:
			return min(nr, r), r
		}
		return 0, max(0, r+nr)
	}
	r0, r1 := rowIdx()
	var out [][]Value
	for _, row := range rows[r0:r1] {
		c := len(row)
		var c0, c1 int
		switch {
		case take && nc >= 0:
			c0, c1 = 0, min(nc, c)
		case take:
			c0, c1 = max(0, c+nc), c
		case nc >= 0:
			c0, c1 = min(nc, c), c
		default:
			c0, c1 = 0, max(0, c+nc)
		}
		out = append(out, row[c0:c1])
	}
	return out
}

func fnStack(horizontal bool) Func {
	return func(e *Evaluator, args []Node) Value {
		var out [][]Value
		for i := range args {
			a, _ := e.rangeOf(args, i)
			if a.Kind == Error {
				return a
			}
			if horizontal {
				for r, row := range a.Rows {
					for len(out) <= r {
						out = append(out, nil)
					}
					out[r] = append(out[r], row...)
				}
			} else {
				out = append(out, a.Rows...)
			}
		}
		width := 0
		for _, row := range out {
			width = max(width, len(row))
		}
		for i := range out {
			for len(out[i]) < width {
				out[i] = append(out[i], errNA)
			}
		}
		return Arr(out)
	}
}

func fnToColRow(toCol bool) Func {
	return func(e *Evaluator, args []Node) Value {
		a, _ := e.rangeOf(args, 0)
		if a.Kind == Error {
			return a
		}
		ignore, _ := e.numDefault(args, 1, 0)
		byCol := false
		if len(args) > 2 {
			byCol, _ = e.boolean(args, 2)
		}
		rows := a.Rows
		if byCol {
			rows = transposeRows(rows)
		}
		var list []Value
		for _, row := range rows {
			for _, v := range row {
				if (ignore == 1 || ignore == 3) && v.Kind == Empty {
					continue
				}
				if (ignore == 2 || ignore == 3) && v.Kind == Error {
					continue
				}
				list = append(list, v)
			}
		}
		if toCol {
			out := make([][]Value, len(list))
			for i, v := range list {
				out[i] = []Value{v}
			}
			return Arr(out)
		}
		return Arr([][]Value{list})
	}
}

func init() {
	register(map[string]Func{
		// Excel stores @range as _xlfn.SINGLE(range) and range# as _xlfn.ANCHORARRAY(range).
		"SINGLE":      func(e *Evaluator, args []Node) Value { return e.unary(UnaryNode{Op: "@", X: args[0]}) },
		"ANCHORARRAY": func(e *Evaluator, args []Node) Value { return e.Arg(args, 0) },
	})
}
