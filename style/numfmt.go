// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package style

// Built-in number format codes (ECMA-376 18.8.30). Setting Style.NumFmt to
// one of these makes the writer use the built-in id instead of a custom
// numFmt entry.
const (
	NumFmtGeneral      = "General"
	NumFmtInt          = "0"
	NumFmtDecimal2     = "0.00"
	NumFmtThousands    = "#,##0"
	NumFmtThousands2   = "#,##0.00"
	NumFmtPercent      = "0%"
	NumFmtPercent2     = "0.00%"
	NumFmtScientific   = "0.00E+00"
	NumFmtFraction1    = "# ?/?"
	NumFmtFraction2    = "# ??/??"
	NumFmtDate         = "mm-dd-yy"
	NumFmtDateDMY      = "d-mmm-yy"
	NumFmtDateDM       = "d-mmm"
	NumFmtDateMY       = "mmm-yy"
	NumFmtTime12       = "h:mm AM/PM"
	NumFmtTime12s      = "h:mm:ss AM/PM"
	NumFmtTime24       = "h:mm"
	NumFmtTime24s      = "h:mm:ss"
	NumFmtDateTime     = "m/d/yy h:mm"
	NumFmtAccountInt   = "#,##0 ;(#,##0)"
	NumFmtAccountIntR  = "#,##0 ;[Red](#,##0)"
	NumFmtAccount2     = "#,##0.00;(#,##0.00)"
	NumFmtAccount2R    = "#,##0.00;[Red](#,##0.00)"
	NumFmtDuration     = "mm:ss"
	NumFmtDurationH    = "[h]:mm:ss"
	NumFmtDurationMs   = "mmss.0"
	NumFmtScientific1  = "##0.0E+0"
	NumFmtText         = "@"
	NumFmtISODate      = "yyyy-mm-dd"
	NumFmtISODateTime  = "yyyy-mm-dd hh:mm:ss"
	NumFmtCurrency     = `"$"#,##0.00`
	NumFmtCurrencyRed  = `"$"#,##0.00;[Red]\-"$"#,##0.00`
	NumFmtAccounting   = `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`
	NumFmtAccountingNo = `_(* #,##0.00_);_(* \(#,##0.00\);_(* "-"??_);_(@_)`
)

// builtinNumFmts maps the built-in ids to their codes (en-US locale).
var builtinNumFmts = map[int]string{
	0:  "General",
	1:  "0",
	2:  "0.00",
	3:  "#,##0",
	4:  "#,##0.00",
	9:  "0%",
	10: "0.00%",
	11: "0.00E+00",
	12: "# ?/?",
	13: "# ??/??",
	14: "mm-dd-yy",
	15: "d-mmm-yy",
	16: "d-mmm",
	17: "mmm-yy",
	18: "h:mm AM/PM",
	19: "h:mm:ss AM/PM",
	20: "h:mm",
	21: "h:mm:ss",
	22: "m/d/yy h:mm",
	37: "#,##0 ;(#,##0)",
	38: "#,##0 ;[Red](#,##0)",
	39: "#,##0.00;(#,##0.00)",
	40: "#,##0.00;[Red](#,##0.00)",
	45: "mm:ss",
	46: "[h]:mm:ss",
	47: "mmss.0",
	48: "##0.0E+0",
	49: "@",
}

var builtinNumFmtIDs = func() map[string]int {
	m := make(map[string]int, len(builtinNumFmts))
	for id, code := range builtinNumFmts {
		m[code] = id
	}
	// Aliases Excel writes for the same built-ins in some locales/apps.
	m["m/d/yyyy"] = 14
	m["mm-dd-yy"] = 14
	m["_(* #,##0_);_(* \\(#,##0\\);_(* \"-\"_);_(@_)"] = 41
	m["_(\"$\"* #,##0_);_(\"$\"* \\(#,##0\\);_(\"$\"* \"-\"_);_(@_)"] = 42
	m["_(* #,##0.00_);_(* \\(#,##0.00\\);_(* \"-\"??_);_(@_)"] = 43
	m["_(\"$\"* #,##0.00_);_(\"$\"* \\(#,##0.00\\);_(\"$\"* \"-\"??_);_(@_)"] = 44
	return m
}()

// BuiltinNumFmt returns the code of a built-in number format id.
func BuiltinNumFmt(id int) (string, bool) {
	code, ok := builtinNumFmts[id]
	if !ok {
		switch {
		case id >= 41 && id <= 44:
			// Accounting formats; the codes depend on the locale.
			for c, i := range builtinNumFmtIDs {
				if i == id {
					return c, true
				}
			}
		case id >= 5 && id <= 8:
			// Currency formats (locale dependent); en-US forms.
			return [...]string{`"$"#,##0_);("$"#,##0)`, `"$"#,##0_);[Red]("$"#,##0)`,
				`"$"#,##0.00_);("$"#,##0.00)`, `"$"#,##0.00_);[Red]("$"#,##0.00)`}[id-5], true
		case id >= 27 && id <= 36, id >= 50 && id <= 58:
			// East Asian date formats; keep a date-like code so values show as dates.
			return "yyyy-mm-dd", true
		}
	}
	return code, ok
}

// BuiltinNumFmtID returns the built-in id for a code, if it is one.
func BuiltinNumFmtID(code string) (int, bool) {
	id, ok := builtinNumFmtIDs[code]
	return id, ok
}

// FirstCustomNumFmtID is the first id the writer assigns to custom formats.
const FirstCustomNumFmtID = 164

// IsDateFormat reports whether a number format shows dates or times, so
// that values with it should be read back as time.Time. It ignores quoted
// literals, bracketed color/condition sections and escaped characters.
func IsDateFormat(code string) bool {
	if code == "" || code == NumFmtGeneral {
		return false
	}
	if id, ok := builtinNumFmtIDs[code]; ok {
		return id >= 14 && id <= 22 || id >= 27 && id <= 36 || id >= 45 && id <= 47 || id >= 50 && id <= 58
	}
	inQuote := false
	depth := 0
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case inQuote:
			if c == '"' {
				inQuote = false
			}
		case c == '"':
			inQuote = true
		case c == '\\':
			i++
		case c == '[':
			depth++
			// [h], [mm], [ss] elapsed-time sections count as time formats.
			j := i + 1
			for j < len(code) && code[j] != ']' {
				j++
			}
			inner := code[i+1 : j]
			if inner != "" && isAll(inner, "hHmMsS") {
				return true
			}
		case c == ']':
			if depth > 0 {
				depth--
			}
		case depth > 0:
		case c == ';':
			// Only the first section decides.
			return false
		case c == 'y' || c == 'Y' || c == 'd' || c == 'D' || c == 'h' || c == 'H' || c == 's' || c == 'S':
			return true
		case c == 'm' || c == 'M':
			return true
		case c == 'A' || c == 'a':
			if i+4 < len(code)+1 && len(code)-i >= 3 && (code[i:i+3] == "A/P" || code[i:i+3] == "a/p") {
				return true
			}
			if len(code)-i >= 5 && (code[i:i+5] == "AM/PM" || code[i:i+5] == "am/pm") {
				return true
			}
		}
	}
	return false
}

func isAll(s, set string) bool {
	for i := 0; i < len(s); i++ {
		found := false
		for j := 0; j < len(set); j++ {
			if s[i] == set[j] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
