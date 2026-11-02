# gexcel — code style and conventions

- Mirror gpdf: interfaces in the root package (`gexcel.go`), implementations in `*_impl.go`,
  `New*` constructors return interfaces, plain structs in `model/`.
- One concern per file (`writer/styles.go`, `writer/comments.go`, `reader/drawing.go`, ...).
- Errors: package-level sentinels (`ErrInvalidRef`, `ErrSheetExists`, `reader.ErrEncrypted`,
  `formula.ErrCycle`, `formula.ErrSyntax`) wrapped with `fmt.Errorf("...: %w", err)`.
- Setters on `Cell`/`Row` return the receiver for chaining; invalid references yield a detached cell
  whose `Err()` is set so chains never panic.
- `Cell.Set(any)` accepts Go scalars, `time.Time`, `time.Duration`, `[]model.RichRun`,
  `model.Value`, `error`, `fmt.Stringer`; anything else is `fmt.Sprint`ed.
- Excel semantics win over Go convenience: 1-based rows/columns, A1 strings in the API, formulas
  without `=`, case-insensitive sheet lookup.
- XML writing goes through `ooxml.Builder` (attributes with empty values are skipped unless the name
  ends with `!`); XML reading uses `ooxml.ParseDOM` for small parts and `encoding/xml` tokens for
  `sheetData`.
- Worksheet functions live in `formula/funcs*.go` tables keyed by upper-case name; add new ones with
  `register(map[string]Func{...})` and a case in `formula_test.go`'s table; add the name to
  `formula/prefix.go` if Excel stores it with `_xlfn.`.
- Tests: table-driven, white-box next to code, black-box in `tests/`; `writer/wtest.SampleWorkbook`
  is the shared feature-complete fixture. `GEXCEL_OUT=dir` makes some tests write the files for
  manual inspection.
- Copyright header on every Go file; commits by gsoultan only, no AI credits.
