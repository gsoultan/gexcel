# Graph Report - gexcel  (2026-11-02)

## Corpus Check
- 76 files · ~86,133 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 2)

## Summary
- 1462 nodes · 4835 edges · 78 communities (55 shown, 23 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 690 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Workbook
- Evaluator
- Sheet
- cell
- funcs_lookup.go
- chartWriter
- Format
- .Arg
- docReader
- Builder
- funcs_date.go
- Sheet
- Node
- ParseBool
- Node
- packer
- steps
- steps
- Value
- Workbook
- Color
- Row
- testing.T
- ParseCell
- NewBuilder
- Range
- Calculator
- codeql.yml
- Token
- Layout
- Func
- New
- Reader
- Writer
- gexcel — Excel feature coverage
- Cell
- Data flow
- Border
- time.Time
- CondFmtRule
- siXML
- Font
- New
- protect.go
- StreamSheet
- sheetWriter
- each
- parseAlignment
- ParseArea
- project.yml
- scanRange
- DataValidation
- .Rows
- Anchor
- Table
- writeNode
- ValidSheetName
- .IsDate
- Comment
- ToValue
- Image
- date.go
- DefinedName
- ColName
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
2. `Sheet` - 122 edges
3. `Sheet` - 115 edges
4. `Num()` - 98 edges
5. `cell` - 84 edges
6. `Workbook` - 83 edges
7. `Style` - 81 edges
8. `Value` - 72 edges
9. `Styles` - 70 edges
10. `Cell` - 53 edges

## Surprising Connections (you probably didn't know these)
- `Data flow` --references--> `Chart`  [INFERRED]
  .serena/memories/architecture.md → chart/chart.go
- `Formula engine` --references--> `Calculator`  [INFERRED]
  .serena/memories/architecture.md → formula/calc.go
- `Layout` --references--> `Calculator`  [INFERRED]
  .serena/memories/project_overview.md → formula/calc.go
- `Data flow` --references--> `AddPrefixes()`  [INFERRED]
  .serena/memories/architecture.md → formula/prefix.go
- `Layout` --references--> `AddPrefixes()`  [INFERRED]
  .serena/memories/project_overview.md → formula/prefix.go

## Import Cycles
- None detected.

## Communities (78 total, 23 thin omitted)

### Community 0 - "Workbook"
Cohesion: 0.05
Nodes (55): numberOrError(), AnchorRef, Chart, DataTable, View3D, main(), BoolNode, Calculate() (+47 more)

### Community 1 - "Evaluator"
Cohesion: 0.11
Nodes (73): Evaluator, aggAverage(), aggCount(), aggMax(), aggMin(), aggSum(), fnAtan2(), fnAverage() (+65 more)

### Community 4 - "funcs_lookup.go"
Cohesion: 0.19
Nodes (37): column(), flatten(), fnAddress(), fnChooseRC(), fnDrop(), fnFilter(), fnHLookup(), fnIndex() (+29 more)

### Community 5 - "chartWriter"
Cohesion: 0.11
Nodes (16): Axis, DataLabels, LineStyle, Series, Title, Type, DataPoint, ErrorBars (+8 more)

### Community 6 - "Format"
Cohesion: 0.12
Nodes (22): Format, bestFraction(), digitFirst(), digitGroup(), fixed(), General(), general(), group() (+14 more)

### Community 7 - ".Arg"
Cohesion: 0.18
Nodes (26): roundHalfAway(), fnChar(), fnCode(), fnConcat(), fnDollar(), fnExact(), fnFindSearch(), fnFixed() (+18 more)

### Community 8 - "docReader"
Cohesion: 0.17
Nodes (13): SheetState, TestPackageRoundTrip(), FindRel(), Relationship, ParseRels(), RelByID(), RelsName(), Resolve() (+5 more)

### Community 9 - "Builder"
Cohesion: 0.14
Nodes (23): Builder, FormatRanges(), autoFilterXML(), breaksXML(), cfvoXML(), dataValidationXML(), escapeQuotes(), filterColumnXML() (+15 more)

### Community 10 - "funcs_date.go"
Cohesion: 0.20
Nodes (25): FromSerial(), addMonths(), datePart(), days360(), fnDate(), fnDateDif(), fnDateValue(), fnDays() (+17 more)

### Community 11 - "Sheet"
Cohesion: 0.10
Nodes (20): ColorFilter, CustomFilter, DateGroupItem, DynamicFilter, IconFilter, Pane, PrintOptions, Relationship (+12 more)

### Community 12 - "Node"
Cohesion: 0.12
Nodes (15): ArrayNode, binPrec(), Parse(), parseRefNode(), References(), String(), walk(), writeNode() (+7 more)

### Community 13 - "ParseBool"
Cohesion: 0.20
Nodes (15): ParseBool(), ParseFloat(), ParseInt(), parseAutoFilter(), parseSortState(), sheetParser, parseBorder(), parseColor() (+7 more)

### Community 14 - "Node"
Cohesion: 0.17
Nodes (16): Node, chartTypeOf(), parseAxis(), parseChart(), parseLineStyle(), parseRPr(), parseSeries(), parseSeriesList() (+8 more)

### Community 15 - "packer"
Cohesion: 0.12
Nodes (15): PivotTable, RawPart, ExternalLink, NewWriter(), externalLinkOf(), Options, packer, sharedStrings (+7 more)

### Community 16 - "steps"
Cohesion: 0.08
Nodes (25): runs-on, steps, strategy, jobs, fuzz, target, name, on (+17 more)

### Community 17 - "steps"
Cohesion: 0.09
Nodes (25): name, runs-on, steps, GEXCEL_EXTERNAL, name, needs, runs-on, steps (+17 more)

### Community 18 - "Value"
Cohesion: 0.19
Nodes (5): Context, binaryScalar(), fnIs(), Ref(), Formula engine

### Community 19 - "Workbook"
Cohesion: 0.10
Nodes (4): Workbook, NewEmpty(), Open(), Wrap()

### Community 20 - "Color"
Cohesion: 0.10
Nodes (17): parseLine(), ColorKind, Gradient, GradientStop, GradientType, LineStyle, PatternType, Auto() (+9 more)

### Community 22 - "testing.T"
Cohesion: 0.19
Nodes (18): TestNumbers(), TestOpenSample(), ReadBytes(), FuzzRead(), roundTrip(), TestReadInlineStrings(), TestReadLimits(), TestRoundTripSample() (+10 more)

### Community 24 - "NewBuilder"
Cohesion: 0.20
Nodes (12): DefaultPageMargins(), CustomProperty, Relative(), NewBuilder(), guidOf(), threadedID(), vmlXML(), appPropsXML() (+4 more)

### Community 25 - "Range"
Cohesion: 0.14
Nodes (5): Formula, FormulaKind, IgnoredError, Range, NewRange()

### Community 26 - "Calculator"
Cohesion: 0.20
Nodes (5): splitSpec(), Calculator, cellKey, Err(), FromModel()

### Community 28 - "codeql.yml"
Cohesion: 0.11
Nodes (17): runs-on, steps, jobs, analyze, name, on, pull_request, push (+9 more)

### Community 29 - "Token"
Cohesion: 0.23
Nodes (15): TestPrefixes(), AddPrefixes(), StripPrefixes(), Token, classifyRef(), decodeRune(), isAlnum(), isCellOrLine() (+7 more)

### Community 30 - "Layout"
Cohesion: 0.16
Nodes (13): TestShift(), Shift(), shiftRef(), OpenReader(), StreamFile(), StreamReader(), Read(), OpenStream() (+5 more)

### Community 31 - "Func"
Cohesion: 0.23
Nodes (15): Func, fnAndOr(), fnEvenOdd(), fnLargeSmall(), fnMinMax(), fnParity(), fnRound(), fnStdVar() (+7 more)

### Community 32 - "New"
Cohesion: 0.16
Nodes (13): New(), New(), OpenBytes(), OpenStream(), build(), TestAPIRoundTrip(), TestSheetManagement(), Solid() (+5 more)

### Community 33 - "Reader"
Cohesion: 0.17
Nodes (5): Attr(), Reader, Open(), OpenBytes(), TestWriteSample()

### Community 35 - "gexcel — Excel feature coverage"
Cohesion: 0.13
Nodes (13): Development, Feature coverage, gexcel, License, Packages, Cells and data, Drawings, Formatting (+5 more)

### Community 38 - "Cell"
Cohesion: 0.18
Nodes (8): CellRange(), Cell, MustCell(), NewCell(), ParseR1C1(), r1c1Part(), FuzzParse(), TestCells()

### Community 39 - "Data flow"
Cohesion: 0.17
Nodes (5): IsDynamicArray(), Data flow, gexcel — architecture, Reader limits, Streaming

### Community 40 - "Border"
Cohesion: 0.21
Nodes (5): All(), Border, mergeBorder(), mergeFont(), Thin()

### Community 41 - "time.Time"
Cohesion: 0.21
Nodes (6): ToSerial(), TestDates(), Time(), Date(), Properties, corePropsXML()

### Community 42 - "CondFmtRule"
Cohesion: 0.21
Nodes (12): CfValueType, ColorScale, CondFmtOperator, CondFmtType, CustomIcon, CfValue, CondFmtRule, ConditionalFormat (+4 more)

### Community 43 - "siXML"
Cohesion: 0.20
Nodes (11): TestBuilder(), Clean(), EscapeAttr(), EscapeText(), EscapeUnicode(), isHex4(), leftPad(), UnescapeUnicode() (+3 more)

### Community 44 - "Font"
Cohesion: 0.22
Nodes (9): Legend, LegendPosition, parseFont(), parseRichText(), Font, Underline, VertAlign, modelFontBold() (+1 more)

### Community 45 - "New"
Cohesion: 0.24
Nodes (10): FuzzFormat(), FuzzParse(), mustRange(), TestCalculateWorkbook(), TestEvaluate(), testWorkbook(), New(), MustRange() (+2 more)

### Community 46 - "protect.go"
Cohesion: 0.25
Nodes (5): hashPassword(), HashPasswordWithSalt(), LegacyPasswordHash(), VerifyPassword(), TestPasswordHash()

### Community 47 - "StreamSheet"
Cohesion: 0.25
Nodes (4): SheetKind, StreamSheet, StreamWriter, sheetPartName()

### Community 48 - "sheetWriter"
Cohesion: 0.22
Nodes (6): SheetProtection, hasAttr(), orWidth(), outlineLevelCol(), outlineLevelRow(), sheetWriter

### Community 49 - "each"
Cohesion: 0.29
Nodes (9): aggregator, criterion, each(), fnAverageIf(), fnCountIf(), fnIfs(), fnSumIf(), parseCriterion() (+1 more)

### Community 50 - "parseAlignment"
Cohesion: 0.28
Nodes (6): Xf, parseAlignment(), HAlign, Alignment, Protection, VAlign

### Community 51 - "ParseArea"
Cohesion: 0.31
Nodes (8): Area, allDigits(), NeedsQuote(), ParseArea(), ParseAreas(), QuoteSheet(), splitSheet(), TestAreas()

### Community 52 - "project.yml"
Cohesion: 0.22
Nodes (8): encoding, excluded_tools, ignore_all_files_in_gitignore, ignored_paths, initial_prompt, languages, project_name, read_only

### Community 54 - "scanRange"
Cohesion: 0.39
Nodes (7): ColNumber(), isLetter(), scanCell(), scanRange(), scanWholeCols(), scanWholeRows(), TestColNames()

### Community 55 - "DataValidation"
Cohesion: 0.38
Nodes (5): DataValidationType, DataValidation, DataValidationOperator, parseDataValidation(), OpBetween()

### Community 57 - "Anchor"
Cohesion: 0.33
Nodes (7): AnchorKind, Anchor, CellOffset, Shape, anchorStart(), markerXML(), xfrmXML()

### Community 58 - "Table"
Cohesion: 0.40
Nodes (4): DefaultTableStyle(), Table, TableColumn, TableStyle

### Community 59 - "writeNode"
Cohesion: 0.40
Nodes (5): attrName(), prefixFor(), qname(), rawXML(), writeNode()

### Community 60 - "ValidSheetName"
Cohesion: 0.40
Nodes (3): ValidSheetName(), quoteSheet(), Validate()

### Community 62 - "Comment"
Cohesion: 0.40
Nodes (3): CommentReply, RichRun, Comment

### Community 63 - "ToValue"
Cohesion: 0.40
Nodes (3): ToValue(), IsErrorCode(), Rich()

### Community 64 - "Image"
Cohesion: 0.40
Nodes (3): imageSize(), sniffImage(), Image

### Community 65 - "date.go"
Cohesion: 0.60
Nodes (3): DurationToSerial(), SerialToDuration(), TestSerial()

## Knowledge Gaps
- **1 isolated node(s):** `github.com/gsoultan/gexcel`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 289 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **23 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Sheet` connect `Sheet` to `Workbook`, `Sheet`, `docReader`, `ParseBool`, `packer`, `Color`, `NewBuilder`, `Range`, `Calculator`, `CondFmtRule`, `StreamSheet`, `sheetWriter`, `DataValidation`, `.Rows`, `Anchor`, `Table`, `Comment`, `Image`, `HeaderFooter`, `PageMargins`, `PageSetup`, `Hyperlink`?**
  _High betweenness centrality (0.162) - this node is a cross-community bridge._
- **Are the 95 inferred relationships involving `Num()` (e.g. with `binaryScalar()` and `.argOrZero()`) actually correct?**
  _`Num()` has 95 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/gsoultan/gexcel` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Workbook` be split into smaller, more focused modules?**
  _Cohesion score 0.05297157622739018 - nodes in this community are weakly interconnected._
- **Why does `Sheet` connect `Sheet` to `Workbook`, `Image`, `cell`, `Cell`, `HeaderFooter`, `PageMargins`, `PageSetup`, `.setCols`, `Sheet`, `Color`, `Row`, `ParseCell`, `DataValidation`, `Row`, `Range`, `Table`, `.Workbook`, `ValidSheetName`?**
  _High betweenness centrality (0.110) - this node is a cross-community bridge._
- **Should `Evaluator` be split into smaller, more focused modules?**
  _Cohesion score 0.10900331225534478 - nodes in this community are weakly interconnected._
- **Why does `cell` connect `cell` to `Workbook`, `.edit`, `Data flow`, `Border`, `Hyperlink`, `time.Time`, `Font`, `parseAlignment`, `Color`, `.IsDate`, `Comment`, `ToValue`?**
  _High betweenness centrality (0.074) - this node is a cross-community bridge._