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

// Each formula may build maxArrayCells array cells in total. The nightly
// FuzzParse run killed its worker a second time on arrays that are computed
// rather than read: ROW(A:A)*COLUMN(A:XFD) asks for seventeen billion cells,
// and a chain of operations on a million rows repeats the cost at each step.
func TestArrayBudget(t *testing.T) {
	wb := testWorkbook()
	chain := "ROW(A:A)" + strings.Repeat("+1", 200)
	nested := strings.Repeat("TRANSPOSE(", 200) + "ROW(A:A)" + strings.Repeat(")", 200)
	for name, f := range map[string]string{
		"product":   "ROW(A:A)*COLUMN(A:XFD)",
		"sum":       "SUM(ROW(A:A)*COLUMN(A:XFD))",
		"concat":    "ROW(A:A)&COLUMN(A:XFD)",
		"chain":     chain,
		"nested":    nested,
		"unary":     strings.Repeat("-", 200) + "ROW(A:A)",
		"percent":   "ROW(A:A)" + strings.Repeat("%", 200),
		"if":        "IF(ROW(A:A)*COLUMN(A:Z)>0,1,2)",
		"sequences": "SEQUENCE(1000,1000)" + strings.Repeat("*SEQUENCE(1000,1000)", 20),
	} {
		start := time.Now()
		done := make(chan string, 1)
		go func() { done <- NewCalculator(wb).Evaluate("Data", f).ToModel().Text() }()
		select {
		case got := <-done:
			if got != "#NUM!" {
				t.Errorf("%s = %.40q, want #NUM!", name, got)
			}
			if d := time.Since(start); d > 20*time.Second { // generous: the race detector is slow
				t.Errorf("%s took %v", name, d)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("%s did not finish", name)
		}
	}
	// Within the budget nothing changes, and each formula starts afresh.
	calc := NewCalculator(wb)
	for f, want := range map[string]string{
		"SUM(ROW(A:A))=549756338176":            "TRUE",
		"SUM(ROW(A1:A1000)*COLUMN(A1:J1))":      "27527500",
		"ROWS(ROW(A:A)*2)":                      "1048576",
		"SUM(SEQUENCE(1000,1000))=500000500000": "TRUE",
		"SUM(ROW(A:A)+1)=549757386752":          "TRUE",
	} {
		if got := calc.Evaluate("Data", f).ToModel().Text(); got != want {
			t.Errorf("%s = %q, want %q", f, got, want)
		}
	}
}
