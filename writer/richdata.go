// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// Pictures in cells (Excel 365 "Place in Cell") are rich values: the cell
// holds #VALUE! for older readers and a value-metadata index (vm) that leads
// through xl/metadata.xml to a rich value in xl/richData/rdrichvalue.xml,
// which names a relationship in richValueRel.xml that points at the media
// part. The layout below is the one Excel writes for local images.

// CalcOrigin values of a local image rich value.
const (
	calcOriginImage      = 5 // picture inserted by the user
	calcOriginDecorative = 6 // picture marked as decorative
)

// cellImages collects the pictures placed in cells while the sheets are
// written; the parts are emitted once by finish.
type cellImages struct {
	values []cellImageValue // rich values, in vm order
	index  map[[32]byte]int // value key -> index into values
	media  []cellImageMedia // media parts, in relationship order
	byData map[[32]byte]int // image bytes -> index into media
	err    error            // first picture that could not be written
}

// cellImageMedia is a media part that is written once the sheets are done:
// a sheet part is still open while its cells are written.
type cellImageMedia struct {
	name string
	data []byte
}

type cellImageValue struct {
	rel        int // index into media
	altText    string
	decorative bool
}

// cellImageVM registers a picture and returns the vm attribute of its cell,
// or 0 when the picture cannot be written.
func (p *packer) cellImageVM(img *model.CellImage) int {
	ci := &p.cellImages
	if ci.index == nil {
		ci.index, ci.byData = map[[32]byte]int{}, map[[32]byte]int{}
	}
	format := strings.ToLower(img.Format)
	if format == "" {
		format = ooxml.SniffImage(img.Data)
	}
	ct, ok := imageContentTypes[format]
	if !ok || len(img.Data) == 0 {
		if ci.err == nil {
			ci.err = fmt.Errorf("%w: cell image format %q", ErrUnsupported, img.Format)
		}
		return 0
	}
	dataKey := sha256.Sum256(img.Data)
	alt := img.AltText
	if img.Decorative {
		alt = ""
	}
	h := sha256.New()
	h.Write(dataKey[:])
	fmt.Fprintf(h, "|%t|%s", img.Decorative, alt)
	var key [32]byte
	h.Sum(key[:0])
	if i, ok := ci.index[key]; ok {
		return i + 1
	}
	rel, ok := ci.byData[dataKey]
	if !ok {
		p.nImage++
		media := fmt.Sprintf("xl/media/image%d.%s", p.nImage, ct[1])
		p.pw.Default(ct[1], ct[0])
		rel = len(ci.media)
		ci.media = append(ci.media, cellImageMedia{name: media, data: img.Data})
		ci.byData[dataKey] = rel
	}
	ci.values = append(ci.values, cellImageValue{rel: rel, altText: alt, decorative: img.Decorative})
	ci.index[key] = len(ci.values) - 1
	return len(ci.values)
}

// writeCellImages emits the rich data parts for the pictures placed in
// cells and links them from the workbook.
func (p *packer) writeCellImages() error {
	ci := &p.cellImages
	if ci.err != nil {
		return ci.err
	}
	if len(ci.values) == 0 {
		return nil
	}
	if len(p.wb.RichData) > 0 {
		return fmt.Errorf("%w: cell images in a workbook that keeps other rich data types", ErrUnsupported)
	}
	const (
		relPart   = "xl/richData/richValueRel.xml"
		valuePart = "xl/richData/rdrichvalue.xml"
		strucPart = "xl/richData/rdrichvaluestructure.xml"
		typesPart = "xl/richData/rdRichValueTypes.xml"
	)
	// Relationships to the media parts.
	b := ooxml.NewBuilder()
	b.Start("richValueRels", "xmlns", ooxml.NSRichValueRel, "xmlns:r", ooxml.NSRel)
	for _, media := range ci.media {
		if err := p.pw.Add(media.name, "", media.data); err != nil {
			return err
		}
		id := p.pw.Rel(relPart, ooxml.RelImage, ooxml.Relative(relPart, media.name), "")
		b.Empty("rel", "r:id", id)
	}
	b.End()
	if err := p.pw.Add(relPart, ooxml.CTRichValueRel, b.Bytes()); err != nil {
		return err
	}
	// Rich values. Structure 0 is a plain local image; a second structure
	// with a Text key is added when any picture has alternative text.
	withText := -1
	for _, v := range ci.values {
		if v.altText != "" {
			withText = 1
		}
	}
	b = ooxml.NewBuilder()
	b.Start("rvData", "xmlns", ooxml.NSRichData, "count", ooxml.Int(len(ci.values)))
	for _, v := range ci.values {
		origin := calcOriginImage
		if v.decorative {
			origin = calcOriginDecorative
		}
		if v.altText != "" {
			b.Start("rv", "s", ooxml.Int(withText))
		} else {
			b.Start("rv", "s", "0")
		}
		b.Elem("v", ooxml.Int(v.rel))
		b.Elem("v", ooxml.Int(origin))
		if v.altText != "" {
			b.Elem("v", v.altText)
		}
		b.End()
	}
	b.End()
	if err := p.pw.Add(valuePart, ooxml.CTRdRichValue, b.Bytes()); err != nil {
		return err
	}
	b = ooxml.NewBuilder()
	n := 1
	if withText > 0 {
		n = 2
	}
	b.Start("rvStructures", "xmlns", ooxml.NSRichData, "count", ooxml.Int(n))
	for i := 0; i < n; i++ {
		b.Start("s", "t", "_localImage")
		b.Empty("k", "n", "_rvRel:LocalImageIdentifier", "t", "i")
		b.Empty("k", "n", "CalcOrigin", "t", "i")
		if i == 1 {
			b.Empty("k", "n", "Text", "t", "s")
		}
		b.End()
	}
	b.End()
	if err := p.pw.Add(strucPart, ooxml.CTRdRichValueStruc, b.Bytes()); err != nil {
		return err
	}
	if err := p.pw.Add(typesPart, ooxml.CTRdRichValueTypes, richValueTypesXML()); err != nil {
		return err
	}
	const wb = "xl/workbook.xml"
	p.pw.Rel(wb, ooxml.RelRichValueRel, ooxml.Relative(wb, relPart), "")
	p.pw.Rel(wb, ooxml.RelRdRichValue, ooxml.Relative(wb, valuePart), "")
	p.pw.Rel(wb, ooxml.RelRdRichValueStruc, ooxml.Relative(wb, strucPart), "")
	p.pw.Rel(wb, ooxml.RelRdRichValueTypes, ooxml.Relative(wb, typesPart), "")
	return nil
}

// richValueTypesXML is the fixed xl/richData/rdRichValueTypes.xml part.
func richValueTypesXML() []byte {
	var sb strings.Builder
	sb.WriteString(ooxml.Header + `<rvTypesInfo xmlns="` + ooxml.NSRichData2 + `" xmlns:mc="` + ooxml.NSMC +
		`" mc:Ignorable="x" xmlns:x="` + ooxml.NSMain + `"><global><keyFlags>` +
		`<key name="_Self"><flag name="ExcludeFromFile" value="1"/><flag name="ExcludeFromCalcComparison" value="1"/></key>`)
	for _, k := range []string{"_DisplayString", "_Flags", "_Format", "_SubLabel", "_Attribution", "_Icon", "_Display", "_CanonicalPropertyNames", "_ClassificationId"} {
		sb.WriteString(`<key name="` + k + `"><flag name="ExcludeFromCalcComparison" value="1"/></key>`)
	}
	sb.WriteString(`</keyFlags></global></rvTypesInfo>`)
	return []byte(sb.String())
}

// metadataXML is the xl/metadata.xml part: the dynamic array marker that
// cm="1" refers to and one value-metadata block per picture in a cell.
func metadataXML(dynamic bool, images int) []byte {
	const flags = ` minSupportedVersion="120000" copy="1" pasteAll="1" pasteValues="1" merge="1" splitFirst="1" rowColShift="1" clearFormats="1" clearComments="1" assign="1" coerce="1"`
	var sb strings.Builder
	sb.WriteString(ooxml.Header + `<metadata xmlns="` + ooxml.NSMain + `"`)
	if images > 0 {
		sb.WriteString(` xmlns:xlrd="` + ooxml.NSRichData + `"`)
	}
	if dynamic {
		sb.WriteString(` xmlns:xda="` + ooxml.NSXDA + `"`)
	}
	types := 0
	if dynamic {
		types++
	}
	richType := 0
	if images > 0 {
		types++
		richType = types
	}
	fmt.Fprintf(&sb, `><metadataTypes count="%d">`, types)
	if dynamic {
		sb.WriteString(`<metadataType name="XLDAPR"` + flags + ` cellMeta="1"/>`)
	}
	if images > 0 {
		sb.WriteString(`<metadataType name="XLRICHVALUE"` + flags + `/>`)
	}
	sb.WriteString(`</metadataTypes>`)
	if dynamic {
		sb.WriteString(`<futureMetadata name="XLDAPR" count="1"><bk><extLst><ext uri="{bdbb8cdc-fa1e-496e-a857-3c3f30c029c3}"><xda:dynamicArrayProperties fDynamic="1" fCollapsed="0"/></ext></extLst></bk></futureMetadata>`)
	}
	if images > 0 {
		fmt.Fprintf(&sb, `<futureMetadata name="XLRICHVALUE" count="%d">`, images)
		for i := 0; i < images; i++ {
			fmt.Fprintf(&sb, `<bk><extLst><ext uri="{3e2802c4-a4d2-4d8b-9148-e3be6c30e623}"><xlrd:rvb i="%d"/></ext></extLst></bk>`, i)
		}
		sb.WriteString(`</futureMetadata>`)
	}
	if dynamic {
		sb.WriteString(`<cellMetadata count="1"><bk><rc t="1" v="0"/></bk></cellMetadata>`)
	}
	if images > 0 {
		fmt.Fprintf(&sb, `<valueMetadata count="%d">`, images)
		for i := 0; i < images; i++ {
			fmt.Fprintf(&sb, `<bk><rc t="%d" v="%d"/></bk>`, richType, i)
		}
		sb.WriteString(`</valueMetadata>`)
	}
	sb.WriteString(`</metadata>`)
	return []byte(sb.String())
}
