# gexcel — review findings and known gaps

Open items found while building and validating; pick these up in follow-up PRs.

- chartEx charts (treemap, sunburst, waterfall, funnel, histogram, box & whisker) are modelled but
  `writer.chartXML` returns `ErrUnsupported`; reading keeps the raw part.
- Pivot tables are preserved raw; no model, refresh or creation yet.
- Threaded comments on read are matched to legacy comments by the `tc=` author marker; comments
  created by other tools without the legacy mirror are read from `threadedComments` directly.
- Formula engine: LAMBDA and the higher-order functions (MAP, REDUCE, SCAN, BYROW, BYCOL,
  MAKEARRAY) are not implemented; implicit intersection (`@`) on ranges outside the formula's row or
  column returns `#VALUE!` as Excel does; `#` spill references use the anchor's value.
- `MATCH`/`VLOOKUP` approximate mode fall back to a linear scan on unsorted data instead of the
  binary-search-only result Excel gives.
- `reader.Options.Strict` is off by default: malformed XML is parsed best-effort.
- The writer does not emit `calcChain.xml`; Excel rebuilds it (we drop it on read for the same
  reason).
- `Sheet.CopySheet` renames copied tables with a `_copy` suffix; the writer does not yet verify
  table-name uniqueness across sheets (`Validate` should).
- Chart reading maps only the common cases of overlay/secondary-axis combos; exotic axis groups
  survive through `Chart.Raw` when the chart is not edited, but any edit regenerates from the model.
- LibreOffice before 24.8 lacks XLOOKUP/SORT/FILTER; external tests avoid those functions.
- Fixed 2026-11-14: rich data parts linked with the real relationship types (`rdRichValue`,
  `rdRichValueStructure`, `rdRichValueTypes`, `richValueRel`) used to be written back without a
  workbook relationship, so data types and in-cell pictures were silently orphaned on a round
  trip; `RawPart.RelType` now carries the type. The reader also dropped the dynamic-array flag of
  `cm="1"` cells when the `<f>` element followed (files from Excel/XlsxWriter), fixed too.
- Pictures in cells are verified against XlsxWriter output and by round trip, not in real Excel.
  The formula engine sees a picture cell as empty, where Excel sees a rich value.
- The function count is 205 (`len(formula.Functions())`), not the ~230 quoted earlier.
- FuzzFormat (CI smoke) found two faults in `format`: `0.00/0` panicked (a slash after a decimal
  point or exponent was parsed as a fraction and took numerator digits that were not integer
  placeholders; such a slash is now a literal), and a long denominator such as `0/0000000000`
  made `bestFraction` scan up to 10^n candidates (now a continued-fraction search, denominator
  digits capped at 15). Both inputs are kept in `format/testdata/fuzz/FuzzFormat/`.
- FuzzParse (10 minute run) killed its worker on `XLOOKUP(,AF:X,A2:A7)`: `toArray` built every
  cell of a whole-column reference (9.4M values). References are now read only up to the sheet's
  last used row and column (`formula.Extenter`, implemented by the Calculator; COUNTBLANK adds the
  cells left out so its answer is unchanged). Probing the neighbours found more unbounded work,
  fixed with it: NETWORKDAYS/WORKDAY walked day by day over any span (now #NUM! outside Excel's
  dates 0..2958465), FIXED/DOLLAR built a format with any number of decimals (now #VALUE! above
  127, as Excel), and `Parse` accepted formulas of any length, so tree depth and evaluator
  recursion were unbounded (now `formula.MaxLength` = 8192 characters, Excel's limit).
  Whole-row references such as `1:1` still do not parse (#REF!).
- The nightly FuzzParse run failed once more, on arrays that are computed rather than read:
  `ROW(A:A)*COLUMN(A:XFD)` asked for 17 billion cells, and chains such as `ROW(A:A)+1+1+...`
  repeated a million-cell array at every step. Each formula now has a budget of
  `maxArrayCells` = 5,000,000 array cells (`Evaluator.spend`, reset per formula in `EvalNode`),
  charged by `toArray`, `zip`, `mapScalar`, `rangeOf` (computed arrays passed to functions),
  ROW/COLUMN, SEQUENCE and array IF; past it the formula is #NUM!. The cap on a single
  reference was 10,000,000 cells before, so references of 5-10M used cells now give #NUM!.
  New array-building code must call `e.spend(rows, cols)` before it allocates.
