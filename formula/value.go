// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"math"
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
)

// Kind classifies an evaluation result.
type Kind uint8

// Result kinds.
const (
	Empty Kind = iota
	Number
	Text
	Bool
	Error
	Array
	Reference
)

// Value is the result of evaluating an expression. Arrays hold rows of
// values; references keep the sheet and range so functions such as ROW or
// OFFSET can inspect them.
type Value struct {
	Kind  Kind
	Num   float64
	Str   string
	Bool  bool
	Rows  [][]Value
	Sheet string
	Range ref.Range
}

// Constructors.
func Num(f float64) Value {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Err(model.ErrNum)
	}
	return Value{Kind: Number, Num: f}
}
func Str(s string) Value       { return Value{Kind: Text, Str: s} }
func Boolean(b bool) Value     { return Value{Kind: Bool, Bool: b} }
func Err(code string) Value    { return Value{Kind: Error, Str: code} }
func Arr(rows [][]Value) Value { return Value{Kind: Array, Rows: rows} }
func Ref(sheet string, r ref.Range) Value {
	return Value{Kind: Reference, Sheet: sheet, Range: r.Normalize()}
}

// Common errors.
var (
	errValue = Err(model.ErrValue)
	errDiv0  = Err(model.ErrDiv0)
	errNA    = Err(model.ErrNA)
	errRef   = Err(model.ErrRef)
	errName  = Err(model.ErrName)
	errNum   = Err(model.ErrNum)
	errCalc  = Err(model.ErrCalc)
)

// IsError reports whether v is an error value.
func (v Value) IsError() bool { return v.Kind == Error }

// FromModel converts a stored cell value.
func FromModel(m model.Value) Value {
	switch m.Kind {
	case model.KindString:
		return Str(m.Str)
	case model.KindNumber:
		return Num(m.Num)
	case model.KindBool:
		return Boolean(m.Bool)
	case model.KindError:
		return Err(m.Str)
	}
	return Value{}
}

// ToModel converts a result to a storable cell value. Arrays store their
// top-left element.
func (v Value) ToModel() model.Value {
	switch v.Kind {
	case Number:
		return model.Number(v.Num)
	case Text:
		return model.String(v.Str)
	case Bool:
		return model.Boolean(v.Bool)
	case Error:
		return model.Error(v.Str)
	case Array:
		if len(v.Rows) > 0 && len(v.Rows[0]) > 0 {
			return v.Rows[0][0].ToModel()
		}
		return model.Number(0)
	}
	return model.Value{}
}

// Dims returns the array dimensions (1x1 for scalars).
func (v Value) Dims() (rows, cols int) {
	if v.Kind == Array {
		if len(v.Rows) == 0 {
			return 0, 0
		}
		return len(v.Rows), len(v.Rows[0])
	}
	if v.Kind == Reference {
		return v.Range.Rows(), v.Range.Cols()
	}
	return 1, 1
}

// At returns an array element (scalars return themselves).
func (v Value) At(r, c int) Value {
	if v.Kind != Array {
		return v
	}
	if r < len(v.Rows) && c < len(v.Rows[r]) {
		return v.Rows[r][c]
	}
	return errNA
}

// coercions

func toNumber(v Value) (float64, Value) {
	switch v.Kind {
	case Number:
		return v.Num, Value{}
	case Bool:
		if v.Bool {
			return 1, Value{}
		}
		return 0, Value{}
	case Empty:
		return 0, Value{}
	case Text:
		f, ok := parseNumber(v.Str)
		if !ok {
			return 0, errValue
		}
		return f, Value{}
	case Error:
		return 0, v
	case Array:
		return toNumber(v.At(0, 0))
	}
	return 0, errValue
}

// parseNumber parses text as Excel would when a number is required:
// plain decimals, percentages, thousands separators and booleans are
// accepted.
func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	pct := false
	if strings.HasSuffix(s, "%") {
		pct = true
		s = strings.TrimSuffix(s, "%")
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "$")
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = "-" + s[1:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		switch strings.ToUpper(s) {
		case "TRUE":
			return 1, true
		case "FALSE":
			return 0, true
		}
		if t, ok := parseDateText(s); ok {
			return t, true
		}
		return 0, false
	}
	if pct {
		f /= 100
	}
	return f, true
}

func toText(v Value) string {
	switch v.Kind {
	case Text:
		return v.Str
	case Number:
		return formatNum(v.Num)
	case Bool:
		if v.Bool {
			return "TRUE"
		}
		return "FALSE"
	case Error:
		return v.Str
	case Array:
		return toText(v.At(0, 0))
	}
	return ""
}

func formatNum(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'g', 15, 64)
	if strings.Contains(s, "e") {
		return strconv.FormatFloat(f, 'E', -1, 64)
	}
	return s
}

func toBool(v Value) (bool, Value) {
	switch v.Kind {
	case Bool:
		return v.Bool, Value{}
	case Number:
		return v.Num != 0, Value{}
	case Empty:
		return false, Value{}
	case Text:
		switch strings.ToUpper(strings.TrimSpace(v.Str)) {
		case "TRUE":
			return true, Value{}
		case "FALSE":
			return false, Value{}
		}
		f, ok := parseNumber(v.Str)
		if ok {
			return f != 0, Value{}
		}
		return false, errValue
	case Error:
		return false, v
	case Array:
		return toBool(v.At(0, 0))
	}
	return false, errValue
}

// compare orders two scalar values as Excel does: numbers < text <
// booleans; text comparison is case-insensitive. Returns -1, 0, 1.
func compare(a, b Value) int {
	rank := func(v Value) int {
		switch v.Kind {
		case Number, Empty:
			return 0
		case Text:
			return 1
		case Bool:
			return 2
		}
		return 3
	}
	if a.Kind == Empty && b.Kind == Text {
		a = Str("")
	}
	if b.Kind == Empty && a.Kind == Text {
		b = Str("")
	}
	ra, rb := rank(a), rank(b)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}
	switch ra {
	case 0:
		x, y := a.Num, b.Num
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	case 1:
		return strings.Compare(strings.ToLower(a.Str), strings.ToLower(b.Str))
	case 2:
		if a.Bool == b.Bool {
			return 0
		}
		if !a.Bool {
			return -1
		}
		return 1
	}
	return strings.Compare(a.Str, b.Str)
}
