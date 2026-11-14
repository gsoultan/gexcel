# Graph Report - gexcel  (2026-11-14)

## Corpus Check
- 80 files · ~90,380 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 2)

## Summary
- 1490 nodes · 4994 edges · 70 communities (50 shown, 20 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 699 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6ee71aa9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- ref.go
- Evaluator
- Sheet
- cell
- funcs_lookup.go
- chartWriter
- Format
- .Arg
- Node
- Builder
- funcs_date.go
- Sheet
- Node
- Cell
- Value
- NewBuilder
- steps
- steps
- Value
- Workbook
- Fill
- Row
- testing.T
- ParseCell
- packer
- Range
- Calculator
- .Workbook
- codeql.yml
- Token
- New
- Func
- New
- Style
- Writer
- gexcel — Excel feature coverage
- gexcel.go
- .edit
- Cell
- ToValue
- Border
- Workbook
- CondFmtRule
- siXML
- CellImage
- MustRange
- protect.go
- io.Writer
- sheetWriter
- each
- Alignment
- writer/richdata.go
- project.yml
- xlsxwriter_cell_images_4d5b9f35.md
- Row
- Anchor
- Table
- Comment
- Image
- DefinedName
- HeaderFooter
- PageMargins
- PageSetup
- Hyperlink
- CLAUDE.md
- check-headers.sh
- suggested_commands.md
- task_completion_checklist.md
- github.com/gsoultan/gexcel

## God Nodes (most connected - your core abstractions)
1. `Evaluator` - 144 edges
2. `Sheet` - 126 edges
3. `Sheet` - 115 edges
4. `Num()` - 98 edges
5. `cell` - 88 edges
6. `Workbook` - 86 edges
7. `Style` - 81 edges
8. `Value` - 75 edges
9. `Styles` - 73 edges
10. `Cell` - 58 edges

## Surprising Connections (you probably didn't know these)
- `Drawings` --references--> `CellImage`  [INFERRED]
  .serena/memories/feature_coverage.md → model/features.go
- `Data flow` --references--> `Chart`  [INFERRED]
  .serena/memories/architecture.md → chart/chart.go
- `Formula engine` --references--> `Calculator`  [INFERRED]
  .serena/memories/architecture.md → formula/calc.go
- `Layout` --references--> `Calculator`  [INFERRED]
  .serena/memories/project_overview.md → formula/calc.go
- `Layout` --references--> `Wrap()`  [INFERRED]
  .serena/memories/project_overview.md → gexcel.go

## Import Cycles
- 2-file cycle: `reader/reader_test.go -> reader/richdata_test.go -> reader/reader_test.go`

## Communities (70 total, 20 thin omitted)

### Community 0 - "ref.go"
Cohesion: 0.12
Nodes (8): BoolNode, EmptyNode, ErrorNode, NameNode, NumberNode, StringNode, StructuredNode, Theme

### Community 1 - "Evaluator"
Cohesion: 0.11
Nodes (72): Evaluator, aggAverage(), aggCount(), aggMax(), aggMin(), aggSum(), fnAtan2(), fnAverage() (+64 more)

### Community 4 - "funcs_lookup.go"
Cohesion: 0.20
Nodes (36): column(), flatten(), fnAddress(), fnChooseRC(), fnDrop(), fnFilter(), fnHLookup(), fnIndex() (+28 more)

### Community 5 - "chartWriter"
Cohesion: 0.06
Nodes (43): AnchorRef, Axis, Chart, DataLabels, LineStyle, Series, Title, Type (+35 more)

### Community 6 - "Format"
Cohesion: 0.08
Nodes (31): DurationToSerial(), SerialToDuration(), ToSerial(), Format, bestFraction(), digitFirst(), digitGroup(), fixed() (+23 more)

### Community 7 - ".Arg"
Cohesion: 0.17
Nodes (28): TestEvaluate(), roundHalfAway(), fnChar(), fnCode(), fnConcat(), fnDollar(), fnExact(), fnFindSearch() (+20 more)

### Community 8 - "Node"
Cohesion: 0.05
Nodes (59): DataValidationType, DataValidation, DataValidationOperator, SheetState, TestPackageRoundTrip(), Attr(), FindRel(), Reader (+51 more)

### Community 9 - "Builder"
Cohesion: 0.13
Nodes (26): DefaultPageMargins(), PageBreak, SheetProtection, Builder, FormatRanges(), gexcel — code style and conventions, autoFilterXML(), breaksXML() (+18 more)

### Community 10 - "funcs_date.go"
Cohesion: 0.19
Nodes (25): FromSerial(), addMonths(), datePart(), days360(), fnDate(), fnDateDif(), fnDateValue(), fnDays() (+17 more)

### Community 11 - "Sheet"
Cohesion: 0.12
Nodes (17): ColorFilter, CustomFilter, DateGroupItem, DynamicFilter, IconFilter, Pane, PrintOptions, Relationship (+9 more)

### Community 12 - "Node"
Cohesion: 0.09
Nodes (22): ArrayNode, binPrec(), Parse(), parseRefNode(), References(), String(), walk(), writeNode() (+14 more)

### Community 13 - "Cell"
Cohesion: 0.21
Nodes (3): Cell, cellAt(), rangeAttr()

### Community 14 - "Value"
Cohesion: 0.13
Nodes (5): numberOrError(), Kind, Value, Value, ValueKind

### Community 15 - "NewBuilder"
Cohesion: 0.17
Nodes (14): PivotTable, RawPart, CustomProperty, ExternalLink, NewBuilder(), externalLinkOf(), appPropsXML(), boolWord() (+6 more)

### Community 16 - "steps"
Cohesion: 0.08
Nodes (25): runs-on, steps, strategy, jobs, fuzz, target, name, on (+17 more)

### Community 17 - "steps"
Cohesion: 0.09
Nodes (25): name, runs-on, steps, GEXCEL_EXTERNAL, name, needs, runs-on, steps (+17 more)

### Community 18 - "Value"
Cohesion: 0.17
Nodes (7): Context, binaryScalar(), each(), fnCount(), fnIs(), Ref(), Formula engine

### Community 19 - "Workbook"
Cohesion: 0.08
Nodes (12): main(), Workbook, New(), NewEmpty(), Open(), OpenBytes(), OpenStream(), TestCellImage() (+4 more)

### Community 20 - "Fill"
Cohesion: 0.13
Nodes (7): builtinStyleID(), fillKey(), NewStyles(), BuiltinNumFmtID(), PatternType, Fill, TestNumFmt()

### Community 22 - "testing.T"
Cohesion: 0.15
Nodes (24): TestParseRoundTrip(), ReadBytes(), FuzzRead(), roundTrip(), TestReadInlineStrings(), TestReadLimits(), TestRoundTripSample(), TestStream() (+16 more)

### Community 24 - "packer"
Cohesion: 0.16
Nodes (9): SheetKind, Relative(), guidOf(), threadedID(), packer, richValueTypesXML(), orDefault(), invBool() (+1 more)

### Community 25 - "Range"
Cohesion: 0.09
Nodes (18): Formula, FormulaKind, IgnoredError, AutoFilter, SortState, SortCondition, CellRange(), ColName() (+10 more)

### Community 26 - "Calculator"
Cohesion: 0.15
Nodes (9): Calculate(), NewCalculator(), splitSpec(), Calculator, cellKey, NewEvaluator(), FuzzParse(), Err() (+1 more)

### Community 27 - ".Workbook"
Cohesion: 0.10
Nodes (4): Properties, ValidSheetName(), quoteSheet(), Validate()

### Community 28 - "codeql.yml"
Cohesion: 0.11
Nodes (17): runs-on, steps, jobs, analyze, name, on, pull_request, push (+9 more)

### Community 29 - "Token"
Cohesion: 0.11
Nodes (26): TestPrefixes(), TestShift(), AddPrefixes(), IsDynamicArray(), StripPrefixes(), Shift(), shiftRef(), Token (+18 more)

### Community 30 - "New"
Cohesion: 0.31
Nodes (8): OpenReader(), StreamReader(), New(), Read(), TestCellImageUnsupportedFormat(), OpenStream(), TestStreamLargeSheet(), TestValidate()

### Community 31 - "Func"
Cohesion: 0.20
Nodes (17): aggregator, Func, fnAndOr(), fnEvenOdd(), fnIfs(), fnLargeSmall(), fnMinMax(), fnParity() (+9 more)

### Community 32 - "New"
Cohesion: 0.25
Nodes (8): New(), build(), TestAPIRoundTrip(), RGB(), Solid(), TestStyleMerge(), itoa(), TestQuickStart()

### Community 33 - "Style"
Cohesion: 0.21
Nodes (8): CellStyle, Styles, Gradient, GradientStop, GradientType, BuiltinNumFmt(), Style, vmlXML()

### Community 35 - "gexcel — Excel feature coverage"
Cohesion: 0.20
Nodes (8): Cells and data, Drawings, Formatting, gexcel — Excel feature coverage, Page setup, Pivot tables, Protection, Workbook

### Community 36 - "gexcel.go"
Cohesion: 0.21
Nodes (9): Permission, StreamFile(), TableNoFilter(), TableOption, TableStripes(), TableStyle(), TableTotals(), Stream (+1 more)

### Community 38 - "Cell"
Cohesion: 0.14
Nodes (8): SparklineGroup, Sparkline, Cell, MustCell(), NewCell(), ParseR1C1(), r1c1Part(), TestCells()

### Community 39 - "ToValue"
Cohesion: 0.17
Nodes (4): ToValue(), IsErrorCode(), Rich(), ThemeColor()

### Community 40 - "Border"
Cohesion: 0.36
Nodes (6): parseLine(), LineStyle, All(), Border, Line, Thin()

### Community 41 - "Workbook"
Cohesion: 0.24
Nodes (6): CalcProperties, RawXML, Workbook, WorkbookProtection, WorkbookView, sampleWorkbook()

### Community 42 - "CondFmtRule"
Cohesion: 0.31
Nodes (12): CfValueType, ColorScale, CondFmtOperator, CondFmtType, CustomIcon, CfValue, CondFmtRule, ConditionalFormat (+4 more)

### Community 43 - "siXML"
Cohesion: 0.12
Nodes (13): TestBuilder(), Clean(), EscapeAttr(), EscapeText(), EscapeUnicode(), isHex4(), leftPad(), UnescapeUnicode() (+5 more)

### Community 45 - "MustRange"
Cohesion: 0.29
Nodes (8): mustRange(), TestCalculateWorkbook(), testWorkbook(), NewSheet(), OpenBytes(), MustRange(), TestStreamWriter(), TestWriteSample()

### Community 46 - "protect.go"
Cohesion: 0.25
Nodes (5): hashPassword(), HashPasswordWithSalt(), LegacyPasswordHash(), VerifyPassword(), TestPasswordHash()

### Community 47 - "io.Writer"
Cohesion: 0.15
Nodes (11): NewWriter(), Development, Feature coverage, gexcel, License, Packages, Options, NewStreamWriter() (+3 more)

### Community 48 - "sheetWriter"
Cohesion: 0.33
Nodes (4): hasAttr(), orWidth(), sheetWriter, intAttr()

### Community 49 - "each"
Cohesion: 0.43
Nodes (6): criterion, fnAverageIf(), fnCountIf(), fnSumIf(), parseCriterion(), wildcardMatch()

### Community 50 - "Alignment"
Cohesion: 0.14
Nodes (7): Xf, HAlign, Alignment, Protection, mergeBorder(), mergeFont(), VAlign

### Community 51 - "writer/richdata.go"
Cohesion: 0.60
Nodes (3): cellImageMedia, cellImages, cellImageValue

### Community 52 - "project.yml"
Cohesion: 0.22
Nodes (8): encoding, excluded_tools, ignore_all_files_in_gitignore, ignored_paths, initial_prompt, languages, project_name, read_only

### Community 57 - "Anchor"
Cohesion: 0.31
Nodes (8): AnchorKind, Anchor, CellOffset, Shape, anchorStart(), chartAnchor(), markerXML(), xfrmXML()

### Community 58 - "Table"
Cohesion: 0.40
Nodes (4): DefaultTableStyle(), Table, TableColumn, TableStyle

### Community 62 - "Comment"
Cohesion: 0.40
Nodes (3): CommentReply, RichRun, Comment

### Community 64 - "Image"
Cohesion: 0.40
Nodes (3): imageSize(), sniffImage(), Image

## Knowledge Gaps
- **1 isolated node(s):** `github.com/gsoultan/gexcel`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 290 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **20 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Sheet` connect `Sheet` to `ref.go`, `Sheet`, `chartWriter`, `Node`, `Builder`, `Cell`, `Value`, `NewBuilder`, `packer`, `Range`, `Calculator`, `Style`, `gexcel.go`, `Cell`, `Workbook`, `CondFmtRule`, `MustRange`, `io.Writer`, `sheetWriter`, `writer/richdata.go`, `Anchor`, `Table`, `Comment`, `Image`, `HeaderFooter`, `PageMargins`, `PageSetup`, `Hyperlink`?**
  _High betweenness centrality (0.166) - this node is a cross-community bridge._
- **Are the 95 inferred relationships involving `Num()` (e.g. with `binaryScalar()` and `.argOrZero()`) actually correct?**
  _`Num()` has 95 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/gsoultan/gexcel` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `ref.go` be split into smaller, more focused modules?**
  _Cohesion score 0.125 - nodes in this community are weakly interconnected._
- **Why does `Sheet` connect `Sheet` to `Image`, `ref.go`, `cell`, `gexcel.go`, `HeaderFooter`, `Cell`, `PageMargins`, `Node`, `PageSetup`, `Sheet`, `Row`, `Row`, `ParseCell`, `Range`, `Table`, `.Workbook`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Should `Evaluator` be split into smaller, more focused modules?**
  _Cohesion score 0.11018518518518519 - nodes in this community are weakly interconnected._
- **Why does `cell` connect `cell` to `ref.go`, `gexcel.go`, `.edit`, `chartWriter`, `ToValue`, `Border`, `Hyperlink`, `CellImage`, `Cell`, `Alignment`, `Fill`, `Comment`?**
  _High betweenness centrality (0.071) - this node is a cross-community bridge._