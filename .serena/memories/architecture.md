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

`reader.Options{MaxPartSize (1 GiB default), MaxCells, SkipDrawings, Sheets, Strict}`; CFB/OLE
(encrypted) files return `ErrEncrypted`.
