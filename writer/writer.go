// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package writer serialises a model.Workbook as an Office Open XML
// SpreadsheetML package (.xlsx / .xlsm / .xltx). StreamWriter writes very
// large sheets row by row without holding them in memory.
package writer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gsoultan/gexcel/formula"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/gexcel/ooxml"
	"github.com/gsoultan/gexcel/ref"
)

// Options tune the output.
type Options struct {
	// Template writes an .xltx/.xltm content type.
	Template bool
	// InlineStrings stores every string inline instead of in the shared
	// string table (bigger files, less memory).
	InlineStrings bool
	// Modified is the timestamp stored in the zip entries and, when the
	// workbook properties have no Modified time, in docProps/core.xml.
	// Zero uses a fixed 1980 timestamp so output is reproducible.
	Modified time.Time
	// NoFullCalcOnLoad leaves calcPr fullCalcOnLoad unset: cached formula
	// values are trusted as they are.
	NoFullCalcOnLoad bool
}

// Errors.
var (
	ErrNoSheets         = errors.New("writer: workbook has no sheets")
	ErrInvalidSheetName = errors.New("writer: invalid sheet name")
	ErrDuplicateSheet   = errors.New("writer: duplicate sheet name")
	ErrUnsupported      = errors.New("writer: unsupported feature")
	ErrStreaming        = errors.New("writer: operation not allowed in this streaming state")
)

// Write serialises the workbook to w.
func Write(wb *model.Workbook, w io.Writer, opts ...Options) error {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if err := Validate(wb); err != nil {
		return err
	}
	p := newPacker(wb, w, o)
	for i, sh := range wb.Sheets {
		if err := p.writeSheet(i, sh, nil); err != nil {
			return err
		}
	}
	return p.finish()
}

// WriteFile writes the workbook to a file path.
func WriteFile(wb *model.Workbook, path string, opts ...Options) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := Write(wb, f, opts...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Validate checks what Excel refuses: sheet names and duplicates.
func Validate(wb *model.Workbook) error {
	if len(wb.Sheets) == 0 {
		return ErrNoSheets
	}
	seen := map[string]bool{}
	for _, s := range wb.Sheets {
		if !model.ValidSheetName(s.Name) {
			return fmt.Errorf("%w: %q", ErrInvalidSheetName, s.Name)
		}
		k := strings.ToLower(s.Name)
		if seen[k] {
			return fmt.Errorf("%w: %q", ErrDuplicateSheet, s.Name)
		}
		seen[k] = true
	}
	return nil
}

// packer holds the state of one package being written.
type packer struct {
	wb   *model.Workbook
	pw   *ooxml.Writer
	opts Options
	sst  *sharedStrings

	sheetParts []sheetPart
	// counters for part names
	nDrawing, nChart, nComments, nTable, nImage, nVML, nThreaded int
	persons                                                      map[string]string // author -> person id
	personOrder                                                  []string
	hasDynamic                                                   bool
	workbookRels                                                 []ooxml.Relationship
	extraDefinedNames                                            []model.DefinedName
}

type sheetPart struct {
	name  string // part name
	relID string
	sheet *model.Sheet
}

func newPacker(wb *model.Workbook, w io.Writer, o Options) *packer {
	p := &packer{wb: wb, pw: ooxml.NewWriter(w), opts: o, sst: newSharedStrings(), persons: map[string]string{}}
	if !o.Modified.IsZero() {
		p.pw.SetModified(o.Modified.UTC())
	}
	return p
}

// sheetPartName returns the part name of sheet i.
func sheetPartName(i int, kind model.SheetKind) string {
	switch kind {
	case model.Chartsheet:
		return fmt.Sprintf("xl/chartsheets/sheet%d.xml", i+1)
	case model.MacroSheet:
		return fmt.Sprintf("xl/macrosheets/sheet%d.xml", i+1)
	case model.DialogSheet:
		return fmt.Sprintf("xl/dialogsheets/sheet%d.xml", i+1)
	}
	return fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
}

// writeSheet writes sheet i. When rows is non-nil (streaming), the row
// data is produced by the callback instead of the model.
func (p *packer) writeSheet(i int, sh *model.Sheet, rows func(io.Writer) error) error {
	name := sheetPartName(i, sh.Kind)
	part := sheetPart{name: name, sheet: sh}
	relType := ooxml.RelWorksheet
	ct := ooxml.CTWorksheet
	switch sh.Kind {
	case model.Chartsheet:
		relType, ct = ooxml.RelChartsheet, ooxml.CTChartsheet
	case model.MacroSheet:
		relType, ct = ooxml.RelMacrosheet, ooxml.CTMacrosheet
	case model.DialogSheet:
		relType, ct = ooxml.RelDialogsheet, ooxml.CTDialogsheet
	}
	part.relID = p.pw.Rel("xl/workbook.xml", relType, ooxml.Relative("xl/workbook.xml", name), "")
	p.sheetParts = append(p.sheetParts, part)

	if sh.Raw != nil && sh.Kind != model.Worksheet && (sh.Kind != model.Chartsheet || len(sh.Charts) == 0) {
		return p.writeRawSheet(name, ct, sh)
	}
	if sh.Kind == model.Chartsheet {
		return p.writeChartsheet(i, name, sh)
	}

	// Sub-parts referenced from the sheet must have their relationships
	// registered before the sheet XML is written (it needs the ids).
	sw := &sheetWriter{p: p, sh: sh, part: name, index: i}
	if err := sw.prepare(); err != nil {
		return err
	}
	w, err := p.pw.Create(name, ct)
	if err != nil {
		return err
	}
	if err := sw.writeHead(w); err != nil {
		return err
	}
	if rows != nil {
		if err := rows(w); err != nil {
			return err
		}
	} else {
		if err := sw.writeRows(w); err != nil {
			return err
		}
	}
	return sw.writeTail(w)
}

func (p *packer) writeRawSheet(name, ct string, sh *model.Sheet) error {
	if err := p.pw.Add(name, ct, sh.Raw.Data); err != nil {
		return err
	}
	return p.writeRawRelated(name, sh.Raw)
}

// writeRawRelated writes the preserved relationships and parts of a raw part.
func (p *packer) writeRawRelated(name string, raw *model.RawPart) error {
	for _, r := range raw.Rels {
		p.pw.RelWithID(name, r.ID, r.Type, r.Target, r.TargetMode)
		if r.TargetMode == "External" {
			continue
		}
		target := ooxml.Resolve(name, r.Target)
		if sub, ok := raw.Related[target]; ok && !p.pw.Has(target) {
			if err := p.pw.Add(target, sub.ContentType, sub.Data); err != nil {
				return err
			}
			if err := p.writeRawRelated(target, sub); err != nil {
				return err
			}
		}
	}
	return nil
}

// finish writes the workbook-level parts and closes the package.
func (p *packer) finish() error {
	wb := p.wb
	// Styles.
	if err := p.pw.Add("xl/styles.xml", ooxml.CTStyles, stylesXML(wb.Styles)); err != nil {
		return err
	}
	p.pw.Rel("xl/workbook.xml", ooxml.RelStyles, "styles.xml", "")
	// Theme.
	theme := wb.Theme
	var themeData []byte
	if theme != nil && len(theme.Raw) > 0 {
		themeData = theme.Raw
	} else {
		themeData = themeXML(theme)
	}
	if err := p.pw.Add("xl/theme/theme1.xml", ooxml.CTTheme, themeData); err != nil {
		return err
	}
	p.pw.Rel("xl/workbook.xml", ooxml.RelTheme, "theme/theme1.xml", "")
	// Shared strings.
	if p.sst.count > 0 {
		if err := p.pw.Add("xl/sharedStrings.xml", ooxml.CTSharedStrings, p.sst.xml()); err != nil {
			return err
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelSharedStrings, "sharedStrings.xml", "")
	}
	// Dynamic array metadata.
	if p.hasDynamic || len(wb.Metadata) > 0 {
		data := wb.Metadata
		if len(data) == 0 {
			data = metadataXML()
		}
		if err := p.pw.Add("xl/metadata.xml", ooxml.CTMetadata, data); err != nil {
			return err
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelMetadata, "metadata.xml", "")
	}
	// Persons for threaded comments.
	if len(p.personOrder) > 0 {
		if err := p.pw.Add("xl/persons/person.xml", ooxml.CTPersons, p.personsXML()); err != nil {
			return err
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelPersons, "persons/person.xml", "")
	}
	// External links.
	for i, el := range wb.ExternalLinks {
		name := fmt.Sprintf("xl/externalLinks/externalLink%d.xml", i+1)
		if el.Raw != nil {
			if err := p.pw.Add(name, ooxml.CTExternalLink, el.Raw.Data); err != nil {
				return err
			}
			if err := p.writeRawRelated(name, el.Raw); err != nil {
				return err
			}
		} else {
			if err := p.pw.Add(name, ooxml.CTExternalLink, externalLinkXML(el)); err != nil {
				return err
			}
			p.pw.Rel(name, ooxml.RelExternalLinkPath, el.Target, "External")
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelExternalLink, ooxml.Relative("xl/workbook.xml", name), "")
	}
	// Pivot caches referenced from the workbook.
	var cacheIDs []int
	for id := range wb.PivotCaches {
		cacheIDs = append(cacheIDs, id)
	}
	sort.Ints(cacheIDs)
	cacheRels := map[int]string{}
	for _, id := range cacheIDs {
		raw := wb.PivotCaches[id]
		if !p.pw.Has(raw.Name) {
			if err := p.pw.Add(raw.Name, raw.ContentType, raw.Data); err != nil {
				return err
			}
			if err := p.writeRawRelated(raw.Name, raw); err != nil {
				return err
			}
		}
		cacheRels[id] = p.pw.Rel("xl/workbook.xml", ooxml.RelPivotCacheDef, ooxml.Relative("xl/workbook.xml", raw.Name), "")
	}
	// VBA project.
	if len(wb.VBAProject) > 0 {
		if err := p.pw.Add("xl/vbaProject.bin", ooxml.CTVBA, wb.VBAProject); err != nil {
			return err
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelVBA, "vbaProject.bin", "")
	}
	// Preserved parts (custom XML, connections, rich data, ...).
	for _, raw := range wb.Preserved {
		if p.pw.Has(raw.Name) {
			continue
		}
		if err := p.pw.Add(raw.Name, raw.ContentType, raw.Data); err != nil {
			return err
		}
		if err := p.writeRawRelated(raw.Name, raw); err != nil {
			return err
		}
		if relType := relTypeForPreserved(raw); relType != "" {
			p.pw.Rel("xl/workbook.xml", relType, ooxml.Relative("xl/workbook.xml", raw.Name), "")
		}
	}
	for _, raw := range wb.RichData {
		if p.pw.Has(raw.Name) {
			continue
		}
		if err := p.pw.Add(raw.Name, raw.ContentType, raw.Data); err != nil {
			return err
		}
		if err := p.writeRawRelated(raw.Name, raw); err != nil {
			return err
		}
		p.pw.Rel("xl/workbook.xml", ooxml.RelRichValue, ooxml.Relative("xl/workbook.xml", raw.Name), "")
	}
	// Workbook.
	ct := ooxml.CTWorkbook
	switch {
	case len(wb.VBAProject) > 0 && p.opts.Template:
		ct = ooxml.CTTemplateMacro
	case len(wb.VBAProject) > 0:
		ct = ooxml.CTWorkbookMacro
	case p.opts.Template:
		ct = ooxml.CTTemplate
	}
	if err := p.pw.Add("xl/workbook.xml", ct, p.workbookXML(cacheIDs, cacheRels)); err != nil {
		return err
	}
	p.pw.Rel("", ooxml.RelOfficeDocument, "xl/workbook.xml", "")
	// Properties.
	now := p.opts.Modified
	if now.IsZero() {
		now = time.Now()
	}
	if err := p.pw.Add("docProps/core.xml", ooxml.CTCoreProps, corePropsXML(wb.Properties, now)); err != nil {
		return err
	}
	p.pw.Rel("", ooxml.RelCoreProps, "docProps/core.xml", "")
	if err := p.pw.Add("docProps/app.xml", ooxml.CTExtProps, appPropsXML(wb)); err != nil {
		return err
	}
	p.pw.Rel("", ooxml.RelExtProps, "docProps/app.xml", "")
	if len(wb.CustomProps) > 0 {
		if err := p.pw.Add("docProps/custom.xml", ooxml.CTCustomProps, customPropsXML(wb.CustomProps)); err != nil {
			return err
		}
		p.pw.Rel("", ooxml.RelCustomProps, "docProps/custom.xml", "")
	}
	return p.pw.Close()
}

func relTypeForPreserved(raw *model.RawPart) string {
	switch raw.ContentType {
	case ooxml.CTConnections:
		return ooxml.RelConnections
	case ooxml.CTSlicerCache:
		return ooxml.RelSlicerCache
	}
	if strings.HasPrefix(raw.Name, "customXml/item") && !strings.Contains(raw.Name, "Props") {
		return ooxml.RelCustomXML
	}
	return ""
}

// workbookXML builds xl/workbook.xml.
func (p *packer) workbookXML(cacheIDs []int, cacheRels map[int]string) []byte {
	wb := p.wb
	b := ooxml.NewBuilder()
	b.Start("workbook", "xmlns", ooxml.NSMain, "xmlns:r", ooxml.NSRel, "xmlns:mc", ooxml.NSMC, "mc:Ignorable", "x15 xr xr6 xr10 xr2",
		"xmlns:x15", ooxml.NSX15, "xmlns:xr", ooxml.NSXR, "xmlns:xr6", ooxml.NSXR6, "xmlns:xr10", ooxml.NSXR10, "xmlns:xr2", ooxml.NSXR2)
	b.Empty("fileVersion", "appName", "xl", "lastEdited", "7", "lowestEdited", "7", "rupBuild", "27425")
	attrs := []string{"date1904", boolAttr(wb.Date1904), "codeName", wb.CodeName, "filterPrivacy", boolAttr(wb.FilterPrivacy),
		"hidePivotFieldList", boolAttr(wb.HidePivotFieldList), "updateLinks", wb.UpdateLinks}
	if wb.ShowInkAnnotation != nil {
		attrs = append(attrs, "showInkAnnotation", ooxml.Bool(*wb.ShowInkAnnotation))
	}
	if wb.DefaultThemeVersion != 0 {
		attrs = append(attrs, "defaultThemeVersion", ooxml.Int(wb.DefaultThemeVersion))
	} else {
		attrs = append(attrs, "defaultThemeVersion", "166925")
	}
	b.Empty("workbookPr", attrs...)
	if wp := wb.Protection; wp != nil {
		attrs := []string{"lockStructure", boolAttr(wp.LockStructure), "lockWindows", boolAttr(wp.LockWindows), "lockRevision", boolAttr(wp.LockRevision)}
		if wp.Password != "" || wp.HashValue != "" {
			alg, hash, salt, spin := wp.Algorithm, wp.HashValue, wp.SaltValue, wp.SpinCount
			if wp.Password != "" {
				alg, hash, salt, spin = hashPassword(wp.Password)
			}
			attrs = append(attrs, "workbookAlgorithmName", alg, "workbookHashValue", hash, "workbookSaltValue", salt, "workbookSpinCount", ooxml.Int(spin))
		} else if wp.LegacyHash != "" {
			attrs = append(attrs, "workbookPassword", wp.LegacyHash)
		}
		if wp.RevisionPassword != "" {
			alg, hash, salt, spin := hashPassword(wp.RevisionPassword)
			attrs = append(attrs, "revisionsAlgorithmName", alg, "revisionsHashValue", hash, "revisionsSaltValue", salt, "revisionsSpinCount", ooxml.Int(spin))
		} else if wp.RevisionsHash != "" {
			attrs = append(attrs, "revisionsAlgorithmName", wp.RevisionsAlgorithm, "revisionsHashValue", wp.RevisionsHash, "revisionsSaltValue", wp.RevisionsSalt, "revisionsSpinCount", ooxml.Int(wp.RevisionsSpin))
		}
		b.Empty("workbookProtection", attrs...)
	}
	v := wb.View
	b.Start("bookViews")
	va := []string{"xWindow", ooxml.Int(v.XWindow), "yWindow", ooxml.Int(v.YWindow)}
	if v.WindowWidth > 0 {
		va = append(va, "windowWidth", ooxml.Int(v.WindowWidth), "windowHeight", ooxml.Int(v.WindowHeight))
	} else {
		va = append(va, "windowWidth", "28800", "windowHeight", "12000")
	}
	va = append(va, "activeTab", intAttr(v.ActiveTab), "firstSheet", intAttr(v.FirstSheet), "visibility", v.Visibility,
		"minimized", boolAttr(v.Minimized), "showHorizontalScroll", invBool(v.HideHorizontalScroll), "showVerticalScroll", invBool(v.HideVerticalScroll),
		"showSheetTabs", invBool(v.HideSheetTabs), "tabRatio", intAttr(v.TabRatio))
	if v.AutoFilterDateGrouping != nil {
		va = append(va, "autoFilterDateGrouping", ooxml.Bool(*v.AutoFilterDateGrouping))
	}
	b.Empty("workbookView", va...)
	b.End()
	b.Start("sheets")
	for i, sp := range p.sheetParts {
		b.Empty("sheet", "name", sp.sheet.Name, "sheetId", ooxml.Int(i+1), "state", string(sp.sheet.State), "r:id", sp.relID)
	}
	b.End()
	if n := len(wb.ExternalLinks); n > 0 {
		b.Start("externalReferences")
		for i := range wb.ExternalLinks {
			// The external link rels were added after the sheets, styles, theme, sst...
			b.Empty("externalReference", "r:id", p.externalLinkRelID(i))
		}
		b.End()
	}
	names := p.definedNames()
	if len(names) > 0 {
		b.Start("definedNames")
		for _, d := range names {
			attrs := []string{"name", d.Name, "comment", d.Comment, "hidden", boolAttr(d.Hidden), "function", boolAttr(d.Function), "vbProcedure", boolAttr(d.VBProc)}
			if d.Local {
				attrs = append(attrs, "localSheetId", ooxml.Int(d.SheetIndex))
			}
			b.Elem("definedName", formula.AddPrefixes(d.RefersTo), attrs...)
		}
		b.End()
	}
	c := wb.Calc
	ca := []string{"calcId", "191029"}
	if c.CalcID != 0 {
		ca[1] = ooxml.Int(c.CalcID)
	}
	ca = append(ca, "calcMode", c.CalcMode)
	if c.FullCalcOnLoad || !p.opts.NoFullCalcOnLoad && p.needsCalc() {
		ca = append(ca, "fullCalcOnLoad", "1")
	}
	ca = append(ca, "refMode", map[bool]string{true: "R1C1", false: ""}[c.RefModeR1C1], "iterate", boolAttr(c.Iterate),
		"iterateCount", intAttr(c.IterateCount), "fullPrecision", map[bool]string{true: "0", false: ""}[c.NoFullPrecision], "forceFullCalc", boolAttr(c.ForceFullCalc))
	if c.IterateDelta != 0 {
		ca = append(ca, "iterateDelta", ooxml.Float(c.IterateDelta))
	}
	if c.CalcCompleted != nil {
		ca = append(ca, "calcCompleted", ooxml.Bool(*c.CalcCompleted))
	}
	if c.CalcOnSave != nil {
		ca = append(ca, "calcOnSave", ooxml.Bool(*c.CalcOnSave))
	}
	if c.ConcurrentCalc != nil {
		ca = append(ca, "concurrentCalc", ooxml.Bool(*c.ConcurrentCalc))
	}
	b.Empty("calcPr", ca...)
	if len(cacheIDs) > 0 {
		b.Start("pivotCaches")
		for _, id := range cacheIDs {
			b.Empty("pivotCache", "cacheId", ooxml.Int(id), "r:id", cacheRels[id])
		}
		b.End()
	}
	if len(wb.Extensions) > 0 {
		b.Start("extLst")
		for _, e := range wb.Extensions {
			b.Write(e.Data)
		}
		b.End()
	}
	b.End()
	return b.Bytes()
}

func (p *packer) externalLinkRelID(i int) string {
	// Find the i-th externalLink relationship on the workbook part.
	// Relationship ids are assigned sequentially; recompute by scanning.
	return p.relIDFor("xl/workbook.xml", ooxml.RelExternalLink, i)
}

// relIDFor returns the id of the n-th relationship of a type from a part.
func (p *packer) relIDFor(part, relType string, n int) string {
	return p.pw.RelIDByType(part, relType, n)
}

// needsCalc reports whether any cell has a formula without a cached value,
// so Excel should recalculate on load.
func (p *packer) needsCalc() bool {
	for _, sp := range p.sheetParts {
		for _, r := range sp.sheet.Rows {
			for _, c := range r.Cells {
				if c.Formula != nil && c.Value.IsEmpty() {
					return true
				}
			}
		}
	}
	return false
}

// definedNames merges the workbook's names with the ones derived from
// sheet print settings and autofilters.
func (p *packer) definedNames() []model.DefinedName {
	var out []model.DefinedName
	skip := map[string]bool{}
	for i, sp := range p.sheetParts {
		sh := sp.sheet
		q := ref.QuoteSheet(sh.Name)
		if len(sh.PrintArea) > 0 {
			parts := make([]string, len(sh.PrintArea))
			for j, r := range sh.PrintArea {
				parts[j] = q + "!" + r.Abs().String()
			}
			out = append(out, model.DefinedName{Name: model.NamePrintArea, RefersTo: strings.Join(parts, ","), Local: true, SheetIndex: i})
			skip[fmt.Sprintf("%s/%d", model.NamePrintArea, i)] = true
		}
		if sh.PrintTitleRows != nil || sh.PrintTitleCols != nil {
			var parts []string
			if sh.PrintTitleCols != nil {
				c := *sh.PrintTitleCols
				parts = append(parts, fmt.Sprintf("%s!$%s:$%s", q, ref.ColName(c.From.Col), ref.ColName(c.To.Col)))
			}
			if sh.PrintTitleRows != nil {
				r := *sh.PrintTitleRows
				parts = append(parts, fmt.Sprintf("%s!$%d:$%d", q, r.From.Row, r.To.Row))
			}
			out = append(out, model.DefinedName{Name: model.NamePrintTitles, RefersTo: strings.Join(parts, ","), Local: true, SheetIndex: i})
			skip[fmt.Sprintf("%s/%d", model.NamePrintTitles, i)] = true
		}
		if sh.AutoFilter != nil {
			out = append(out, model.DefinedName{Name: model.NameFilterDatabase, RefersTo: q + "!" + sh.AutoFilter.Range.Abs().String(), Local: true, SheetIndex: i, Hidden: true})
			skip[fmt.Sprintf("%s/%d", model.NameFilterDatabase, i)] = true
		}
	}
	for _, d := range p.wb.DefinedNames {
		if d.Local && skip[fmt.Sprintf("%s/%d", d.Name, d.SheetIndex)] {
			continue
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
		if out[i].Local != out[j].Local {
			return !out[i].Local
		}
		return out[i].SheetIndex < out[j].SheetIndex
	})
	return out
}

func boolAttr(v bool) string {
	if v {
		return "1"
	}
	return ""
}

func invBool(hidden bool) string {
	if hidden {
		return "0"
	}
	return ""
}

func intAttr(v int) string {
	if v == 0 {
		return ""
	}
	return ooxml.Int(v)
}

func externalLinkXML(el *model.ExternalLink) []byte {
	b := ooxml.NewBuilder()
	b.Start("externalLink", "xmlns", ooxml.NSMain, "xmlns:r", ooxml.NSRel)
	b.Start("externalBook", "r:id", "rId1")
	if len(el.SheetNames) > 0 {
		b.Start("sheetNames")
		for _, n := range el.SheetNames {
			b.Empty("sheetName", "val", n)
		}
		b.End()
	}
	b.End().End()
	return b.Bytes()
}

// metadataXML is the xl/metadata.xml part that marks dynamic array formulas.
func metadataXML() []byte {
	return []byte(ooxml.Header + `<metadata xmlns="` + ooxml.NSMain + `" xmlns:xda="` + ooxml.NSXDA + `">` +
		`<metadataTypes count="1"><metadataType name="XLDAPR" minSupportedVersion="120000" copy="1" pasteAll="1" pasteValues="1" merge="1" splitFirst="1" rowColShift="1" clearFormats="1" clearComments="1" assign="1" coerce="1" cellMeta="1"/></metadataTypes>` +
		`<futureMetadata name="XLDAPR" count="1"><bk><extLst><ext uri="{bdbb8cdc-fa1e-496e-a857-3c3f30c029c3}"><xda:dynamicArrayProperties fDynamic="1" fCollapsed="0"/></ext></extLst></bk></futureMetadata>` +
		`<cellMetadata count="1"><bk><rc t="1" v="0"/></bk></cellMetadata></metadata>`)
}
