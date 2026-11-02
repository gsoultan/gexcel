// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package model is the in-memory representation of a workbook: plain
// structs that the reader fills, the writer serialises and the root
// gexcel package wraps in its interfaces. It has no behaviour beyond small
// helpers, so every other package can share it.
package model

import (
	"math"
	"strconv"
	"time"

	"github.com/gsoultan/gexcel/format"
	"github.com/gsoultan/gexcel/style"
)

// ValueKind is the type of a cell value.
type ValueKind uint8

// Cell value kinds.
const (
	KindEmpty  ValueKind = iota
	KindString           // plain or rich text
	KindNumber           // float64 (dates are numbers with a date format)
	KindBool
	KindError // "#DIV/0!", "#N/A", ...
)

// Value is a typed cell value. Numbers hold dates too: Excel stores a date
// as a serial number, and the cell's number format makes it show as a date.
type Value struct {
	Kind ValueKind
	Str  string    // KindString: the text; KindError: the error code
	Num  float64   // KindNumber
	Bool bool      // KindBool
	Rich []RichRun // KindString with per-run formatting; Str holds the plain text
	// Inline marks a string that was stored as an inline string (t="inlineStr")
	// rather than in the shared string table; the writer keeps that choice.
	Inline bool
}

// RichRun is a piece of rich text with its own font. A nil Font uses the
// cell's font.
type RichRun struct {
	Text string
	Font *style.Font
}

// String returns a string value.
func String(s string) Value { return Value{Kind: KindString, Str: s} }

// Number returns a numeric value.
func Number(f float64) Value { return Value{Kind: KindNumber, Num: f} }

// Boolean returns a boolean value.
func Boolean(b bool) Value { return Value{Kind: KindBool, Bool: b} }

// Error returns an error value such as "#N/A".
func Error(code string) Value { return Value{Kind: KindError, Str: code} }

// Date returns the serial number of a time in the given date system.
func Date(t time.Time, date1904 bool) Value {
	return Number(format.ToSerial(t, date1904))
}

// Rich returns a rich-text value; the plain text is the concatenation.
func Rich(runs ...RichRun) Value {
	var s string
	for _, r := range runs {
		s += r.Text
	}
	return Value{Kind: KindString, Str: s, Rich: runs}
}

// IsEmpty reports whether the value is empty.
func (v Value) IsEmpty() bool { return v.Kind == KindEmpty }

// Time converts a numeric value to a time; ok is false for other kinds.
func (v Value) Time(date1904 bool) (time.Time, bool) {
	if v.Kind != KindNumber {
		return time.Time{}, false
	}
	return format.FromSerial(v.Num, date1904), true
}

// Float returns the value as a number: numbers as they are, booleans as
// 0/1, numeric-looking strings parsed; ok is false otherwise.
func (v Value) Float() (float64, bool) {
	switch v.Kind {
	case KindNumber:
		return v.Num, true
	case KindBool:
		if v.Bool {
			return 1, true
		}
		return 0, true
	case KindString:
		f, err := strconv.ParseFloat(v.Str, 64)
		return f, err == nil
	}
	return 0, false
}

// Text returns the value as plain text without applying a number format:
// numbers in General format, booleans as TRUE/FALSE, errors as their code.
func (v Value) Text() string {
	switch v.Kind {
	case KindString, KindError:
		return v.Str
	case KindNumber:
		return format.General(v.Num)
	case KindBool:
		return format.Bool(v.Bool)
	}
	return ""
}

// Formatted renders the value with a number format code.
func (v Value) Formatted(numFmt string, date1904 bool) string {
	switch v.Kind {
	case KindNumber:
		if math.IsNaN(v.Num) || math.IsInf(v.Num, 0) {
			return "#NUM!"
		}
		return format.Number(v.Num, numFmt, date1904)
	case KindString:
		return format.Text(v.Str, numFmt)
	}
	return v.Text()
}

// Equal compares two values including rich runs.
func (v Value) Equal(o Value) bool {
	if v.Kind != o.Kind || v.Str != o.Str || v.Num != o.Num || v.Bool != o.Bool || len(v.Rich) != len(o.Rich) {
		return false
	}
	for i := range v.Rich {
		a, b := v.Rich[i], o.Rich[i]
		if a.Text != b.Text || (a.Font == nil) != (b.Font == nil) || a.Font != nil && *a.Font != *b.Font {
			return false
		}
	}
	return true
}

// Error codes Excel knows.
const (
	ErrNull        = "#NULL!"
	ErrDiv0        = "#DIV/0!"
	ErrValue       = "#VALUE!"
	ErrRef         = "#REF!"
	ErrName        = "#NAME?"
	ErrNum         = "#NUM!"
	ErrNA          = "#N/A"
	ErrGettingData = "#GETTING_DATA"
	ErrSpill       = "#SPILL!"
	ErrCalc        = "#CALC!"
	ErrField       = "#FIELD!"
	ErrUnknown     = "#UNKNOWN!"
)

// IsErrorCode reports whether s is one of the Excel error literals.
func IsErrorCode(s string) bool {
	switch s {
	case ErrNull, ErrDiv0, ErrValue, ErrRef, ErrName, ErrNum, ErrNA, ErrGettingData, ErrSpill, ErrCalc, ErrField, ErrUnknown:
		return true
	}
	return false
}
