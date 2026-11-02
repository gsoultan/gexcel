// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package format

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Number renders a number with a format code, as Excel displays it in the
// en-US locale. date1904 selects the workbook date system for date codes.
func Number(v float64, code string, date1904 bool) string {
	f := Parse(code)
	return f.Number(v, date1904)
}

// Text renders a string with a format code: the text section (fourth, or
// "@") applies; other formats show the string unchanged.
func Text(s, code string) string { return Parse(code).Text(s) }

// Bool renders a boolean as Excel shows it.
func Bool(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

// Format is a parsed number format with up to four sections.
type Format struct {
	Code     string
	Sections []Section
}

// Section is one ';'-separated part of a format.
type Section struct {
	Color     string // "Red", "Blue", ... or ""
	Cond      string // ">", ">=", "<", "<=", "=", "<>" or ""
	CondValue float64
	Tokens    []Token
	IsDate    bool
	IsText    bool // contains @
	Percent   int  // number of % signs
	Scale     int  // trailing commas (÷1000 each)
	Grouping  bool
	IntDigits []Token // digit placeholders before the decimal point
	DecDigits []Token // after it
	ExpDigits []Token // exponent digits
	HasDec    bool
	HasExp    bool
	ExpPlus   bool
	Fraction  bool
	FracNum   []Token // numerator placeholders
	FracDen   []Token // denominator placeholders
	FracFixed int     // fixed denominator, 0 if not fixed
	General   bool
	Elapsed   byte // 'h', 'm' or 's' for [h] style elapsed time
	HasAMPM   bool
}

// TokenKind classifies format tokens.
type TokenKind uint8

// Token kinds.
const (
	TokLiteral TokenKind = iota // literal text
	TokDigit                    // 0, # or ?
	TokDecimal                  // .
	TokPercent                  // %
	TokExp                      // E+ / E-
	TokSlash                    // / in fractions
	TokText                     // @
	TokGeneral                  // General
	TokDate                     // date/time element: Text holds the code (yyyy, mm, h, AM/PM...)
	TokFill                     // *x : repeat (rendered as nothing)
	TokSpace                    // _x : a space of x's width (rendered as a space)
	TokComma                    // thousands separator or scaling comma
)

// Token is one element of a format section.
type Token struct {
	Kind TokenKind
	Text string
}

var colorNames = map[string]bool{"black": true, "blue": true, "cyan": true, "green": true, "magenta": true, "red": true, "white": true, "yellow": true}

// Parse parses a number format code. It never fails: unknown constructs
// become literals.
func Parse(code string) *Format {
	f := &Format{Code: code}
	if code == "" || strings.EqualFold(code, "General") {
		f.Sections = []Section{{General: true, Tokens: []Token{{Kind: TokGeneral}}}}
		return f
	}
	for _, part := range splitSections(code) {
		f.Sections = append(f.Sections, parseSection(part))
	}
	if len(f.Sections) == 0 {
		f.Sections = []Section{{General: true, Tokens: []Token{{Kind: TokGeneral}}}}
	}
	return f
}

func splitSections(code string) []string {
	var parts []string
	var b strings.Builder
	inQuote, inBracket := false, false
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case inQuote:
			b.WriteByte(c)
			if c == '"' {
				inQuote = false
			}
		case inBracket:
			b.WriteByte(c)
			if c == ']' {
				inBracket = false
			}
		case c == '"':
			inQuote = true
			b.WriteByte(c)
		case c == '[':
			inBracket = true
			b.WriteByte(c)
		case c == '\\' && i+1 < len(code):
			b.WriteByte(c)
			b.WriteByte(code[i+1])
			i++
		case c == ';':
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	parts = append(parts, b.String())
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return parts
}

func parseSection(code string) Section {
	var s Section
	seenDec, seenExp, seenSlash := false, false, false
	runes := []rune(code)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '"':
			j := i + 1
			for j < len(runes) && runes[j] != '"' {
				j++
			}
			s.Tokens = append(s.Tokens, Token{TokLiteral, string(runes[i+1 : min(j, len(runes))])})
			i = j
		case r == '\\':
			if i+1 < len(runes) {
				s.Tokens = append(s.Tokens, Token{TokLiteral, string(runes[i+1])})
				i++
			}
		case r == '_':
			if i+1 < len(runes) {
				s.Tokens = append(s.Tokens, Token{TokSpace, string(runes[i+1])})
				i++
			}
		case r == '*':
			if i+1 < len(runes) {
				s.Tokens = append(s.Tokens, Token{TokFill, string(runes[i+1])})
				i++
			}
		case r == '[':
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			inner := string(runes[i+1 : min(j, len(runes))])
			i = j
			s.parseBracket(inner)
		case r == '0' || r == '#' || r == '?':
			t := Token{TokDigit, string(r)}
			s.Tokens = append(s.Tokens, t)
			switch {
			case seenExp:
				s.ExpDigits = append(s.ExpDigits, t)
			case seenSlash:
				s.FracDen = append(s.FracDen, t)
			case seenDec:
				s.DecDigits = append(s.DecDigits, t)
			default:
				s.IntDigits = append(s.IntDigits, t)
			}
		case r == '.':
			if !seenDec && !s.IsDate {
				seenDec = true
				s.HasDec = true
			}
			if s.IsDate {
				s.Tokens = append(s.Tokens, Token{TokLiteral, "."})
			} else {
				s.Tokens = append(s.Tokens, Token{TokDecimal, "."})
			}
		case r == ',':
			if s.IsDate {
				s.Tokens = append(s.Tokens, Token{TokLiteral, ","})
			} else {
				s.Tokens = append(s.Tokens, Token{TokComma, ","})
			}
		case r == '%':
			s.Percent++
			s.Tokens = append(s.Tokens, Token{TokPercent, "%"})
		case (r == 'E' || r == 'e') && i+1 < len(runes) && (runes[i+1] == '+' || runes[i+1] == '-') && !s.IsDate:
			seenExp = true
			s.HasExp = true
			s.ExpPlus = runes[i+1] == '+'
			s.Tokens = append(s.Tokens, Token{TokExp, string(runes[i : i+2])})
			i++
		case r == '/' && !s.IsDate && len(s.IntDigits) > 0:
			seenSlash = true
			s.Fraction = true
			// Digits before the slash (after the last space) are the numerator.
			s.Tokens = append(s.Tokens, Token{TokSlash, "/"})
			// A fixed denominator: digits following the slash.
			j := i + 1
			for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
				j++
			}
			if j > i+1 {
				s.FracFixed, _ = strconv.Atoi(string(runes[i+1 : j]))
				s.Tokens = append(s.Tokens, Token{TokLiteral, string(runes[i+1 : j])})
				i = j - 1
			}
		case r == '@':
			s.IsText = true
			s.Tokens = append(s.Tokens, Token{TokText, "@"})
		case r == 'G' || r == 'g':
			if len(runes)-i >= 7 && strings.EqualFold(string(runes[i:i+7]), "General") {
				s.General = true
				s.Tokens = append(s.Tokens, Token{TokGeneral, "General"})
				i += 6
			} else {
				s.Tokens = append(s.Tokens, Token{TokLiteral, string(r)})
			}
		case isDateLetter(r):
			// AM/PM and A/P
			if (r == 'A' || r == 'a') && len(runes)-i >= 5 && strings.EqualFold(string(runes[i:i+5]), "AM/PM") {
				s.Tokens = append(s.Tokens, Token{TokDate, "AM/PM"})
				s.HasAMPM = true
				s.IsDate = true
				i += 4
				continue
			}
			if (r == 'A' || r == 'a') && len(runes)-i >= 3 && strings.EqualFold(string(runes[i:i+3]), "A/P") {
				s.Tokens = append(s.Tokens, Token{TokDate, "A/P"})
				s.HasAMPM = true
				s.IsDate = true
				i += 2
				continue
			}
			if r == 'A' || r == 'a' {
				s.Tokens = append(s.Tokens, Token{TokLiteral, string(r)})
				continue
			}
			j := i
			lr := unicode.ToLower(r)
			for j < len(runes) && unicode.ToLower(runes[j]) == lr {
				j++
			}
			s.IsDate = true
			s.Tokens = append(s.Tokens, Token{TokDate, strings.ToLower(string(runes[i:j]))})
			i = j - 1
		case r == '-' || r == '+' || r == '(' || r == ')' || r == ':' || r == ' ' || r == '/' || r == '$' || r == '!' || r == '&' || r == '~' || r == '{' || r == '}' || r == '<' || r == '>' || r == '=' || r == '^' || r == '\'':
			s.Tokens = append(s.Tokens, Token{TokLiteral, string(r)})
		default:
			s.Tokens = append(s.Tokens, Token{TokLiteral, string(r)})
		}
	}
	// Decide the fraction numerator: the digit run right before the slash.
	if s.Fraction {
		var num []Token
		for i := len(s.Tokens) - 1; i >= 0; i-- {
			if s.Tokens[i].Kind == TokSlash {
				for j := i - 1; j >= 0 && s.Tokens[j].Kind == TokDigit; j-- {
					num = append([]Token{s.Tokens[j]}, num...)
				}
				break
			}
		}
		s.FracNum = num
		s.IntDigits = s.IntDigits[:len(s.IntDigits)-len(num)]
	}
	// Grouping and scaling commas: a comma between digit placeholders groups
	// thousands; commas right after the last placeholder divide by 1000 each.
	if !s.IsDate {
		lastDigit := -1
		for i, t := range s.Tokens {
			if t.Kind == TokDigit || t.Kind == TokDecimal {
				lastDigit = i
			}
		}
		for i, t := range s.Tokens {
			if t.Kind != TokComma {
				continue
			}
			if i > lastDigit {
				if i == lastDigit+1 || s.Tokens[i-1].Kind == TokComma {
					s.Scale++
				}
				continue
			}
			if i > 0 && s.Tokens[i-1].Kind == TokDigit {
				s.Grouping = true
			}
		}
	}
	return s
}

func (s *Section) parseBracket(inner string) {
	low := strings.ToLower(inner)
	switch {
	case colorNames[low]:
		s.Color = inner
	case strings.HasPrefix(low, "color"):
		s.Color = inner
	case strings.HasPrefix(low, "$"):
		// Locale/currency: [$€-407]; keep the currency symbol as a literal.
		sym := inner[1:]
		if i := strings.IndexByte(sym, '-'); i >= 0 {
			sym = sym[:i]
		}
		if sym != "" {
			s.Tokens = append(s.Tokens, Token{TokLiteral, sym})
		}
	case low == "h" || low == "hh" || low == "m" || low == "mm" || low == "s" || low == "ss":
		s.Elapsed = low[0]
		s.IsDate = true
		s.Tokens = append(s.Tokens, Token{TokDate, "[" + low + "]"})
	case strings.HasPrefix(low, "dbnum"), strings.HasPrefix(low, "natnum"), low == "t", low == "lt":
	default:
		for _, op := range []string{">=", "<=", "<>", ">", "<", "="} {
			if strings.HasPrefix(inner, op) {
				v, err := strconv.ParseFloat(strings.TrimSpace(inner[len(op):]), 64)
				if err == nil {
					s.Cond, s.CondValue = op, v
				}
				return
			}
		}
	}
}

func isDateLetter(r rune) bool {
	switch unicode.ToLower(r) {
	case 'y', 'm', 'd', 'h', 's', 'a', 'b', 'e', 'g':
		return r != 'e' && r != 'E' && r != 'g' && r != 'G' // e/g handled elsewhere
	}
	return false
}

// IsDate reports whether the first numeric section shows a date or time.
func (f *Format) IsDate() bool {
	for _, s := range f.Sections {
		if s.IsText {
			continue
		}
		return s.IsDate
	}
	return false
}

func (s *Section) matches(v float64) bool {
	switch s.Cond {
	case ">":
		return v > s.CondValue
	case ">=":
		return v >= s.CondValue
	case "<":
		return v < s.CondValue
	case "<=":
		return v <= s.CondValue
	case "=":
		return v == s.CondValue
	case "<>":
		return v != s.CondValue
	}
	return true
}

// section picks the section for a value and says whether the sign must be
// rendered by the caller (true when the negative section is implicit).
func (f *Format) section(v float64) (*Section, bool) {
	secs := f.Sections
	var numeric []*Section
	for i := range secs {
		if !secs[i].IsText || len(secs) == 1 {
			numeric = append(numeric, &secs[i])
		}
	}
	if len(numeric) == 0 {
		return &secs[0], v < 0
	}
	// Conditional sections.
	hasCond := false
	for _, s := range numeric {
		if s.Cond != "" {
			hasCond = true
		}
	}
	if hasCond {
		for _, s := range numeric {
			if s.Cond != "" && s.matches(v) {
				return s, v < 0
			}
		}
		for _, s := range numeric {
			if s.Cond == "" {
				return s, v < 0
			}
		}
		return numeric[len(numeric)-1], v < 0
	}
	switch {
	case len(numeric) == 1:
		return numeric[0], v < 0
	case v > 0 || len(numeric) == 2 && v == 0:
		return numeric[0], false
	case v < 0:
		return numeric[1], false
	default:
		return numeric[2], false
	}
}

// Number renders a number through the format.
func (f *Format) Number(v float64, date1904 bool) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "#NUM!"
	}
	sec, implicitSign := f.section(v)
	if sec.IsText && len(f.Sections) > 1 {
		return general(v)
	}
	neg := v < 0
	av := math.Abs(v)
	var out string
	switch {
	case sec.General:
		out = renderWith(sec, general(av), nil)
	case sec.IsDate:
		out = sec.date(v, date1904)
		implicitSign = false
	case sec.Fraction:
		out = sec.fraction(av)
	default:
		out = sec.number(av)
	}
	if neg && implicitSign && !sec.IsDate {
		out = "-" + out
	}
	return out
}

// Text renders a string through the format's text section.
func (f *Format) Text(s string) string {
	var sec *Section
	for i := range f.Sections {
		if f.Sections[i].IsText {
			sec = &f.Sections[i]
			break
		}
	}
	if sec == nil {
		if len(f.Sections) == 4 {
			sec = &f.Sections[3]
		} else {
			return s
		}
	}
	var b strings.Builder
	for _, t := range sec.Tokens {
		switch t.Kind {
		case TokText:
			b.WriteString(s)
		case TokLiteral:
			b.WriteString(t.Text)
		case TokSpace:
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// general renders a number as Excel's General format does: up to 11
// characters, switching to scientific notation for very large or small values.
func general(v float64) string {
	if v == 0 {
		return "0"
	}
	av := math.Abs(v)
	if av >= 1e11 || av < 1e-4 {
		s := strconv.FormatFloat(v, 'E', 5, 64)
		// Trim trailing zeros in the mantissa: 1.00000E+11 -> 1E+11
		mant, exp, _ := strings.Cut(s, "E")
		if strings.Contains(mant, ".") {
			mant = strings.TrimRight(strings.TrimRight(mant, "0"), ".")
		}
		if len(exp) > 0 && (exp[0] == '+' || exp[0] == '-') {
			sign := exp[0]
			digits := strings.TrimLeft(exp[1:], "0")
			if len(digits) < 2 {
				digits = strings.Repeat("0", 2-len(digits)) + digits
			}
			exp = string(sign) + digits
		}
		return mant + "E" + exp
	}
	// Up to 10 significant digits for integers... Excel shows up to 11 digits total.
	intDigits := 1
	if av >= 1 {
		intDigits = int(math.Floor(math.Log10(av))) + 1
	}
	dec := max(0, 10-intDigits)
	s := strconv.FormatFloat(v, 'f', dec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// General renders a number with the General format.
func General(v float64) string { return general(v) }

// number renders av (non-negative) with the digit placeholders of the section.
func (s *Section) number(av float64) string {
	if s.Percent > 0 {
		av *= math.Pow(100, float64(s.Percent))
	}
	if s.Scale > 0 {
		av /= math.Pow(1000, float64(s.Scale))
	}
	dec := len(s.DecDigits)
	var intPart, decPart, expPart string
	if s.HasExp {
		exp := 0
		if av != 0 {
			exp = int(math.Floor(math.Log10(av)))
			// Engineering style: ##0.0E+0 groups exponent by the integer digit count.
			n := len(s.IntDigits)
			if n > 1 {
				exp = int(math.Floor(float64(exp)/float64(n))) * n
			}
		}
		mant := av / math.Pow(10, float64(exp))
		str := fixed(mant, dec)
		if ip, _, _ := strings.Cut(str, "."); len(ip) > max(1, len(s.IntDigits)) {
			exp += len(ip) - max(1, len(s.IntDigits))
			str = fixed(av/math.Pow(10, float64(exp)), dec)
		}
		intPart, decPart, _ = strings.Cut(str, ".")
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		} else if !s.ExpPlus {
			sign = ""
		}
		expStr := strconv.Itoa(exp)
		for len(expStr) < len(s.ExpDigits) {
			expStr = "0" + expStr
		}
		expPart = sign + expStr
	} else {
		str := fixed(av, dec)
		intPart, decPart, _ = strings.Cut(str, ".")
	}
	if intPart == "0" && len(s.IntDigits) > 0 && s.IntDigits[len(s.IntDigits)-1].Text != "0" {
		// "#" and "?" show nothing for a zero integer part.
		if s.IntDigits[len(s.IntDigits)-1].Text == "?" {
			intPart = " "
		} else {
			intPart = ""
		}
	}
	// Pad the integer part to the number of leading 0/? placeholders.
	lead := 0
	for _, t := range s.IntDigits {
		if t.Text == "0" || t.Text == "?" {
			lead++
		}
	}
	for len(strings.TrimSpace(intPart)) < lead && len(intPart) < len(s.IntDigits) {
		if s.IntDigits[len(s.IntDigits)-1-len(intPart)].Text == "?" {
			intPart = " " + intPart
		} else {
			intPart = "0" + intPart
		}
	}
	if s.Grouping {
		intPart = group(intPart)
	}
	// Decimal digits: trim trailing zeros for # placeholders, spaces for ?.
	decPart = trimDec(decPart, s.DecDigits)
	return renderWith(s, intPart, func(b *strings.Builder) {
		if s.HasDec {
			if decPart != "" || hasZeroDigit(s.DecDigits) {
				b.WriteByte('.')
				b.WriteString(decPart)
			} else if decPart == "" && !hasZeroDigit(s.DecDigits) && len(s.DecDigits) > 0 {
				// "0.##" with integer value: Excel shows "5." — keep the point.
				b.WriteByte('.')
			}
		}
		if s.HasExp {
			b.WriteByte('E')
			b.WriteString(expPart)
		}
	})
}

func hasZeroDigit(ts []Token) bool {
	for _, t := range ts {
		if t.Text == "0" {
			return true
		}
	}
	return false
}

func trimDec(dec string, ph []Token) string {
	b := []byte(dec)
	for i := len(b) - 1; i >= 0 && i < len(ph); i-- {
		if b[i] != '0' {
			break
		}
		switch ph[i].Text {
		case "#":
			b = b[:i]
		case "?":
			b[i] = ' '
		default:
			return string(b)
		}
	}
	return string(b)
}

// renderWith writes the section's literal tokens around the integer digits
// and the decimal/exponent part produced by after.
func renderWith(s *Section, intPart string, after func(*strings.Builder)) string {
	var b strings.Builder
	intWritten := false
	for i, t := range s.Tokens {
		switch t.Kind {
		case TokLiteral:
			b.WriteString(t.Text)
		case TokSpace:
			b.WriteByte(' ')
		case TokFill:
		case TokGeneral:
			b.WriteString(intPart)
		case TokDigit:
			if !intWritten {
				b.WriteString(intPart)
				intWritten = true
			}
			// skip remaining integer placeholders
			if isDecPlaceholder(s, i) {
				// decimal placeholders are handled by after()
			}
		case TokComma:
		case TokDecimal:
			if after != nil {
				after(&b)
				after = nil
			}
			// skip the decimal digit tokens that follow
		case TokExp:
			if after != nil {
				after(&b)
				after = nil
			}
		case TokPercent:
			b.WriteByte('%')
		case TokText:
		}
	}
	if after != nil {
		after(&b)
	}
	return b.String()
}

func isDecPlaceholder(s *Section, i int) bool {
	for j := i - 1; j >= 0; j-- {
		if s.Tokens[j].Kind == TokDecimal {
			return true
		}
	}
	return false
}

func group(intPart string) string {
	trimmed := strings.TrimLeft(intPart, " ")
	pad := intPart[:len(intPart)-len(trimmed)]
	if len(trimmed) <= 3 {
		return intPart
	}
	var b strings.Builder
	b.WriteString(pad)
	first := len(trimmed) % 3
	if first > 0 {
		b.WriteString(trimmed[:first])
	}
	for i := first; i < len(trimmed); i += 3 {
		if b.Len() > len(pad) {
			b.WriteByte(',')
		}
		b.WriteString(trimmed[i : i+3])
	}
	return b.String()
}

// fixed formats a non-negative value with dec decimals, rounding half away
// from zero on its 15-significant-digit decimal representation, as Excel
// does (so 1.005 with "0.00" gives 1.01, not the binary 1.00).
func fixed(v float64, dec int) string {
	if v == 0 {
		return strconv.FormatFloat(0, 'f', dec, 64)
	}
	e := strconv.FormatFloat(v, 'e', 14, 64) // d.dddddddddddddde±XX
	mant, expStr, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := []byte(strings.Replace(mant, ".", "", 1)) // 15 digits, value = 0.d... × 10^(exp+1)
	intLen := exp + 1
	keep := intLen + dec // digits kept before rounding position
	if keep < 0 {
		return strconv.FormatFloat(0, 'f', dec, 64)
	}
	// Pad so that index keep exists.
	for len(digits) <= keep {
		digits = append(digits, '0')
	}
	if digits[keep] >= '5' {
		i := keep - 1
		for i >= 0 {
			if digits[i] == '9' {
				digits[i] = '0'
				i--
				continue
			}
			digits[i]++
			break
		}
		if i < 0 {
			digits = append([]byte{'1'}, digits...)
			intLen++
			keep++
		}
	}
	digits = digits[:keep]
	var b strings.Builder
	if intLen <= 0 {
		b.WriteByte('0')
		if dec > 0 {
			b.WriteByte('.')
			b.WriteString(strings.Repeat("0", -intLen))
			b.Write(digits)
		}
		return b.String()
	}
	b.Write(digits[:intLen])
	if dec > 0 {
		b.WriteByte('.')
		b.Write(digits[intLen:])
	}
	return b.String()
}

// fraction renders av as a mixed or simple fraction.
func (s *Section) fraction(av float64) string {
	whole := 0.0
	rem := av
	if len(s.IntDigits) > 0 {
		whole = math.Floor(av)
		rem = av - whole
	}
	var num, den int
	if s.FracFixed > 0 {
		den = s.FracFixed
		num = int(math.Round(rem * float64(den)))
	} else {
		maxDen := int(math.Pow(10, float64(len(s.FracDen)))) - 1
		num, den = bestFraction(rem, maxDen)
	}
	if num == den && den > 0 {
		whole++
		num = 0
	}
	var b strings.Builder
	for i, t := range s.Tokens {
		switch t.Kind {
		case TokLiteral:
			b.WriteString(t.Text)
		case TokSpace:
			b.WriteByte(' ')
		case TokDigit:
			// Determine which group this digit belongs to.
			switch digitGroup(s, i) {
			case 0:
				if digitFirst(s, i, 0) {
					if len(s.IntDigits) > 0 {
						if whole == 0 && num != 0 {
							b.WriteString(strings.Repeat(" ", len(s.IntDigits)))
						} else {
							b.WriteString(strconv.Itoa(int(whole)))
						}
					}
				}
			case 1:
				if digitFirst(s, i, 1) {
					if num == 0 && len(s.IntDigits) > 0 {
						b.WriteString(strings.Repeat(" ", len(s.FracNum)))
					} else {
						b.WriteString(pad(strconv.Itoa(num), len(s.FracNum)))
					}
				}
			case 2:
				if digitFirst(s, i, 2) {
					if num == 0 && len(s.IntDigits) > 0 {
						b.WriteString(strings.Repeat(" ", len(s.FracDen)))
					} else {
						b.WriteString(strconv.Itoa(den))
					}
				}
			}
		case TokSlash:
			if num == 0 && len(s.IntDigits) > 0 {
				b.WriteByte(' ')
			} else {
				b.WriteByte('/')
			}
		}
	}
	return b.String()
}

func pad(s string, n int) string {
	for len(s) < n {
		s = " " + s
	}
	return s
}

// digitGroup says whether digit token i is in the whole part (0),
// numerator (1) or denominator (2).
func digitGroup(s *Section, i int) int {
	slash := -1
	for j, t := range s.Tokens {
		if t.Kind == TokSlash {
			slash = j
			break
		}
	}
	if i > slash {
		return 2
	}
	// Numerator: contiguous digits right before the slash.
	j := slash - 1
	for j >= 0 && s.Tokens[j].Kind == TokDigit {
		j--
	}
	if i > j {
		return 1
	}
	return 0
}

func digitFirst(s *Section, i, grp int) bool {
	for j := i - 1; j >= 0; j-- {
		if s.Tokens[j].Kind == TokDigit {
			return digitGroup(s, j) != grp
		}
	}
	return true
}

// bestFraction finds the fraction closest to v with a denominator up to maxDen
// (Stern-Brocot / continued fractions).
func bestFraction(v float64, maxDen int) (int, int) {
	if maxDen < 1 {
		maxDen = 1
	}
	bestN, bestD := 0, 1
	bestErr := v
	for d := 1; d <= maxDen; d++ {
		n := int(math.Round(v * float64(d)))
		err := math.Abs(v - float64(n)/float64(d))
		if err < bestErr-1e-12 {
			bestN, bestD, bestErr = n, d, err
			if err == 0 {
				break
			}
		}
	}
	return bestN, bestD
}

var monthNames = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
var dayNames = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// date renders a serial number through a date/time section.
func (s *Section) date(v float64, date1904 bool) string {
	// Round to the precision the format shows so 0.99999 seconds show as the next unit.
	hasSec := false
	fracDigits := 0
	for i, t := range s.Tokens {
		if t.Kind == TokDate && (t.Text[0] == 's' || t.Text == "[s]" || t.Text == "[ss]") {
			hasSec = true
			if i+2 < len(s.Tokens) && s.Tokens[i+1].Kind == TokLiteral && s.Tokens[i+1].Text == "." && s.Tokens[i+2].Kind == TokDigit {
				for j := i + 2; j < len(s.Tokens) && s.Tokens[j].Kind == TokDigit; j++ {
					fracDigits++
				}
			}
		}
	}
	unit := 1.0 / 1440 // minutes
	if hasSec {
		unit = 1.0 / 86400
		if fracDigits > 0 {
			unit /= math.Pow(10, float64(fracDigits))
		}
	}
	if s.Elapsed == 0 {
		v = math.Floor(v/unit+0.5) * unit
	}
	neg := v < 0
	if neg && s.Elapsed == 0 {
		return strings.Repeat("#", 7)
	}
	av := math.Abs(v)
	t := FromSerial(av, date1904)
	days := math.Floor(av)
	dayFrac := av - days
	totalSec := math.Floor(dayFrac*86400 + 0.5*unit*86400)
	hour, minute, sec := int(totalSec)/3600, int(totalSec)/60%60, int(totalSec)%60
	fracSec := dayFrac*86400 - math.Floor(dayFrac*86400)
	if s.Elapsed != 0 {
		hour += int(days) * 24
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	skipDigits := 0
	for i, tok := range s.Tokens {
		switch tok.Kind {
		case TokLiteral:
			if tok.Text == "." && i+1 < len(s.Tokens) && s.Tokens[i+1].Kind == TokDigit && i > 0 && s.Tokens[i-1].Kind == TokDate {
				// fractional seconds
				n := 0
				for j := i + 1; j < len(s.Tokens) && s.Tokens[j].Kind == TokDigit; j++ {
					n++
				}
				b.WriteByte('.')
				fs := strconv.FormatFloat(fracSec, 'f', n, 64)
				b.WriteString(fs[2:])
				skipDigits = n
				continue
			}
			b.WriteString(tok.Text)
		case TokSpace:
			b.WriteByte(' ')
		case TokDigit:
			if skipDigits > 0 {
				skipDigits--
			}
		case TokDate:
			switch tok.Text {
			case "yy":
				b.WriteString(two(t.Year() % 100))
			case "y", "yyy", "yyyy", "yyyyy":
				b.WriteString(strconv.Itoa(t.Year()))
			case "m":
				if s.minuteContext(i) {
					b.WriteString(strconv.Itoa(minute))
				} else {
					b.WriteString(strconv.Itoa(int(t.Month())))
				}
			case "mm":
				if s.minuteContext(i) {
					b.WriteString(two(minute))
				} else {
					b.WriteString(two(int(t.Month())))
				}
			case "mmm":
				b.WriteString(monthNames[t.Month()-1][:3])
			case "mmmm":
				b.WriteString(monthNames[t.Month()-1])
			case "mmmmm":
				b.WriteString(monthNames[t.Month()-1][:1])
			case "d":
				b.WriteString(strconv.Itoa(t.Day()))
			case "dd":
				b.WriteString(two(t.Day()))
			case "ddd":
				b.WriteString(dayNames[t.Weekday()][:3])
			case "dddd", "ddddd":
				b.WriteString(dayNames[t.Weekday()])
			case "h", "hh":
				h := hour
				if s.HasAMPM {
					h = hour % 12
					if h == 0 {
						h = 12
					}
				}
				if tok.Text == "hh" {
					b.WriteString(two(h))
				} else {
					b.WriteString(strconv.Itoa(h))
				}
			case "s":
				b.WriteString(strconv.Itoa(sec))
			case "ss":
				b.WriteString(two(sec))
			case "[h]", "[hh]":
				b.WriteString(strconv.Itoa(hour))
			case "[m]", "[mm]":
				b.WriteString(strconv.Itoa(hour*60 + minute))
			case "[s]", "[ss]":
				b.WriteString(strconv.Itoa(hour*3600 + minute*60 + sec))
			case "AM/PM":
				if hour%24 < 12 {
					b.WriteString("AM")
				} else {
					b.WriteString("PM")
				}
			case "A/P":
				if hour%24 < 12 {
					b.WriteString("A")
				} else {
					b.WriteString("P")
				}
			case "b", "bb", "bbbb":
				b.WriteString(strconv.Itoa(t.Year() + 543)) // Buddhist year
			default:
				b.WriteString(tok.Text)
			}
		}
	}
	return b.String()
}

// minuteContext reports whether an m/mm token means minutes: it follows an
// hour token or precedes a seconds token.
func (s *Section) minuteContext(i int) bool {
	for j := i - 1; j >= 0; j-- {
		if s.Tokens[j].Kind != TokDate {
			continue
		}
		c := s.Tokens[j].Text[0]
		if c == 'h' || c == '[' && s.Tokens[j].Text[1] == 'h' {
			return true
		}
		if c == 's' || c == 'd' || c == 'y' || c == 'm' {
			break
		}
	}
	for j := i + 1; j < len(s.Tokens); j++ {
		if s.Tokens[j].Kind != TokDate {
			continue
		}
		c := s.Tokens[j].Text[0]
		if c == 's' || c == '[' && s.Tokens[j].Text[1] == 's' {
			return true
		}
		break
	}
	return false
}

func two(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// Time renders a time.Time with a date format code.
func Time(t time.Time, code string) string {
	return Number(ToSerial(t, false), code, false)
}
