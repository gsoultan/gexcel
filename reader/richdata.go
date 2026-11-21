// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package reader

import (
	"strings"

	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
)

// isRichDataRel reports whether a workbook relationship points at a rich
// data part (rich values, their structures and types, image relationships,
// arrays, property bags, rich styles).
func isRichDataRel(relType string) bool {
	return relType == ooxml.RelRichValue || relType == ooxml.RelRichValueRel || strings.HasPrefix(relType, ooxml.RelRichDataPrefix)
}

// readCellImages resolves the pictures placed in cells: value metadata index
// (the vm attribute of a cell) -> picture. It only takes over when every
// rich value of the workbook is a local image; a workbook that also holds
// other rich data (stocks, geography, web images) keeps all of it verbatim,
// because the parts index each other and cannot be rewritten piecemeal.
func (d *docReader) readCellImages(wbRels []ooxml.Relationship) {
	if len(d.wb.Metadata) == 0 {
		return
	}
	var valuePart, strucPart, relPart, typesPart string
	for _, rel := range wbRels {
		name := ooxml.Resolve(d.wbPart, rel.Target)
		switch rel.Type {
		case ooxml.RelRdRichValue:
			valuePart = name
		case ooxml.RelRdRichValueStruc:
			strucPart = name
		case ooxml.RelRichValueRel:
			relPart = name
		case ooxml.RelRdRichValueTypes:
			typesPart = name
		default:
			if isRichDataRel(rel.Type) {
				return
			}
		}
	}
	if valuePart == "" || strucPart == "" || relPart == "" {
		return
	}
	// Structures: the keys of each rich value type.
	strucs, err := d.dom(strucPart)
	if err != nil || strucs == nil {
		return
	}
	var keys [][]string
	for _, s := range strucs.Each("s") {
		if s.Attr("t") != "_localImage" {
			return
		}
		var ks []string
		for _, k := range s.Each("k") {
			ks = append(ks, k.Attr("n"))
		}
		keys = append(keys, ks)
	}
	// Relationships to the media parts, in <rel> order.
	relDOM, err := d.dom(relPart)
	if err != nil || relDOM == nil {
		return
	}
	rels, _ := d.pr.Rels(relPart)
	var media []string
	for _, r := range relDOM.Each("rel") {
		rel, ok := ooxml.RelByID(rels, r.Attr("id"))
		if !ok || rel.TargetMode == "External" {
			return
		}
		media = append(media, ooxml.Resolve(relPart, rel.Target))
	}
	// Rich values.
	values, err := d.dom(valuePart)
	if err != nil || values == nil {
		return
	}
	var images []*model.CellImage
	for _, rv := range values.Each("rv") {
		s := ooxml.ParseInt(rv.Attr("s"), -1)
		if s < 0 || s >= len(keys) {
			return
		}
		img := &model.CellImage{}
		rel := -1
		for i, v := range rv.Each("v") {
			if i >= len(keys[s]) {
				break
			}
			switch keys[s][i] {
			case "_rvRel:LocalImageIdentifier":
				rel = ooxml.ParseInt(v.Text, -1)
			case "CalcOrigin":
				img.Decorative = ooxml.ParseInt(v.Text, 0) == 6
			case "Text":
				img.AltText = v.Text
			}
		}
		if rel < 0 || rel >= len(media) {
			return
		}
		data, err := d.pr.Part(media[rel])
		if err != nil {
			return
		}
		img.Data = data
		img.Format = ooxml.SniffImage(data)
		if img.Format == "" {
			if i := strings.LastIndexByte(media[rel], '.'); i >= 0 {
				img.Format = strings.ToLower(media[rel][i+1:])
			}
		}
		images = append(images, img)
	}
	// Metadata: vm -> metadata type and index -> rich value.
	meta, err := ooxml.ParseDOM(d.wb.Metadata)
	if err != nil || meta == nil {
		return
	}
	richType := 0
	for i, t := range meta.Child("metadataTypes").Each("metadataType") {
		switch t.Attr("name") {
		case "XLRICHVALUE":
			richType = i + 1
		case "XLDAPR":
		default:
			return
		}
	}
	if richType == 0 {
		return
	}
	var blocks []int // XLRICHVALUE block -> rich value index
	for _, fm := range meta.Each("futureMetadata") {
		if fm.Attr("name") != "XLRICHVALUE" {
			continue
		}
		for _, bk := range fm.Each("bk") {
			rvb := bk.Path("extLst", "ext", "rvb")
			if rvb == nil {
				return
			}
			blocks = append(blocks, ooxml.ParseInt(rvb.Attr("i"), -1))
		}
	}
	byVM := map[int]*model.CellImage{}
	for i, bk := range meta.Child("valueMetadata").Each("bk") {
		rc := bk.Child("rc")
		if rc == nil || ooxml.ParseInt(rc.Attr("t"), 0) != richType {
			return
		}
		v := ooxml.ParseInt(rc.Attr("v"), -1)
		if v < 0 || v >= len(blocks) || blocks[v] < 0 || blocks[v] >= len(images) {
			return
		}
		byVM[i+1] = images[blocks[v]]
	}
	if len(byVM) == 0 {
		return
	}
	// The pictures are in the model now; the writer regenerates the parts.
	d.cellImages = byVM
	d.wb.Metadata = nil
	for _, name := range append([]string{valuePart, strucPart, relPart, typesPart}, media...) {
		if name != "" {
			d.use(name)
			d.use(ooxml.RelsName(name))
		}
	}
}
