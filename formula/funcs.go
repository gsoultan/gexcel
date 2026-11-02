// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"math"
	"math/rand/v2"
	"sort"
	"strings"
)

// builtins maps upper-case function names to implementations. The
// remaining families live in funcs_text.go, funcs_lookup.go and
// funcs_date.go and register themselves in init.
var builtins = map[string]Func{}

func register(m map[string]Func) {
	for k, v := range m {
		builtins[k] = v
	}
}

func init() {
	register(map[string]Func{
		// Math
		"SUM":          fnSum,
		"PRODUCT":      fnProduct,
		"ABS":          math1(math.Abs),
		"SQRT":         math1Check(math.Sqrt, func(x float64) bool { return x >= 0 }),
		"EXP":          math1(math.Exp),
		"LN":           math1Check(math.Log, func(x float64) bool { return x > 0 }),
		"LOG10":        math1Check(math.Log10, func(x float64) bool { return x > 0 }),
		"SIN":          math1(math.Sin),
		"COS":          math1(math.Cos),
		"TAN":          math1(math.Tan),
		"ASIN":         math1(math.Asin),
		"ACOS":         math1(math.Acos),
		"ATAN":         math1(math.Atan),
		"SINH":         math1(math.Sinh),
		"COSH":         math1(math.Cosh),
		"TANH":         math1(math.Tanh),
		"PI":           func(e *Evaluator, a []Node) Value { return Num(math.Pi) },
		"RAND":         func(e *Evaluator, a []Node) Value { return Num(rand.Float64()) },
		"RANDBETWEEN":  fnRandBetween,
		"LOG":          fnLog,
		"POWER":        fnPower,
		"MOD":          fnMod,
		"QUOTIENT":     fnQuotient,
		"INT":          math1(math.Floor),
		"TRUNC":        fnTrunc,
		"ROUND":        fnRound(roundHalfAway),
		"ROUNDUP":      fnRound(roundUp),
		"ROUNDDOWN":    fnRound(roundDown),
		"CEILING":      fnCeiling,
		"CEILING.MATH": fnCeilingMath,
		"FLOOR":        fnFloor,
		"FLOOR.MATH":   fnFloorMath,
		"MROUND":       fnMRound,
		"EVEN":         fnEvenOdd(2),
		"ODD":          fnEvenOdd(1),
		"SIGN":         math1(func(x float64) float64 { return float64(sign(x)) }),
		"FACT":         fnFact,
		"GCD":          fnGCD,
		"LCM":          fnLCM,
		"SUMPRODUCT":   fnSumProduct,
		"SUMSQ":        fnSumSq,
		"ATAN2":        fnAtan2,
		"DEGREES":      math1(func(x float64) float64 { return x * 180 / math.Pi }),
		"RADIANS":      math1(func(x float64) float64 { return x * math.Pi / 180 }),
		// Statistics
		"AVERAGE":        fnAverage,
		"AVERAGEA":       fnCountA, // replaced below
		"COUNT":          fnCount,
		"COUNTA":         fnCountA,
		"COUNTBLANK":     fnCountBlank,
		"MIN":            fnMinMax(true),
		"MAX":            fnMinMax(false),
		"MEDIAN":         fnMedian,
		"MODE":           fnMode,
		"MODE.SNGL":      fnMode,
		"STDEV":          fnStdVar(true, true),
		"STDEV.S":        fnStdVar(true, true),
		"STDEV.P":        fnStdVar(true, false),
		"STDEVP":         fnStdVar(true, false),
		"VAR":            fnStdVar(false, true),
		"VAR.S":          fnStdVar(false, true),
		"VAR.P":          fnStdVar(false, false),
		"VARP":           fnStdVar(false, false),
		"LARGE":          fnLargeSmall(true),
		"SMALL":          fnLargeSmall(false),
		"RANK":           fnRank,
		"RANK.EQ":        fnRank,
		"PERCENTILE":     fnPercentile(true),
		"PERCENTILE.INC": fnPercentile(true),
		"PERCENTILE.EXC": fnPercentile(false),
		"QUARTILE":       fnQuartile,
		"QUARTILE.INC":   fnQuartile,
		"SUMIF":          fnSumIf,
		"SUMIFS":         fnIfs(aggSum),
		"COUNTIF":        fnCountIf,
		"COUNTIFS":       fnIfs(aggCount),
		"AVERAGEIF":      fnAverageIf,
		"AVERAGEIFS":     fnIfs(aggAverage),
		"MINIFS":         fnIfs(aggMin),
		"MAXIFS":         fnIfs(aggMax),
		"GEOMEAN":        fnGeoMean,
		"CORREL":         fnCorrel,
		"PEARSON":        fnCorrel,
		"SLOPE":          fnSlope,
		"INTERCEPT":      fnIntercept,
		// Logical
		"IF":      fnIf,
		"IFS":     fnIfs2,
		"IFERROR": fnIfError,
		"IFNA":    fnIfNA,
		"AND":     fnAndOr(true),
		"OR":      fnAndOr(false),
		"XOR":     fnXor,
		"NOT":     fnNot,
		"TRUE":    func(e *Evaluator, a []Node) Value { return Boolean(true) },
		"FALSE":   func(e *Evaluator, a []Node) Value { return Boolean(false) },
		"SWITCH":  fnSwitch,
		"CHOOSE":  fnChoose,
		"LET":     fnLet,
		// Information
		"ISBLANK":    fnIs(func(v Value) bool { return v.Kind == Empty }),
		"ISNUMBER":   fnIs(func(v Value) bool { return v.Kind == Number }),
		"ISTEXT":     fnIs(func(v Value) bool { return v.Kind == Text }),
		"ISNONTEXT":  fnIs(func(v Value) bool { return v.Kind != Text }),
		"ISLOGICAL":  fnIs(func(v Value) bool { return v.Kind == Bool }),
		"ISERROR":    fnIs(func(v Value) bool { return v.Kind == Error }),
		"ISERR":      fnIs(func(v Value) bool { return v.Kind == Error && v.Str != "#N/A" }),
		"ISNA":       fnIs(func(v Value) bool { return v.Kind == Error && v.Str == "#N/A" }),
		"ISEVEN":     fnParity(0),
		"ISODD":      fnParity(1),
		"NA":         func(e *Evaluator, a []Node) Value { return errNA },
		"N":          fnN,
		"TYPE":       fnType,
		"ERROR.TYPE": fnErrorType,
		"ISFORMULA":  func(e *Evaluator, a []Node) Value { return Boolean(false) },
		// Financial
		"PMT":  fnPMT,
		"FV":   fnFV,
		"PV":   fnPV,
		"NPV":  fnNPV,
		"IPMT": fnIPMT,
		"PPMT": fnPPMT,
		"NPER": fnNPER,
		"RATE": fnRATE,
		"IRR":  fnIRR,
	})
	builtins["AVERAGEA"] = fnAverageA
}

func sign(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

func math1(f func(float64) float64) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
			x, err := toNumber(v)
			if err.Kind == Error {
				return err
			}
			return Num(f(x))
		})
	}
}

func math1Check(f func(float64) float64, ok func(float64) bool) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
			x, err := toNumber(v)
			if err.Kind == Error {
				return err
			}
			if !ok(x) {
				return errNum
			}
			return Num(f(x))
		})
	}
}

func fnSum(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	s := 0.0
	for _, n := range nums {
		s += n
	}
	return Num(s)
}

func fnSumSq(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	s := 0.0
	for _, n := range nums {
		s += n * n
	}
	return Num(s)
}

func fnProduct(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	if len(nums) == 0 {
		return Num(0)
	}
	p := 1.0
	for _, n := range nums {
		p *= n
	}
	return Num(p)
}

func fnRandBetween(e *Evaluator, args []Node) Value {
	lo, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	hi, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	lo, hi = math.Ceil(lo), math.Floor(hi)
	if hi < lo {
		return errNum
	}
	return Num(lo + math.Floor(rand.Float64()*(hi-lo+1)))
}

func fnLog(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	base, err := e.numDefault(args, 1, 10)
	if err.Kind == Error {
		return err
	}
	if x <= 0 || base <= 0 || base == 1 {
		return errNum
	}
	return Num(math.Log(x) / math.Log(base))
}

func fnPower(e *Evaluator, args []Node) Value {
	return e.zip(e.Arg(args, 0), e.Arg(args, 1), func(a, b Value) Value { return binaryScalar("^", a, b) })
}

func fnMod(e *Evaluator, args []Node) Value {
	return e.zip(e.Arg(args, 0), e.Arg(args, 1), func(a, b Value) Value {
		x, err := toNumber(a)
		if err.Kind == Error {
			return err
		}
		y, err := toNumber(b)
		if err.Kind == Error {
			return err
		}
		if y == 0 {
			return errDiv0
		}
		r := math.Mod(x, y)
		if r != 0 && (r < 0) != (y < 0) {
			r += y
		}
		return Num(r)
	})
}

func fnQuotient(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	y, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	if y == 0 {
		return errDiv0
	}
	return Num(math.Trunc(x / y))
}

func fnTrunc(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	d, err := e.numDefault(args, 1, 0)
	if err.Kind == Error {
		return err
	}
	return Num(roundDown(x, int(d)))
}

func pow10(d int) float64 { return math.Pow(10, float64(d)) }

func roundHalfAway(x float64, d int) float64 {
	p := pow10(d)
	v := x * p
	// Correct binary representation error (e.g. 2.675*100 = 267.49999).
	v = math.Round(v*1e8) / 1e8
	return math.Round(v) / p
}

func roundUp(x float64, d int) float64 {
	p := pow10(d)
	v := x * p
	v = math.Round(v*1e8) / 1e8
	if v >= 0 {
		return math.Ceil(v) / p
	}
	return math.Floor(v) / p
}

func roundDown(x float64, d int) float64 {
	p := pow10(d)
	v := x * p
	v = math.Round(v*1e8) / 1e8
	return math.Trunc(v) / p
}

func fnRound(f func(float64, int) float64) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.zip(e.Arg(args, 0), e.argOrZero(args, 1), func(a, b Value) Value {
			x, err := toNumber(a)
			if err.Kind == Error {
				return err
			}
			d, err := toNumber(b)
			if err.Kind == Error {
				return err
			}
			return Num(f(x, int(d)))
		})
	}
}

func (e *Evaluator) argOrZero(args []Node, i int) Value {
	v := e.Arg(args, i)
	if v.Kind == Empty {
		return Num(0)
	}
	return v
}

func fnCeiling(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	s, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	if s == 0 {
		return Num(0)
	}
	if x > 0 && s < 0 {
		return errNum
	}
	return Num(math.Ceil(x/s) * s)
}

func fnCeilingMath(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	s, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	mode, _ := e.numDefault(args, 2, 0)
	if s == 0 {
		return Num(0)
	}
	s = math.Abs(s)
	if x < 0 && mode != 0 {
		return Num(-math.Ceil(-x/s) * s)
	}
	return Num(math.Ceil(x/s) * s)
}

func fnFloor(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	s, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	if s == 0 {
		return errDiv0
	}
	if x > 0 && s < 0 {
		return errNum
	}
	return Num(math.Floor(x/s) * s)
}

func fnFloorMath(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	s, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	mode, _ := e.numDefault(args, 2, 0)
	if s == 0 {
		return Num(0)
	}
	s = math.Abs(s)
	if x < 0 && mode != 0 {
		return Num(-math.Floor(-x/s) * s)
	}
	return Num(math.Floor(x/s) * s)
}

func fnMRound(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	m, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	if m == 0 {
		return Num(0)
	}
	if (x < 0) != (m < 0) {
		return errNum
	}
	return Num(roundHalfAway(x/m, 0) * m)
}

func fnEvenOdd(parity int) Func {
	return func(e *Evaluator, args []Node) Value {
		x, err := e.num(args, 0)
		if err.Kind == Error {
			return err
		}
		a := math.Ceil(math.Abs(x))
		if parity == 2 {
			if math.Mod(a, 2) != 0 {
				a++
			}
		} else if math.Mod(a, 2) == 0 {
			a++
		}
		if x < 0 {
			a = -a
		}
		return Num(a)
	}
}

func fnFact(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	if x < 0 || x > 170 {
		return errNum
	}
	r := 1.0
	for i := 2; i <= int(x); i++ {
		r *= float64(i)
	}
	return Num(r)
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func fnGCD(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	var g int64
	for _, n := range nums {
		if n < 0 {
			return errNum
		}
		g = gcd(g, int64(n))
	}
	return Num(float64(g))
}

func fnLCM(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	l := int64(1)
	for _, n := range nums {
		if n < 0 {
			return errNum
		}
		v := int64(n)
		if v == 0 {
			return Num(0)
		}
		l = l / gcd(l, v) * v
	}
	return Num(float64(l))
}

func fnAtan2(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	y, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	if x == 0 && y == 0 {
		return errDiv0
	}
	return Num(math.Atan2(y, x))
}

func fnSumProduct(e *Evaluator, args []Node) Value {
	var arrays []Value
	for i := range args {
		a, _ := e.rangeOf(args, i)
		if a.Kind == Error {
			return a
		}
		arrays = append(arrays, a)
	}
	if len(arrays) == 0 {
		return errValue
	}
	r, c := arrays[0].Dims()
	for _, a := range arrays[1:] {
		if ar, ac := a.Dims(); ar != r || ac != c {
			return errValue
		}
	}
	s := 0.0
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			p := 1.0
			for _, a := range arrays {
				v := a.At(i, j)
				if v.Kind == Error {
					return v
				}
				if v.Kind == Number {
					p *= v.Num
				} else if v.Kind == Bool && v.Bool {
					// booleans count as 0 in SUMPRODUCT unless coerced; keep Excel: TRUE*... -> 0? Excel treats non-numeric as 0.
					p = 0
				} else {
					p = 0
				}
			}
			s += p
		}
	}
	return Num(s)
}

func fnAverage(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	if len(nums) == 0 {
		return errDiv0
	}
	s := 0.0
	for _, n := range nums {
		s += n
	}
	return Num(s / float64(len(nums)))
}

func fnAverageA(e *Evaluator, args []Node) Value {
	s, n := 0.0, 0
	for i := range args {
		v := e.Arg(args, i)
		var err Value
		each(v, func(x Value) bool {
			switch x.Kind {
			case Number:
				s += x.Num
				n++
			case Bool:
				if x.Bool {
					s++
				}
				n++
			case Text:
				n++
			case Error:
				err = x
				return false
			}
			return true
		})
		if err.Kind == Error {
			return err
		}
	}
	if n == 0 {
		return errDiv0
	}
	return Num(s / float64(n))
}

func fnCount(e *Evaluator, args []Node) Value {
	n := 0
	for i := range args {
		raw := e.ArgRaw(args, i)
		fromRef := raw.Kind == Reference || raw.Kind == Array
		each(e.deref(raw), func(x Value) bool {
			switch x.Kind {
			case Number:
				n++
			case Bool:
				if !fromRef {
					n++
				}
			case Text:
				if !fromRef {
					if _, ok := parseNumber(x.Str); ok {
						n++
					}
				}
			}
			return true
		})
	}
	return Num(float64(n))
}

func fnCountA(e *Evaluator, args []Node) Value {
	n := 0
	for i := range args {
		each(e.Arg(args, i), func(x Value) bool {
			if x.Kind != Empty {
				n++
			}
			return true
		})
	}
	return Num(float64(n))
}

func fnCountBlank(e *Evaluator, args []Node) Value {
	n := 0
	for i := range args {
		each(e.Arg(args, i), func(x Value) bool {
			if x.Kind == Empty || x.Kind == Text && x.Str == "" {
				n++
			}
			return true
		})
	}
	return Num(float64(n))
}

func fnMinMax(isMin bool) Func {
	return func(e *Evaluator, args []Node) Value {
		nums, err := e.numbersOf(args)
		if err.Kind == Error {
			return err
		}
		if len(nums) == 0 {
			return Num(0)
		}
		r := nums[0]
		for _, n := range nums[1:] {
			if isMin && n < r || !isMin && n > r {
				r = n
			}
		}
		return Num(r)
	}
}

func fnMedian(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	if len(nums) == 0 {
		return errNum
	}
	sort.Float64s(nums)
	n := len(nums)
	if n%2 == 1 {
		return Num(nums[n/2])
	}
	return Num((nums[n/2-1] + nums[n/2]) / 2)
}

func fnMode(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	counts := map[float64]int{}
	best, bestN := 0.0, 0
	for _, n := range nums {
		counts[n]++
		if counts[n] > bestN {
			best, bestN = n, counts[n]
		}
	}
	if bestN < 2 {
		return errNA
	}
	return Num(best)
}

func fnStdVar(stdev, sample bool) Func {
	return func(e *Evaluator, args []Node) Value {
		nums, err := e.numbersOf(args)
		if err.Kind == Error {
			return err
		}
		n := float64(len(nums))
		if sample && n < 2 || !sample && n < 1 {
			return errDiv0
		}
		mean := 0.0
		for _, x := range nums {
			mean += x
		}
		mean /= n
		ss := 0.0
		for _, x := range nums {
			ss += (x - mean) * (x - mean)
		}
		if sample {
			ss /= n - 1
		} else {
			ss /= n
		}
		if stdev {
			return Num(math.Sqrt(ss))
		}
		return Num(ss)
	}
}

func fnLargeSmall(large bool) Func {
	return func(e *Evaluator, args []Node) Value {
		nums, err := e.numbersOf(args[:1])
		if err.Kind == Error {
			return err
		}
		k, err := e.num(args, 1)
		if err.Kind == Error {
			return err
		}
		ki := int(k)
		if ki < 1 || ki > len(nums) {
			return errNum
		}
		sort.Float64s(nums)
		if large {
			return Num(nums[len(nums)-ki])
		}
		return Num(nums[ki-1])
	}
}

func fnRank(e *Evaluator, args []Node) Value {
	x, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	nums, err := e.numbersOf(args[1:2])
	if err.Kind == Error {
		return err
	}
	asc, _ := e.numDefault(args, 2, 0)
	rank := 1
	found := false
	for _, n := range nums {
		if n == x {
			found = true
		}
		if asc == 0 && n > x || asc != 0 && n < x {
			rank++
		}
	}
	if !found {
		return errNA
	}
	return Num(float64(rank))
}

func percentile(nums []float64, p float64, inclusive bool) (float64, bool) {
	if len(nums) == 0 {
		return 0, false
	}
	sort.Float64s(nums)
	n := float64(len(nums))
	var pos float64
	if inclusive {
		if p < 0 || p > 1 {
			return 0, false
		}
		pos = p * (n - 1)
	} else {
		if p <= 0 || p >= 1 {
			return 0, false
		}
		pos = p*(n+1) - 1
		if pos < 0 || pos > n-1 {
			return 0, false
		}
	}
	lo := math.Floor(pos)
	frac := pos - lo
	i := int(lo)
	if i+1 >= len(nums) {
		return nums[len(nums)-1], true
	}
	return nums[i] + frac*(nums[i+1]-nums[i]), true
}

func fnPercentile(inclusive bool) Func {
	return func(e *Evaluator, args []Node) Value {
		nums, err := e.numbersOf(args[:1])
		if err.Kind == Error {
			return err
		}
		p, err := e.num(args, 1)
		if err.Kind == Error {
			return err
		}
		v, ok := percentile(nums, p, inclusive)
		if !ok {
			return errNum
		}
		return Num(v)
	}
}

func fnQuartile(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args[:1])
	if err.Kind == Error {
		return err
	}
	q, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	if q < 0 || q > 4 {
		return errNum
	}
	v, ok := percentile(nums, q/4, true)
	if !ok {
		return errNum
	}
	return Num(v)
}

func fnGeoMean(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args)
	if err.Kind == Error {
		return err
	}
	if len(nums) == 0 {
		return errNum
	}
	s := 0.0
	for _, n := range nums {
		if n <= 0 {
			return errNum
		}
		s += math.Log(n)
	}
	return Num(math.Exp(s / float64(len(nums))))
}

func pairs(e *Evaluator, args []Node) (xs, ys []float64, err Value) {
	a, _ := e.rangeOf(args, 0)
	b, _ := e.rangeOf(args, 1)
	if a.Kind == Error {
		return nil, nil, a
	}
	if b.Kind == Error {
		return nil, nil, b
	}
	ar, ac := a.Dims()
	br, bc := b.Dims()
	if ar*ac != br*bc {
		return nil, nil, errNA
	}
	for i := 0; i < ar; i++ {
		for j := 0; j < ac; j++ {
			x, y := a.At(i, j), b.At(i*ac+j/bc, (i*ac+j)%bc)
			if x.Kind == Number && y.Kind == Number {
				xs = append(xs, x.Num)
				ys = append(ys, y.Num)
			}
		}
	}
	return xs, ys, Value{}
}

func linreg(xs, ys []float64) (slope, intercept, r float64, ok bool) {
	n := float64(len(xs))
	if n == 0 {
		return 0, 0, 0, false
	}
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/n, sy/n
	var sxy, sxx, syy float64
	for i := range xs {
		sxy += (xs[i] - mx) * (ys[i] - my)
		sxx += (xs[i] - mx) * (xs[i] - mx)
		syy += (ys[i] - my) * (ys[i] - my)
	}
	if sxx == 0 || syy == 0 {
		return 0, 0, 0, false
	}
	slope = sxy / sxx
	return slope, my - slope*mx, sxy / math.Sqrt(sxx*syy), true
}

func fnCorrel(e *Evaluator, args []Node) Value {
	xs, ys, err := pairs(e, args)
	if err.Kind == Error {
		return err
	}
	_, _, r, ok := linreg(xs, ys)
	if !ok {
		return errDiv0
	}
	return Num(r)
}

func fnSlope(e *Evaluator, args []Node) Value {
	ys, xs, err := pairs(e, args)
	if err.Kind == Error {
		return err
	}
	s, _, _, ok := linreg(xs, ys)
	if !ok {
		return errDiv0
	}
	return Num(s)
}

func fnIntercept(e *Evaluator, args []Node) Value {
	ys, xs, err := pairs(e, args)
	if err.Kind == Error {
		return err
	}
	_, b, _, ok := linreg(xs, ys)
	if !ok {
		return errDiv0
	}
	return Num(b)
}

// Criteria matching for *IF functions.

type criterion struct {
	op     string
	num    float64
	isNum  bool
	text   string
	empty  bool
	isBool bool
	b      bool
}

func parseCriterion(v Value) criterion {
	if v.Kind == Bool {
		return criterion{op: "=", isBool: true, b: v.Bool}
	}
	if v.Kind == Number {
		return criterion{op: "=", num: v.Num, isNum: true}
	}
	if v.Kind == Empty {
		return criterion{op: "=", empty: true}
	}
	s := toText(v)
	op := "="
	for _, p := range []string{"<>", ">=", "<=", "=", ">", "<"} {
		if strings.HasPrefix(s, p) {
			op = p
			s = s[len(p):]
			break
		}
	}
	c := criterion{op: op, text: s}
	switch strings.ToUpper(s) {
	case "TRUE":
		return criterion{op: op, isBool: true, b: true}
	case "FALSE":
		return criterion{op: op, isBool: true, b: false}
	}
	if f, ok := parseNumber(s); ok && s != "" {
		c.num, c.isNum = f, true
	}
	if s == "" && op == "=" {
		c.empty = true
	}
	return c
}

func (c criterion) match(v Value) bool {
	if c.isBool {
		eq := v.Kind == Bool && v.Bool == c.b
		if c.op == "<>" {
			return !eq
		}
		return eq
	}
	if c.empty {
		isBlank := v.Kind == Empty || v.Kind == Text && v.Str == ""
		if c.op == "<>" {
			return !isBlank
		}
		return isBlank
	}
	if c.isNum {
		if v.Kind == Empty {
			// Blank cells do not match numeric criteria except "<>n".
			return c.op == "<>"
		}
		if v.Kind == Bool {
			return false
		}
		f, ok := v.Num, v.Kind == Number
		if !ok {
			if v.Kind == Text {
				// Text never equals a number unless the criterion was text-only.
				if c.op == "<>" {
					return true
				}
				return false
			}
			return false
		}
		switch c.op {
		case "=":
			return f == c.num
		case "<>":
			return f != c.num
		case ">":
			return f > c.num
		case "<":
			return f < c.num
		case ">=":
			return f >= c.num
		case "<=":
			return f <= c.num
		}
		return false
	}
	// Text criteria with wildcards.
	s := toText(v)
	switch c.op {
	case "=":
		return wildcardMatch(strings.ToLower(c.text), strings.ToLower(s))
	case "<>":
		return !wildcardMatch(strings.ToLower(c.text), strings.ToLower(s))
	}
	if v.Kind != Text {
		return false
	}
	cmp := strings.Compare(strings.ToLower(s), strings.ToLower(c.text))
	switch c.op {
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	}
	return false
}

// wildcardMatch matches Excel patterns: * any run, ? one char, ~ escapes.
func wildcardMatch(pat, s string) bool {
	p, i := 0, 0
	starP, starI := -1, -1
	for i < len(s) {
		if p < len(pat) {
			switch pat[p] {
			case '~':
				if p+1 < len(pat) && pat[p+1] == s[i] {
					p += 2
					i++
					continue
				}
			case '?':
				p++
				i++
				continue
			case '*':
				starP, starI = p, i
				p++
				continue
			default:
				if pat[p] == s[i] {
					p++
					i++
					continue
				}
			}
		}
		if starP >= 0 {
			p = starP + 1
			starI++
			i = starI
			continue
		}
		return false
	}
	for p < len(pat) && pat[p] == '*' {
		p++
	}
	return p == len(pat)
}

func fnSumIf(e *Evaluator, args []Node) Value {
	rng, _ := e.rangeOf(args, 0)
	if rng.Kind == Error {
		return rng
	}
	crit := parseCriterion(e.Arg(args, 1))
	sumRng := rng
	if len(args) > 2 {
		sumRng, _ = e.rangeOf(args, 2)
		if sumRng.Kind == Error {
			return sumRng
		}
	}
	s := 0.0
	r, c := rng.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if crit.match(rng.At(i, j)) {
				if v := sumRng.At(i, j); v.Kind == Number {
					s += v.Num
				}
			}
		}
	}
	return Num(s)
}

func fnCountIf(e *Evaluator, args []Node) Value {
	rng, _ := e.rangeOf(args, 0)
	if rng.Kind == Error {
		return rng
	}
	crit := parseCriterion(e.Arg(args, 1))
	n := 0
	each(rng, func(v Value) bool {
		if crit.match(v) {
			n++
		}
		return true
	})
	return Num(float64(n))
}

func fnAverageIf(e *Evaluator, args []Node) Value {
	rng, _ := e.rangeOf(args, 0)
	if rng.Kind == Error {
		return rng
	}
	crit := parseCriterion(e.Arg(args, 1))
	avgRng := rng
	if len(args) > 2 {
		avgRng, _ = e.rangeOf(args, 2)
		if avgRng.Kind == Error {
			return avgRng
		}
	}
	s, n := 0.0, 0
	r, c := rng.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if crit.match(rng.At(i, j)) {
				if v := avgRng.At(i, j); v.Kind == Number {
					s += v.Num
					n++
				}
			}
		}
	}
	if n == 0 {
		return errDiv0
	}
	return Num(s / float64(n))
}

type aggregator func(vals []float64, matched int) Value

func aggSum(vals []float64, _ int) Value {
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return Num(s)
}

func aggCount(_ []float64, matched int) Value { return Num(float64(matched)) }

func aggAverage(vals []float64, _ int) Value {
	if len(vals) == 0 {
		return errDiv0
	}
	return Num(aggSum(vals, 0).Num / float64(len(vals)))
}

func aggMin(vals []float64, _ int) Value {
	if len(vals) == 0 {
		return Num(0)
	}
	m := vals[0]
	for _, v := range vals {
		m = math.Min(m, v)
	}
	return Num(m)
}

func aggMax(vals []float64, _ int) Value {
	if len(vals) == 0 {
		return Num(0)
	}
	m := vals[0]
	for _, v := range vals {
		m = math.Max(m, v)
	}
	return Num(m)
}

// fnIfs implements SUMIFS/COUNTIFS/AVERAGEIFS/MINIFS/MAXIFS. For COUNTIFS
// the pairs start at argument 0; otherwise argument 0 is the value range.
func fnIfs(agg aggregator) Func {
	return func(e *Evaluator, args []Node) Value {
		start := 1
		var valRng Value
		isCount := false
		if len(args)%2 == 0 {
			// COUNTIFS: (crit_range, crit)+
			start = 0
			isCount = true
		} else {
			valRng, _ = e.rangeOf(args, 0)
			if valRng.Kind == Error {
				return valRng
			}
		}
		type pair struct {
			rng  Value
			crit criterion
		}
		var ps []pair
		for i := start; i+1 < len(args); i += 2 {
			rng, _ := e.rangeOf(args, i)
			if rng.Kind == Error {
				return rng
			}
			ps = append(ps, pair{rng, parseCriterion(e.Arg(args, i+1))})
		}
		if len(ps) == 0 {
			return errValue
		}
		r, c := ps[0].rng.Dims()
		for _, p := range ps {
			if pr, pc := p.rng.Dims(); pr != r || pc != c {
				return errValue
			}
		}
		if !isCount {
			if vr, vc := valRng.Dims(); vr != r || vc != c {
				return errValue
			}
		}
		var vals []float64
		matched := 0
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				ok := true
				for _, p := range ps {
					if !p.crit.match(p.rng.At(i, j)) {
						ok = false
						break
					}
				}
				if !ok {
					continue
				}
				matched++
				if !isCount {
					if v := valRng.At(i, j); v.Kind == Number {
						vals = append(vals, v.Num)
					}
				}
			}
		}
		return agg(vals, matched)
	}
}

// Logical

func fnIf(e *Evaluator, args []Node) Value {
	cond := e.Arg(args, 0)
	if cond.Kind == Array {
		yes, no := e.Arg(args, 1), Boolean(false)
		if len(args) > 2 {
			no = e.Arg(args, 2)
		}
		if yes.Kind == Empty {
			yes = Num(0)
		}
		if no.Kind == Empty && len(args) > 2 {
			no = Num(0)
		}
		pick := func(v Value, i, j int) Value {
			r, c := v.Dims()
			if r == 1 {
				i = 0
			}
			if c == 1 {
				j = 0
			}
			if i >= r || j >= c {
				return errNA
			}
			return v.At(i, j)
		}
		out := make([][]Value, len(cond.Rows))
		for i, row := range cond.Rows {
			out[i] = make([]Value, len(row))
			for j, c := range row {
				b, err := toBool(c)
				if err.Kind == Error {
					out[i][j] = err
					continue
				}
				if b {
					out[i][j] = pick(yes, i, j)
				} else {
					out[i][j] = pick(no, i, j)
				}
			}
		}
		return Arr(out)
	}
	b, err := toBool(cond)
	if err.Kind == Error {
		return err
	}
	if b {
		if len(args) < 2 {
			return Boolean(true)
		}
		v := e.Arg(args, 1)
		if v.Kind == Empty {
			return Num(0)
		}
		return v
	}
	if len(args) < 3 {
		return Boolean(false)
	}
	v := e.Arg(args, 2)
	if v.Kind == Empty {
		return Num(0)
	}
	return v
}

func fnIfs2(e *Evaluator, args []Node) Value {
	for i := 0; i+1 < len(args); i += 2 {
		b, err := e.boolean(args, i)
		if err.Kind == Error {
			return err
		}
		if b {
			return e.Arg(args, i+1)
		}
	}
	return errNA
}

func fnIfError(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind == Error {
		return e.Arg(args, 1)
	}
	return v
}

func fnIfNA(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind == Error && v.Str == "#N/A" {
		return e.Arg(args, 1)
	}
	return v
}

func fnAndOr(isAnd bool) Func {
	return func(e *Evaluator, args []Node) Value {
		result := isAnd
		seen := false
		for i := range args {
			v := e.Arg(args, i)
			var err Value
			each(v, func(x Value) bool {
				if x.Kind == Empty || x.Kind == Text && v.Kind == Array {
					return true
				}
				b, e2 := toBool(x)
				if e2.Kind == Error {
					err = e2
					return false
				}
				seen = true
				if isAnd {
					result = result && b
				} else {
					result = result || b
				}
				return true
			})
			if err.Kind == Error {
				return err
			}
		}
		if !seen {
			return errValue
		}
		return Boolean(result)
	}
}

func fnXor(e *Evaluator, args []Node) Value {
	n := 0
	for i := range args {
		var err Value
		each(e.Arg(args, i), func(x Value) bool {
			if x.Kind == Empty {
				return true
			}
			b, e2 := toBool(x)
			if e2.Kind == Error {
				err = e2
				return false
			}
			if b {
				n++
			}
			return true
		})
		if err.Kind == Error {
			return err
		}
	}
	return Boolean(n%2 == 1)
}

func fnNot(e *Evaluator, args []Node) Value {
	return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
		b, err := toBool(v)
		if err.Kind == Error {
			return err
		}
		return Boolean(!b)
	})
}

func fnSwitch(e *Evaluator, args []Node) Value {
	if len(args) < 3 {
		return errNA
	}
	x := e.Arg(args, 0)
	if x.Kind == Error {
		return x
	}
	i := 1
	for ; i+1 < len(args); i += 2 {
		if compare(x, e.Arg(args, i)) == 0 {
			return e.Arg(args, i+1)
		}
	}
	if i < len(args) {
		return e.Arg(args, i)
	}
	return errNA
}

func fnChoose(e *Evaluator, args []Node) Value {
	k, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	i := int(k)
	if i < 1 || i >= len(args) {
		return errValue
	}
	return e.ArgRaw(args, i)
}

// LET(name, value, ..., calculation): names are bound as temporary
// functions so the body can refer to them.
func fnLet(e *Evaluator, args []Node) Value {
	if len(args) < 3 || len(args)%2 == 0 {
		return errValue
	}
	saved := e.Funcs
	e.Funcs = map[string]Func{}
	for k, v := range saved {
		e.Funcs[k] = v
	}
	defer func() { e.Funcs = saved }()
	body := args[len(args)-1]
	// Names appear as NameNode; resolve them through a substitution walk.
	bindings := map[string]Value{}
	for i := 0; i+1 < len(args)-1; i += 2 {
		n, ok := args[i].(NameNode)
		if !ok {
			return errValue
		}
		bindings[strings.ToUpper(n.Name)] = e.deref(e.evalWithBindings(args[i+1], bindings))
	}
	return e.deref(e.evalWithBindings(body, bindings))
}

func (e *Evaluator) evalWithBindings(n Node, b map[string]Value) Value {
	return e.eval(substitute(n, b))
}

func substitute(n Node, b map[string]Value) Node {
	switch x := n.(type) {
	case NameNode:
		if v, ok := b[strings.ToUpper(x.Name)]; ok {
			return valueNode{v}
		}
		return x
	case UnaryNode:
		x.X = substitute(x.X, b)
		return x
	case BinaryNode:
		x.L, x.R = substitute(x.L, b), substitute(x.R, b)
		return x
	case CallNode:
		args := make([]Node, len(x.Args))
		for i, a := range x.Args {
			args[i] = substitute(a, b)
		}
		x.Args = args
		return x
	case ArrayNode:
		rows := make([][]Node, len(x.Rows))
		for i, r := range x.Rows {
			rows[i] = make([]Node, len(r))
			for j, c := range r {
				rows[i][j] = substitute(c, b)
			}
		}
		x.Rows = rows
		return x
	}
	return n
}

// valueNode carries an already-evaluated value through the tree.
type valueNode struct{ v Value }

func (valueNode) node() {}

// Information

func fnIs(pred func(Value) bool) Func {
	return func(e *Evaluator, args []Node) Value {
		raw := e.ArgRaw(args, 0)
		if raw.Kind == Reference && !raw.Range.IsCell() {
			return e.mapScalar(e.toArray(raw), func(v Value) Value { return Boolean(pred(v)) })
		}
		v := e.deref(raw)
		if v.Kind == Array {
			return e.mapScalar(v, func(x Value) Value { return Boolean(pred(x)) })
		}
		return Boolean(pred(v))
	}
}

func fnParity(want int) Func {
	return func(e *Evaluator, args []Node) Value {
		x, err := e.num(args, 0)
		if err.Kind == Error {
			return err
		}
		return Boolean(int(math.Abs(math.Trunc(x)))%2 == want)
	}
}

func fnN(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	switch v.Kind {
	case Number:
		return v
	case Bool:
		if v.Bool {
			return Num(1)
		}
		return Num(0)
	case Error:
		return v
	}
	return Num(0)
}

func fnType(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	switch v.Kind {
	case Number, Empty:
		return Num(1)
	case Text:
		return Num(2)
	case Bool:
		return Num(4)
	case Error:
		return Num(16)
	case Array:
		return Num(64)
	}
	return Num(1)
}

func fnErrorType(e *Evaluator, args []Node) Value {
	v := e.Arg(args, 0)
	if v.Kind != Error {
		return errNA
	}
	codes := map[string]float64{"#NULL!": 1, "#DIV/0!": 2, "#VALUE!": 3, "#REF!": 4, "#NAME?": 5, "#NUM!": 6, "#N/A": 7, "#GETTING_DATA": 8, "#SPILL!": 9, "#CALC!": 14}
	if c, ok := codes[v.Str]; ok {
		return Num(c)
	}
	return errNA
}

// Financial

func fnPMT(e *Evaluator, args []Node) Value {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	nper, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	pv, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	fv, _ := e.numDefault(args, 3, 0)
	typ, _ := e.numDefault(args, 4, 0)
	return Num(pmt(rate, nper, pv, fv, typ != 0))
}

func pmt(rate, nper, pv, fv float64, begin bool) float64 {
	if rate == 0 {
		return -(pv + fv) / nper
	}
	f := math.Pow(1+rate, nper)
	p := -(pv*f + fv) * rate / (f - 1)
	if begin {
		p /= 1 + rate
	}
	return p
}

func fnFV(e *Evaluator, args []Node) Value {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	nper, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	p, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	pv, _ := e.numDefault(args, 3, 0)
	typ, _ := e.numDefault(args, 4, 0)
	if rate == 0 {
		return Num(-(pv + p*nper))
	}
	f := math.Pow(1+rate, nper)
	t := 0.0
	if typ != 0 {
		t = 1
	}
	return Num(-(pv*f + p*(1+rate*t)*(f-1)/rate))
}

func fnPV(e *Evaluator, args []Node) Value {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	nper, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	p, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	fv, _ := e.numDefault(args, 3, 0)
	typ, _ := e.numDefault(args, 4, 0)
	if rate == 0 {
		return Num(-(fv + p*nper))
	}
	f := math.Pow(1+rate, nper)
	t := 0.0
	if typ != 0 {
		t = 1
	}
	return Num(-(fv + p*(1+rate*t)*(f-1)/rate) / f)
}

func fnNPV(e *Evaluator, args []Node) Value {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	nums, err := e.numbersOf(args[1:])
	if err.Kind == Error {
		return err
	}
	s := 0.0
	for i, v := range nums {
		s += v / math.Pow(1+rate, float64(i+1))
	}
	return Num(s)
}

func fnIPMT(e *Evaluator, args []Node) Value {
	ip, _, err := ipmtPpmt(e, args)
	if err.Kind == Error {
		return err
	}
	return Num(ip)
}

func fnPPMT(e *Evaluator, args []Node) Value {
	_, pp, err := ipmtPpmt(e, args)
	if err.Kind == Error {
		return err
	}
	return Num(pp)
}

func ipmtPpmt(e *Evaluator, args []Node) (float64, float64, Value) {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return 0, 0, err
	}
	per, err := e.num(args, 1)
	if err.Kind == Error {
		return 0, 0, err
	}
	nper, err := e.num(args, 2)
	if err.Kind == Error {
		return 0, 0, err
	}
	pv, err := e.num(args, 3)
	if err.Kind == Error {
		return 0, 0, err
	}
	fv, _ := e.numDefault(args, 4, 0)
	typ, _ := e.numDefault(args, 5, 0)
	if per < 1 || per > nper {
		return 0, 0, errNum
	}
	begin := typ != 0
	p := pmt(rate, nper, pv, fv, begin)
	var ipmt float64
	switch {
	case per == 1 && begin:
		ipmt = 0
	case per == 1:
		ipmt = -pv * rate
	case begin:
		ipmt = (futureValue(rate, per-2, p, pv, true) - p) * rate
	default:
		ipmt = futureValue(rate, per-1, p, pv, false) * rate
	}
	return ipmt, p - ipmt, Value{}
}

// futureValue is FV() for the cash-flow sign convention used by IPMT.
func futureValue(rate, nper, pmt, pv float64, begin bool) float64 {
	if rate == 0 {
		return -(pv + pmt*nper)
	}
	g := math.Pow(1+rate, nper)
	t := 0.0
	if begin {
		t = 1
	}
	return -(pv*g + pmt*(1+rate*t)*(g-1)/rate)
}

func fnNPER(e *Evaluator, args []Node) Value {
	rate, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	p, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	pv, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	fv, _ := e.numDefault(args, 3, 0)
	typ, _ := e.numDefault(args, 4, 0)
	if rate == 0 {
		if p == 0 {
			return errNum
		}
		return Num(-(pv + fv) / p)
	}
	t := 0.0
	if typ != 0 {
		t = 1
	}
	a := p * (1 + rate*t) / rate
	num := -fv + a
	den := pv + a
	if num/den <= 0 {
		return errNum
	}
	return Num(math.Log(num/den) / math.Log(1+rate))
}

func fnRATE(e *Evaluator, args []Node) Value {
	nper, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	p, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	pv, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	fv, _ := e.numDefault(args, 3, 0)
	typ, _ := e.numDefault(args, 4, 0)
	guess, _ := e.numDefault(args, 5, 0.1)
	begin := typ != 0
	f := func(r float64) float64 {
		if r == 0 {
			return pv + p*nper + fv
		}
		g := math.Pow(1+r, nper)
		t := 0.0
		if begin {
			t = 1
		}
		return pv*g + p*(1+r*t)*(g-1)/r + fv
	}
	r := guess
	for i := 0; i < 100; i++ {
		y := f(r)
		d := (f(r+1e-6) - y) / 1e-6
		if d == 0 {
			break
		}
		nr := r - y/d
		if math.Abs(nr-r) < 1e-10 {
			return Num(nr)
		}
		r = nr
	}
	if math.Abs(f(r)) < 1e-6 {
		return Num(r)
	}
	return errNum
}

func fnIRR(e *Evaluator, args []Node) Value {
	nums, err := e.numbersOf(args[:1])
	if err.Kind == Error {
		return err
	}
	guess, _ := e.numDefault(args, 1, 0.1)
	npv := func(r float64) float64 {
		s := 0.0
		for i, v := range nums {
			s += v / math.Pow(1+r, float64(i))
		}
		return s
	}
	r := guess
	for i := 0; i < 200; i++ {
		y := npv(r)
		d := (npv(r+1e-7) - y) / 1e-7
		if d == 0 {
			break
		}
		nr := r - y/d
		if math.Abs(nr-r) < 1e-10 {
			return Num(nr)
		}
		r = nr
	}
	if math.Abs(npv(r)) < 1e-6 {
		return Num(r)
	}
	return errNum
}
