// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/reader"
	"github.com/gsoultan/gexcel/ref"
	"github.com/gsoultan/gexcel/writer"
	"github.com/gsoultan/gexcel/writer/wtest"
)

// cellImagesFixture was written by XlsxWriter's embed_image: pictures in A1,
// B2 (with alternative text) and C3 (the picture of A1 again), text in D1
// and a dynamic array formula in F1.
const cellImagesFixture = "testdata/xlsxwriter_cell_images.xlsx"

func unzip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	return out
}

func rezip(t *testing.T, parts map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(body)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func cellAt(t *testing.T, sh *model.Sheet, a1 string) *model.Cell {
	t.Helper()
	cr := ref.MustCell(a1)
	c := sh.Lookup(cr.Row, cr.Col)
	if c == nil {
		t.Fatalf("cell %s missing", a1)
	}
	return c
}

func TestReadCellImagesFromXlsxWriter(t *testing.T) {
	src, err := os.ReadFile(cellImagesFixture)
	if err != nil {
		t.Fatal(err)
	}
	check := func(wb *model.Workbook) {
		t.Helper()
		sh := wb.Sheets[0]
		a1, b2, c3 := cellAt(t, sh, "A1"), cellAt(t, sh, "B2"), cellAt(t, sh, "C3")
		for _, c := range []*model.Cell{a1, b2, c3} {
			if c.Image == nil || c.Image.Format != "png" || len(c.Image.Data) == 0 {
				t.Fatalf("%s: no picture: %+v", c.Ref(), c.Image)
			}
			if !c.Value.IsEmpty() || c.ValueMetadata != 0 {
				t.Errorf("%s: placeholder leaked into the model: %+v vm=%d", c.Ref(), c.Value, c.ValueMetadata)
			}
		}
		if !bytes.Equal(a1.Image.Data, c3.Image.Data) || bytes.Equal(a1.Image.Data, b2.Image.Data) {
			t.Error("pictures resolved to the wrong media")
		}
		if a1.Image.AltText != "" || b2.Image.AltText != "A blue box" {
			t.Errorf("alt text: %q, %q", a1.Image.AltText, b2.Image.AltText)
		}
		if d1 := cellAt(t, sh, "D1"); d1.Value.Str != "text" || d1.Image != nil {
			t.Errorf("D1 = %+v", d1)
		}
		if f1 := cellAt(t, sh, "F1"); f1.Formula == nil || !f1.Formula.Dynamic {
			t.Errorf("F1 formula = %+v", f1.Formula)
		}
		if len(wb.RichData) != 0 || len(wb.Metadata) != 0 {
			t.Errorf("rich data kept raw although modelled: %d parts, %d metadata bytes", len(wb.RichData), len(wb.Metadata))
		}
	}
	wb, err := reader.ReadBytes(src)
	if err != nil {
		t.Fatal(err)
	}
	check(wb)

	// Writing it back gives the same parts XlsxWriter produced.
	var out bytes.Buffer
	if err := writer.Write(wb, &out); err != nil {
		t.Fatal(err)
	}
	want, got := unzip(t, src), unzip(t, out.Bytes())
	for _, name := range []string{"xl/metadata.xml", "xl/richData/richValueRel.xml", "xl/richData/rdRichValueTypes.xml"} {
		if !bytes.Equal(bytes.TrimSpace(got[name]), bytes.TrimSpace(want[name])) {
			t.Errorf("%s differs\n got %s\nwant %s", name, got[name], want[name])
		}
	}
	rels := string(got["xl/_rels/workbook.xml.rels"])
	for _, typ := range []string{"2022/10/relationships/richValueRel", "relationships/rdRichValue\"", "relationships/rdRichValueStructure", "relationships/rdRichValueTypes", "relationships/sheetMetadata"} {
		if !strings.Contains(rels, typ) {
			t.Errorf("workbook relationships lack %s", typ)
		}
	}
	ct := string(got["[Content_Types].xml"])
	for _, typ := range []string{"rdrichvalue+xml", "rdrichvaluestructure+xml", "rdrichvaluetypes+xml", "richvaluerel+xml"} {
		if !strings.Contains(ct, typ) {
			t.Errorf("content types lack %s", typ)
		}
	}
	sheet := string(got["xl/worksheets/sheet1.xml"])
	for _, c := range []string{`<c r="A1" t="e" vm="1"><v>#VALUE!</v></c>`, `<c r="B2" t="e" vm="2"><v>#VALUE!</v></c>`, `<c r="C3" t="e" vm="1"><v>#VALUE!</v></c>`} {
		if !strings.Contains(sheet, c) {
			t.Errorf("sheet lacks %s in\n%s", c, sheet)
		}
	}
	media := 0
	for name := range got {
		if strings.HasPrefix(name, "xl/media/") {
			media++
		}
	}
	if media != 2 {
		t.Errorf("media parts = %d, want 2 (one per distinct picture)", media)
	}
	again, err := reader.ReadBytes(out.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	check(again)
}

func TestCellImagesRoundTrip(t *testing.T) {
	wb := wtest.SampleWorkbook() // has a dynamic array formula and a floating picture
	sh := wb.Sheets[0]
	dot := wtest.PNGDot()
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
	sh.Cell(20, 1).Image = &model.CellImage{Data: dot}
	sh.Cell(20, 2).Image = &model.CellImage{Data: dot, AltText: "red <dot> & more"}
	sh.Cell(21, 1).Image = &model.CellImage{Data: dot, Decorative: true, AltText: "dropped"}
	sh.Cell(21, 2).Image = &model.CellImage{Format: "jpeg", Data: jpeg}
	sh.Cell(22, 1).Image = &model.CellImage{Data: dot}

	got := roundTrip(t, wb)
	gs := got.Sheets[0]
	type want struct {
		ref        string
		format     string
		alt        string
		decorative bool
	}
	for _, w := range []want{{"A20", "png", "", false}, {"B20", "png", "red <dot> & more", false}, {"A21", "png", "", true}, {"B21", "jpeg", "", false}, {"A22", "png", "", false}} {
		c := cellAt(t, gs, w.ref)
		if c.Image == nil {
			t.Fatalf("%s: picture lost", w.ref)
		}
		if c.Image.Format != w.format || c.Image.AltText != w.alt || c.Image.Decorative != w.decorative {
			t.Errorf("%s: %+v, want %+v", w.ref, *c.Image, w)
		}
	}
	if cellAt(t, gs, "A20").Image != cellAt(t, gs, "A22").Image {
		t.Error("identical pictures should share one rich value")
	}
	// The floating picture and the dynamic array of the sample survive next to them.
	if len(got.Sheets[1].Images) != 1 {
		t.Errorf("floating pictures = %d", len(got.Sheets[1].Images))
	}
	if f := got.Sheets[1].Cell(1, 8).Formula; f == nil || !f.Dynamic {
		t.Errorf("dynamic array lost: %+v", f)
	}

	// Streaming readers see the pictures too.
	var buf bytes.Buffer
	if err := writer.Write(wb, &buf); err != nil {
		t.Fatal(err)
	}
	st, err := reader.OpenStreamBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	found := 0
	for row, err := range st.Rows(sh.Name) {
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range row.Cells {
			if c.Image != nil {
				found++
			}
		}
	}
	if found != 5 {
		t.Errorf("stream saw %d pictures, want 5", found)
	}
}

func TestCellImageUnsupportedFormat(t *testing.T) {
	wb := model.New()
	sh := wb.AddSheet("S")
	sh.Cell(1, 1).Image = &model.CellImage{Data: []byte("not an image")}
	if err := writer.Write(wb, io.Discard); !errors.Is(err, writer.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

// A workbook whose rich values are not all local images keeps every rich
// data part verbatim, linked from the workbook again.
func TestOtherRichDataPreserved(t *testing.T) {
	src, err := os.ReadFile(cellImagesFixture)
	if err != nil {
		t.Fatal(err)
	}
	parts := unzip(t, src)
	const struc = "xl/richData/rdrichvaluestructure.xml"
	parts[struc] = bytes.ReplaceAll(parts[struc], []byte(`t="_localImage"`), []byte(`t="_entity"`))
	wb, err := reader.ReadBytes(rezip(t, parts))
	if err != nil {
		t.Fatal(err)
	}
	a1 := cellAt(t, wb.Sheets[0], "A1")
	if a1.Image != nil || a1.ValueMetadata != 1 || a1.Value.Str != "#VALUE!" {
		t.Fatalf("A1 = %+v", a1)
	}
	if len(wb.RichData) != 4 || len(wb.Metadata) == 0 {
		t.Fatalf("rich data parts = %d, metadata = %d bytes", len(wb.RichData), len(wb.Metadata))
	}
	var out bytes.Buffer
	if err := writer.Write(wb, &out); err != nil {
		t.Fatal(err)
	}
	got := unzip(t, out.Bytes())
	for _, name := range []string{"xl/metadata.xml", struc, "xl/richData/rdrichvalue.xml", "xl/richData/richValueRel.xml", "xl/richData/_rels/richValueRel.xml.rels", "xl/richData/rdRichValueTypes.xml", "xl/media/image1.png", "xl/media/image2.png"} {
		if !bytes.Equal(got[name], parts[name]) {
			t.Errorf("%s not preserved", name)
		}
	}
	rels := string(got["xl/_rels/workbook.xml.rels"])
	for _, typ := range []string{"2022/10/relationships/richValueRel", "relationships/rdRichValue\"", "relationships/rdRichValueStructure", "relationships/rdRichValueTypes"} {
		if !strings.Contains(rels, typ) {
			t.Errorf("workbook relationships lack %s", typ)
		}
	}
	if !strings.Contains(string(got["xl/worksheets/sheet1.xml"]), `<c r="A1" t="e" vm="1">`) {
		t.Error("vm attribute lost")
	}
	// New pictures cannot be mixed into rich data that is kept verbatim.
	wb.Sheets[0].Cell(9, 9).Image = &model.CellImage{Data: wtest.PNGDot()}
	if err := writer.Write(wb, io.Discard); !errors.Is(err, writer.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}
