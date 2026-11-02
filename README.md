# gexcel

A pure-Go library for reading and writing Microsoft Excel workbooks (`.xlsx`, `.xlsm`, `.xltx`),
built the same way as [gpdf](https://github.com/gsoultan/gpdf): no external dependencies, an
interface-based API, a plain data model underneath, fuzzed parsers and output validated against
LibreOffice and openpyxl in CI.

```go
import (
    "github.com/gsoultan/gexcel"
    "github.com/gsoultan/gexcel/style"
)

wb := gexcel.New()
sh := wb.SheetAt(0)
sh.SetName("Report")
sh.SetRow("A1", "Item", "Qty", "Price", "Total")
sh.SetStyle("A1:D1", style.Style{Font: style.Font{Bold: true}, Fill: style.Solid(style.RGB("DDEBF7"))})
sh.SetRows("A2", [][]any{{"Widget", 4, 2.5}, {"Gadget", 2, 10}})
sh.Cell("D2").SetFormula("B2*C2").SetNumFmt(style.NumFmtCurrency)
sh.Cell("D3").SetFormula("B3*C3").SetNumFmt(style.NumFmtCurrency)
sh.Cell("D4").SetFormula("SUM(D2:D3)").SetNumFmt(style.NumFmtCurrency).SetBold(true)
sh.Cell("F1").Set(time.Now())            // stored as a date serial with a date format
sh.SetColWidth(1, 1, 16)
sh.Freeze(1, 0)
sh.AddTable("A1:D3", "Orders")
wb.Calculate()                           // optional: fill cached formula results
err := wb.SaveFile("report.xlsx")
```

Reading:

```go
wb, err := gexcel.Open("report.xlsx")
sh, _ := wb.Sheet("Report")
for row := range sh.Rows() {
    for cell := range row.Cells() {
        fmt.Println(cell.Ref(), cell.Any(), cell.Formatted())
    }
}
```

Very large files can be streamed row by row without loading a sheet into memory:

```go
st, _ := gexcel.StreamFile("big.xlsx")
defer st.Close()
for row, err := range st.Rows("Data") { ... }
```

and written with `writer.NewStreamWriter`, which emits rows straight into the zip entry.

## Packages

| Package   | Purpose |
|-----------|---------|
| `gexcel`  | Interface-based API: `Workbook`, `Sheet`, `Row`, `Cell`; `New`, `Open*`, `Stream*`. |
| `model`   | Plain data model of a workbook (sheets, cells, styles registry, tables, validations, conditional formats, comments, protection, print setup, raw-part preservation). |
| `style`   | Fonts, fills, borders, alignment, colors (RGB, theme + tint, indexed), number-format ids and the document theme. |
| `format`  | Excel number-format engine (`format.Number`, dates, elapsed time, fractions, sections, conditions) and date serial conversion. |
| `ref`     | A1 / R1C1 cell and range references, areas, sheet-name quoting. |
| `formula` | Tokenizer, parser, evaluator (~230 functions incl. dynamic arrays, XLOOKUP, LET, structured references, 3-D references), workbook calculator with spilling, `_xlfn.` prefix handling, reference shifting. |
| `chart`   | Chart model: column/bar/line/pie/doughnut/area/scatter/bubble/radar/stock/surface (+3-D variants), axes, data labels, trendlines, error bars, overlays and secondary axes. |
| `reader`  | OOXML parser: token-streamed rows, styles, theme, shared strings, drawings, charts, comments and threaded comments, tables, sparklines, data bars, VBA/pivot/customXml preservation, zip-bomb limits, row streaming. |
| `writer`  | OOXML writer with a streaming mode; sheet protection hashing; `Validate` for pre-save checks. |
| `ooxml`   | OPC package layer (zip, content types, relationships) and a small XML builder/DOM. |

## Feature coverage

See [`.serena/memories/feature_coverage.md`](.serena/memories/feature_coverage.md) for the
checklist of Excel features, what is implemented, what is preserved on round trip, and what is
planned (pivot table editing, chartEx charts, slicers, encryption, legacy `.xls`).

## Development

```
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...
GEXCEL_EXTERNAL=1 go test -run External ./tests/   # needs LibreOffice and openpyxl
```

Every Go file starts with the copyright header enforced by `scripts/check-headers.sh`.

## License

MIT, see [LICENSE](LICENSE).
