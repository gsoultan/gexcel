# gexcel — architecture

## Data flow

```
file/bytes ──ooxml.Reader──▶ reader (DOM for small parts, xml.Decoder token stream for sheetData)
          ──▶ model.Workbook ◀── gexcel.Workbook/Sheet/Row/Cell (thin wrappers, no copies)
model.Workbook ──writer.packer──▶ ooxml.Writer (zip) ──▶ file/bytes
```

- The public API wraps the model: `gexcel.Wrap(wb)`; every `Model()` returns the live struct.
- Styles are deduplicated by `model.Styles.Add(style.Style) int` (key-based registry of fonts, fills,
  borders, numFmts, cellXfs, dxfs). `Cell.SetStyle` stores only the xf id.
- Strings go to the shared string table unless `Value.Inline` or `writer.Options.InlineStrings`.
- Dates are serial numbers; `Cell.Set(time.Time)` applies `yyyy-mm-dd hh:mm:ss` if the cell has no
  number format; `Cell.Time()/Any()` convert back using `style.IsDateFormat`.
- Formulas: stored without `=`; writer adds `_xlfn.`/`_xlfn._xlws.` prefixes
  (`formula.AddPrefixes`), reader strips them. `Cell.SetFormula` marks formulas whose outer call is
  a dynamic-array function (`formula.IsDynamicArray`) as dynamic arrays (`cm="1"` + metadata.xml).
  Shared formulas are expanded per cell on read with `formula.Shift`.
- Conditional formatting data bars use the x14 extension; sparklines are x14; both read and written.
- Sheet protection: SHA-512 with 100000 spins (`writer/protect.go`).
- Raw-part preservation: chart sheets, dialog/macro sheets, VBA (`vbaProject.bin`), pivot caches and
  tables, external links, customXml, rich data and unknown `extLst` entries are kept as
  `model.RawPart`/`RawXML` and written back.
- Pictures in cells: `model.Cell.Image *model.CellImage`. The writer emits the cell as
  `t="e" vm="N"` with `#VALUE!` and collects the pictures in `packer.cellImages`
  (`writer/richdata.go`); `finish` then writes the media parts (deferred: a sheet zip entry is
  still open while cells are written), `xl/richData/{richValueRel,rdrichvalue,rdrichvaluestructure,
  rdRichValueTypes}.xml` and regenerates `xl/metadata.xml` (`metadataXML(dynamic, images)`:
  XLDAPR type first so `cm="1"` stays valid, then XLRICHVALUE). `reader/richdata.go`
  (`readCellImages`) resolves vm -> valueMetadata -> futureMetadata rvb -> rich value -> rel ->
  media before sheets are parsed and takes over only when every rich value structure is
  `_localImage` and the metadata types are XLDAPR/XLRICHVALUE; otherwise all rich data stays raw
  in `Workbook.RichData` (each `RawPart.RelType` keeps the workbook relationship type) and adding
  `Cell.Image` to such a workbook fails with `writer.ErrUnsupported`. The model keeps the cell
  value empty for picture cells; `#VALUE!` exists only in the file.
- Charts: `chart.Chart` is written as a full `c:chartSpace`; on read the chart is parsed into the
  model and the raw bytes kept in `Chart.Raw`. chartEx types (treemap, sunburst, waterfall, funnel,
  histogram, box & whisker) are modelled but the writer returns `ErrUnsupported` for them.

## Streaming

- Write: `writer.NewStreamWriter(w, wb, opts)` → `Sheet(*model.Sheet)` → `Row(index, cells...)` →
  `Close()`; rows are written straight into the zip entry through a bufio writer.
- Read: `reader.OpenStream*` loads the workbook skeleton (styles, SST, names) and parses sheets on
  demand with `sheetParser.onRow`; `Stream.Rows(name)` is an `iter.Seq2[*model.Row, error]`.

## Formula engine

- `Tokenize` → `Parse` (precedence climbing; `:` folds `Sheet1:'Sheet 2'!A1` into a 3-D ref) →
  `Evaluator.eval` with `Value{Kind: Number|Text|Bool|Error|Array|Reference}`.
- Functions are `Func(e *Evaluator, args []Node) Value`; lazy evaluation lets IF/IFERROR skip
  branches. Arithmetic and comparisons broadcast over arrays (`zip`).
- `Calculator` implements `Context` over a `model.Workbook`: resolves cells, defined names (sheet
  scope first), structured references (`Table[[#Headers],[Col]]`, `[@Col]`), 3-D sheet ranges;
  computes cells on demand with cycle detection (`ErrCycle`) and spills arrays (dynamic → sets
  `Formula.Ref`; legacy CSE arrays fill their `Ref`).
- Any panic inside a function becomes `#VALUE!` (`Evaluator.call` recovers).

## Reader limits

`reader.Options{MaxPartSize (1 GiB default), MaxCells, SkipDrawings, Sheets, Strict, Password}`; a
CFB/OLE (encrypted) file returns `ErrEncrypted` without a password, `ErrWrongPassword` with a wrong
one, and is otherwise decrypted in memory (`ooxml.Decrypt`, so it must fit MaxPartSize).

## Encryption (ooxml/encrypt.go, ooxml/cfb.go)

`writer.Options{Password}` buffers the plain package and wraps it with `ooxml.Encrypt`: ECMA-376
agile encryption as Excel writes it — AES-256-CBC over 4096-byte segments (segment IV =
SHA-512(salt, segment index)), password key via SHA-512 with 100,000 spins (iterator prefixed) and
the MS-OFFCRYPTO block keys for verifierHashInput/Value, KeyValue, HmacKey/HmacValue; HMAC-SHA512
dataIntegrity; `EncryptionInfo` v4.4 XML. `ooxml/cfb.go` is a minimal compound-file writer/reader
(512-byte sectors, mini stream for streams < 4096 bytes) holding `EncryptionInfo` and
`EncryptedPackage` (8-byte plain length prefix). Verified against LibreOffice (opens with the
password) and round-trips through `ooxml.Decrypt`. Style keys (`style.Style.Key`) are built with
hand-written writers instead of fmt for the writer's deduplication hot path.
