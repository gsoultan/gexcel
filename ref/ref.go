// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package ref parses and formats spreadsheet cell references: column
// letters, A1 and R1C1 cell addresses, ranges, and sheet-qualified
// references. Rows and columns are 1-based everywhere in gexcel.
package ref

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Excel sheet limits (Office 2007 and later, "BIFF12" grid).
const (
	MaxRows = 1_048_576
	MaxCols = 16_384
)

// ErrInvalid reports a reference that cannot be parsed.
var ErrInvalid = errors.New("ref: invalid reference")

// ColName returns the letters for a 1-based column number ("A", "Z", "AA").
// It returns "" for col < 1.
func ColName(col int) string {
	if col < 1 {
		return ""
	}
	var buf [8]byte
	i := len(buf)
	for col > 0 {
		col--
		i--
		buf[i] = byte('A' + col%26)
		col /= 26
	}
	return string(buf[i:])
}

// ColNumber returns the 1-based column number for column letters,
// case-insensitively. It returns 0 for an invalid name.
func ColNumber(name string) int {
	if name == "" || len(name) > 3 {
		return 0
	}
	n := 0
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z':
			n = n*26 + int(c-'A') + 1
		case c >= 'a' && c <= 'z':
			n = n*26 + int(c-'a') + 1
		default:
			return 0
		}
	}
	if n > MaxCols {
		return 0
	}
	return n
}

// Cell is a 1-based cell address with optional absolute markers.
type Cell struct {
	Row, Col       int
	AbsRow, AbsCol bool
}

// NewCell returns a relative cell address.
func NewCell(row, col int) Cell { return Cell{Row: row, Col: col} }

// Abs returns the same address with both markers set ($A$1).
func (c Cell) Abs() Cell { c.AbsRow, c.AbsCol = true, true; return c }

// Valid reports whether the address lies inside the Excel grid.
func (c Cell) Valid() bool {
	return c.Row >= 1 && c.Row <= MaxRows && c.Col >= 1 && c.Col <= MaxCols
}

// String formats the address in A1 notation, with $ for absolute parts.
func (c Cell) String() string {
	var b strings.Builder
	if c.AbsCol {
		b.WriteByte('$')
	}
	b.WriteString(ColName(c.Col))
	if c.AbsRow {
		b.WriteByte('$')
	}
	b.WriteString(strconv.Itoa(c.Row))
	return b.String()
}

// R1C1 formats the address in R1C1 notation relative to origin; absolute
// parts are written as R<n>/C<n>, relative ones as R[d]/C[d].
func (c Cell) R1C1(origin Cell) string {
	var b strings.Builder
	b.WriteByte('R')
	if c.AbsRow {
		b.WriteString(strconv.Itoa(c.Row))
	} else if d := c.Row - origin.Row; d != 0 {
		fmt.Fprintf(&b, "[%d]", d)
	}
	b.WriteByte('C')
	if c.AbsCol {
		b.WriteString(strconv.Itoa(c.Col))
	} else if d := c.Col - origin.Col; d != 0 {
		fmt.Fprintf(&b, "[%d]", d)
	}
	return b.String()
}

// Offset moves a relative address by the given deltas; absolute parts are
// kept. The result may be outside the grid, so check Valid.
func (c Cell) Offset(rows, cols int) Cell {
	if !c.AbsRow {
		c.Row += rows
	}
	if !c.AbsCol {
		c.Col += cols
	}
	return c
}

// ParseCell parses an A1 address such as "B7" or "$B$7".
func ParseCell(s string) (Cell, error) {
	c, rest, ok := scanCell(s)
	if !ok || rest != "" {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return c, nil
}

// MustCell is ParseCell that panics on error, for constants in code.
func MustCell(s string) Cell {
	c, err := ParseCell(s)
	if err != nil {
		panic(err)
	}
	return c
}

// scanCell reads an A1 address from the start of s and returns the rest.
func scanCell(s string) (Cell, string, bool) {
	var c Cell
	i := 0
	if i < len(s) && s[i] == '$' {
		c.AbsCol = true
		i++
	}
	j := i
	for j < len(s) && isLetter(s[j]) {
		j++
	}
	if j == i || j-i > 3 {
		return Cell{}, s, false
	}
	c.Col = ColNumber(s[i:j])
	if c.Col == 0 {
		return Cell{}, s, false
	}
	i = j
	if i < len(s) && s[i] == '$' {
		c.AbsRow = true
		i++
	}
	j = i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == i || j-i > 7 {
		return Cell{}, s, false
	}
	n, err := strconv.Atoi(s[i:j])
	if err != nil || n < 1 || n > MaxRows {
		return Cell{}, s, false
	}
	c.Row = n
	return c, s[j:], true
}

func isLetter(b byte) bool { return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }

// Range is an inclusive rectangular cell range. A whole-column range has
// Rows 1..MaxRows and a whole-row range Cols 1..MaxCols; WholeCols and
// WholeRows remember that the range was written that way ("A:C", "2:5").
type Range struct {
	From, To  Cell
	WholeCols bool
	WholeRows bool
}

// NewRange returns a range from (r1,c1) to (r2,c2), normalised so that From
// is the top-left corner.
func NewRange(r1, c1, r2, c2 int) Range {
	return Range{From: Cell{Row: r1, Col: c1}, To: Cell{Row: r2, Col: c2}}.Normalize()
}

// CellRange returns a single-cell range.
func CellRange(c Cell) Range { return Range{From: c, To: c} }

// Normalize orders the corners so that From.Row <= To.Row and From.Col <= To.Col.
func (r Range) Normalize() Range {
	if r.From.Row > r.To.Row {
		r.From.Row, r.To.Row = r.To.Row, r.From.Row
		r.From.AbsRow, r.To.AbsRow = r.To.AbsRow, r.From.AbsRow
	}
	if r.From.Col > r.To.Col {
		r.From.Col, r.To.Col = r.To.Col, r.From.Col
		r.From.AbsCol, r.To.AbsCol = r.To.AbsCol, r.From.AbsCol
	}
	return r
}

// Valid reports whether both corners are inside the grid and ordered.
func (r Range) Valid() bool {
	return r.From.Valid() && r.To.Valid() && r.From.Row <= r.To.Row && r.From.Col <= r.To.Col
}

// Rows returns the number of rows in the range.
func (r Range) Rows() int { return r.To.Row - r.From.Row + 1 }

// Cols returns the number of columns in the range.
func (r Range) Cols() int { return r.To.Col - r.From.Col + 1 }

// Size returns the number of cells.
func (r Range) Size() int { return r.Rows() * r.Cols() }

// IsCell reports whether the range covers a single cell.
func (r Range) IsCell() bool { return r.From.Row == r.To.Row && r.From.Col == r.To.Col }

// Contains reports whether the cell lies inside the range.
func (r Range) Contains(row, col int) bool {
	return row >= r.From.Row && row <= r.To.Row && col >= r.From.Col && col <= r.To.Col
}

// Overlaps reports whether two ranges share at least one cell.
func (r Range) Overlaps(o Range) bool {
	return r.From.Row <= o.To.Row && o.From.Row <= r.To.Row && r.From.Col <= o.To.Col && o.From.Col <= r.To.Col
}

// Intersect returns the common cells of two ranges.
func (r Range) Intersect(o Range) (Range, bool) {
	if !r.Overlaps(o) {
		return Range{}, false
	}
	return NewRange(max(r.From.Row, o.From.Row), max(r.From.Col, o.From.Col),
		min(r.To.Row, o.To.Row), min(r.To.Col, o.To.Col)), true
}

// Union returns the smallest range containing both.
func (r Range) Union(o Range) Range {
	return NewRange(min(r.From.Row, o.From.Row), min(r.From.Col, o.From.Col),
		max(r.To.Row, o.To.Row), max(r.To.Col, o.To.Col))
}

// Offset shifts the range.
func (r Range) Offset(rows, cols int) Range {
	r.From = r.From.Offset(rows, cols)
	r.To = r.To.Offset(rows, cols)
	return r
}

// Abs returns the range with every part absolute.
func (r Range) Abs() Range { r.From, r.To = r.From.Abs(), r.To.Abs(); return r }

// Each calls fn for every cell in row-major order and stops when it returns false.
func (r Range) Each(fn func(row, col int) bool) {
	for row := r.From.Row; row <= r.To.Row; row++ {
		for col := r.From.Col; col <= r.To.Col; col++ {
			if !fn(row, col) {
				return
			}
		}
	}
}

// String formats the range in A1 notation ("A1:C3", "A1", "A:C", "2:5").
func (r Range) String() string {
	switch {
	case r.WholeCols:
		return dollar(r.From.AbsCol) + ColName(r.From.Col) + ":" + dollar(r.To.AbsCol) + ColName(r.To.Col)
	case r.WholeRows:
		return dollar(r.From.AbsRow) + strconv.Itoa(r.From.Row) + ":" + dollar(r.To.AbsRow) + strconv.Itoa(r.To.Row)
	case r.IsCell() && r.From == r.To:
		return r.From.String()
	}
	return r.From.String() + ":" + r.To.String()
}

func dollar(abs bool) string {
	if abs {
		return "$"
	}
	return ""
}

// ParseRange parses "A1:C3", "A1", "A:C", "$A:$C", "2:5" or "B7:B7".
func ParseRange(s string) (Range, error) {
	r, rest, ok := scanRange(s)
	if !ok || rest != "" {
		return Range{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return r, nil
}

// MustRange is ParseRange that panics on error.
func MustRange(s string) Range {
	r, err := ParseRange(s)
	if err != nil {
		panic(err)
	}
	return r
}

func scanRange(s string) (Range, string, bool) {
	if c, rest, ok := scanCell(s); ok {
		if !strings.HasPrefix(rest, ":") {
			return CellRange(c), rest, true
		}
		c2, rest2, ok2 := scanCell(rest[1:])
		if !ok2 {
			return Range{}, s, false
		}
		return Range{From: c, To: c2}.Normalize(), rest2, true
	}
	// Whole columns: [$]A:[$]C
	if r, rest, ok := scanWholeCols(s); ok {
		return r, rest, true
	}
	if r, rest, ok := scanWholeRows(s); ok {
		return r, rest, true
	}
	return Range{}, s, false
}

func scanWholeCols(s string) (Range, string, bool) {
	i := 0
	abs1 := false
	if i < len(s) && s[i] == '$' {
		abs1 = true
		i++
	}
	j := i
	for j < len(s) && isLetter(s[j]) {
		j++
	}
	if j == i || j >= len(s) || s[j] != ':' {
		return Range{}, s, false
	}
	c1 := ColNumber(s[i:j])
	k := j + 1
	abs2 := false
	if k < len(s) && s[k] == '$' {
		abs2 = true
		k++
	}
	m := k
	for m < len(s) && isLetter(s[m]) {
		m++
	}
	if m == k {
		return Range{}, s, false
	}
	c2 := ColNumber(s[k:m])
	if c1 == 0 || c2 == 0 {
		return Range{}, s, false
	}
	r := Range{From: Cell{Row: 1, Col: c1, AbsCol: abs1, AbsRow: true}, To: Cell{Row: MaxRows, Col: c2, AbsCol: abs2, AbsRow: true}, WholeCols: true}.Normalize()
	return r, s[m:], true
}

func scanWholeRows(s string) (Range, string, bool) {
	i := 0
	abs1 := false
	if i < len(s) && s[i] == '$' {
		abs1 = true
		i++
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == i || j >= len(s) || s[j] != ':' {
		return Range{}, s, false
	}
	r1, _ := strconv.Atoi(s[i:j])
	k := j + 1
	abs2 := false
	if k < len(s) && s[k] == '$' {
		abs2 = true
		k++
	}
	m := k
	for m < len(s) && s[m] >= '0' && s[m] <= '9' {
		m++
	}
	if m == k {
		return Range{}, s, false
	}
	r2, _ := strconv.Atoi(s[k:m])
	if r1 < 1 || r2 < 1 || r1 > MaxRows || r2 > MaxRows {
		return Range{}, s, false
	}
	r := Range{From: Cell{Row: r1, Col: 1, AbsRow: abs1, AbsCol: true}, To: Cell{Row: r2, Col: MaxCols, AbsRow: abs2, AbsCol: true}, WholeRows: true}.Normalize()
	return r, s[m:], true
}

// Area is a range on a named sheet, as written in formulas and defined
// names: 'My Sheet'!A1:B2 or Sheet1!A1. An empty Sheet means the current sheet.
type Area struct {
	Sheet string
	Range Range
}

// String formats the area, quoting the sheet name when needed.
func (a Area) String() string {
	if a.Sheet == "" {
		return a.Range.String()
	}
	return QuoteSheet(a.Sheet) + "!" + a.Range.String()
}

// ParseArea parses an optionally sheet-qualified range.
func ParseArea(s string) (Area, error) {
	sheet, rest, err := splitSheet(s)
	if err != nil {
		return Area{}, err
	}
	r, err := ParseRange(rest)
	if err != nil {
		return Area{}, err
	}
	return Area{Sheet: sheet, Range: r}, nil
}

// ParseAreas parses a space- or comma-separated list of areas, as in
// sqref attributes ("A1:B2 D4") and print areas ("Sheet1!$A$1:$B$2,Sheet1!$D$4").
func ParseAreas(s string) ([]Area, error) {
	var out []Area
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' }) {
		a, err := ParseArea(part)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// ParseRanges parses a space-separated sqref list without sheet names.
func ParseRanges(s string) ([]Range, error) {
	var out []Range
	for part := range strings.FieldsSeq(s) {
		r, err := ParseRange(part)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// FormatRanges joins ranges as an sqref attribute value.
func FormatRanges(rs []Range) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		parts[i] = r.String()
	}
	return strings.Join(parts, " ")
}

func splitSheet(s string) (sheet, rest string, err error) {
	if strings.HasPrefix(s, "'") {
		// Quoted sheet name; '' is an escaped apostrophe.
		var b strings.Builder
		i := 1
		for i < len(s) {
			if s[i] == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					b.WriteByte('\'')
					i += 2
					continue
				}
				break
			}
			b.WriteByte(s[i])
			i++
		}
		if i+1 >= len(s) || s[i] != '\'' || s[i+1] != '!' {
			return "", "", fmt.Errorf("%w: %q", ErrInvalid, s)
		}
		return b.String(), s[i+2:], nil
	}
	if i := strings.LastIndexByte(s, '!'); i >= 0 {
		return s[:i], s[i+1:], nil
	}
	return "", s, nil
}

// QuoteSheet quotes a sheet name for use in a formula when it contains
// characters other than letters, digits and underscores or starts with a digit.
func QuoteSheet(name string) string {
	if !NeedsQuote(name) {
		return name
	}
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

// NeedsQuote reports whether QuoteSheet would quote the name.
func NeedsQuote(name string) bool {
	if name == "" {
		return true
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r > 0x7f:
		case r >= '0' && r <= '9', r == '.':
			if i == 0 {
				return true
			}
		default:
			return true
		}
	}
	// A name that looks like a cell reference or R1C1 must be quoted too.
	if _, err := ParseCell(name); err == nil {
		return true
	}
	up := strings.ToUpper(name)
	if (strings.HasPrefix(up, "R") || strings.HasPrefix(up, "C")) && len(name) > 1 && allDigits(name[1:]) {
		return true
	}
	return false
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// ParseR1C1 parses an R1C1 address relative to origin.
func ParseR1C1(s string, origin Cell) (Cell, error) {
	up := strings.ToUpper(s)
	if !strings.HasPrefix(up, "R") {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	ci := strings.IndexByte(up, 'C')
	if ci < 0 {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	row, absRow, err := r1c1Part(up[1:ci], origin.Row)
	if err != nil {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	col, absCol, err := r1c1Part(up[ci+1:], origin.Col)
	if err != nil {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	c := Cell{Row: row, Col: col, AbsRow: absRow, AbsCol: absCol}
	if !c.Valid() {
		return Cell{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return c, nil
}

func r1c1Part(s string, base int) (int, bool, error) {
	if s == "" {
		return base, false, nil
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		d, err := strconv.Atoi(s[1 : len(s)-1])
		if err != nil {
			return 0, false, err
		}
		return base + d, false, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}
