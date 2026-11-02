// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ref"
)

func testWorkbook() *model.Workbook {
	wb := model.New()
	sh := wb.AddSheet("Data")
	rows := [][]any{
		{"Name", "Qty", "Price", "Date", "Flag"},
		{"apple", 3, 1.5, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), true},
		{"banana", 5, 0.25, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), false},
		{"cherry", 10, 4, time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC), true},
		{"", 0, 0, nil, nil},
	}
	for r, row := range rows {
		for c, v := range row {
			cell := sh.Cell(r+1, c+1)
			switch x := v.(type) {
			case string:
				if x != "" {
					cell.Value = model.String(x)
				}
			case int:
				cell.Value = model.Number(float64(x))
			case float64:
				cell.Value = model.Number(x)
			case bool:
				cell.Value = model.Boolean(x)
			case time.Time:
				cell.Value = model.Date(x, false)
			}
		}
	}
	sh.Tables = append(sh.Tables, &model.Table{Name: "Items", DisplayName: "Items", Range: mustRange("A1:E4"), Columns: []model.TableColumn{{Name: "Name"}, {Name: "Qty"}, {Name: "Price"}, {Name: "Date"}, {Name: "Flag"}}})
	wb.DefinedNames = append(wb.DefinedNames, model.DefinedName{Name: "Prices", RefersTo: "Data!$C$2:$C$4"}, model.DefinedName{Name: "Rate", RefersTo: "0.1"})
	other := wb.AddSheet("Other Sheet")
	other.Cell(1, 1).Value = model.Number(100)
	other.Cell(2, 1).Value = model.Number(200)
	return wb
}

func mustRange(s string) ref.Range { return ref.MustRange(s) }

var cases = []struct {
	f    string
	want string
}{
	{"1+2*3", "7"}, {"(1+2)*3", "9"}, {"2^3^2", "64"}, {"-2^2", "4"}, {"10/4", "2.5"}, {"1/0", "#DIV/0!"},
	{`"a"&"b"&1`, "ab1"}, {"1=1", "TRUE"}, {`"A"="a"`, "TRUE"}, {"1<>1", "FALSE"}, {"2>=2", "TRUE"}, {"50%", "0.5"},
	{"SUM(B2:B4)", "18"}, {"SUM(B:B)", "18"}, {"SUM(Qty)", "#NAME?"}, {"SUM(Prices)", "5.75"}, {"SUM(Items[Qty])", "18"}, {"SUM(Items[[#Data],[Price]])", "5.75"},
	{"AVERAGE(B2:B4)", "6"}, {"MIN(B2:B4)", "3"}, {"MAX(C2:C4)", "4"}, {"COUNT(A1:E4)", "9"}, {"COUNTA(A1:E5)", "22"}, {"COUNTBLANK(A1:E5)", "3"},
	{"SUMPRODUCT(B2:B4,C2:C4)", "45.75"}, {"SUMIF(B2:B4,\">4\",C2:C4)", "4.25"}, {"SUMIFS(C2:C4,B2:B4,\">=5\",A2:A4,\"b*\")", "0.25"},
	{"COUNTIF(A2:A4,\"*an*\")", "1"}, {"COUNTIFS(B2:B4,\">2\",E2:E4,TRUE)", "2"}, {"AVERAGEIF(B2:B4,\">3\")", "7.5"}, {"MAXIFS(C2:C4,E2:E4,TRUE)", "4"},
	{"ROUND(2.675,2)", "2.68"}, {"ROUND(-1.5,0)", "-2"}, {"ROUNDUP(1.21,1)", "1.3"}, {"ROUNDDOWN(-1.29,1)", "-1.2"}, {"INT(-1.5)", "-2"}, {"TRUNC(-1.5)", "-1"},
	{"MOD(-3,2)", "1"}, {"MOD(3,-2)", "-1"}, {"CEILING(2.1,0.5)", "2.5"}, {"FLOOR(2.9,0.5)", "2.5"}, {"MROUND(7,3)", "6"}, {"EVEN(3)", "4"}, {"ODD(4)", "5"},
	{"ABS(-3)", "3"}, {"SQRT(16)", "4"}, {"SQRT(-1)", "#NUM!"}, {"POWER(2,10)", "1024"}, {"LOG(100)", "2"}, {"LOG(8,2)", "3"}, {"FACT(5)", "120"}, {"GCD(12,18)", "6"}, {"LCM(4,6)", "12"},
	{"MEDIAN(1,2,3,4)", "2.5"}, {"MODE(1,2,2,3)", "2"}, {"STDEV(2,4,4,4,5,5,7,9)", "2.13808993529939"}, {"VAR.P(2,4,4,4,5,5,7,9)", "4"}, {"LARGE(B2:B4,2)", "5"}, {"SMALL(B2:B4,1)", "3"},
	{"RANK(5,B2:B4)", "2"}, {"PERCENTILE(B2:B4,0.5)", "5"}, {"QUARTILE(B2:B4,3)", "7.5"},
	{"IF(1>0,\"yes\",\"no\")", "yes"}, {"IF(FALSE,1)", "FALSE"}, {"IFS(1>2,\"a\",2>1,\"b\")", "b"}, {"IFERROR(1/0,\"x\")", "x"}, {"IFNA(NA(),0)", "0"}, {"AND(TRUE,1)", "TRUE"}, {"OR(FALSE,0)", "FALSE"}, {"XOR(TRUE,TRUE)", "FALSE"}, {"NOT(TRUE)", "FALSE"},
	{"SWITCH(2,1,\"a\",2,\"b\",\"z\")", "b"}, {"CHOOSE(2,\"x\",\"y\")", "y"}, {"LET(x,2,y,3,x*y)", "6"},
	{"ISBLANK(A5)", "TRUE"}, {"ISNUMBER(B2)", "TRUE"}, {"ISTEXT(A2)", "TRUE"}, {"ISERROR(1/0)", "TRUE"}, {"ISNA(NA())", "TRUE"}, {"ISEVEN(4)", "TRUE"}, {"N(TRUE)", "1"}, {"TYPE(\"a\")", "2"}, {"ERROR.TYPE(#REF!)", "4"},
	{"CONCAT(A2,\"-\",B2)", "apple-3"}, {"TEXTJOIN(\",\",TRUE,A2:A5)", "apple,banana,cherry"}, {"LEFT(A2,3)", "app"}, {"RIGHT(A2,2)", "le"}, {"MID(A3,2,3)", "ana"}, {"LEN(A4)", "6"},
	{"UPPER(A2)", "APPLE"}, {"PROPER(\"hello world\")", "Hello World"}, {"TRIM(\"  a   b  \")", "a b"}, {"SUBSTITUTE(\"aaa\",\"a\",\"b\",2)", "aba"}, {"REPLACE(\"abcdef\",2,3,\"X\")", "aXef"},
	{"FIND(\"n\",A3)", "3"}, {"SEARCH(\"AN\",A3)", "2"}, {"SEARCH(\"a?a\",A3)", "2"}, {"REPT(\"ab\",3)", "ababab"}, {"TEXT(1234.5,\"#,##0.00\")", "1,234.50"}, {"TEXT(D2,\"yyyy-mm-dd\")", "2026-01-15"},
	{"VALUE(\"1,234.5\")", "1234.5"}, {"VALUE(\"12%\")", "0.12"}, {"EXACT(\"a\",\"A\")", "FALSE"}, {"CHAR(65)", "A"}, {"CODE(\"a\")", "97"}, {"T(1)", ""}, {"TEXTBEFORE(\"a-b-c\",\"-\",2)", "a-b"}, {"TEXTAFTER(\"a-b-c\",\"-\",-1)", "c"},
	{"FIXED(1234.567,1)", "1,234.6"}, {"DOLLAR(1234.567)", "$1,234.57"},
	{"VLOOKUP(\"banana\",A2:C4,3,FALSE)", "0.25"}, {"VLOOKUP(\"BAN*\",A2:C4,2,0)", "5"}, {"VLOOKUP(\"zzz\",A2:C4,2,FALSE)", "#N/A"}, {"VLOOKUP(7,B2:C4,2,TRUE)", "0.25"},
	{"HLOOKUP(\"Qty\",A1:C4,2,FALSE)", "3"}, {"XLOOKUP(\"cherry\",A2:A4,C2:C4)", "4"}, {"XLOOKUP(\"x\",A2:A4,C2:C4,\"none\")", "none"}, {"LOOKUP(5,B2:B4,A2:A4)", "banana"},
	{"INDEX(A2:C4,2,1)", "banana"}, {"INDEX(B2:B4,3)", "10"}, {"MATCH(\"cherry\",A2:A4,0)", "3"}, {"MATCH(6,B2:B4,1)", "2"}, {"XMATCH(\"banana\",A2:A4)", "2"}, {"INDEX(A2:A4,MATCH(10,B2:B4,0))", "cherry"},
	{"SUM(OFFSET(B2,1,0,2,1))", "15"}, {"OFFSET(A1,1,1)", "3"}, {"INDIRECT(\"C3\")", "0.25"}, {"INDIRECT(\"Data!B4\")", "10"}, {"SUM(INDIRECT(\"B2:B4\"))", "18"}, {"ROW()", "1"}, {"ROW(B5)", "5"}, {"COLUMN(C1)", "3"}, {"ROWS(A1:B3)", "3"}, {"COLUMNS(A1:B3)", "2"},
	{"ADDRESS(2,3)", "$C$2"}, {"ADDRESS(2,3,4,TRUE,\"My Sheet\")", "'My Sheet'!C2"},
	{"SUM('Other Sheet'!A1:A2)", "300"}, {"'Other Sheet'!A2*2", "400"}, {"SUM(Data:'Other Sheet'!A2)", "200"}, {"Rate*100", "10"},
	{"DATE(2026,3,14)", "46095"}, {"DATE(2026,13,1)", "46388"}, {"TIME(12,30,0)", "0.520833333333333"}, {"YEAR(D3)", "2026"}, {"MONTH(D3)", "2"}, {"DAY(D3)", "20"}, {"HOUR(D4)", "12"}, {"MINUTE(0.5208333)", "30"},
	{"WEEKDAY(D2)", "5"}, {"WEEKDAY(D2,2)", "4"}, {"WEEKNUM(D2)", "3"}, {"ISOWEEKNUM(D2)", "3"}, {"DAYS(D3,D2)", "36"}, {"DATEDIF(D2,D4,\"m\")", "2"}, {"DATEDIF(D2,D4,\"d\")", "75"},
	{"EDATE(D2,1)", "46068"}, {"EOMONTH(D2,0)", "46053"}, {"DATEVALUE(\"2026-01-15\")", "46037"}, {"TIMEVALUE(\"6:00 PM\")", "0.75"}, {"NETWORKDAYS(D2,D3)", "27"}, {"WORKDAY(D2,5)", "46044"}, {"YEARFRAC(D2,D3)", "0.0972222222222222"}, {"DAYS360(D2,D3)", "35"},
	{"D2+1", "46038"}, {"\"2026-01-15\"+0", "46037"}, {"YEAR(\"3/14/2026\")", "2026"},
	{"PMT(0.05/12,360,200000)", "-1073.64324604517"}, {"FV(0.1,2,0,-100)", "121"}, {"PV(0.1,1,0,-110)", "100"}, {"NPV(0.1,100,100)", "173.553719008264"}, {"NPER(0.1,-100,500)", "7.27254089734171"},
	{"SUM({1,2;3,4})", "10"}, {"SUM(B2:B4*C2:C4)", "45.75"}, {"MAX(IF(E2:E4,B2:B4))", "10"}, {"SUM(SEQUENCE(4))", "10"}, {"SUM(FILTER(B2:B4,E2:E4))", "13"}, {"INDEX(SORT(A2:A4,1,-1),1)", "cherry"}, {"INDEX(UNIQUE({1;1;2}),2)", "2"},
	{"TRANSPOSE(A2:A3)", "apple"}, {"ROWS(UNIQUE({1;1;2}))", "2"}, {"INDEX(TEXTSPLIT(\"a,b,c\",\",\"),3)", "c"}, {"SUMPRODUCT((B2:B4>3)*C2:C4)", "4.25"}, {"@B2:B4", "#VALUE!"},
	{"SUM(1,\"2\",TRUE)", "4"}, {"SUM(A2:A4)", "0"}, {"\"5\"*2", "10"}, {"\"abc\"*2", "#VALUE!"}, {"TRUE+1", "2"}, {"A5+1", "1"}, {"NOSUCHFUNC(1)", "#NAME?"}, {"#N/A", "#N/A"},
	{"IF(B2>2,\"big\")", "big"}, {"SUM(B2:B4)/COUNT(B2:B4)", "6"}, {"ROUND(AVERAGE(C2:C4),3)", "1.917"}, {"_xlfn.CONCAT(\"a\",\"b\")", "ab"}, {"_xlfn.XLOOKUP(5,B2:B4,A2:A4)", "banana"},
}

func TestEvaluate(t *testing.T) {
	wb := testWorkbook()
	calc := NewCalculator(wb)
	for _, tc := range cases {
		v := calc.Evaluate("Data", tc.f)
		got := toText(v)
		if v.Kind == Number {
			got = formatNum(v.Num)
		}
		if v.Kind == Array {
			got = toText(v.At(0, 0))
		}
		if got != tc.want {
			if f, ok := parseNumber(tc.want); ok && v.Kind == Number && math.Abs(f-v.Num) < 1e-9*math.Max(1, math.Abs(f)) {
				continue
			}
			t.Errorf("%s = %q, want %q", tc.f, got, tc.want)
		}
	}
}

func TestCalculateWorkbook(t *testing.T) {
	wb := testWorkbook()
	sh := wb.Sheets[0]
	sh.Cell(1, 7).Formula = &model.Formula{Text: "SUM(B2:B4)"}
	sh.Cell(2, 7).Formula = &model.Formula{Text: "G1*2"}
	sh.Cell(3, 7).Formula = &model.Formula{Text: "G2+G1"}
	sh.Cell(7, 7).Formula = &model.Formula{Text: "SEQUENCE(1,3)", Dynamic: true, Kind: model.FormulaArray, Master: true}
	sh.Cell(8, 7).Formula = &model.Formula{Text: "{1,2;3,4}", Kind: model.FormulaArray, Master: true, Ref: ref.MustRange("G8:H9")}
	sh.Cell(6, 7).Formula = &model.Formula{Text: "[Price]*[Qty]"} // bare structured ref outside a table -> error
	for r := 2; r <= 4; r++ {
		sh.Cell(r, 8).Formula = &model.Formula{Text: "Items[@Qty]*Items[@Price]"}
	}
	if err := Calculate(wb); err != nil {
		t.Fatal(err)
	}
	if v := sh.Cell(1, 7).Value; v.Num != 18 {
		t.Errorf("G1 = %+v", v)
	}
	if v := sh.Cell(3, 7).Value; v.Num != 54 {
		t.Errorf("G3 = %+v", v)
	}
	if v := sh.Cell(7, 9).Value; v.Num != 3 {
		t.Errorf("spill I7 = %+v", v)
	}
	if r := sh.Cell(7, 7).Formula.Ref.String(); r != "G7:I7" {
		t.Errorf("spill ref %s", r)
	}
	if v := sh.Cell(9, 8).Value; v.Num != 4 {
		t.Errorf("legacy array H9 = %+v", v)
	}
	if v := sh.Cell(3, 8).Value; v.Num != 1.25 {
		t.Errorf("H3 = %+v", v)
	}
	// Cycle.
	sh.Cell(10, 1).Formula = &model.Formula{Text: "A11+1"}
	sh.Cell(11, 1).Formula = &model.Formula{Text: "A10+1"}
	if err := Calculate(wb); err != ErrCycle {
		t.Errorf("cycle: %v", err)
	}
	if v := sh.Cell(10, 1).Value; v.Kind != model.KindError {
		t.Errorf("A10 = %+v", v)
	}
}

func TestShift(t *testing.T) {
	cases := map[string]string{
		"A1+$B$2":     "B2+$B$2",
		"SUM(A1:B2)":  "SUM(B2:C3)",
		"Sheet1!A$1":  "Sheet1!B$1",
		"\"A1\"&A1":   "\"A1\"&B2",
		"A:A":         "B:B",
		"1:1":         "2:2",
		"A1#":         "B2#",
		"Table1[Col]": "Table1[Col]",
	}
	for in, want := range cases {
		if got := Shift(in, 1, 1); got != want {
			t.Errorf("Shift(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseRoundTrip(t *testing.T) {
	for _, f := range []string{"SUM(A1:B2,3)", "IF(A1>1,\"x\",\"y\")", "{1,2;3,4}", "-A1^2", "'My Sheet'!$A$1", "A1:INDEX(B:B,3)", "Table1[[#Headers],[Col]]", "1+2*3", "(1+2)*3", "a&b", "LEN(\"\"\"\")"} {
		n, err := Parse(f)
		if err != nil {
			t.Errorf("Parse(%q): %v", f, err)
			continue
		}
		got := String(n)
		if strings.ReplaceAll(got, " ", "") != strings.ReplaceAll(f, " ", "") {
			t.Errorf("String(Parse(%q)) = %q", f, got)
		}
	}
	for _, bad := range []string{"", "SUM(", "1+", ")", "A1:", "{1,2"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, c := range cases {
		f.Add(c.f)
	}
	wb := testWorkbook()
	f.Fuzz(func(t *testing.T, s string) {
		n, err := Parse(s)
		if err != nil {
			return
		}
		_ = String(n)
		calc := NewCalculator(wb)
		_ = calc.Evaluate("Data", s)
	})
}

func TestPrefixes(t *testing.T) {
	cases := map[string]string{
		"SUM(A1)":                      "SUM(A1)",
		"XLOOKUP(1,A:A,B:B)":           "_xlfn.XLOOKUP(1,A:A,B:B)",
		"SORT(FILTER(A1:B9,C1:C9))":    "_xlfn._xlws.SORT(_xlfn._xlws.FILTER(A1:B9,C1:C9))",
		"_xlfn.CONCAT(\"a\")":          "_xlfn.CONCAT(\"a\")",
		"IFS(A1>1,\"concat(\",TRUE,1)": "_xlfn.IFS(A1>1,\"concat(\",TRUE,1)",
	}
	for in, want := range cases {
		if got := AddPrefixes(in); got != want {
			t.Errorf("AddPrefixes(%q) = %q, want %q", in, got, want)
		}
		if got := StripPrefixes(want); got != in && !strings.HasPrefix(in, "_xlfn") {
			t.Errorf("StripPrefixes(%q) = %q, want %q", want, got, in)
		}
	}
	if !IsDynamicArray("SORT(A1:A3)") || !IsDynamicArray("IFERROR(UNIQUE(A:A),\"\")") || IsDynamicArray("SUM(SORT(A1:A3))") || IsDynamicArray("@SORT(A1:A3)") {
		t.Error("IsDynamicArray")
	}
}
