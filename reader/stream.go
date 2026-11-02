// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"iter"
	"os"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// Stream reads worksheet rows one at a time, so very large files can be
// processed without holding a sheet in memory. Workbook-level data
// (sheet list, styles, shared strings, names) is loaded up front and
// available through Workbook().
type Stream struct {
	d      *docReader
	closer io.Closer
}

// OpenStream prepares a streaming reader over a package.
func OpenStream(r io.ReaderAt, size int64, opts ...Options) (*Stream, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.MaxPartSize == 0 {
		o.MaxPartSize = 1 << 30
	}
	o.SkipDrawings = true
	pr, err := ooxml.Open(r, size)
	if err != nil {
		return nil, err
	}
	pr.MaxPartSize = o.MaxPartSize
	rd := &docReader{pr: pr, opts: o, wb: model.New(), skipSheets: true}
	if err := rd.read(); err != nil {
		return nil, err
	}
	return &Stream{d: rd}, nil
}

// OpenStreamFile opens a file for streaming; Close releases it.
func OpenStreamFile(name string, opts ...Options) (*Stream, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	s, err := OpenStream(f, st.Size(), opts...)
	if err != nil {
		f.Close()
		return nil, err
	}
	s.closer = f
	return s, nil
}

// OpenStreamBytes streams a package held in memory.
func OpenStreamBytes(data []byte, opts ...Options) (*Stream, error) {
	return OpenStream(bytes.NewReader(data), int64(len(data)), opts...)
}

// Workbook returns the workbook skeleton: sheets (without rows), styles,
// theme, defined names and properties.
func (s *Stream) Workbook() *model.Workbook { return s.d.wb }

// SheetNames lists the worksheets that can be streamed.
func (s *Stream) SheetNames() []string {
	var out []string
	for _, sh := range s.d.wb.Sheets {
		if sh.Kind == model.Worksheet {
			out = append(out, sh.Name)
		}
	}
	return out
}

// Close releases the underlying file, if any.
func (s *Stream) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// ErrNoSheet is returned when a streamed sheet does not exist.
var ErrNoSheet = errors.New("reader: no such sheet")

// Rows iterates the rows of a sheet in file order. Each row is complete
// (cells, formulas, style ids) and is not retained after the iteration
// step. Sheet-level settings (merges, column widths, ...) are filled on
// the model sheet as they are encountered after the row data.
func (s *Stream) Rows(sheet string) iter.Seq2[*model.Row, error] {
	return func(yield func(*model.Row, error) bool) {
		part, ok := s.d.sheetParts[sheet]
		if !ok {
			for name, p := range s.d.sheetParts {
				if equalFold(name, sheet) {
					part, ok = p, true
				}
			}
		}
		if !ok {
			yield(nil, ErrNoSheet)
			return
		}
		sh := s.d.wb.Sheet(sheet)
		if sh == nil {
			sh = model.NewSheet(sheet)
		}
		rc, err := s.d.pr.Open(part)
		if err != nil {
			yield(nil, err)
			return
		}
		defer rc.Close()
		rels, _ := s.d.pr.Rels(part)
		dec := xml.NewDecoder(io.LimitReader(rc, s.d.opts.MaxPartSize))
		dec.Strict = false
		sp := &sheetParser{d: s.d, sh: sh, part: part, rels: rels, dec: dec}
		var stopped bool
		sp.onRow = func(r *model.Row) bool {
			if !yield(r, nil) {
				stopped = true
				return false
			}
			return true
		}
		if err := sp.parse(); err != nil && !errors.Is(err, errStop) && !stopped {
			yield(nil, err)
		}
	}
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
