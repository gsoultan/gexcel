// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package tests holds black-box tests, including the validation of gexcel
// output through LibreOffice and openpyxl. The external tests run when
// GEXCEL_EXTERNAL=1 and the tools are installed.
package tests

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gsoultan/gexcel"
	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/gexcel/writer/wtest"
)

func needExternal(t *testing.T) {
	t.Helper()
	if os.Getenv("GEXCEL_EXTERNAL") == "" {
		t.Skip("set GEXCEL_EXTERNAL=1 to run LibreOffice/openpyxl validation")
	}
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice not installed")
	}
}

func reportWorkbook(t *testing.T) gexcel.Workbook {
	t.Helper()
	wb := gexcel.New()
	sh := wb.SheetAt(0)
	if err := sh.SetName("Report"); err != nil {
		t.Fatal(err)
	}
	if err := sh.SetRow("A1", "Region", "Sales", "When", "Share"); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC)
	if err := sh.SetRows("A2", [][]any{{"North", 1200.5, when, 0.25}, {"South", 980, when.AddDate(0, 0, 1), 0.2}, {"East", 1510.25, when.AddDate(0, 0, 2), 0.31}}); err != nil {
		t.Fatal(err)
	}
	sh.SetStyle("A1:D1", style.Style{Font: style.Font{Bold: true}})
	for c := range sh.Range("D2:D4") {
		c.SetNumFmt("0%")
	}
	sh.Cell("B5").SetFormula("SUM(B2:B4)")
	sh.Cell("C5").SetFormula("TEXT(C2,\"yyyy-mm-dd\")")
	// Functions every LibreOffice release in CI understands (XLOOKUP/SORT
	// need LibreOffice 24.8+ and would show #NAME? on older runners).
	sh.Cell("D5").SetFormula("INDEX(B2:B4,MATCH(\"East\",A2:A4,0))")
	sh.Cell("E1").SetFormula("LARGE(B2:B4,1)")
	sh.Cell("E3").SetFormula("SMALL(B2:B4,1)")
	return wb
}

// TestExternalLibreOffice converts a workbook to CSV with LibreOffice and
// checks the values and formula results it computed.
func TestExternalLibreOffice(t *testing.T) {
	needExternal(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.xlsx")
	if err := reportWorkbook(t).SaveFile(path); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("soffice", "--headless", "--convert-to", "csv", "--outdir", dir, path)
	cmd.Env = append(os.Environ(), "HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("soffice: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "report.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := map[[2]int]string{{0, 0}: "Region", {1, 1}: "1200.5", {1, 3}: "25%", {4, 1}: "3690.75", {4, 2}: "2026-03-14", {4, 3}: "1510.25", {0, 4}: "1510.25", {2, 4}: "980"}
	_ = want
	for pos, w := range want {
		if pos[0] >= len(rows) || pos[1] >= len(rows[pos[0]]) {
			t.Errorf("cell %v missing in %v", pos, rows)
			continue
		}
		if got := rows[pos[0]][pos[1]]; got != w {
			t.Errorf("cell %v = %q, want %q", pos, got, w)
		}
	}
	// The feature-complete sample must also convert without LibreOffice complaining.
	sample := filepath.Join(dir, "sample.xlsx")
	if err := gexcel.Wrap(wtest.SampleWorkbook()).SaveFile(sample); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", dir, sample)
	cmd.Env = append(os.Environ(), "HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("soffice pdf: %v\n%s", err, out)
	}
	if st, err := os.Stat(filepath.Join(dir, "sample.pdf")); err != nil || st.Size() == 0 {
		t.Fatalf("no pdf produced: %v", err)
	}
	// Files written by LibreOffice must read back.
	cmd = exec.Command("soffice", "--headless", "--convert-to", "xlsx", "--outdir", filepath.Join(dir, "lo"), sample)
	cmd.Env = append(os.Environ(), "HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("soffice xlsx: %v\n%s", err, out)
	}
	back, err := gexcel.Open(filepath.Join(dir, "lo", "sample.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	if v := back.SheetAt(0).Cell("A1").String(); v == "" {
		t.Error("LibreOffice-saved file lost A1")
	}
}

// TestExternalOpenpyxl loads the output with openpyxl and prints a digest
// that is compared with the expectations below.
func TestExternalOpenpyxl(t *testing.T) {
	needExternal(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.xlsx")
	if err := gexcel.Wrap(wtest.SampleWorkbook()).SaveFile(path); err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from openpyxl import load_workbook
wb = load_workbook(sys.argv[1])
ws = wb.worksheets[0]
print("sheets", len(wb.sheetnames))
print("a1", ws["A1"].value, ws["A1"].font.b)
print("charts", sum(len(s._charts) for s in wb.worksheets))
print("images", sum(len(s._images) for s in wb.worksheets))
print("tables", sum(len(s.tables) for s in wb.worksheets))
print("names", sorted(wb.defined_names.keys()) if hasattr(wb.defined_names, "keys") else len(wb.defined_names.definedName))
print("merged", sum(len(s.merged_cells.ranges) for s in wb.worksheets))
print("dv", sum(len(s.data_validations.dataValidation) for s in wb.worksheets))
print("cf", sum(len(s.conditional_formatting) for s in wb.worksheets))
`
	sp := filepath.Join(dir, "check.py")
	if err := os.WriteFile(sp, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("python3", "-I", sp, path).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No module named") {
			t.Skip("openpyxl not installed")
		}
		t.Fatalf("openpyxl: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"a1 Item True", "charts 2", "images 1", "tables 1", "merged 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("openpyxl digest lacks %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "sheets 3") && !strings.Contains(got, "sheets 2") {
		t.Errorf("unexpected sheet count:\n%s", got)
	}
}
