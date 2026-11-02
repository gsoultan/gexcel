// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"strings"
	"unicode"
)

// TokKind classifies a formula token.
type TokKind int

// Token kinds.
const (
	TokUnknown    TokKind = iota
	TokNumber             // 1, 1.5, 1e3
	TokString             // "text" (Text holds the raw quoted form)
	TokBool               // TRUE / FALSE
	TokError              // #N/A, #REF! ...
	TokRef                // A1, $A$1, A1:B2, Sheet1!A1, 'My Sheet'!A:A
	TokName               // defined name or table/structured reference head
	TokFunc               // function name followed by '(' (Text excludes the paren)
	TokOp                 // + - * / ^ & = <> <= >= < > % : , ;
	TokOpen               // (
	TokClose              // )
	TokSep                // , or ; argument separator
	TokSpace              // whitespace (kept so formulas round-trip verbatim)
	TokArrayOpen          // {
	TokArrayClose         // }
	TokStructured         // Table1[[#Headers],[Col]] structured reference
)

// Token is one lexical unit of a formula.
type Token struct {
	Kind TokKind
	Text string
}

// Tokenize splits a formula (without the leading '=') into tokens. It never
// fails: unrecognised input becomes TokUnknown so the caller can rebuild
// the exact source text by concatenating Token.Text.
func Tokenize(src string) []Token {
	src = strings.TrimPrefix(src, "=")
	var toks []Token
	i := 0
	n := len(src)
	for i < n {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			j := i
			for j < n && (src[j] == ' ' || src[j] == '\t' || src[j] == '\n' || src[j] == '\r') {
				j++
			}
			toks = append(toks, Token{TokSpace, src[i:j]})
			i = j
		case c == '"':
			j := i + 1
			for j < n {
				if src[j] == '"' {
					if j+1 < n && src[j+1] == '"' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			if j < n {
				j++
			}
			toks = append(toks, Token{TokString, src[i:j]})
			i = j
		case c == '\'':
			// Quoted sheet name: 'Sheet 1'!A1 or 'Book.xlsx'!Name
			j := i + 1
			for j < n {
				if src[j] == '\'' {
					if j+1 < n && src[j+1] == '\'' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			if j < n {
				j++
			}
			// include the ! and the following reference
			if j < n && src[j] == '!' {
				k := j + 1
				k = scanRefOrName(src, k)
				toks = append(toks, classifyRef(src[i:k]))
				i = k
			} else {
				toks = append(toks, Token{TokUnknown, src[i:j]})
				i = j
			}
		case c == '{':
			toks = append(toks, Token{TokArrayOpen, "{"})
			i++
		case c == '}':
			toks = append(toks, Token{TokArrayClose, "}"})
			i++
		case c == '(':
			toks = append(toks, Token{TokOpen, "("})
			i++
		case c == ')':
			toks = append(toks, Token{TokClose, ")"})
			i++
		case c == ',' || c == ';':
			toks = append(toks, Token{TokSep, string(c)})
			i++
		case c == '#':
			// error literal or spill operator (#) after a reference
			j := i + 1
			for j < n && (isAlnum(src[j]) || src[j] == '/' || src[j] == '!' || src[j] == '?' || src[j] == '.') {
				j++
				if src[j-1] == '!' || src[j-1] == '?' {
					break
				}
			}
			t := src[i:j]
			if j == i+1 || t == "#" {
				toks = append(toks, Token{TokOp, "#"})
				i++
			} else {
				toks = append(toks, Token{TokError, t})
				i = j
			}
		case c == '<' || c == '>':
			if i+1 < n && (src[i+1] == '=' || (c == '<' && src[i+1] == '>')) {
				toks = append(toks, Token{TokOp, src[i : i+2]})
				i += 2
			} else {
				toks = append(toks, Token{TokOp, string(c)})
				i++
			}
		case c == '+' || c == '-' || c == '*' || c == '/' || c == '^' || c == '&' || c == '=' || c == '%' || c == ':' || c == '@':
			toks = append(toks, Token{TokOp, string(c)})
			i++
		case c >= '0' && c <= '9' || c == '.' && i+1 < n && src[i+1] >= '0' && src[i+1] <= '9':
			j := i
			for j < n && (src[j] >= '0' && src[j] <= '9' || src[j] == '.') {
				j++
			}
			if j < n && (src[j] == 'e' || src[j] == 'E') {
				k := j + 1
				if k < n && (src[k] == '+' || src[k] == '-') {
					k++
				}
				if k < n && src[k] >= '0' && src[k] <= '9' {
					for k < n && src[k] >= '0' && src[k] <= '9' {
						k++
					}
					j = k
				}
			}
			toks = append(toks, Token{TokNumber, src[i:j]})
			i = j
		case c == '$' || c == '_' || c == '\\' || c == '[' || isLetterByte(src, i):
			j := scanRefOrName(src, i)
			if j == i {
				toks = append(toks, Token{TokUnknown, string(c)})
				i++
				continue
			}
			word := src[i:j]
			if j < n && src[j] == '(' {
				toks = append(toks, Token{TokFunc, word})
				i = j
				continue
			}
			toks = append(toks, classifyRef(word))
			i = j
		default:
			toks = append(toks, Token{TokUnknown, string(c)})
			i++
		}
	}
	return toks
}

func isAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '.'
}

func isLetterByte(s string, i int) bool {
	c := s[i]
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= 0x80 {
		for _, r := range s[i:] {
			return unicode.IsLetter(r)
		}
	}
	return false
}

// scanRefOrName scans an identifier, cell reference, range or structured
// reference starting at i and returns the end index.
func scanRefOrName(s string, i int) int {
	n := len(s)
	j := i
	// External workbook index: [1]Sheet1!A1
	if j < n && s[j] == '[' {
		k := strings.IndexByte(s[j:], ']')
		if k < 0 {
			return j
		}
		j += k + 1
	}
	for j < n {
		c := s[j]
		switch {
		case isAlnum(c) || c == '$' || c == '\\' || c == '?':
			j++
		case c >= 0x80:
			r, size := decodeRune(s[j:])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return j
			}
			j += size
		case c == '!':
			// sheet separator: continue scanning the reference part
			j++
		case c == ':':
			// range operator inside reference: only if followed by ref-like char
			if j+1 < n && (isAlnum(s[j+1]) || s[j+1] == '$') {
				j++
			} else {
				return j
			}
		case c == '[':
			// structured reference: match brackets
			depth := 0
			k := j
			for k < n {
				if s[k] == '[' {
					depth++
				} else if s[k] == ']' {
					depth--
					if depth == 0 {
						k++
						break
					}
				} else if s[k] == '\'' && k+1 < n {
					k++ // escape
				}
				k++
			}
			j = k
			return j
		default:
			return j
		}
	}
	return j
}

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		if i > 0 {
			return r, i
		}
		_ = r
	}
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 1
}

// classifyRef decides whether a scanned word is a reference, a name, a
// boolean or an error.
func classifyRef(word string) Token {
	switch strings.ToUpper(word) {
	case "TRUE", "FALSE":
		return Token{TokBool, word}
	}
	if strings.ContainsRune(word, '[') && !strings.HasPrefix(word, "[") {
		return Token{TokStructured, word}
	}
	body := word
	if i := strings.LastIndexByte(word, '!'); i >= 0 {
		body = word[i+1:]
	}
	if IsRef(body) {
		return Token{TokRef, word}
	}
	return Token{TokName, word}
}

// IsRef reports whether s (without a sheet prefix) looks like an A1-style
// cell, range, whole-column or whole-row reference.
func IsRef(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.SplitN(s, ":", 2)
	for _, p := range parts {
		if !isCellOrLine(p) {
			return false
		}
	}
	if len(parts) == 2 {
		// A:A, 1:1, A1:B2 all fine; A1:B (mixed) is not
		a, b := refShape(parts[0]), refShape(parts[1])
		if a != b && a != 3 && b != 3 {
			return false
		}
	} else {
		return refShape(parts[0]) == 3
	}
	return true
}

// refShape returns 1 for column-only, 2 for row-only, 3 for cell, 0 otherwise.
func refShape(p string) int {
	p = strings.ReplaceAll(p, "$", "")
	letters := 0
	for letters < len(p) && (p[letters] >= 'A' && p[letters] <= 'Z' || p[letters] >= 'a' && p[letters] <= 'z') {
		letters++
	}
	digits := 0
	for letters+digits < len(p) && p[letters+digits] >= '0' && p[letters+digits] <= '9' {
		digits++
	}
	if letters+digits != len(p) {
		return 0
	}
	switch {
	case letters > 0 && digits > 0:
		if letters > 3 || digits > 7 {
			return 0
		}
		return 3
	case letters > 0:
		if letters > 3 {
			return 0
		}
		return 1
	case digits > 0:
		return 2
	}
	return 0
}

func isCellOrLine(p string) bool {
	if p == "" || p == "$" {
		return false
	}
	// at most two $ signs: one before letters, one before digits
	if strings.Count(p, "$") > 2 {
		return false
	}
	return refShape(p) != 0
}

// Join rebuilds the source text of a token list.
func Join(toks []Token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.Text)
	}
	return b.String()
}
