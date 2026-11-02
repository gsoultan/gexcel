// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package formula tokenizes, parses and evaluates Excel formulas.
package formula

import (
	"strings"

	"github.com/gsoultan/gexcel/ref"
)

// Shift moves every relative reference in a formula by the given row and
// column deltas, as Excel does when a shared formula is expanded or a
// formula is copied. Absolute parts ($) stay. Text in quotes is untouched.
func Shift(text string, rows, cols int) string {
	if rows == 0 && cols == 0 {
		return text
	}
	toks := Tokenize(text)
	var b strings.Builder
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.Kind == TokRef {
			b.WriteString(shiftRef(t.Text, rows, cols))
			continue
		}
		// Whole-row ranges (1:1) tokenize as number : number.
		if t.Kind == TokNumber && i+2 < len(toks) && toks[i+1].Kind == TokOp && toks[i+1].Text == ":" && toks[i+2].Kind == TokNumber &&
			!strings.ContainsAny(t.Text, ".eE") && !strings.ContainsAny(toks[i+2].Text, ".eE") {
			b.WriteString(shiftRef(t.Text+":"+toks[i+2].Text, rows, cols))
			i += 2
			continue
		}
		b.WriteString(t.Text)
	}
	return b.String()
}

func shiftRef(s string, rows, cols int) string {
	sheet := ""
	rest := s
	if i := strings.LastIndexByte(s, '!'); i >= 0 {
		sheet, rest = s[:i+1], s[i+1:]
	}
	r, err := ref.ParseRange(rest)
	if err != nil {
		return s
	}
	r = r.Offset(rows, cols)
	if !r.Valid() {
		return sheet + "#REF!"
	}
	return sheet + r.String()
}
