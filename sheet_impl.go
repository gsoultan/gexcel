// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package gexcel

import (
	"fmt"
	"iter"
	"sort"
	"strings"

	"github.com/gsoultan/gexcel/chart"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

type sheet struct {
	wb *workbook
	m  *model.Sheet
}

func (s *sheet) Model() *model.Sheet { return s.m }
func (s *sheet) Workbook() Workbook  { return s.wb }
func (s *sheet) Name() string        { return s.m.Name }
func (s *sheet) Index() int          { return s.wb.m.SheetIndex(s.m.Name) }
func (s *sheet) Hidden() bool        { return s.m.State != "" && s.m.State != model.SheetVisible }

func (s *sheet) SetName(name string) error {
	if !model.ValidSheetName(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	if i := s.wb.m.SheetIndex(name); i >= 0 && s.wb.m.Sheets[i] != s.m {
		return fmt.Errorf("%w: %q", ErrSheetExists, name)
	}
	old := s.m.Name
	s.m.Name = name
	// Keep names and formulas that mention the sheet in sync.
	oq, nq := quoteSheet(old)+"!", quoteSheet(name)+"!"
	for i := range s.wb.m.DefinedNames {
		s.wb.m.DefinedNames[i].RefersTo = strings.ReplaceAll(s.wb.m.DefinedNames[i].RefersTo, oq, nq)
	}
	for _, sh := range s.wb.m.Sheets {
		for _, row := range sh.Rows {
			for _, c := range row.Cells {
				if c.Formula != nil && strings.Contains(c.Formula.Text, oq) {
					c.Formula.Text = strings.ReplaceAll(c.Formula.Text, oq, nq)
				}
			}
		}
		for _, ch := range sh.Charts {
			for i := range ch.Series {
				ch.Series[i].Values = strings.ReplaceAll(ch.Series[i].Values, oq, nq)
				ch.Series[i].Categories = strings.ReplaceAll(ch.Series[i].Categories, oq, nq)
				ch.Series[i].NameRef = strings.ReplaceAll(ch.Series[i].NameRef, oq, nq)
			}
		}
	}
	return nil
}

func (s *sheet) SetHidden(h bool) {
	if h {
		s.m.State = model.SheetHidden
	} else {
		s.m.State = model.SheetVisible
	}
}

func (s *sheet) SetTabColor(c style.Color) { s.m.TabColor = c }

func (s *sheet) Cell(a1 string) Cell {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return &cell{sh: s, m: &model.Cell{}, err: fmt.Errorf("%w: %q", ErrInvalidRef, a1)}
	}
	return s.CellAt(cr.Row, cr.Col)
}

func (s *sheet) CellAt(row, col int) Cell {
	if row < 1 || col < 1 || row > ref.MaxRows || col > ref.MaxCols {
		return &cell{sh: s, m: &model.Cell{}, err: fmt.Errorf("%w: row %d col %d", ErrInvalidRef, row, col)}
	}
	c := s.m.Cell(row, col)
	return &cell{sh: s, m: c}
}

func (s *sheet) Lookup(a1 string) (Cell, bool) {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return nil, false
	}
	c := s.m.Lookup(cr.Row, cr.Col)
	if c == nil {
		return nil, false
	}
	return &cell{sh: s, m: c}, true
}

func (s *sheet) Row(index int) Row { return &row{sh: s, m: s.m.Row(index)} }

func (s *sheet) Rows() iter.Seq[Row] {
	return func(yield func(Row) bool) {
		for _, r := range s.m.SortedRows() {
			if !yield(&row{sh: s, m: r}) {
				return
			}
		}
	}
}

func (s *sheet) Cells() iter.Seq[Cell] {
	return func(yield func(Cell) bool) {
		for _, r := range s.m.SortedRows() {
			for _, c := range r.SortedCells() {
				if !yield(&cell{sh: s, m: c}) {
					return
				}
			}
		}
	}
}

func (s *sheet) Range(rng string) iter.Seq[Cell] {
	r, err := ref.ParseRange(rng)
	return func(yield func(Cell) bool) {
		if err != nil {
			return
		}
		r = r.Normalize()
		if r.WholeCols || r.WholeRows {
			// Only existing cells for open-ended ranges.
			for _, mr := range s.m.SortedRows() {
				for _, c := range mr.SortedCells() {
					if r.Contains(c.Row, c.Col) && !yield(&cell{sh: s, m: c}) {
						return
					}
				}
			}
			return
		}
		for rr := r.From.Row; rr <= r.To.Row; rr++ {
			for cc := r.From.Col; cc <= r.To.Col; cc++ {
				if !yield(&cell{sh: s, m: s.m.Cell(rr, cc)}) {
					return
				}
			}
		}
	}
}

func (s *sheet) UsedRange() (ref.Range, bool) { return s.m.UsedRange() }

func (s *sheet) MaxRow() int {
	r, ok := s.m.UsedRange()
	if !ok {
		return 0
	}
	return r.To.Row
}

func (s *sheet) MaxCol() int {
	r, ok := s.m.UsedRange()
	if !ok {
		return 0
	}
	return r.To.Col
}

func (s *sheet) Set(a1 string, v any) Cell { return s.Cell(a1).Set(v) }

func (s *sheet) SetRow(a1 string, values ...any) error {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, a1)
	}
	for i, v := range values {
		s.CellAt(cr.Row, cr.Col+i).Set(v)
	}
	return nil
}

func (s *sheet) SetRows(a1 string, rows [][]any) error {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, a1)
	}
	for r, vals := range rows {
		for c, v := range vals {
			s.CellAt(cr.Row+r, cr.Col+c).Set(v)
		}
	}
	return nil
}

func (s *sheet) Value(a1 string) model.Value {
	c, ok := s.Lookup(a1)
	if !ok {
		return model.Value{}
	}
	return c.Value()
}

func (s *sheet) AppendRow(values ...any) Row {
	r := s.Row(s.MaxRow() + 1)
	return r.Set(values...)
}

func (s *sheet) Merge(rng string) error {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	r = r.Normalize()
	for _, m := range s.m.Merges {
		if m.Overlaps(r) {
			return fmt.Errorf("gexcel: merge %s overlaps existing merge %s", r, m)
		}
	}
	s.m.Merges = append(s.m.Merges, r)
	return nil
}

func (s *sheet) Unmerge(rng string) error {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	r = r.Normalize()
	out := s.m.Merges[:0]
	for _, m := range s.m.Merges {
		if m != r && !m.Overlaps(r) {
			out = append(out, m)
		}
	}
	s.m.Merges = out
	return nil
}

func (s *sheet) Merges() []ref.Range {
	out := make([]ref.Range, len(s.m.Merges))
	copy(out, s.m.Merges)
	return out
}

func (s *sheet) setCols(from, to int, fn func(*model.Col)) {
	if to < from {
		to = from
	}
	for c := from; c <= to; c++ {
		s.m.SetCol(c, fn)
	}
}

func (s *sheet) SetColWidth(from, to int, width float64) {
	s.setCols(from, to, func(c *model.Col) { c.Width, c.CustomWidth = width, true })
}

func (s *sheet) SetColHidden(from, to int, hidden bool) {
	s.setCols(from, to, func(c *model.Col) { c.Hidden = hidden })
}

func (s *sheet) SetColStyle(from, to int, st style.Style) {
	id := s.wb.m.Styles.Add(st)
	s.setCols(from, to, func(c *model.Col) { c.StyleID = id })
}

func (s *sheet) SetRowHeight(r int, h float64) {
	mr := s.m.Row(r)
	mr.Height, mr.CustomHeight = h, true
}

func (s *sheet) SetStyle(rng string, st style.Style) error {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	r = r.Normalize()
	switch {
	case r.WholeCols:
		s.SetColStyle(r.From.Col, r.To.Col, st)
		for c := range s.Range(rng) {
			c.SetStyle(st)
		}
	case r.WholeRows:
		id := s.wb.m.Styles.Add(st)
		for rr := r.From.Row; rr <= r.To.Row; rr++ {
			mr := s.m.Row(rr)
			mr.StyleID, mr.CustomFormat = id, true
		}
		for c := range s.Range(rng) {
			c.SetStyle(st)
		}
	default:
		for c := range s.Range(rng) {
			c.SetStyle(st)
		}
	}
	return nil
}

func (s *sheet) Freeze(rows, cols int) {
	if rows <= 0 && cols <= 0 {
		s.m.View.Freeze = nil
		return
	}
	s.m.View.Freeze = &model.Pane{Rows: rows, Cols: cols}
}

func (s *sheet) SetZoom(pct int) { s.m.View.ZoomScale = pct }

func (s *sheet) SetGridLines(show bool) { s.m.View.HideGridLines = !show }

func (s *sheet) AddHyperlink(rng, target, tooltip string) error {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	s.removeHyperlink(r)
	s.m.Hyperlinks = append(s.m.Hyperlinks, model.Hyperlink{Range: r.Normalize(), Target: target, Tooltip: tooltip})
	return nil
}

func (s *sheet) AddInternalLink(rng, location, tooltip string) error {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	s.removeHyperlink(r)
	s.m.Hyperlinks = append(s.m.Hyperlinks, model.Hyperlink{Range: r.Normalize(), Location: location, Tooltip: tooltip})
	return nil
}

func (s *sheet) removeHyperlink(r ref.Range) {
	out := s.m.Hyperlinks[:0]
	for _, h := range s.m.Hyperlinks {
		if h.Range != r.Normalize() {
			out = append(out, h)
		}
	}
	s.m.Hyperlinks = out
}

func (s *sheet) AddComment(a1, author, text string) error {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, a1)
	}
	s.removeComment(cr)
	s.m.Comments = append(s.m.Comments, model.Comment{Ref: cr, Author: author, Text: text})
	return nil
}

func (s *sheet) AddThreadedComment(a1, author, text string, replies ...model.CommentReply) error {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, a1)
	}
	s.removeComment(cr)
	s.m.Comments = append(s.m.Comments, model.Comment{Ref: cr, Author: author, Text: text, Threaded: true, Replies: replies})
	return nil
}

func (s *sheet) removeComment(cr ref.Cell) {
	out := s.m.Comments[:0]
	for _, c := range s.m.Comments {
		if c.Ref.Row != cr.Row || c.Ref.Col != cr.Col {
			out = append(out, c)
		}
	}
	s.m.Comments = out
}

func (s *sheet) AddImage(a1 string, img model.Image) error {
	cr, err := ref.ParseCell(a1)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, a1)
	}
	if len(img.Data) == 0 {
		return fmt.Errorf("gexcel: image has no data")
	}
	if img.Format == "" {
		img.Format = sniffImage(img.Data)
	}
	if img.Anchor.Kind == 0 {
		img.Anchor = model.Anchor{Kind: model.AnchorOneCell, From: model.CellOffset{Row: cr.Row, Col: cr.Col}, Width: img.Anchor.Width, Height: img.Anchor.Height}
		if img.Anchor.Width == 0 || img.Anchor.Height == 0 {
			w, h := imageSize(img.Data)
			img.Anchor.Width, img.Anchor.Height = int64(w)*model.EMUPerPixel, int64(h)*model.EMUPerPixel
		}
	}
	s.m.Images = append(s.m.Images, &img)
	return nil
}

func (s *sheet) AddChart(from, to string, c *chart.Chart) error {
	f, err := ref.ParseCell(from)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, from)
	}
	t, err := ref.ParseCell(to)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, to)
	}
	c.Anchor = chart.AnchorRef{FromRow: f.Row, FromCol: f.Col, ToRow: t.Row, ToCol: t.Col}
	s.m.Charts = append(s.m.Charts, c)
	return nil
}

func (s *sheet) AddTable(rng, name string, opts ...TableOption) (*model.Table, error) {
	r, err := ref.ParseRange(rng)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	r = r.Normalize()
	t := &model.Table{Name: name, DisplayName: name, Range: r, Style: model.DefaultTableStyle()}
	seen := map[string]bool{}
	for c := r.From.Col; c <= r.To.Col; c++ {
		hc := s.m.Cell(r.From.Row, c)
		h := hc.Value.Text()
		if h == "" {
			h = fmt.Sprintf("Column%d", c-r.From.Col+1)
			hc.Value = model.String(h)
		}
		base := h
		for n := 2; seen[strings.ToLower(h)]; n++ {
			h = fmt.Sprintf("%s%d", base, n)
		}
		seen[strings.ToLower(h)] = true
		if hc.Value.Text() != h {
			hc.Value = model.String(h)
		}
		t.Columns = append(t.Columns, model.TableColumn{ID: c - r.From.Col + 1, Name: h})
	}
	for _, o := range opts {
		o(t)
	}
	s.m.Tables = append(s.m.Tables, t)
	return t, nil
}

func (s *sheet) AddDataValidation(rng string, dv model.DataValidation) error {
	rs, err := ref.ParseRanges(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	dv.Ranges = rs
	s.m.DataValidations = append(s.m.DataValidations, dv)
	return nil
}

func (s *sheet) AddConditionalFormat(rng string, rules ...model.CondFmtRule) error {
	rs, err := ref.ParseRanges(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	s.m.ConditionalFormats = append(s.m.ConditionalFormats, model.ConditionalFormat{Ranges: rs, Rules: rules})
	return nil
}

func (s *sheet) AddSparklines(g model.SparklineGroup) { s.m.Sparklines = append(s.m.Sparklines, g) }

func (s *sheet) SetAutoFilter(rng string) error {
	if rng == "" {
		s.m.AutoFilter = nil
		return nil
	}
	r, err := ref.ParseRange(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	s.m.AutoFilter = &model.AutoFilter{Range: r.Normalize()}
	return nil
}

func (s *sheet) Protect(password string, allow ...Permission) {
	p := &model.SheetProtection{Password: password, Sheet: true}
	for _, a := range allow {
		switch a {
		case AllowFormatCells:
			p.FormatCells = true
		case AllowFormatColumns:
			p.FormatColumns = true
		case AllowFormatRows:
			p.FormatRows = true
		case AllowInsertColumns:
			p.InsertColumns = true
		case AllowInsertRows:
			p.InsertRows = true
		case AllowInsertHyperlinks:
			p.InsertHyperlinks = true
		case AllowDeleteColumns:
			p.DeleteColumns = true
		case AllowDeleteRows:
			p.DeleteRows = true
		case AllowSort:
			p.Sort = true
		case AllowAutoFilter:
			p.AutoFilter = true
		case AllowPivotTables:
			p.PivotTables = true
		case AllowObjects:
			p.Objects = true
		case AllowScenarios:
			p.Scenarios = true
		}
	}
	s.m.Protection = p
}

func (s *sheet) Unprotect() { s.m.Protection = nil }

func (s *sheet) PageSetup() *model.PageSetup       { return &s.m.PageSetup }
func (s *sheet) PageMargins() *model.PageMargins   { return &s.m.PageMargins }
func (s *sheet) HeaderFooter() *model.HeaderFooter { return &s.m.HeaderFooter }

func (s *sheet) SetPrintArea(rng string) error {
	if rng == "" {
		s.m.PrintArea = nil
		return nil
	}
	rs, err := ref.ParseRanges(rng)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRef, rng)
	}
	s.m.PrintArea = rs
	return nil
}

func (s *sheet) SetPrintTitleRows(from, to int) {
	if from <= 0 {
		s.m.PrintTitleRows = nil
		return
	}
	if to < from {
		to = from
	}
	r := ref.Range{From: ref.Cell{Row: from, AbsRow: true, Col: 1}, To: ref.Cell{Row: to, AbsRow: true, Col: ref.MaxCols}, WholeRows: true}
	s.m.PrintTitleRows = &r
}

func (s *sheet) SetPrintTitleCols(from, to int) {
	if from <= 0 {
		s.m.PrintTitleCols = nil
		return
	}
	if to < from {
		to = from
	}
	r := ref.Range{From: ref.Cell{Col: from, AbsCol: true, Row: 1}, To: ref.Cell{Col: to, AbsCol: true, Row: ref.MaxRows}, WholeCols: true}
	s.m.PrintTitleCols = &r
}

// row implements Row.
type row struct {
	sh *sheet
	m  *model.Row
}

func (r *row) Model() *model.Row { return r.m }
func (r *row) Index() int        { return r.m.Index }
func (r *row) Cell(col int) Cell { return r.sh.CellAt(r.m.Index, col) }

func (r *row) Cells() iter.Seq[Cell] {
	return func(yield func(Cell) bool) {
		for _, c := range r.m.SortedCells() {
			if !yield(&cell{sh: r.sh, m: c}) {
				return
			}
		}
	}
}

func (r *row) Values() []model.Value {
	max := r.m.MaxCol()
	out := make([]model.Value, max)
	for col, c := range r.m.Cells {
		if col >= 1 && col <= max {
			out[col-1] = c.Value
		}
	}
	return out
}

func (r *row) Set(values ...any) Row {
	for i, v := range values {
		r.Cell(i + 1).Set(v)
	}
	return r
}

func (r *row) SetHeight(h float64) Row { r.m.Height, r.m.CustomHeight = h, true; return r }
func (r *row) SetHidden(h bool) Row    { r.m.Hidden = h; return r }

func (r *row) SetStyle(st style.Style) Row {
	r.m.StyleID, r.m.CustomFormat = r.sh.wb.m.Styles.Add(st), true
	cols := make([]int, 0, len(r.m.Cells))
	for c := range r.m.Cells {
		cols = append(cols, c)
	}
	sort.Ints(cols)
	for _, c := range cols {
		r.Cell(c).SetStyle(st)
	}
	return r
}

func (r *row) SetOutlineLevel(level int, collapsed bool) Row {
	r.m.OutlineLevel, r.m.Collapsed = level, collapsed
	if level > r.sh.m.OutlineLevelRow {
		r.sh.m.OutlineLevelRow = level
	}
	return r
}
