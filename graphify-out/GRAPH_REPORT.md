# Graph Report - gexcel  (2026-11-14)

## Corpus Check
- 80 files · ~90,999 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 4)

## Summary
- 1492 nodes · 5002 edges · 71 communities (51 shown, 20 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 701 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a82c77fa`
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
- Token
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
- New
- Func
- New
- Styles
- Writer
- gexcel — Excel feature coverage
- gexcel.go
- Cell
- Color
- Wrap
- Workbook
- CondFmtRule
- siXML
- CellImage
- New
- protect.go
- StreamSheet
- sheetWriter
- each
- Style
- ParseArea
- project.yml
- xlsxwriter_cell_images_4d5b9f35.md
- scanRange
- Anchor
- ParseRange
- ColName
- SparklineGroup
- Font
- Image
- HeaderFooter
- PageMargins
- PageSetup
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
- `Data flow` --references--> `Chart`  [INFERRED]
  .serena/memories/architecture.md → chart/chart.go
- `Formula engine` --references--> `Calculator`  [INFERRED]
  .serena/memories/architecture.md → formula/calc.go
- `Layout` --references--> `Calculator`  [INFERRED]
  .serena/memories/project_overview.md → formula/calc.go
- `Formula engine` --references--> `Context`  [INFERRED]
  .serena/memories/architecture.md → formula/eval.go
- `Layout` --references--> `Wrap()`  [INFERRED]
  .serena/memories/project_overview.md → gexcel.go

## Import Cycles
- 2-file cycle: `reader/reader_test.go -> reader/richdata_test.go -> reader/reader_test.go`

## Communities (71 total, 20 thin omitted)

### Community 0 - "ref.go"
Cohesion: 0.12
Nodes (10): BoolNode, EmptyNode, ErrorNode, Kind, NameNode, NumberNode, StringNode, StructuredNode (+2 more)

### Community 1 - "Evaluator"
Cohesion: 0.11
Nodes (71): Evaluator, aggAverage(), aggCount(), aggMax(), aggMin(), aggSum(), fnAtan2(), fnAverage() (+63 more)

### Community 3 - "cell"
Cohesion: 0.04
Nodes (5): ToValue(), cell, Hyperlink, IsErrorCode(), Rich()

### Community 4 - "funcs_lookup.go"
Cohesion: 0.20
Nodes (36): column(), flatten(), fnAddress(), fnChooseRC(), fnDrop(), fnFilter(), fnHLookup(), fnIndex() (+28 more)

### Community 5 - "chartWriter"
Cohesion: 0.10
Nodes (21): AnchorRef, Axis, Chart, DataLabels, LineStyle, Series, Type, DataPoint (+13 more)

### Community 6 - "Format"
Cohesion: 0.10
Nodes (27): Format, bestFraction(), digitFirst(), digitGroup(), fixed(), General(), general(), group() (+19 more)

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
Cohesion: 0.15
Nodes (29): DurationToSerial(), FromSerial(), SerialToDuration(), ToSerial(), TestSerial(), addMonths(), datePart(), days360() (+21 more)

### Community 11 - "Sheet"
Cohesion: 0.12
Nodes (17): ColorFilter, CustomFilter, DateGroupItem, DynamicFilter, IconFilter, Pane, PrintOptions, Relationship (+9 more)

### Community 12 - "Token"
Cohesion: 0.06
Nodes (43): ArrayNode, binPrec(), Parse(), parseRefNode(), References(), String(), walk(), writeNode() (+35 more)

### Community 14 - "Value"
Cohesion: 0.16
Nodes (5): Value, ValueKind, cellImageMedia, cellImages, cellImageValue

### Community 15 - "NewBuilder"
Cohesion: 0.15
Nodes (15): PivotTable, RawPart, CustomProperty, ExternalLink, NewBuilder(), externalLinkOf(), appPropsXML(), boolWord() (+7 more)

### Community 16 - "steps"
Cohesion: 0.08
Nodes (25): runs-on, steps, strategy, jobs, fuzz, target, name, on (+17 more)

### Community 17 - "steps"
Cohesion: 0.09
Nodes (25): name, runs-on, steps, GEXCEL_EXTERNAL, name, needs, runs-on, steps (+17 more)

### Community 18 - "Value"
Cohesion: 0.17
Nodes (7): Context, binaryScalar(), fnCount(), fnIs(), Err(), FromModel(), Ref()

### Community 20 - "Fill"
Cohesion: 0.14
Nodes (5): builtinStyleID(), fillKey(), BuiltinNumFmtID(), PatternType, Fill

### Community 22 - "testing.T"
Cohesion: 0.20
Nodes (23): OpenBytes(), ReadBytes(), FuzzRead(), roundTrip(), TestReadInlineStrings(), TestReadLimits(), TestRoundTripSample(), TestStream() (+15 more)

### Community 24 - "packer"
Cohesion: 0.20
Nodes (6): Relative(), guidOf(), threadedID(), packer, orDefault(), invBool()

### Community 25 - "Range"
Cohesion: 0.13
Nodes (8): Formula, FormulaKind, IgnoredError, AutoFilter, SortState, SortCondition, Range, NewRange()

### Community 26 - "Calculator"
Cohesion: 0.23
Nodes (3): splitSpec(), Calculator, cellKey

### Community 27 - ".Workbook"
Cohesion: 0.07
Nodes (5): DefinedName, Properties, ValidSheetName(), quoteSheet(), Validate()

### Community 28 - "codeql.yml"
Cohesion: 0.11
Nodes (17): runs-on, steps, jobs, analyze, name, on, pull_request, push (+9 more)

### Community 30 - "New"
Cohesion: 0.38
Nodes (6): main(), OpenReader(), StreamReader(), Read(), ReadFile(), OpenStream()

### Community 31 - "Func"
Cohesion: 0.17
Nodes (20): aggregator, each(), Func, timePart(), fnAndOr(), fnEvenOdd(), fnIfs(), fnLargeSmall() (+12 more)

### Community 32 - "New"
Cohesion: 0.20
Nodes (14): New(), New(), OpenBytes(), build(), TestAPIRoundTrip(), TestCellImage(), TestSheetManagement(), RGB() (+6 more)

### Community 33 - "Styles"
Cohesion: 0.18
Nodes (9): numberOrError(), CellStyle, RawXML, Styles, BuiltinNumFmt(), isAll(), IsDateFormat(), TestNumFmt() (+1 more)

### Community 34 - "Writer"
Cohesion: 0.13
Nodes (5): Writer, NewWriter(), RelsXML(), Options, newPacker()

### Community 35 - "gexcel — Excel feature coverage"
Cohesion: 0.14
Nodes (12): Development, Feature coverage, gexcel, License, Packages, Cells and data, Formatting, gexcel — Excel feature coverage (+4 more)

### Community 36 - "gexcel.go"
Cohesion: 0.18
Nodes (10): Permission, StreamFile(), TableNoFilter(), TableOption, TableStripes(), TableStyle(), TableTotals(), Stream (+2 more)

### Community 38 - "Cell"
Cohesion: 0.16
Nodes (8): FuzzFormat(), CellRange(), Cell, NewCell(), ParseR1C1(), r1c1Part(), FuzzParse(), TestCells()

### Community 39 - "Color"
Cohesion: 0.12
Nodes (17): schemeColor(), ColorKind, Auto(), Color, Indexed(), RGBA(), TestColors(), ThemeColor() (+9 more)

### Community 40 - "Wrap"
Cohesion: 0.28
Nodes (8): NewEmpty(), Open(), OpenStream(), TestOpenSample(), Wrap(), needExternal(), TestExternalLibreOffice(), TestExternalOpenpyxl()

### Community 41 - "Workbook"
Cohesion: 0.21
Nodes (8): Calculate(), NewCalculator(), NewEvaluator(), CalcProperties, NewSheet(), Workbook, WorkbookProtection, WorkbookView

### Community 42 - "CondFmtRule"
Cohesion: 0.33
Nodes (11): CfValueType, ColorScale, CondFmtOperator, CondFmtType, CustomIcon, CfValue, CondFmtRule, ConditionalFormat (+3 more)

### Community 43 - "siXML"
Cohesion: 0.12
Nodes (13): TestBuilder(), Clean(), EscapeAttr(), EscapeText(), EscapeUnicode(), isHex4(), leftPad(), UnescapeUnicode() (+5 more)

### Community 44 - "CellImage"
Cohesion: 0.29
Nodes (3): CellImage, SniffImage(), Drawings

### Community 45 - "New"
Cohesion: 0.25
Nodes (11): mustRange(), TestCalculateWorkbook(), testWorkbook(), NewStyles(), New(), TestCellImageUnsupportedFormat(), MustRange(), TestRanges() (+3 more)

### Community 46 - "protect.go"
Cohesion: 0.25
Nodes (5): hashPassword(), HashPasswordWithSalt(), LegacyPasswordHash(), VerifyPassword(), TestPasswordHash()

### Community 47 - "StreamSheet"
Cohesion: 0.23
Nodes (4): SheetKind, StreamSheet, StreamWriter, sheetPartName()

### Community 48 - "sheetWriter"
Cohesion: 0.39
Nodes (4): hasAttr(), orWidth(), sheetWriter, intAttr()

### Community 49 - "each"
Cohesion: 0.43
Nodes (6): criterion, fnAverageIf(), fnCountIf(), fnSumIf(), parseCriterion(), wildcardMatch()

### Community 50 - "Style"
Cohesion: 0.09
Nodes (16): Xf, Gradient, GradientStop, GradientType, HAlign, LineStyle, All(), Alignment (+8 more)

### Community 51 - "ParseArea"
Cohesion: 0.31
Nodes (8): Area, allDigits(), NeedsQuote(), ParseArea(), ParseAreas(), QuoteSheet(), splitSheet(), TestAreas()

### Community 52 - "project.yml"
Cohesion: 0.22
Nodes (8): encoding, excluded_tools, ignore_all_files_in_gitignore, ignored_paths, initial_prompt, languages, project_name, read_only

### Community 55 - "scanRange"
Cohesion: 0.39
Nodes (7): ColNumber(), isLetter(), scanCell(), scanRange(), scanWholeCols(), scanWholeRows(), TestColNames()

### Community 57 - "Anchor"
Cohesion: 0.31
Nodes (8): AnchorKind, Anchor, CellOffset, Shape, anchorStart(), chartAnchor(), markerXML(), xfrmXML()

### Community 58 - "ParseRange"
Cohesion: 0.18
Nodes (5): DefaultTableStyle(), Table, TableColumn, TableStyle, ParseRange()

### Community 62 - "Font"
Cohesion: 0.14
Nodes (9): Title, CommentReply, RichRun, Comment, Font, Underline, VertAlign, modelFontBold() (+1 more)

### Community 64 - "Image"
Cohesion: 0.40
Nodes (3): imageSize(), sniffImage(), Image

## Knowledge Gaps
- **1 isolated node(s):** `github.com/gsoultan/gexcel`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 290 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **20 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Sheet` connect `Sheet` to `ref.go`, `Sheet`, `cell`, `chartWriter`, `Node`, `Builder`, `Cell`, `Value`, `NewBuilder`, `testing.T`, `packer`, `Range`, `Calculator`, `Styles`, `gexcel.go`, `Color`, `Workbook`, `CondFmtRule`, `StreamSheet`, `sheetWriter`, `Style`, `Anchor`, `ParseRange`, `SparklineGroup`, `Font`, `Image`, `HeaderFooter`, `PageMargins`, `PageSetup`?**
  _High betweenness centrality (0.156) - this node is a cross-community bridge._
- **Are the 95 inferred relationships involving `Num()` (e.g. with `binaryScalar()` and `.argOrZero()`) actually correct?**
  _`Num()` has 95 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/gsoultan/gexcel` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `ref.go` be split into smaller, more focused modules?**
  _Cohesion score 0.11951219512195121 - nodes in this community are weakly interconnected._
- **Why does `Sheet` connect `Sheet` to `cell`, `chartWriter`, `Node`, `Sheet`, `Row`, `ParseCell`, `Range`, `.Workbook`, `Cell`, `Styles`, `gexcel.go`, `Color`, `Style`, `Row`, `.SetStyle`, `ParseRange`, `SparklineGroup`, `Image`, `HeaderFooter`, `PageMargins`, `PageSetup`?**
  _High betweenness centrality (0.101) - this node is a cross-community bridge._
- **Should `Evaluator` be split into smaller, more focused modules?**
  _Cohesion score 0.11170886075949367 - nodes in this community are weakly interconnected._
- **Why does `cell` connect `cell` to `Styles`, `gexcel.go`, `.edit`, `Color`, `CellImage`, `Cell`, `Style`, `Fill`, `Font`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._