# gexcel — project overview

Pure-Go library (module `github.com/gsoultan/gexcel`, Go 1.26, zero dependencies) for reading and
writing Microsoft Excel workbooks in the Office Open XML format (`.xlsx`, `.xlsm`, `.xltx`, `.xltm`).
It is the Excel counterpart of `gpdf` and follows the same conventions: interface-based public API at
the package root, plain data model underneath, small files per concern, sentinel errors wrapped with
`%w`, white-box tests next to code plus black-box tests under `tests/`, fuzz targets, and external
validation (LibreOffice, openpyxl) in CI.

Goal stated by the owner: full support for current Microsoft 365 Excel, phased one PR per area, with
`feature_coverage` tracking status. Once gexcel has a release, `proteus-converter` switches from its
own `formats/xlsx` writer to gexcel.

## Layout

- `gexcel.go`, `workbook_impl.go`, `sheet_impl.go`, `cell_impl.go` — public interfaces `Workbook`,
  `Sheet`, `Row`, `Cell`, constructors `New`, `Open`, `OpenBytes`, `OpenReader`, `OpenStream`,
  `StreamFile`, `StreamReader`, `Wrap`.
- `model/` — data model (`Workbook`, `Sheet`, `Row`, `Cell`, `Value`, `Styles` registry, tables,
  validations, conditional formats, comments, images, defined names, protection, print setup).
- `style/` — `Style`, `Font`, `Fill`, `Border`, `Alignment`, `Color` (RGB/theme+tint/indexed), builtin
  number formats, `Theme`.
- `format/` — number-format engine and date serials.
- `ref/` — A1/R1C1 references, ranges, areas.
- `formula/` — tokenizer, parser (AST), evaluator, ~230 functions, `Calculator` (workbook calc with
  spilling and cycle detection), `Shift`, `AddPrefixes`/`StripPrefixes` (`_xlfn.`), `IsDynamicArray`.
- `chart/` — chart model.
- `ooxml/` — OPC zip package reader/writer, content types, relationships, XML builder and DOM.
- `reader/` — package parser (`Read*`), streaming row reader (`OpenStream*`).
- `writer/` — package writer (`Write*`, `StreamWriter`), `Validate`, protection hashing;
  `writer/wtest` builds the feature-complete sample workbook shared by tests.
- `cmd/gexcel-dump` — CLI that prints what the reader sees in a file.
- `tests/` — black-box tests; `External*` tests run LibreOffice/openpyxl when `GEXCEL_EXTERNAL=1`.
- `scripts/check-headers.sh` — copyright header check used by CI.

## Validation status

- Writer output: opened by openpyxl and LibreOffice; LibreOffice evaluates the formulas and renders
  the charts (checked via CSV and PDF export).
- Reader: round-trips the writer sample; reads openpyxl- and LibreOffice-written files.
- Formula engine: 110 formulas cross-checked against LibreOffice (differences only where LibreOffice
  lacks a function or formats a result).
