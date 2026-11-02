// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/gexcel/writer/wtest"
)

func sampleWorkbook() *model.Workbook { return wtest.SampleWorkbook() }

func TestWriteSample(t *testing.T) {
	wb := sampleWorkbook()
	var buf bytes.Buffer
	if err := Write(wb, &buf); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GEXCEL_OUT"); dir != "" {
		os.WriteFile(filepath.Join(dir, "sample.xlsx"), buf.Bytes(), 0o644)
	}
	pr, err := ooxml.OpenBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	// Every XML part must be well-formed.
	for _, name := range pr.Names() {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		data, err := pr.Part(name)
		if err != nil {
			t.Fatal(err)
		}
		dec := xml.NewDecoder(bytes.NewReader(data))
		for {
			_, err := dec.Token()
			if err != nil {
				if err.Error() != "EOF" {
					t.Errorf("%s: %v", name, err)
				}
				break
			}
		}
	}
	for _, want := range []string{"xl/workbook.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml", "xl/styles.xml", "xl/sharedStrings.xml",
		"xl/theme/theme1.xml", "xl/comments1.xml", "xl/drawings/vmlDrawing1.vml", "xl/tables/table1.xml", "xl/charts/chart1.xml", "xl/charts/chart2.xml",
		"xl/drawings/drawing1.xml", "xl/media/image1.png", "xl/metadata.xml", "docProps/core.xml", "docProps/app.xml", "docProps/custom.xml"} {
		if !pr.Has(want) {
			t.Errorf("missing part %s", want)
		}
	}
	sheet, _ := pr.Part("xl/worksheets/sheet1.xml")
	for _, want := range []string{`<c r="A1" s="1" t="s"><v>0</v></c>`, `<f>B2*C2</f>`, `<mergeCell ref="A8:C8"/>`, `<pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/>`,
		`algorithmName="SHA-512"`, `<dataValidation type="whole"`, `<cfRule type="cellIs" dxfId="0" priority="1" operator="greaterThan"><formula>100</formula></cfRule>`,
		`<hyperlink ref="A2" r:id="rId`, `location="'Data'!A1"`, `<legacyDrawing r:id=`, `<is><t>inline</t></is>`, `<pageSetup paperSize="9" orientation="landscape"/>`} {
		if !strings.Contains(string(sheet), want) {
			t.Errorf("sheet1 lacks %s\n%s", want, sheet)
		}
	}
	sst, _ := pr.Part("xl/sharedStrings.xml")
	if !strings.Contains(string(sst), `<si><t xml:space="preserve">  spaced  </t></si>`) || !strings.Contains(string(sst), `<si><r><t xml:space="preserve">Rich </t></r><r><rPr><b/><color rgb="FFFF0000"/></rPr><t>bold</t></r></si>`) {
		t.Errorf("shared strings: %s", sst)
	}
	wbx, _ := pr.Part("xl/workbook.xml")
	for _, want := range []string{`<definedName name="_xlnm._FilterDatabase" hidden="1" localSheetId="0">Data!$A$1:$E$4</definedName>`, `<definedName name="_xlnm.Print_Titles" localSheetId="0">Data!$1:$1</definedName>`,
		`<definedName name="Prices" localSheetId="0">Data!$C$2:$C$4</definedName>`, `<definedName name="TaxRate">0.2</definedName>`, `<sheet name="Report" sheetId="2" r:id="rId2"/>`} {
		if !strings.Contains(string(wbx), want) {
			t.Errorf("workbook lacks %s\n%s", want, wbx)
		}
	}
	sheet2, _ := pr.Part("xl/worksheets/sheet2.xml")
	if !strings.Contains(string(sheet2), `<c r="H1" cm="1"><f t="array" ref="H1:H4">_xlfn._xlws.SORT(B2:B5)</f></c>`) {
		t.Errorf("dynamic array formula\n%s", sheet2)
	}
	if !strings.Contains(string(sheet2), "x14:sparklineGroups") {
		t.Error("sparklines missing")
	}
}

func TestStreamWriter(t *testing.T) {
	var buf bytes.Buffer
	wb := model.New()
	sw := NewStreamWriter(&buf, wb)
	bold := wb.Styles.Add(style.Style{Font: style.Font{Bold: true}})
	sh := model.NewSheet("Big")
	sh.View.Freeze = &model.Pane{Rows: 1}
	ss := sw.Sheet(sh)
	if err := ss.Row(1, &model.Cell{Col: 1, Value: model.String("n"), StyleID: bold}, &model.Cell{Col: 2, Value: model.String("sq")}); err != nil {
		t.Fatal(err)
	}
	const n = 20000
	for i := 2; i <= n; i++ {
		if err := ss.Row(i, &model.Cell{Col: 2, Value: model.Number(float64(i * i))}, &model.Cell{Col: 1, Value: model.Number(float64(i))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := ss.Row(5, &model.Cell{Col: 1}); err == nil {
		t.Error("out-of-order row accepted")
	}
	sh.Merges = append(sh.Merges, ref.MustRange("D1:E1"))
	if err := ss.Close(); err != nil {
		t.Fatal(err)
	}
	ss2 := sw.Sheet(model.NewSheet("Second"))
	ss2.Row(1, &model.Cell{Col: 1, Value: model.String("x")})
	if err := sw.Close(); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GEXCEL_OUT"); dir != "" {
		os.WriteFile(filepath.Join(dir, "stream.xlsx"), buf.Bytes(), 0o644)
	}
	pr, err := ooxml.OpenBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	data, _ := pr.Part("xl/worksheets/sheet1.xml")
	s := string(data)
	if !strings.Contains(s, `<row r="20000" spans="1:2"><c r="A20000"><v>20000</v></c><c r="B20000"><v>400000000</v></c></row>`) || !strings.Contains(s, `<mergeCell ref="D1:E1"/>`) || !strings.Contains(s, `state="frozen"`) {
		t.Errorf("stream sheet content wrong: %s", s[len(s)-600:])
	}
	if !pr.Has("xl/worksheets/sheet2.xml") || !pr.Has("xl/sharedStrings.xml") {
		t.Error("parts missing")
	}
}

func TestValidate(t *testing.T) {
	wb := model.New()
	if err := Write(wb, &bytes.Buffer{}); err != ErrNoSheets {
		t.Error(err)
	}
	wb.AddSheet("a")
	wb.AddSheet("A")
	if err := Write(wb, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Error(err)
	}
	wb.Sheets = wb.Sheets[:1]
	wb.Sheets[0].Name = "bad[name]"
	if err := Write(wb, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "invalid sheet name") {
		t.Error(err)
	}
}

func TestPasswordHash(t *testing.T) {
	if LegacyPasswordHash("password") != "83AF" {
		t.Error(LegacyPasswordHash("password"))
	}
	alg, hash, salt, spin := hashPassword("secret")
	if alg != "SHA-512" || spin != 100000 || !VerifyPassword("secret", hash, salt, spin) || VerifyPassword("wrong", hash, salt, spin) {
		t.Error("hash round trip")
	}
}
