// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
)

// Content types of the parts gexcel writes.
const (
	CTWorkbook         = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"
	CTWorkbookMacro    = "application/vnd.ms-excel.sheet.macroEnabled.main+xml"
	CTTemplate         = "application/vnd.openxmlformats-officedocument.spreadsheetml.template.main+xml"
	CTTemplateMacro    = "application/vnd.ms-excel.template.macroEnabled.main+xml"
	CTWorksheet        = "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"
	CTChartsheet       = "application/vnd.openxmlformats-officedocument.spreadsheetml.chartsheet+xml"
	CTDialogsheet      = "application/vnd.openxmlformats-officedocument.spreadsheetml.dialogsheet+xml"
	CTMacrosheet       = "application/vnd.ms-excel.macrosheet+xml"
	CTStyles           = "application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"
	CTSharedStrings    = "application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"
	CTTheme            = "application/vnd.openxmlformats-officedocument.theme+xml"
	CTCoreProps        = "application/vnd.openxmlformats-package.core-properties+xml"
	CTExtProps         = "application/vnd.openxmlformats-officedocument.extended-properties+xml"
	CTCustomProps      = "application/vnd.openxmlformats-officedocument.custom-properties+xml"
	CTDrawing          = "application/vnd.openxmlformats-officedocument.drawing+xml"
	CTChart            = "application/vnd.openxmlformats-officedocument.drawingml.chart+xml"
	CTChartEx          = "application/vnd.ms-office.chartex+xml"
	CTChartStyle       = "application/vnd.ms-office.chartstyle+xml"
	CTChartColorStyle  = "application/vnd.ms-office.chartcolorstyle+xml"
	CTComments         = "application/vnd.openxmlformats-officedocument.spreadsheetml.comments+xml"
	CTThreadedComments = "application/vnd.ms-excel.threadedcomments+xml"
	CTPersons          = "application/vnd.ms-excel.person+xml"
	CTTable            = "application/vnd.openxmlformats-officedocument.spreadsheetml.table+xml"
	CTVML              = "application/vnd.openxmlformats-officedocument.vmlDrawing"
	CTRels             = "application/vnd.openxmlformats-package.relationships+xml"
	CTXML              = "application/xml"
	CTVBA              = "application/vnd.ms-office.vbaProject"
	CTCalcChain        = "application/vnd.openxmlformats-officedocument.spreadsheetml.calcChain+xml"
	CTExternalLink     = "application/vnd.openxmlformats-officedocument.spreadsheetml.externalLink+xml"
	CTPivotTable       = "application/vnd.openxmlformats-officedocument.spreadsheetml.pivotTable+xml"
	CTPivotCacheDef    = "application/vnd.openxmlformats-officedocument.spreadsheetml.pivotCacheDefinition+xml"
	CTPivotCacheRec    = "application/vnd.openxmlformats-officedocument.spreadsheetml.pivotCacheRecords+xml"
	CTMetadata         = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheetMetadata+xml"
	CTRichValue        = "application/vnd.ms-excel.richvalue+xml"
	CTConnections      = "application/vnd.openxmlformats-officedocument.spreadsheetml.connections+xml"
	CTCustomXML        = "application/vnd.openxmlformats-officedocument.customXmlProperties+xml"
	CTSlicerCache      = "application/vnd.ms-excel.slicerCache+xml"
	CTSlicer           = "application/vnd.ms-excel.slicer+xml"
)

// Relationship types.
const (
	RelOfficeDocument   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	RelCoreProps        = "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"
	RelExtProps         = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties"
	RelCustomProps      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/custom-properties"
	RelWorksheet        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"
	RelChartsheet       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet"
	RelDialogsheet      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/dialogsheet"
	RelMacrosheet       = "http://schemas.microsoft.com/office/2006/relationships/xlMacrosheet"
	RelStyles           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles"
	RelSharedStrings    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings"
	RelTheme            = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"
	RelDrawing          = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing"
	RelImage            = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	RelChart            = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	RelChartEx          = "http://schemas.microsoft.com/office/2014/relationships/chartEx"
	RelChartStyle       = "http://schemas.microsoft.com/office/2011/relationships/chartStyle"
	RelChartColorStyle  = "http://schemas.microsoft.com/office/2011/relationships/chartColorStyle"
	RelHyperlink        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"
	RelComments         = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/comments"
	RelThreadedComments = "http://schemas.microsoft.com/office/2017/10/relationships/threadedComment"
	RelPersons          = "http://schemas.microsoft.com/office/2017/10/relationships/person"
	RelVMLDrawing       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/vmlDrawing"
	RelTable            = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/table"
	RelVBA              = "http://schemas.microsoft.com/office/2006/relationships/vbaProject"
	RelCalcChain        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/calcChain"
	RelExternalLink     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/externalLink"
	RelExternalLinkPath = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/externalLinkPath"
	RelPivotTable       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/pivotTable"
	RelPivotCacheDef    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/pivotCacheDefinition"
	RelPivotCacheRec    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/pivotCacheRecords"
	RelMetadata         = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/sheetMetadata"
	RelConnections      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/connections"
	RelCustomXML        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/customXml"
	RelCustomXMLProps   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/customXmlProps"
	RelSlicerCache      = "http://schemas.microsoft.com/office/2007/relationships/slicerCache"
	RelSlicer           = "http://schemas.microsoft.com/office/2007/relationships/slicer"
	RelRichValue        = "http://schemas.microsoft.com/office/2017/06/relationships/richValue"
)

// ErrNotPackage is returned when the input is not an OPC zip package.
var ErrNotPackage = errors.New("ooxml: not an Office Open XML package")

// Writer assembles a package in a zip stream. Parts are added in order;
// content types and the root relationships are written on Close.
type Writer struct {
	zw        *zip.Writer
	defaults  map[string]string // extension -> content type
	overrides map[string]string // part name -> content type
	rels      map[string][]Relationship
	names     map[string]bool
	modified  time.Time
	err       error
}

// Relationship is an OPC relationship.
type Relationship struct {
	ID         string
	Type       string
	Target     string
	TargetMode string
}

// NewWriter starts a package on w.
func NewWriter(w io.Writer) *Writer {
	pw := &Writer{
		zw:        zip.NewWriter(w),
		defaults:  map[string]string{"rels": CTRels, "xml": CTXML},
		overrides: map[string]string{},
		rels:      map[string][]Relationship{},
		names:     map[string]bool{},
		modified:  time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	return pw
}

// SetModified sets the timestamp stored in the zip entries.
func (w *Writer) SetModified(t time.Time) { w.modified = t }

// Default registers a content type for an extension.
func (w *Writer) Default(ext, contentType string) { w.defaults[strings.ToLower(ext)] = contentType }

// Override registers a content type for a part name ("xl/workbook.xml").
func (w *Writer) Override(name, contentType string) { w.overrides[name] = contentType }

// Has reports whether a part was written.
func (w *Writer) Has(name string) bool { return w.names[name] }

// Create opens a part for streaming writes. The returned writer is valid
// until the next Create or Add. contentType "" uses the extension default.
func (w *Writer) Create(name, contentType string) (io.Writer, error) {
	if w.err != nil {
		return nil, w.err
	}
	if w.names[name] {
		return nil, fmt.Errorf("ooxml: duplicate part %q", name)
	}
	w.names[name] = true
	if contentType != "" {
		ext := strings.TrimPrefix(path.Ext(name), ".")
		if w.defaults[strings.ToLower(ext)] != contentType {
			w.overrides[name] = contentType
		}
	}
	method := zip.Deflate
	if ext := strings.ToLower(path.Ext(name)); ext == ".png" || ext == ".jpeg" || ext == ".jpg" || ext == ".gif" {
		method = zip.Store
	}
	f, err := w.zw.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: w.modified})
	if err != nil {
		w.err = err
		return nil, err
	}
	return f, nil
}

// Add writes a whole part.
func (w *Writer) Add(name, contentType string, data []byte) error {
	f, err := w.Create(name, contentType)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		w.err = err
	}
	return err
}

// Rel adds a relationship from a part (or "" for the package root) and
// returns its id.
func (w *Writer) Rel(from, relType, target, targetMode string) string {
	id := fmt.Sprintf("rId%d", len(w.rels[from])+1)
	w.rels[from] = append(w.rels[from], Relationship{ID: id, Type: relType, Target: target, TargetMode: targetMode})
	return id
}

// RelWithID adds a relationship with an explicit id.
func (w *Writer) RelWithID(from, id, relType, target, targetMode string) {
	w.rels[from] = append(w.rels[from], Relationship{ID: id, Type: relType, Target: target, TargetMode: targetMode})
}

// RelCount returns how many relationships a part has so far.
func (w *Writer) RelCount(from string) int { return len(w.rels[from]) }

// Close writes the relationship parts, the content types and the zip
// central directory.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	for from, rels := range w.rels {
		if err := w.Add(RelsName(from), "", RelsXML(rels)); err != nil {
			return err
		}
	}
	if err := w.Add("[Content_Types].xml", "", w.contentTypes()); err != nil {
		return err
	}
	return w.zw.Close()
}

// RelsName returns the relationships part name for a part ("" = root).
func RelsName(part string) string {
	if part == "" {
		return "_rels/.rels"
	}
	dir, base := path.Split(part)
	return dir + "_rels/" + base + ".rels"
}

// RelsXML serialises relationships.
func RelsXML(rels []Relationship) []byte {
	b := NewBuilder()
	b.Start("Relationships", "xmlns", NSPkgRel)
	for _, r := range rels {
		b.Empty("Relationship", "Id", r.ID, "Type", r.Type, "Target", r.Target, "TargetMode", r.TargetMode)
	}
	b.End()
	return b.Bytes()
}

func (w *Writer) contentTypes() []byte {
	b := NewBuilder()
	b.Start("Types", "xmlns", NSContentTypes)
	exts := make([]string, 0, len(w.defaults))
	for e := range w.defaults {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	for _, e := range exts {
		b.Empty("Default", "Extension", e, "ContentType", w.defaults[e])
	}
	names := make([]string, 0, len(w.overrides))
	for n := range w.overrides {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b.Empty("Override", "PartName", "/"+n, "ContentType", w.overrides[n])
	}
	b.End()
	return b.Bytes()
}

// Reader is an opened package.
type Reader struct {
	zr        *zip.Reader
	files     map[string]*zip.File
	defaults  map[string]string
	overrides map[string]string
	// MaxPartSize limits the decompressed size of a part read with Part.
	MaxPartSize int64
}

// Open reads the package directory. r must stay open while parts are read.
func Open(r io.ReaderAt, size int64) (*Reader, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotPackage, err)
	}
	pr := &Reader{zr: zr, files: map[string]*zip.File{}, defaults: map[string]string{}, overrides: map[string]string{}, MaxPartSize: 1 << 31}
	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "/")
		pr.files[name] = f
	}
	ct, err := pr.Part("[Content_Types].xml")
	if err != nil {
		return nil, fmt.Errorf("%w: missing [Content_Types].xml", ErrNotPackage)
	}
	if err := pr.parseContentTypes(ct); err != nil {
		return nil, err
	}
	return pr, nil
}

// OpenBytes opens a package held in memory.
func OpenBytes(data []byte) (*Reader, error) {
	return Open(bytes.NewReader(data), int64(len(data)))
}

func (r *Reader) parseContentTypes(data []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("ooxml: content types: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "Default":
			r.defaults[strings.ToLower(Attr(se, "Extension"))] = Attr(se, "ContentType")
		case "Override":
			r.overrides[strings.TrimPrefix(Attr(se, "PartName"), "/")] = Attr(se, "ContentType")
		}
	}
}

// Attr returns an attribute by local name.
func Attr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// ContentType returns the content type of a part.
func (r *Reader) ContentType(name string) string {
	if ct, ok := r.overrides[name]; ok {
		return ct
	}
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	return r.defaults[ext]
}

// Has reports whether the part exists.
func (r *Reader) Has(name string) bool {
	_, ok := r.files[name]
	return ok
}

// Names lists every part in the package.
func (r *Reader) Names() []string {
	out := make([]string, 0, len(r.files))
	for n := range r.files {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Part reads a whole part. It is limited by MaxPartSize.
func (r *Reader) Part(name string) ([]byte, error) {
	rc, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, r.MaxPartSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > r.MaxPartSize {
		return nil, fmt.Errorf("ooxml: part %q exceeds %d bytes", name, r.MaxPartSize)
	}
	return data, nil
}

// Open streams a part.
func (r *Reader) Open(name string) (io.ReadCloser, error) {
	f, ok := r.files[name]
	if !ok {
		// Part names are case-insensitive in OPC.
		for n, g := range r.files {
			if strings.EqualFold(n, name) {
				f = g
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("ooxml: part %q not found", name)
	}
	return f.Open()
}

// Size returns the uncompressed size a part declares.
func (r *Reader) Size(name string) int64 {
	if f, ok := r.files[name]; ok {
		return int64(f.UncompressedSize64)
	}
	return 0
}

// Rels reads the relationships of a part ("" for the package root).
func (r *Reader) Rels(part string) ([]Relationship, error) {
	data, err := r.Part(RelsName(part))
	if err != nil {
		return nil, nil
	}
	return ParseRels(data)
}

// ParseRels parses a relationships part.
func ParseRels(data []byte) ([]Relationship, error) {
	var rels []Relationship
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return rels, nil
		}
		if err != nil {
			return rels, fmt.Errorf("ooxml: relationships: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "Relationship" {
			rels = append(rels, Relationship{ID: Attr(se, "Id"), Type: Attr(se, "Type"), Target: Attr(se, "Target"), TargetMode: Attr(se, "TargetMode")})
		}
	}
}

// Resolve turns a relationship target relative to a source part into a
// package part name. Absolute targets ("/xl/...") are returned without the
// leading slash.
func Resolve(source, target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}
	dir := path.Dir(source)
	if source == "" || dir == "." {
		dir = ""
	}
	return path.Clean(path.Join(dir, target))
}

// Relative returns the target path for a relationship from source to part.
func Relative(source, part string) string {
	dir := path.Dir(source)
	if source == "" || dir == "." {
		return part
	}
	// Walk up from dir to the common ancestor.
	srcParts := strings.Split(dir, "/")
	dstParts := strings.Split(part, "/")
	i := 0
	for i < len(srcParts) && i < len(dstParts)-1 && srcParts[i] == dstParts[i] {
		i++
	}
	var b strings.Builder
	for j := i; j < len(srcParts); j++ {
		b.WriteString("../")
	}
	b.WriteString(strings.Join(dstParts[i:], "/"))
	return b.String()
}

// FindRel returns the first relationship of the type.
func FindRel(rels []Relationship, relType string) (Relationship, bool) {
	for _, r := range rels {
		if r.Type == relType {
			return r, true
		}
	}
	return Relationship{}, false
}

// RelByID returns the relationship with the id.
func RelByID(rels []Relationship, id string) (Relationship, bool) {
	for _, r := range rels {
		if r.ID == id {
			return r, true
		}
	}
	return Relationship{}, false
}

// RelIDByType returns the id of the n-th (0-based) relationship of the
// type from a part, or "".
func (w *Writer) RelIDByType(from, relType string, n int) string {
	for _, r := range w.rels[from] {
		if r.Type == relType {
			if n == 0 {
				return r.ID
			}
			n--
		}
	}
	return ""
}
