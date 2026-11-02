// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package gexcel

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"strings"
	"time"

	// Register decoders used by imageSize.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/gsoultan/gexcel/format"
	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
)

type cell struct {
	sh  *sheet
	m   *model.Cell
	err error
}

func (c *cell) Model() *model.Cell { return c.m }
func (c *cell) Sheet() Sheet       { return c.sh }
func (c *cell) Err() error         { return c.err }
func (c *cell) Row() int           { return c.m.Row }
func (c *cell) Col() int           { return c.m.Col }
func (c *cell) Ref() string        { return ref.Cell{Row: c.m.Row, Col: c.m.Col}.String() }
func (c *cell) StyleID() int       { return c.m.StyleID }
func (c *cell) Value() model.Value { return c.m.Value }
func (c *cell) IsEmpty() bool      { return c.m.Value.IsEmpty() && c.m.Formula == nil }

func (c *cell) Formula() string {
	if c.m.Formula == nil {
		return ""
	}
	return c.m.Formula.Text
}

func (c *cell) styles() *model.Styles { return c.sh.wb.m.Styles }

func (c *cell) Set(v any) Cell {
	if c.err != nil {
		return c
	}
	c.m.Formula = nil
	val, ok := ToValue(v, c.sh.wb.m.Date1904)
	if !ok {
		c.m.Value = model.String(fmt.Sprint(v))
		return c
	}
	c.m.Value = val
	switch v.(type) {
	case time.Time:
		if !c.IsDate() {
			c.SetNumFmt(style.NumFmtISODateTime)
		}
	case time.Duration:
		if c.Style().NumFmt == "" {
			c.SetNumFmt("[h]:mm:ss")
		}
	}
	return c
}

// ToValue converts a Go value to a cell value. ok is false for types that
// have no natural representation.
func ToValue(v any, date1904 bool) (model.Value, bool) {
	switch x := v.(type) {
	case nil:
		return model.Value{}, true
	case model.Value:
		return x, true
	case *model.Value:
		if x == nil {
			return model.Value{}, true
		}
		return *x, true
	case string:
		return model.String(x), true
	case []byte:
		return model.String(string(x)), true
	case bool:
		return model.Boolean(x), true
	case int:
		return model.Number(float64(x)), true
	case int8:
		return model.Number(float64(x)), true
	case int16:
		return model.Number(float64(x)), true
	case int32:
		return model.Number(float64(x)), true
	case int64:
		return model.Number(float64(x)), true
	case uint:
		return model.Number(float64(x)), true
	case uint8:
		return model.Number(float64(x)), true
	case uint16:
		return model.Number(float64(x)), true
	case uint32:
		return model.Number(float64(x)), true
	case uint64:
		return model.Number(float64(x)), true
	case float32:
		return numberOrError(float64(x)), true
	case float64:
		return numberOrError(x), true
	case time.Time:
		if x.IsZero() {
			return model.Value{}, true
		}
		return model.Date(x, date1904), true
	case *time.Time:
		if x == nil || x.IsZero() {
			return model.Value{}, true
		}
		return model.Date(*x, date1904), true
	case time.Duration:
		return model.Number(format.DurationToSerial(x)), true
	case []model.RichRun:
		return model.Rich(x...), true
	case error:
		msg := x.Error()
		if model.IsErrorCode(msg) {
			return model.Error(msg), true
		}
		return model.Error(model.ErrValue), true
	case fmt.Stringer:
		return model.String(x.String()), true
	}
	return model.Value{}, false
}

func numberOrError(f float64) model.Value {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return model.Error(model.ErrNum)
	}
	return model.Number(f)
}

func (c *cell) SetFormula(f string) Cell {
	if c.err != nil {
		return c
	}
	f = strings.TrimPrefix(strings.TrimSpace(f), "=")
	if f == "" {
		c.m.Formula = nil
		return c
	}
	if formula.IsDynamicArray(f) {
		return c.SetArrayFormula(f, c.Ref(), true)
	}
	c.m.Formula = &model.Formula{Text: f}
	c.m.Value = model.Value{}
	return c
}

func (c *cell) SetArrayFormula(f, rng string, dynamic bool) Cell {
	if c.err != nil {
		return c
	}
	r, err := ref.ParseRange(rng)
	if err != nil {
		r = ref.Range{From: ref.Cell{Row: c.m.Row, Col: c.m.Col}, To: ref.Cell{Row: c.m.Row, Col: c.m.Col}}
	}
	c.m.Formula = &model.Formula{Text: strings.TrimPrefix(strings.TrimSpace(f), "="), Kind: model.FormulaArray, Ref: r.Normalize(), Master: true, Dynamic: dynamic}
	c.m.Value = model.Value{}
	return c
}

func (c *cell) SetFormulaResult(f string, result any) Cell {
	c.SetFormula(f)
	if c.err != nil || c.m.Formula == nil {
		return c
	}
	if v, ok := ToValue(result, c.sh.wb.m.Date1904); ok {
		c.m.Value = v
	}
	return c
}

func (c *cell) SetRich(runs ...model.RichRun) Cell {
	if c.err != nil {
		return c
	}
	c.m.Formula = nil
	c.m.Value = model.Rich(runs...)
	return c
}

func (c *cell) Clear() Cell {
	c.m.Formula = nil
	c.m.Value = model.Value{}
	return c
}

func (c *cell) String() string { return c.m.Value.Text() }

func (c *cell) Formatted() string {
	return c.m.Value.Formatted(c.Style().NumFmt, c.sh.wb.m.Date1904)
}

func (c *cell) Float() (float64, bool) { return c.m.Value.Float() }

func (c *cell) Int() (int64, bool) {
	f, ok := c.m.Value.Float()
	if !ok || f != math.Trunc(f) || math.Abs(f) > math.MaxInt64 {
		return 0, false
	}
	return int64(f), true
}

func (c *cell) Bool() (bool, bool) {
	switch c.m.Value.Kind {
	case model.KindBool:
		return c.m.Value.Bool, true
	case model.KindNumber:
		return c.m.Value.Num != 0, true
	case model.KindString:
		switch strings.ToUpper(strings.TrimSpace(c.m.Value.Str)) {
		case "TRUE":
			return true, true
		case "FALSE":
			return false, true
		}
	}
	return false, false
}

func (c *cell) Time() (time.Time, bool) {
	if c.m.Value.Kind != model.KindNumber {
		return time.Time{}, false
	}
	return c.m.Value.Time(c.sh.wb.m.Date1904)
}

func (c *cell) IsDate() bool {
	code := c.Style().NumFmt
	return code != "" && style.IsDateFormat(code)
}

func (c *cell) Any() any {
	v := c.m.Value
	switch v.Kind {
	case model.KindString:
		return v.Str
	case model.KindBool:
		return v.Bool
	case model.KindError:
		return v.Str
	case model.KindNumber:
		if c.IsDate() {
			if t, ok := v.Time(c.sh.wb.m.Date1904); ok {
				return t
			}
		}
		return v.Num
	}
	return nil
}

func (c *cell) Style() style.Style { return c.styles().Get(c.m.StyleID) }

func (c *cell) SetStyle(st style.Style) Cell {
	if c.err != nil {
		return c
	}
	c.m.StyleID = c.styles().Add(st)
	return c
}

func (c *cell) MergeStyle(st style.Style) Cell {
	cur := c.Style()
	return c.SetStyle(cur.Merge(st))
}

func (c *cell) edit(fn func(*style.Style)) Cell {
	if c.err != nil {
		return c
	}
	st := c.Style()
	fn(&st)
	return c.SetStyle(st)
}

func (c *cell) SetNumFmt(code string) Cell { return c.edit(func(s *style.Style) { s.NumFmt = code }) }
func (c *cell) SetFont(f style.Font) Cell  { return c.edit(func(s *style.Style) { s.Font = f }) }
func (c *cell) SetFill(f style.Fill) Cell  { return c.edit(func(s *style.Style) { s.Fill = f }) }
func (c *cell) SetBorder(b style.Border) Cell {
	return c.edit(func(s *style.Style) { s.Border = b })
}
func (c *cell) SetAlignment(a style.Alignment) Cell {
	return c.edit(func(s *style.Style) { s.Alignment = a })
}
func (c *cell) SetBold(b bool) Cell { return c.edit(func(s *style.Style) { s.Font.Bold = b }) }
func (c *cell) SetColor(col style.Color) Cell {
	return c.edit(func(s *style.Style) { s.Font.Color = col })
}

func (c *cell) SetNamedStyle(name string) Cell {
	return c.edit(func(s *style.Style) { s.Named = name })
}

func (c *cell) SetHyperlink(target, tooltip string) Cell {
	if c.err != nil {
		return c
	}
	if strings.HasPrefix(target, "#") {
		_ = c.sh.AddInternalLink(c.Ref(), strings.TrimPrefix(target, "#"), tooltip)
	} else {
		_ = c.sh.AddHyperlink(c.Ref(), target, tooltip)
	}
	if c.m.Value.IsEmpty() {
		c.m.Value = model.String(strings.TrimPrefix(target, "#"))
	}
	if c.m.StyleID == 0 {
		c.SetStyle(style.Style{Font: style.Font{Underline: style.UnderlineSingle, Color: style.ThemeColor(style.ThemeHyperlink, 0)}, Named: "Hyperlink"})
	}
	return c
}

func (c *cell) Hyperlink() (model.Hyperlink, bool) {
	here := ref.Cell{Row: c.m.Row, Col: c.m.Col}
	for _, h := range c.sh.m.Hyperlinks {
		if h.Range.Contains(here.Row, here.Col) {
			return h, true
		}
	}
	return model.Hyperlink{}, false
}

func (c *cell) SetComment(author, text string) Cell {
	if c.err != nil {
		return c
	}
	_ = c.sh.AddComment(c.Ref(), author, text)
	return c
}

func (c *cell) Comment() (model.Comment, bool) {
	for _, cm := range c.sh.m.Comments {
		if cm.Ref.Row == c.m.Row && cm.Ref.Col == c.m.Col {
			return cm, true
		}
	}
	return model.Comment{}, false
}

func sniffImage(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG")):
		return "png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		return "jpeg"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return "gif"
	case bytes.HasPrefix(data, []byte("BM")):
		return "bmp"
	case bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 12 && string(data[8:12]) == "WEBP":
		return "webp"
	case bytes.HasPrefix(data, []byte("II*\x00")) || bytes.HasPrefix(data, []byte("MM\x00*")):
		return "tiff"
	case bytes.Contains(data[:min(len(data), 512)], []byte("<svg")):
		return "svg"
	case bytes.HasPrefix(data, []byte{0x01, 0x00, 0x00, 0x00}):
		return "emf"
	case bytes.HasPrefix(data, []byte{0xD7, 0xCD, 0xC6, 0x9A}):
		return "wmf"
	}
	return "png"
}

func imageSize(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 200, 150
	}
	return cfg.Width, cfg.Height
}
