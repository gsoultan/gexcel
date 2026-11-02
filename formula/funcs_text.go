// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gsoultan/gexcel/format"
)

func init() {
	register(map[string]Func{
		"CONCATENATE": fnConcat,
		"CONCAT":      fnConcat,
		"TEXTJOIN":    fnTextJoin,
		"LEFT":        fnLeftRight(true),
		"RIGHT":       fnLeftRight(false),
		"MID":         fnMid,
		"LEN":         fnLen,
		"LOWER":       text1(strings.ToLower),
		"UPPER":       text1(func(s string) string { return strings.ToUpper(strings.ReplaceAll(s, "ß", "SS")) }),
		"PROPER":      text1(proper),
		"TRIM":        text1(trimExcel),
		"CLEAN":       text1(clean),
		"SUBSTITUTE":  fnSubstitute,
		"REPLACE":     fnReplace,
		"FIND":        fnFindSearch(true),
		"SEARCH":      fnFindSearch(false),
		"REPT":        fnRept,
		"TEXT":        fnText,
		"VALUE":       fnValue,
		"NUMBERVALUE": fnValue,
		"EXACT":       fnExact,
		"CHAR":        fnChar,
		"UNICHAR":     fnChar,
		"CODE":        fnCode,
		"UNICODE":     fnCode,
		"T":           fnT,
		"TEXTBEFORE":  fnTextBeforeAfter(true),
		"TEXTAFTER":   fnTextBeforeAfter(false),
		"TEXTSPLIT":   fnTextSplit,
		"FIXED":       fnFixed,
		"DOLLAR":      fnDollar,
	})
}

func text1(f func(string) string) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
			if v.Kind == Error {
				return v
			}
			return Str(f(toText(v)))
		})
	}
}

func proper(s string) string {
	var b strings.Builder
	prevLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			if prevLetter {
				b.WriteRune(unicode.ToLower(r))
			} else {
				b.WriteRune(unicode.ToUpper(r))
			}
			prevLetter = true
		} else {
			b.WriteRune(r)
			prevLetter = false
		}
	}
	return b.String()
}

func trimExcel(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, " ", " ")), " ")
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, s)
}

func fnConcat(e *Evaluator, args []Node) Value {
	var b strings.Builder
	for i := range args {
		v := e.Arg(args, i)
		var err Value
		each(v, func(x Value) bool {
			if x.Kind == Error {
				err = x
				return false
			}
			b.WriteString(toText(x))
			return true
		})
		if err.Kind == Error {
			return err
		}
	}
	return Str(b.String())
}

func fnTextJoin(e *Evaluator, args []Node) Value {
	sep, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	skipEmpty, err := e.boolean(args, 1)
	if err.Kind == Error {
		return err
	}
	var parts []string
	for i := 2; i < len(args); i++ {
		var e2 Value
		each(e.Arg(args, i), func(x Value) bool {
			if x.Kind == Error {
				e2 = x
				return false
			}
			s := toText(x)
			if skipEmpty && s == "" {
				return true
			}
			parts = append(parts, s)
			return true
		})
		if e2.Kind == Error {
			return e2
		}
	}
	return Str(strings.Join(parts, sep))
}

func fnLeftRight(left bool) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.zip(e.Arg(args, 0), e.argOr(args, 1, Num(1)), func(a, b Value) Value {
			if a.Kind == Error {
				return a
			}
			n, err := toNumber(b)
			if err.Kind == Error {
				return err
			}
			if n < 0 {
				return errValue
			}
			r := []rune(toText(a))
			k := int(n)
			if k > len(r) {
				k = len(r)
			}
			if left {
				return Str(string(r[:k]))
			}
			return Str(string(r[len(r)-k:]))
		})
	}
}

func (e *Evaluator) argOr(args []Node, i int, def Value) Value {
	v := e.Arg(args, i)
	if v.Kind == Empty {
		return def
	}
	return v
}

func fnMid(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	start, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	n, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	if start < 1 || n < 0 {
		return errValue
	}
	r := []rune(s)
	i := int(start) - 1
	if i >= len(r) {
		return Str("")
	}
	j := i + int(n)
	if j > len(r) {
		j = len(r)
	}
	return Str(string(r[i:j]))
}

func fnLen(e *Evaluator, args []Node) Value {
	return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
		if v.Kind == Error {
			return v
		}
		return Num(float64(utf8.RuneCountInString(toText(v))))
	})
}

func fnSubstitute(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	old, err := e.text(args, 1)
	if err.Kind == Error {
		return err
	}
	nw, err := e.text(args, 2)
	if err.Kind == Error {
		return err
	}
	if old == "" {
		return Str(s)
	}
	if len(args) > 3 {
		inst, err := e.num(args, 3)
		if err.Kind == Error {
			return err
		}
		if inst < 1 {
			return errValue
		}
		idx := 0
		for k := 1; ; k++ {
			p := strings.Index(s[idx:], old)
			if p < 0 {
				return Str(s)
			}
			if k == int(inst) {
				return Str(s[:idx+p] + nw + s[idx+p+len(old):])
			}
			idx += p + len(old)
		}
	}
	return Str(strings.ReplaceAll(s, old, nw))
}

func fnReplace(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	start, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	n, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	nw, err := e.text(args, 3)
	if err.Kind == Error {
		return err
	}
	if start < 1 || n < 0 {
		return errValue
	}
	r := []rune(s)
	i := int(start) - 1
	if i > len(r) {
		i = len(r)
	}
	j := i + int(n)
	if j > len(r) {
		j = len(r)
	}
	return Str(string(r[:i]) + nw + string(r[j:]))
}

func fnFindSearch(caseSensitive bool) Func {
	return func(e *Evaluator, args []Node) Value {
		needle, err := e.text(args, 0)
		if err.Kind == Error {
			return err
		}
		hay, err := e.text(args, 1)
		if err.Kind == Error {
			return err
		}
		start, err := e.numDefault(args, 2, 1)
		if err.Kind == Error {
			return err
		}
		hr := []rune(hay)
		if start < 1 || int(start) > len(hr)+1 {
			return errValue
		}
		off := int(start) - 1
		sub := string(hr[off:])
		var idx int
		if caseSensitive {
			idx = strings.Index(sub, needle)
		} else {
			// wildcard search
			nl, sl := []rune(strings.ToLower(needle)), []rune(strings.ToLower(sub))
			idx = -1
			for i := 0; i <= len(sl); i++ {
				if wildcardPrefix(string(nl), string(sl[i:])) {
					idx = len(string(sl[:i]))
					break
				}
			}
		}
		if idx < 0 {
			return errValue
		}
		return Num(float64(off + utf8.RuneCountInString(sub[:idx]) + 1))
	}
}

// wildcardPrefix reports whether pat matches a prefix of s.
func wildcardPrefix(pat, s string) bool {
	if !strings.ContainsAny(pat, "*?~") {
		return strings.HasPrefix(s, pat)
	}
	for i := 0; i <= len(s); i++ {
		if wildcardMatch(pat, s[:i]) {
			return true
		}
	}
	return false
}

func fnRept(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	n, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	if n < 0 || float64(len(s))*n > 32767 {
		return errValue
	}
	return Str(strings.Repeat(s, int(n)))
}

func fnText(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind == Error {
		return v
	}
	code, err := e.text(args, 1)
	if err.Kind == Error {
		return err
	}
	switch v.Kind {
	case Number:
		return Str(format.Number(v.Num, code, e.date1904()))
	case Bool:
		return Str(toText(v))
	}
	f, ok := parseNumber(toText(v))
	if ok {
		return Str(format.Number(f, code, e.date1904()))
	}
	return Str(format.Text(toText(v), code))
}

func fnValue(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind == Error {
		return v
	}
	if v.Kind == Number {
		return v
	}
	f, ok := parseNumber(toText(v))
	if !ok {
		return errValue
	}
	return Num(f)
}

func fnExact(e *Evaluator, args []Node) Value {
	a, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	b, err := e.text(args, 1)
	if err.Kind == Error {
		return err
	}
	return Boolean(a == b)
}

func fnChar(e *Evaluator, args []Node) Value {
	n, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	if n < 1 || n > 1114111 {
		return errValue
	}
	return Str(string(rune(int(n))))
}

func fnCode(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	if s == "" {
		return errValue
	}
	r, _ := utf8.DecodeRuneInString(s)
	return Num(float64(r))
}

func fnT(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind == Text {
		return v
	}
	if v.Kind == Error {
		return v
	}
	return Str("")
}

func fnTextBeforeAfter(before bool) Func {
	return func(e *Evaluator, args []Node) Value {
		s, err := e.text(args, 0)
		if err.Kind == Error {
			return err
		}
		delim, err := e.text(args, 1)
		if err.Kind == Error {
			return err
		}
		inst, err := e.numDefault(args, 2, 1)
		if err.Kind == Error {
			return err
		}
		matchMode, _ := e.numDefault(args, 3, 0)
		hay, d := s, delim
		if matchMode != 0 {
			hay, d = strings.ToLower(s), strings.ToLower(delim)
		}
		if d == "" {
			if before {
				return Str("")
			}
			return Str(s)
		}
		n := int(inst)
		if n == 0 {
			return errValue
		}
		idx := -1
		if n > 0 {
			pos := 0
			for k := 0; k < n; k++ {
				p := strings.Index(hay[pos:], d)
				if p < 0 {
					idx = -1
					break
				}
				idx = pos + p
				pos = idx + len(d)
			}
		} else {
			pos := len(hay)
			for k := 0; k < -n; k++ {
				p := strings.LastIndex(hay[:pos], d)
				if p < 0 {
					idx = -1
					break
				}
				idx = p
				pos = idx
			}
		}
		if idx < 0 {
			if len(args) > 5 {
				return e.Arg(args, 5)
			}
			return errNA
		}
		if before {
			return Str(s[:idx])
		}
		return Str(s[idx+len(d):])
	}
}

func fnTextSplit(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	colDelim, err := e.text(args, 1)
	if err.Kind == Error {
		return err
	}
	rowDelim := ""
	if len(args) > 2 {
		rowDelim, err = e.text(args, 2)
		if err.Kind == Error {
			return err
		}
	}
	ignoreEmpty := false
	if len(args) > 3 {
		ignoreEmpty, _ = e.boolean(args, 3)
	}
	lines := []string{s}
	if rowDelim != "" {
		lines = strings.Split(s, rowDelim)
	}
	var rows [][]Value
	width := 0
	for _, line := range lines {
		cells := []string{line}
		if colDelim != "" {
			cells = strings.Split(line, colDelim)
		}
		var row []Value
		for _, c := range cells {
			if ignoreEmpty && c == "" {
				continue
			}
			if f, ok := parseNumber(c); ok && c != "" {
				row = append(row, Num(f))
			} else {
				row = append(row, Str(c))
			}
		}
		if len(row) > width {
			width = len(row)
		}
		rows = append(rows, row)
	}
	for i := range rows {
		for len(rows[i]) < width {
			rows[i] = append(rows[i], errNA)
		}
	}
	return Arr(rows)
}

func fnFixed(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	d, err := e.numDefault(args, 1, 2)
	if err.Kind == Error {
		return err
	}
	noCommas := false
	if len(args) > 2 {
		noCommas, _ = e.boolean(args, 2)
	}
	code := "#,##0"
	if noCommas {
		code = "0"
	}
	if d > 0 {
		code += "." + strings.Repeat("0", int(d))
	} else if d < 0 {
		x = roundHalfAway(x, int(d))
	}
	return Str(format.Number(x, code, false))
}

func fnDollar(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	d, err := e.numDefault(args, 1, 2)
	if err.Kind == Error {
		return err
	}
	code := `"$"#,##0`
	if d > 0 {
		code += "." + strings.Repeat("0", int(d))
	} else if d < 0 {
		x = roundHalfAway(x, int(d))
	}
	return Str(format.Number(x, code+`;("$"`+strings.TrimPrefix(code, `"$"`)+`)`, false))
}
