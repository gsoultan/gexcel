// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package gexcel

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/gexcel/writer"
)

type workbook struct {
	m *model.Workbook
}

func (w *workbook) Model() *model.Workbook        { return w.m }
func (w *workbook) Styles() *model.Styles         { return w.m.Styles }
func (w *workbook) Theme() *style.Theme           { return w.m.Theme }
func (w *workbook) Properties() *model.Properties { return &w.m.Properties }
func (w *workbook) Date1904() bool                { return w.m.Date1904 }
func (w *workbook) SetFullCalcOnLoad(b bool)      { w.m.Calc.FullCalcOnLoad = b }

func (w *workbook) Sheets() []Sheet {
	out := make([]Sheet, len(w.m.Sheets))
	for i, s := range w.m.Sheets {
		out[i] = &sheet{wb: w, m: s}
	}
	return out
}

func (w *workbook) Sheet(name string) (Sheet, bool) {
	i := w.m.SheetIndex(name)
	if i < 0 {
		return nil, false
	}
	return &sheet{wb: w, m: w.m.Sheets[i]}, true
}

func (w *workbook) SheetAt(i int) Sheet {
	if i < 0 || i >= len(w.m.Sheets) {
		return nil
	}
	return &sheet{wb: w, m: w.m.Sheets[i]}
}

func (w *workbook) AddSheet(name string) (Sheet, error) {
	if !model.ValidSheetName(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	if w.m.SheetIndex(name) >= 0 {
		return nil, fmt.Errorf("%w: %q", ErrSheetExists, name)
	}
	return &sheet{wb: w, m: w.m.AddSheet(name)}, nil
}

func (w *workbook) RemoveSheet(name string) error {
	i := w.m.SheetIndex(name)
	if i < 0 {
		return fmt.Errorf("%w: %q", ErrSheetNotFound, name)
	}
	if len(w.m.Sheets) == 1 {
		return ErrLastSheet
	}
	w.m.Sheets = append(w.m.Sheets[:i], w.m.Sheets[i+1:]...)
	// Drop and renumber sheet-scoped names.
	names := w.m.DefinedNames[:0]
	for _, dn := range w.m.DefinedNames {
		if dn.Local {
			if dn.SheetIndex == i {
				continue
			}
			if dn.SheetIndex > i {
				dn.SheetIndex--
			}
		}
		names = append(names, dn)
	}
	w.m.DefinedNames = names
	if w.m.View.ActiveTab >= len(w.m.Sheets) {
		w.m.View.ActiveTab = len(w.m.Sheets) - 1
	}
	return nil
}

func (w *workbook) MoveSheet(name string, to int) error {
	i := w.m.SheetIndex(name)
	if i < 0 {
		return fmt.Errorf("%w: %q", ErrSheetNotFound, name)
	}
	if to < 0 {
		to = 0
	}
	if to >= len(w.m.Sheets) {
		to = len(w.m.Sheets) - 1
	}
	if to == i {
		return nil
	}
	s := w.m.Sheets[i]
	w.m.Sheets = append(w.m.Sheets[:i], w.m.Sheets[i+1:]...)
	w.m.Sheets = append(w.m.Sheets[:to], append([]*model.Sheet{s}, w.m.Sheets[to:]...)...)
	remap := func(idx int) int {
		switch {
		case idx == i:
			return to
		case i < to && idx > i && idx <= to:
			return idx - 1
		case to < i && idx >= to && idx < i:
			return idx + 1
		}
		return idx
	}
	for k := range w.m.DefinedNames {
		if w.m.DefinedNames[k].Local {
			w.m.DefinedNames[k].SheetIndex = remap(w.m.DefinedNames[k].SheetIndex)
		}
	}
	w.m.View.ActiveTab = remap(w.m.View.ActiveTab)
	return nil
}

func (w *workbook) CopySheet(from, to string) (Sheet, error) {
	i := w.m.SheetIndex(from)
	if i < 0 {
		return nil, fmt.Errorf("%w: %q", ErrSheetNotFound, from)
	}
	if !model.ValidSheetName(to) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, to)
	}
	if w.m.SheetIndex(to) >= 0 {
		return nil, fmt.Errorf("%w: %q", ErrSheetExists, to)
	}
	cp := w.m.Sheets[i].Clone()
	cp.Name = to
	cp.CodeName = ""
	cp.View.TabSelected = false
	w.m.Sheets = append(w.m.Sheets, cp)
	for _, dn := range w.m.DefinedNames {
		if dn.Local && dn.SheetIndex == i {
			dn.SheetIndex = len(w.m.Sheets) - 1
			dn.RefersTo = strings.ReplaceAll(dn.RefersTo, quoteSheet(from)+"!", quoteSheet(to)+"!")
			w.m.DefinedNames = append(w.m.DefinedNames, dn)
		}
	}
	return &sheet{wb: w, m: cp}, nil
}

func (w *workbook) SetActiveSheet(name string) error {
	i := w.m.SheetIndex(name)
	if i < 0 {
		return fmt.Errorf("%w: %q", ErrSheetNotFound, name)
	}
	w.m.View.ActiveTab = i
	for k, s := range w.m.Sheets {
		s.View.TabSelected = k == i
	}
	return nil
}

func (w *workbook) DefinedNames() []model.DefinedName {
	out := make([]model.DefinedName, len(w.m.DefinedNames))
	copy(out, w.m.DefinedNames)
	return out
}

func (w *workbook) SetDefinedName(name, refersTo string) {
	refersTo = strings.TrimPrefix(refersTo, "=")
	for i := range w.m.DefinedNames {
		if !w.m.DefinedNames[i].Local && strings.EqualFold(w.m.DefinedNames[i].Name, name) {
			w.m.DefinedNames[i].RefersTo = refersTo
			return
		}
	}
	w.m.DefinedNames = append(w.m.DefinedNames, model.DefinedName{Name: name, RefersTo: refersTo})
}

func (w *workbook) SetSheetDefinedName(sheetName, name, refersTo string) error {
	i := w.m.SheetIndex(sheetName)
	if i < 0 {
		return fmt.Errorf("%w: %q", ErrSheetNotFound, sheetName)
	}
	refersTo = strings.TrimPrefix(refersTo, "=")
	for k := range w.m.DefinedNames {
		dn := &w.m.DefinedNames[k]
		if dn.Local && dn.SheetIndex == i && strings.EqualFold(dn.Name, name) {
			dn.RefersTo = refersTo
			return nil
		}
	}
	w.m.DefinedNames = append(w.m.DefinedNames, model.DefinedName{Name: name, RefersTo: refersTo, Local: true, SheetIndex: i})
	return nil
}

func (w *workbook) RemoveDefinedName(name string) {
	out := w.m.DefinedNames[:0]
	for _, dn := range w.m.DefinedNames {
		if !dn.Local && strings.EqualFold(dn.Name, name) {
			continue
		}
		out = append(out, dn)
	}
	w.m.DefinedNames = out
}

func (w *workbook) Save(out io.Writer, opts ...writer.Options) error {
	return writer.Write(w.m, out, opts...)
}

func (w *workbook) SaveFile(name string, opts ...writer.Options) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := writer.Write(w.m, f, opts...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (w *workbook) Validate() error { return writer.Validate(w.m) }

func (w *workbook) Calculate() error { return formula.Calculate(w.m) }

func quoteSheet(name string) string {
	if strings.ContainsAny(name, " -!'()+,;") || strings.IndexFunc(name, func(r rune) bool { return r >= '0' && r <= '9' }) == 0 {
		return "'" + strings.ReplaceAll(name, "'", "''") + "'"
	}
	return name
}
