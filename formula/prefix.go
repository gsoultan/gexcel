// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import "strings"

// xlfn lists functions that Excel stores with an _xlfn. prefix because they
// were added after the 2007 file format. _xlws marks the ones that are
// also worksheet-only (FILTER, SORT).
var xlfn = map[string]string{}

func init() {
	for _, n := range strings.Fields(`
ACOT ACOTH AGGREGATE ARABIC BASE BETA.DIST BETA.INV BINOM.DIST BINOM.DIST.RANGE BINOM.INV BITAND BITLSHIFT BITOR BITRSHIFT BITXOR
BYCOL BYROW CEILING.MATH CEILING.PRECISE CHISQ.DIST CHISQ.DIST.RT CHISQ.INV CHISQ.INV.RT CHISQ.TEST CHOOSECOLS CHOOSEROWS COMBINA CONCAT
CONFIDENCE.NORM CONFIDENCE.T COT COTH COVARIANCE.P COVARIANCE.S CSC CSCH DAYS DECIMAL DROP ENCODEURL ERF.PRECISE ERFC.PRECISE EXPAND
EXPON.DIST F.DIST F.DIST.RT F.INV F.INV.RT F.TEST FILTERXML FLOOR.MATH FLOOR.PRECISE FORECAST.ETS FORECAST.ETS.CONFINT FORECAST.ETS.SEASONALITY
FORECAST.ETS.STAT FORECAST.LINEAR FORMULATEXT GAMMA GAMMA.DIST GAMMA.INV GAMMALN.PRECISE GAUSS HSTACK HYPGEOM.DIST IFNA IFS IMCOSH IMCOT IMCSC
IMCSCH IMSEC IMSECH IMSINH IMTAN ISFORMULA ISOMITTED ISOWEEKNUM LAMBDA LET LOGNORM.DIST LOGNORM.INV MAKEARRAY MAP MAXIFS MINIFS MODE.MULT
MODE.SNGL MUNIT NEGBINOM.DIST NORM.DIST NORM.INV NORM.S.DIST NORM.S.INV NUMBERVALUE PDURATION PERCENTILE.EXC PERCENTILE.INC PERCENTRANK.EXC
PERCENTRANK.INC PERMUTATIONA PHI POISSON.DIST QUARTILE.EXC QUARTILE.INC QUERYSTRING RANDARRAY RANK.AVG RANK.EQ REDUCE RRI SCAN SEC SECH
SEQUENCE SHEET SHEETS SKEW.P SORTBY STDEV.P STDEV.S SWITCH T.DIST T.DIST.2T T.DIST.RT T.INV T.INV.2T T.TEST TAKE TEXTAFTER TEXTBEFORE
TEXTJOIN TEXTSPLIT TOCOL TOROW UNICHAR UNICODE UNIQUE VAR.P VAR.S VSTACK WEBSERVICE WEIBULL.DIST WRAPCOLS WRAPROWS XLOOKUP XMATCH XOR Z.TEST
ANCHORARRAY SINGLE IMAGE STOCKHISTORY VALUETOTEXT ARRAYTOTEXT GROUPBY PIVOTBY PERCENTOF REGEXTEST REGEXEXTRACT REGEXREPLACE TRIMRANGE`) {
		xlfn[n] = "_xlfn."
	}
	for _, n := range []string{"FILTER", "SORT"} {
		xlfn[n] = "_xlfn._xlws."
	}
}

// AddPrefixes inserts the _xlfn. / _xlws. prefixes Excel expects in stored
// formulas for functions newer than the 2007 format. Already prefixed
// names are left alone.
func AddPrefixes(text string) string {
	if !strings.ContainsAny(text, "(") {
		return text
	}
	toks := Tokenize(text)
	changed := false
	for i, t := range toks {
		if t.Kind != TokFunc {
			continue
		}
		up := strings.ToUpper(t.Text)
		if strings.HasPrefix(up, "_XL") {
			continue
		}
		if p, ok := xlfn[up]; ok {
			toks[i].Text = p + t.Text
			changed = true
		}
	}
	if !changed {
		return text
	}
	return Join(toks)
}

// StripPrefixes removes _xlfn. / _xlws. / _xlpm. prefixes so formulas read
// the way Excel shows them.
func StripPrefixes(text string) string {
	if !strings.Contains(text, "_xl") {
		return text
	}
	toks := Tokenize(text)
	for i, t := range toks {
		if t.Kind != TokFunc && t.Kind != TokName {
			continue
		}
		s := t.Text
		for {
			low := strings.ToLower(s)
			switch {
			case strings.HasPrefix(low, "_xlfn."):
				s = s[6:]
			case strings.HasPrefix(low, "_xlws."):
				s = s[6:]
			case strings.HasPrefix(low, "_xlpm."):
				s = s[6:]
			default:
				toks[i].Text = s
				goto next
			}
		}
	next:
	}
	return Join(toks)
}

// dynamicArrayFuncs return arrays by nature; a formula whose outermost call
// is one of them should be stored as a dynamic array so Excel spills it
// instead of applying implicit intersection.
var dynamicArrayFuncs = map[string]bool{
	"FILTER": true, "SORT": true, "SORTBY": true, "UNIQUE": true, "SEQUENCE": true, "RANDARRAY": true, "TEXTSPLIT": true, "VSTACK": true,
	"HSTACK": true, "TOCOL": true, "TOROW": true, "CHOOSECOLS": true, "CHOOSEROWS": true, "TAKE": true, "DROP": true, "EXPAND": true,
	"WRAPROWS": true, "WRAPCOLS": true, "MAKEARRAY": true, "BYROW": true, "BYCOL": true, "MAP": true, "SCAN": true, "TRANSPOSE": true,
	"GROUPBY": true, "PIVOTBY": true, "TRIMRANGE": true,
}

// IsDynamicArray reports whether a formula's outermost call is a function
// that produces an array (so it should spill).
func IsDynamicArray(text string) bool {
	n, err := Parse(text)
	if err != nil {
		return false
	}
	for {
		switch x := n.(type) {
		case CallNode:
			if dynamicArrayFuncs[x.Name] {
				return true
			}
			if x.Name == "IFERROR" || x.Name == "IFNA" || x.Name == "LET" {
				if len(x.Args) > 0 {
					n = x.Args[len(x.Args)-1]
					if x.Name != "LET" {
						n = x.Args[0]
					}
					continue
				}
			}
			return false
		case UnaryNode:
			if x.Op == "@" {
				return false
			}
			n = x.X
		default:
			return false
		}
	}
}
