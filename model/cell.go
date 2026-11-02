// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package model

import (
	"sort"

	"github.com/gsoultan/gexcel/ref"
)

// FormulaKind is the formula type (ECMA-376 ST_CellFormulaType).
type FormulaKind uint8

// Formula kinds.
const (
	FormulaNormal    FormulaKind = iota
	FormulaArray                 // legacy CSE array formula or dynamic array
	FormulaShared                // shared formula group
	FormulaDataTable             // what-if data table
)

// Formula is a cell formula without the leading "=". Cached results live
// in the cell's Value.
type Formula struct {
	Text string
	Kind FormulaKind
	// Ref is the range an array or shared formula covers (set on the master
	// cell only).
	Ref ref.Range
	// SharedIndex identifies a shared formula group; the master cell has
	// Text set, the others only the index (the writer expands nothing: the
	// reader fills Text for every cell of the group by translating the master).
	SharedIndex int
	// Master marks the cell that defines a shared or array formula.
	Master bool
	// CalcAlways (ca="1") forces recalculation on load (volatile formulas).
	CalcAlways bool
	// Dynamic marks a dynamic-array formula (Excel 365 spill); the writer
	// adds the cm="1" metadata so Excel does not show it as a legacy CSE
	// array formula.
	Dynamic bool
	// Data table attributes.
	DataTable2D   bool
	DataTableRow  bool
	DataTableRef1 string
	DataTableRef2 string
}

// Cell is one cell of a sheet. Row and Col are 1-based.
type Cell struct {
	Row, Col int
	Value    Value
	Formula  *Formula
	// StyleID indexes Styles.CellXfs; 0 is the default style.
	StyleID int
	// ShowPhonetic (ph="1") shows East Asian phonetic guides.
	ShowPhonetic bool
	// ValueMetadata indexes the value metadata (vm) of rich data values
	// (Excel 365 data types, images in cells). Kept for round trips.
	ValueMetadata int
}

// Ref returns the A1 address of the cell.
func (c *Cell) Ref() string { return ref.Cell{Row: c.Row, Col: c.Col}.String() }

// IsEmpty reports whether the cell has no value, formula or style.
func (c *Cell) IsEmpty() bool {
	return c == nil || c.Value.IsEmpty() && c.Formula == nil && c.StyleID == 0
}

// Row is a sheet row. Index is 1-based.
type Row struct {
	Index int
	Cells map[int]*Cell // by column
	// Height in points; 0 means the sheet default. CustomHeight marks an
	// explicit height (Excel ignores ht without it on some rows).
	Height       float64
	CustomHeight bool
	Hidden       bool
	// StyleID is the row's default cell style (s attribute, customFormat).
	StyleID      int
	CustomFormat bool
	OutlineLevel int
	Collapsed    bool
	ThickTop     bool
	ThickBottom  bool
	ShowPhonetic bool
}

// Cell returns the cell at the column, creating it when missing.
func (r *Row) Cell(col int) *Cell {
	if r.Cells == nil {
		r.Cells = map[int]*Cell{}
	}
	c, ok := r.Cells[col]
	if !ok {
		c = &Cell{Row: r.Index, Col: col}
		r.Cells[col] = c
	}
	return c
}

// Lookup returns the cell at the column or nil.
func (r *Row) Lookup(col int) *Cell { return r.Cells[col] }

// SortedCells returns the cells in column order.
func (r *Row) SortedCells() []*Cell {
	out := make([]*Cell, 0, len(r.Cells))
	for _, c := range r.Cells {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Col < out[j].Col })
	return out
}

// MaxCol returns the highest column with a cell, or 0.
func (r *Row) MaxCol() int {
	m := 0
	for col := range r.Cells {
		if col > m {
			m = col
		}
	}
	return m
}

// MinCol returns the lowest column with a cell, or 0.
func (r *Row) MinCol() int {
	m := 0
	for col := range r.Cells {
		if m == 0 || col < m {
			m = col
		}
	}
	return m
}

// IsEmpty reports whether the row carries no cells and no attributes.
func (r *Row) IsEmpty() bool {
	return len(r.Cells) == 0 && r.Height == 0 && !r.Hidden && r.StyleID == 0 && r.OutlineLevel == 0 && !r.Collapsed && !r.ThickTop && !r.ThickBottom
}
