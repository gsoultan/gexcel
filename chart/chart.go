// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package chart models DrawingML charts: the chart types Excel offers, with
// their series, axes, titles and legend. The writer turns a Chart into a
// chart part and the reader parses one back.
package chart

import "github.com/gsoultan/gexcel/style"

// Type is the chart type.
type Type string

// Chart types.
const (
	Column          Type = "col"
	ColumnStacked   Type = "colStacked"
	ColumnPercent   Type = "colPercentStacked"
	Column3D        Type = "col3D"
	Bar             Type = "bar"
	BarStacked      Type = "barStacked"
	BarPercent      Type = "barPercentStacked"
	Bar3D           Type = "bar3D"
	Line            Type = "line"
	LineStacked     Type = "lineStacked"
	LinePercent     Type = "linePercentStacked"
	LineMarkers     Type = "lineMarkers"
	Line3D          Type = "line3D"
	Pie             Type = "pie"
	Pie3D           Type = "pie3D"
	PieOfPie        Type = "ofPie"
	BarOfPie        Type = "barOfPie"
	Doughnut        Type = "doughnut"
	Area            Type = "area"
	AreaStacked     Type = "areaStacked"
	AreaPercent     Type = "areaPercentStacked"
	Area3D          Type = "area3D"
	Scatter         Type = "scatter"
	ScatterLines    Type = "scatterLines"
	ScatterSmooth   Type = "scatterSmooth"
	ScatterSmoothNM Type = "scatterSmoothNoMarkers"
	ScatterLinesNM  Type = "scatterLinesNoMarkers"
	Bubble          Type = "bubble"
	Bubble3D        Type = "bubble3D"
	Radar           Type = "radar"
	RadarMarkers    Type = "radarMarkers"
	RadarFilled     Type = "radarFilled"
	Stock           Type = "stock"
	Surface         Type = "surface"
	Surface3D       Type = "surface3D"
	SurfaceWire     Type = "surfaceWireframe"
	// Excel 2016 chart types (cx:chart). Written as chartEx parts.
	Treemap    Type = "treemap"
	Sunburst   Type = "sunburst"
	Histogram  Type = "histogram"
	Pareto     Type = "pareto"
	BoxWhisker Type = "boxWhisker"
	Waterfall  Type = "waterfall"
	Funnel     Type = "funnel"
)

// IsChartEx reports whether the type is written as a chartEx part
// (Excel 2016+ chart kinds).
func (t Type) IsChartEx() bool {
	switch t {
	case Treemap, Sunburst, Histogram, Pareto, BoxWhisker, Waterfall, Funnel:
		return true
	}
	return false
}

// Chart is one chart; it is placed on a sheet with an anchor.
type Chart struct {
	Type   Type
	Title  *Title
	Series []Series
	// Combo charts: more series groups with their own type, drawn on the
	// same plot (e.g. columns with a line). Secondary puts them on
	// secondary axes.
	Overlays []Overlay
	Legend   *Legend
	XAxis    Axis // category axis (or X value axis for scatter/bubble)
	YAxis    Axis // value axis
	ZAxis    Axis // series axis of 3-D charts
	// Appearance.
	Style           int // chart style 1..48 (c:style); 0 = 2
	RoundedCorners  bool
	PlotVisibleOnly *bool
	DisplayBlanksAs string // "gap" (default), "zero", "span"
	VaryColors      *bool
	Background      style.Color
	PlotAreaFill    style.Color
	Font            style.Font
	// Gap/overlap for bar charts (percent); 0 means Excel's default.
	GapWidth int
	Overlap  int
	// Pie options.
	FirstSliceAngle int
	HoleSize        int // doughnut, percent
	// 3-D view.
	View3D *View3D
	// DataTable shows the values under the plot.
	DataTable *DataTable
	// Anchor is set by the sheet that owns the chart.
	Anchor AnchorRef
	// Name is the drawing object name.
	Name string
	// Raw keeps the chart part verbatim when read and not modelled.
	Raw []byte
	// Width and Height in EMU when placed through a one-cell anchor.
	Width, Height int64
}

// Overlay is a second series group of a combo chart.
type Overlay struct {
	Type      Type
	Series    []Series
	Secondary bool
	GapWidth  int
	Overlap   int
}

// AnchorRef places the chart on a sheet (1-based cells, EMU offsets).
type AnchorRef struct {
	FromRow, FromCol       int
	FromRowOff, FromColOff int64
	ToRow, ToCol           int
	ToRowOff, ToColOff     int64
	OneCell                bool // use From plus Width/Height
}

// Title is a chart or axis title.
type Title struct {
	Text     string
	Formula  string // a cell reference ("Sheet1!$A$1") instead of fixed text
	Font     style.Font
	Overlay  bool
	Hidden   bool
	Rotation int
}

// Legend position.
type LegendPosition string

// Legend positions.
const (
	LegendRight    LegendPosition = "r"
	LegendLeft     LegendPosition = "l"
	LegendTop      LegendPosition = "t"
	LegendBottom   LegendPosition = "b"
	LegendTopRight LegendPosition = "tr"
)

// Legend describes the legend.
type Legend struct {
	Position LegendPosition
	Overlay  bool
	Font     style.Font
	Hidden   bool
}

// Series is one data series.
type Series struct {
	Name       string // literal name
	NameRef    string // or a reference ("Sheet1!$B$1")
	Categories string // reference to the category labels / X values
	Values     string // reference to the values ("Sheet1!$B$2:$B$10")
	Bubbles    string // bubble sizes for bubble charts
	// Literal data when not referencing cells.
	CatLiterals []string
	ValLiterals []float64
	// Appearance.
	Fill             style.Color
	Line             LineStyle
	Marker           *Marker
	Smooth           bool
	Explosion        int // pie slice explosion percent
	InvertIfNegative *bool
	DataLabels       *DataLabels
	Trendlines       []Trendline
	ErrorBars        []ErrorBars
	// DataPoints overrides single points (pie slices, highlighted bars).
	DataPoints []DataPoint
	// Order and Index in the chart; assigned by the writer when zero-based order matches.
	Index int
	Order int
	// Shape for 3-D bars: "box", "cone", "cylinder", "pyramid".
	Shape string
	// Secondary axis is set on the Overlay.
}

// LineStyle is the formatting of a line (series line, axis line, trendline).
type LineStyle struct {
	Color  style.Color
	Width  float64 // points; 0 = default
	Dash   string  // "solid", "dash", "dot", "lgDash", "sysDash", "sysDot", ...
	NoLine bool
	Smooth bool
}

// Marker on line/scatter series.
type Marker struct {
	Symbol string // "circle", "square", "diamond", "triangle", "x", "star", "dash", "dot", "plus", "none", "auto"
	Size   int
	Fill   style.Color
	Line   style.Color
}

// DataLabels controls value/category labels on points.
type DataLabels struct {
	ShowValue       bool
	ShowCategory    bool
	ShowSeriesName  bool
	ShowPercent     bool
	ShowLegendKey   bool
	ShowBubbleSize  bool
	ShowLeaderLines bool
	Position        string // "ctr", "inEnd", "outEnd", "inBase", "t", "b", "l", "r", "bestFit"
	NumFmt          string
	Separator       string
	Font            style.Font
}

// Trendline on a series.
type Trendline struct {
	Type            string // "linear", "exp", "log", "poly", "power", "movingAvg"
	Order           int    // polynomial order
	Period          int    // moving average period
	Name            string
	Forward         float64
	Backward        float64
	Intercept       *float64
	DisplayEquation bool
	DisplayRSquared bool
	Line            LineStyle
}

// ErrorBars on a series.
type ErrorBars struct {
	Direction string // "x", "y"
	Type      string // "both", "minus", "plus"
	ValueType string // "fixedVal", "percentage", "stdDev", "stdErr", "cust"
	Value     float64
	Plus      string // references for custom
	Minus     string
	NoEndCap  bool
	Line      LineStyle
}

// DataPoint formats one point of a series.
type DataPoint struct {
	Index            int
	Fill             style.Color
	Line             LineStyle
	Explosion        int
	Marker           *Marker
	InvertIfNegative *bool
}

// Axis describes an axis.
type Axis struct {
	Title          *Title
	Hidden         bool
	Deleted        bool
	Min, Max       *float64
	MajorUnit      *float64
	MinorUnit      *float64
	NumFmt         string
	NumFmtLinked   *bool
	LogBase        float64 // 0 = linear
	Reverse        bool
	Crosses        string // "autoZero", "min", "max"
	CrossesAt      *float64
	Position       string // "l", "r", "t", "b"
	MajorGridlines bool
	MinorGridlines bool
	MajorTickMark  string // "none", "in", "out", "cross"
	MinorTickMark  string
	TickLabelPos   string // "nextTo", "high", "low", "none"
	LabelFont      style.Font
	Line           LineStyle
	// Category axis options.
	TickLabelSkip int
	TickMarkSkip  int
	NoMultiLvlLbl bool
	LabelAlign    string // "ctr", "l", "r"
	LabelOffset   int
	// Date axis.
	DateAxis      bool
	BaseTimeUnit  string // "days", "months", "years"
	MajorTimeUnit string
	MinorTimeUnit string
	// DisplayUnits: "hundreds", "thousands", "millions", ...; 0 = none.
	DisplayUnits string
	Orientation  string // "minMax" (default) or "maxMin"
	TextRotation int
	// Crossing between categories (true by default for column charts).
	CrossBetween string // "between", "midCat"
}

// View3D sets the 3-D rotation.
type View3D struct {
	RotX, RotY     int
	Perspective    int
	RightAngleAxes *bool
	DepthPercent   int
	HeightPercent  int
}

// DataTable under the plot area.
type DataTable struct {
	ShowHorzBorder bool
	ShowVertBorder bool
	ShowOutline    bool
	ShowKeys       bool
}

// New returns a chart of the type with the given series.
func New(t Type, series ...Series) *Chart {
	return &Chart{Type: t, Series: series}
}

// IsBar reports whether the type is a horizontal bar chart.
func (t Type) IsBar() bool {
	return t == Bar || t == BarStacked || t == BarPercent || t == Bar3D
}

// Is3D reports whether the chart type is three-dimensional.
func (t Type) Is3D() bool {
	switch t {
	case Column3D, Bar3D, Line3D, Pie3D, Area3D, Bubble3D, Surface3D, Surface:
		return true
	}
	return false
}

// Grouping returns the c:grouping value for the type.
func (t Type) Grouping() string {
	switch t {
	case ColumnStacked, BarStacked, LineStacked, AreaStacked:
		return "stacked"
	case ColumnPercent, BarPercent, LinePercent, AreaPercent:
		return "percentStacked"
	case Column, Bar, Column3D, Bar3D:
		return "clustered"
	}
	return "standard"
}

// Family returns the underlying XML chart element family: "bar", "line",
// "pie", "doughnut", "ofPie", "area", "scatter", "bubble", "radar",
// "stock", "surface".
func (t Type) Family() string {
	switch t {
	case Column, ColumnStacked, ColumnPercent, Column3D, Bar, BarStacked, BarPercent, Bar3D:
		return "bar"
	case Line, LineStacked, LinePercent, LineMarkers, Line3D:
		return "line"
	case Pie, Pie3D:
		return "pie"
	case PieOfPie, BarOfPie:
		return "ofPie"
	case Doughnut:
		return "doughnut"
	case Area, AreaStacked, AreaPercent, Area3D:
		return "area"
	case Scatter, ScatterLines, ScatterSmooth, ScatterSmoothNM, ScatterLinesNM:
		return "scatter"
	case Bubble, Bubble3D:
		return "bubble"
	case Radar, RadarMarkers, RadarFilled:
		return "radar"
	case Stock:
		return "stock"
	case Surface, Surface3D, SurfaceWire:
		return "surface"
	}
	return "bar"
}

// HasCategoryAxis reports whether the chart family uses a category and
// value axis pair.
func (t Type) HasCategoryAxis() bool {
	switch t.Family() {
	case "pie", "doughnut", "ofPie":
		return false
	}
	return !t.IsChartEx()
}
