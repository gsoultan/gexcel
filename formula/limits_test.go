// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Formulas that used to run for minutes or allocate without bound (found
// by FuzzParse and by probing its neighbours) now answer at once.
func TestHostileFormulasAreBounded(t *testing.T) {
	wb := testWorkbook()
	for f, want := range map[string]string{
		`NETWORKDAYS(1,1E12)`:      "#NUM!",
		`NETWORKDAYS(-5,10)`:       "#NUM!",
		`WORKDAY(1,1E12)`:          "#NUM!",
		`WORKDAY(1,-1E12)`:         "#NUM!",
		`WORKDAY(2958460,10)`:      "#NUM!",
		`WORKDAY(3,-10)`:           "#NUM!",
		`FIXED(1,1E9)`:             "#VALUE!",
		`DOLLAR(1,128)`:            "#VALUE!",
		`FIXED(1.5,3)`:             "1.500",
		`DOLLAR(1234.5,1)`:         "$1,234.5",
		`NETWORKDAYS(45292,45298)`: "5", // Mon 1 Jan 2024 .. Sun 7 Jan 2024
		`WORKDAY(45296,1)`:         "45299",
	} {
		done := make(chan string, 1)
		go func() { done <- NewCalculator(wb).Evaluate("Data", f).ToModel().Text() }()
		select {
		case got := <-done:
			if got != want {
				t.Errorf("%s = %q, want %q", f, got, want)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%s did not finish", f)
		}
	}
}

func TestFormulaLengthLimit(t *testing.T) {
	ok := "1" + strings.Repeat("+1", (MaxLength-1)/2)
	if _, err := Parse(ok); err != nil {
		t.Errorf("a %d character formula: %v", len(ok), err)
	}
	for name, f := range map[string]string{
		"parens": strings.Repeat("(", 100000) + "1" + strings.Repeat(")", 100000),
		"unary":  strings.Repeat("-", 500000) + "1",
		"chain":  "1" + strings.Repeat("+1", MaxLength),
		"pct":    "1" + strings.Repeat("%", MaxLength),
	} {
		if _, err := Parse(f); !errors.Is(err, ErrSyntax) {
			t.Errorf("%s: err = %v, want ErrSyntax", name, err)
		}
	}
	// The limit counts characters, not bytes.
	if _, err := Parse(`"` + strings.Repeat("é", MaxLength-2) + `"`); err != nil {
		t.Errorf("%d two-byte characters: %v", MaxLength-2, err)
	}
}

// A reference that reaches past the data is read only up to the last used
// cell: whole columns must cost what the data costs (FuzzParse found
// XLOOKUP(,AF:X,A2:A7) building nine million cells).
func TestWholeColumnReferencesAreClipped(t *testing.T) {
	wb := testWorkbook()
	calc := NewCalculator(wb)
	start := time.Now()
	for f, want := range map[string]string{
		"XLOOKUP(,AF:X,A2:A7)":         "",
		"SUM(A:A)=SUM(A1:A1048576)":    "TRUE",
		"COUNTBLANK(X:AF)":             "9437184",
		"ROWS(A:A)":                    "1048576",
		"COLUMNS(A:XFD)":               "16384",
		"COUNTA(A:XFD)=COUNTA(A1:Z99)": "TRUE",
		"SUM(AA:AF)":                   "0",
	} {
		got := calc.Evaluate("Data", f).ToModel().Text()
		if want != "" && got != want {
			t.Errorf("%s = %q, want %q", f, got, want)
		}
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("whole-column formulas took %v", d)
	}
}
