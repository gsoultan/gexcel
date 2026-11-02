// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"bytes"
	"testing"
)

func TestBuilder(t *testing.T) {
	b := NewBuilder()
	b.Start("a", "x", "1", "empty", "", "keep!", "").Elem("t", "<&>\"", "n", "v\"q").Empty("e").End()
	want := Header + `<a x="1" keep=""><t n="v&#34;q">&lt;&amp;&gt;&#34;</t><e/></a>`
	if b.String() != want {
		t.Errorf("got %s", b.String())
	}
	if Clean("a\x00b\x01c\td") != "abc\td" {
		t.Error("Clean")
	}
	if EscapeUnicode("a\x01b_x0041_c") != "a_x0001_b_x005F_x0041_c" {
		t.Error("EscapeUnicode", EscapeUnicode("a\x01b_x0041_c"))
	}
	if UnescapeUnicode("a_x0001_b_x005F_x0041_c") != "a\x01b_x0041_c" {
		t.Error("UnescapeUnicode", UnescapeUnicode("a_x0001_b_x005F_x0041_c"))
	}
	cases := map[float64]string{1: "1", 1.5: "1.5", 0.1: "0.1", 1e21: "1E+21", 123456789012345: "123456789012345", 1e-7: "1E-07", -2.5: "-2.5", 1e15: "1000000000000000"}
	for v, want := range cases {
		if got := Float(v); got != want {
			t.Errorf("Float(%v)=%q want %q", v, got, want)
		}
	}
}

func TestPackageRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.Add("xl/workbook.xml", CTWorkbook, []byte("<wb/>")); err != nil {
		t.Fatal(err)
	}
	w.Rel("", RelOfficeDocument, "xl/workbook.xml", "")
	id := w.Rel("xl/workbook.xml", RelWorksheet, "worksheets/sheet1.xml", "")
	if id != "rId1" {
		t.Error(id)
	}
	f, err := w.Create("xl/worksheets/sheet1.xml", CTWorksheet)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("<ws/>"))
	if err := w.Add("xl/media/image1.png", "", []byte("png")); err != nil {
		t.Fatal(err)
	}
	w.Default("png", "image/png")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := OpenBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if r.ContentType("xl/workbook.xml") != CTWorkbook || r.ContentType("xl/media/image1.png") != "image/png" || r.ContentType("_rels/.rels") != CTRels {
		t.Error("content types")
	}
	rels, _ := r.Rels("")
	if rel, ok := FindRel(rels, RelOfficeDocument); !ok || rel.Target != "xl/workbook.xml" {
		t.Error("root rels", rels)
	}
	rels, _ = r.Rels("xl/workbook.xml")
	if rel, ok := RelByID(rels, "rId1"); !ok || Resolve("xl/workbook.xml", rel.Target) != "xl/worksheets/sheet1.xml" {
		t.Error("workbook rels", rels)
	}
	data, err := r.Part("xl/worksheets/sheet1.xml")
	if err != nil || string(data) != "<ws/>" {
		t.Error(string(data), err)
	}
	if _, err := r.Part("missing"); err == nil {
		t.Error("missing part")
	}
	if Resolve("xl/worksheets/sheet1.xml", "../drawings/drawing1.xml") != "xl/drawings/drawing1.xml" {
		t.Error("Resolve")
	}
	if Resolve("xl/workbook.xml", "/xl/styles.xml") != "xl/styles.xml" {
		t.Error("Resolve absolute")
	}
	if Relative("xl/worksheets/sheet1.xml", "xl/drawings/drawing1.xml") != "../drawings/drawing1.xml" {
		t.Error("Relative", Relative("xl/worksheets/sheet1.xml", "xl/drawings/drawing1.xml"))
	}
	if Relative("xl/workbook.xml", "xl/worksheets/sheet1.xml") != "worksheets/sheet1.xml" {
		t.Error("Relative", Relative("xl/workbook.xml", "xl/worksheets/sheet1.xml"))
	}
	if Relative("", "xl/workbook.xml") != "xl/workbook.xml" {
		t.Error("Relative root")
	}
	if RelsName("xl/worksheets/sheet1.xml") != "xl/worksheets/_rels/sheet1.xml.rels" {
		t.Error("RelsName")
	}
	if _, err := OpenBytes([]byte("not a zip")); err == nil {
		t.Error("garbage accepted")
	}
}
