// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"bufio"
	"fmt"
	"io"
	"sort"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// StreamWriter writes a workbook whose sheet rows are produced on the fly,
// so sheets of millions of rows never sit in memory. Use it like this:
//
//	sw := writer.NewStreamWriter(w, wb)      // wb holds styles, names, properties
//	sh := sw.Sheet(model.NewSheet("Data"))   // sheet-level settings come from the model sheet
//	for ... { sh.Row(i, cells...) }           // rows in increasing order
//	sh.Close()                                // more sheets may follow
//	sw.Close()
//
// Strings are written inline (no shared string table) unless
// Options.InlineStrings is false and SharedStrings is set.
type StreamWriter struct {
	p      *packer
	wb     *model.Workbook
	cur    *StreamSheet
	closed bool
	err    error
}

// NewStreamWriter starts streaming a workbook. wb provides the styles,
// theme, properties and defined names; its Sheets slice is filled as
// sheets are streamed (with their row data omitted).
func NewStreamWriter(w io.Writer, wb *model.Workbook, opts ...Options) *StreamWriter {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if wb == nil {
		wb = model.New()
	}
	if wb.Styles == nil {
		wb.Styles = model.NewStyles()
	}
	wb.Sheets = nil
	return &StreamWriter{p: newPacker(wb, w, o), wb: wb}
}

// Workbook returns the workbook the streamed sheets belong to, for styles
// (wb.Styles.Add) and workbook-level settings.
func (s *StreamWriter) Workbook() *model.Workbook { return s.wb }

// Sheet starts a new sheet. The sheet's own rows are ignored; everything
// else (columns, merges, freeze panes, validations, images, ...) is
// written. The previous sheet is closed first.
func (s *StreamWriter) Sheet(sh *model.Sheet) *StreamSheet {
	if s.cur != nil {
		s.cur.Close()
	}
	if s.closed || s.err != nil {
		return &StreamSheet{err: fmt.Errorf("%w: writer closed", ErrStreaming)}
	}
	sh.Rows = nil
	if !model.ValidSheetName(sh.Name) {
		s.err = fmt.Errorf("%w: %q", ErrInvalidSheetName, sh.Name)
		return &StreamSheet{err: s.err}
	}
	for _, o := range s.wb.Sheets {
		if o.Name == sh.Name {
			s.err = fmt.Errorf("%w: %q", ErrDuplicateSheet, sh.Name)
			return &StreamSheet{err: s.err}
		}
	}
	index := len(s.wb.Sheets)
	s.wb.Sheets = append(s.wb.Sheets, sh)
	ss := &StreamSheet{s: s, sh: sh, index: index}
	s.cur = ss
	// The packer writes the head immediately through the rows callback;
	// we capture the writer to feed rows into.
	ss.start()
	return ss
}

// Close finishes the current sheet and writes the workbook-level parts.
func (s *StreamWriter) Close() error {
	if s.closed {
		return s.err
	}
	if s.cur != nil {
		if err := s.cur.Close(); err != nil {
			s.err = err
		}
	}
	s.closed = true
	if s.err != nil {
		return s.err
	}
	if len(s.wb.Sheets) == 0 {
		s.err = ErrNoSheets
		return s.err
	}
	s.err = s.p.finish()
	return s.err
}

// StreamSheet receives rows of one sheet.
type StreamSheet struct {
	s       *StreamWriter
	sh      *model.Sheet
	index   int
	sw      *sheetWriter
	out     io.Writer
	bw      *bufio.Writer
	lastRow int
	closed  bool
	err     error
	buf     ooxml.Builder
}

func (ss *StreamSheet) start() {
	p := ss.s.p
	name := sheetPartName(ss.index, model.Worksheet)
	part := sheetPart{name: name, sheet: ss.sh}
	part.relID = p.pw.Rel("xl/workbook.xml", ooxml.RelWorksheet, ooxml.Relative("xl/workbook.xml", name), "")
	p.sheetParts = append(p.sheetParts, part)
	ss.sw = &sheetWriter{p: p, sh: ss.sh, part: name, index: ss.index}
	if err := ss.sw.prepare(); err != nil {
		ss.err = err
		return
	}
	w, err := p.pw.Create(name, ooxml.CTWorksheet)
	if err != nil {
		ss.err = err
		return
	}
	ss.out = w
	if err := ss.sw.writeHead(w); err != nil {
		ss.err = err
		return
	}
	ss.bw = bufio.NewWriterSize(w, 256<<10)
}

// Row writes a row. Rows must come in increasing index order; cells may be
// given in any column order. Cells carry their own Row/Col; a zero Row is
// filled in.
func (ss *StreamSheet) Row(index int, cells ...*model.Cell) error {
	if ss.err != nil {
		return ss.err
	}
	if ss.closed {
		return fmt.Errorf("%w: sheet closed", ErrStreaming)
	}
	if index <= ss.lastRow {
		return fmt.Errorf("%w: row %d after row %d", ErrStreaming, index, ss.lastRow)
	}
	ss.lastRow = index
	for _, c := range cells {
		c.Row = index
	}
	sort.Slice(cells, func(i, j int) bool { return cells[i].Col < cells[j].Col })
	r := &model.Row{Index: index}
	ss.buf.Reset()
	ss.sw.rowXML(&ss.buf, r, cells)
	_, err := ss.bw.Write(ss.buf.Bytes())
	if err != nil {
		ss.err = err
	}
	return err
}

// RowWith writes a row with attributes (height, hidden, style) and cells.
func (ss *StreamSheet) RowWith(r *model.Row, cells ...*model.Cell) error {
	if ss.err != nil {
		return ss.err
	}
	if ss.closed {
		return fmt.Errorf("%w: sheet closed", ErrStreaming)
	}
	if r.Index <= ss.lastRow {
		return fmt.Errorf("%w: row %d after row %d", ErrStreaming, r.Index, ss.lastRow)
	}
	ss.lastRow = r.Index
	for _, c := range cells {
		c.Row = r.Index
	}
	sort.Slice(cells, func(i, j int) bool { return cells[i].Col < cells[j].Col })
	ss.buf.Reset()
	ss.sw.rowXML(&ss.buf, r, cells)
	_, err := ss.bw.Write(ss.buf.Bytes())
	if err != nil {
		ss.err = err
	}
	return err
}

// Close ends the sheet: the parts after sheetData (merges, validations,
// drawings, ...) are written from the model sheet as it is now, so
// post-row settings may still be added before Close.
func (ss *StreamSheet) Close() error {
	if ss.closed {
		return ss.err
	}
	ss.closed = true
	if ss.err != nil {
		return ss.err
	}
	if err := ss.bw.Flush(); err != nil {
		ss.err = err
		return err
	}
	ss.err = ss.sw.writeTail(ss.out)
	return ss.err
}
