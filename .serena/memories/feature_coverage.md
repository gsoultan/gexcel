# gexcel — Excel feature coverage

Legend: ✅ read + write + API · 🔁 preserved on round trip only · 📝 model/write only · ⏳ planned

## Workbook
- ✅ Sheets (add/remove/move/copy/rename/hide, tab color, active tab), worksheet views, freeze/split
- ✅ Core/app/custom document properties
- ✅ Defined names (workbook and sheet scoped), print area, print titles
- ✅ Date systems (1900 with leap bug, 1904)
- ✅ Workbook protection attributes, calc properties, full-calc-on-load
- 🔁 Chart sheets, dialog sheets, macro sheets (raw parts; chart of chart sheet is parsed)
- 🔁 VBA project (`.xlsm`), external links, customXml, rich data, pivot caches
- ⏳ Encryption / password-to-open (CFB container), legacy `.xls` reading, `.xlsb`

## Cells and data
- ✅ Numbers, text (SST or inline), rich text runs, booleans, errors, dates/times, durations
- ✅ Number formats: builtin ids + custom codes; full rendering engine (`Cell.Formatted`)
- ✅ Formulas: normal, shared (expanded on read), legacy array (CSE), dynamic array with spill
  metadata, data-table formulas, `_xlfn.` prefix handling, R1C1 parsing
- ✅ Formula calculation (`Workbook.Calculate`): ~230 functions — math, statistics, logical, text,
  lookup (VLOOKUP/XLOOKUP/INDEX/MATCH/OFFSET/INDIRECT), dynamic arrays (FILTER/SORT/SORTBY/UNIQUE/
  SEQUENCE/TEXTSPLIT/VSTACK/HSTACK/TAKE/DROP/CHOOSEROWS/CHOOSECOLS/TOCOL/TOROW), date/time,
  financial (PMT/FV/PV/NPV/IPMT/PPMT/NPER/RATE/IRR), information, LET, structured and 3-D references
- ⏳ LAMBDA/MAP/REDUCE/SCAN/BYROW/BYCOL, statistical distributions, engineering, database functions,
  cube/web functions, volatile recalculation chains, calcChain.xml output
- ✅ Hyperlinks (external and internal), comments/notes (legacy VML), threaded comments (persons)
- ✅ Merged cells, row heights, column widths, hidden rows/cols, outline levels, cell metadata ids

## Formatting
- ✅ Fonts, fills (solid, pattern, gradient), borders (incl. diagonal), alignment, protection, named
  cell styles, differential styles (dxf), theme colors with tint, indexed colors
- ✅ Conditional formatting: cellIs, expression, text rules, top/bottom, above average, duplicates,
  color scales, data bars (x14 extensions), icon sets, time periods
- ✅ Tables (structured tables) with styles, totals rows, calculated columns, auto filters
- ✅ Auto filter with filter columns (values, custom, top10, dynamic, color, icon), sort state
- ✅ Data validation (all types, operators, messages)
- ✅ Sparklines (line/column/stacked, x14)
- 🔁 Custom table styles, pivot styles (raw `tableStyles` XML)
- ⏳ Slicers, timelines

## Drawings
- ✅ Images (png, jpeg, gif, bmp, tiff, svg, emf, wmf, webp) with one-cell/two-cell/absolute anchors,
  hyperlinks
- ✅ Charts: column/bar (clustered, stacked, percent, 3-D), line (+markers, 3-D), pie/doughnut/
  pie-of-pie/bar-of-pie, area, scatter, bubble, radar, stock, surface; titles, legends, axes
  (scaling, units, number formats, gridlines, log, reverse, date axes), data labels, trendlines,
  error bars, data points, overlays/combo charts with secondary axes, data tables, 3-D view
- 📝 Shapes (preset geometry, fill, line, text) — written; read keeps raw anchor XML
- ⏳ chartEx charts (treemap, sunburst, waterfall, funnel, histogram, box & whisker), SmartArt,
  form controls, OLE objects (preserved raw only)

## Page setup
- ✅ Page setup (orientation, paper, scale, fit-to, DPI, order), margins, print options, headers and
  footers (odd/even/first), row/column page breaks, ignored errors

## Protection
- ✅ Sheet protection with SHA-512 hashing and per-action permissions, workbook structure lock
- ✅ Cell lock/hidden flags
- ⏳ Protected ranges with passwords, file encryption

## Pivot tables
- 🔁 Pivot tables and caches survive a round trip untouched
- ⏳ Pivot table model, refresh and creation

## I/O
- ✅ `Read`/`ReadFile`/`ReadBytes`, streaming row reader, zip-bomb limits, `MaxCells`, sheet filter
- ✅ `Write`/`WriteFile`, streaming writer, `Validate`, template options
- ✅ Validation in CI through LibreOffice (CSV/PDF conversion, re-read) and openpyxl
