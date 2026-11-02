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
