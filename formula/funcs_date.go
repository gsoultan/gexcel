// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package formula

import (
	"math"
	"strings"
	"time"

	"github.com/gsoultan/gexcel/format"
)

func init() {
	register(map[string]Func{
		"DATE":        fnDate,
		"TIME":        fnTime,
		"TODAY":       fnToday,
		"NOW":         fnNow,
		"YEAR":        datePart(func(t time.Time) float64 { return float64(t.Year()) }),
		"MONTH":       datePart(func(t time.Time) float64 { return float64(t.Month()) }),
		"DAY":         datePart(func(t time.Time) float64 { return float64(t.Day()) }),
		"HOUR":        timePart(3600),
		"MINUTE":      timePart(60),
		"SECOND":      timePart(1),
		"WEEKDAY":     fnWeekday,
		"WEEKNUM":     fnWeekNum,
		"ISOWEEKNUM":  fnISOWeekNum,
		"DAYS":        fnDays,
		"DATEDIF":     fnDateDif,
		"EDATE":       fnEDate,
		"EOMONTH":     fnEOMonth,
		"DATEVALUE":   fnDateValue,
		"TIMEVALUE":   fnTimeValue,
		"NETWORKDAYS": fnNetworkDays,
		"WORKDAY":     fnWorkday,
		"YEARFRAC":    fnYearFrac,
		"DAYS360":     fnDays360,
	})
}

// Now is the clock used by TODAY and NOW; tests may replace it.
var Now = time.Now

func (e *Evaluator) serialArg(args []Node, i int) (float64, Value) {
	v := e.Arg(args, i)
	if v.Kind == Text {
		if f, ok := parseDateText(v.Str); ok {
			return f, Value{}
		}
		return 0, errValue
	}
	return toNumber(v)
}

func (e *Evaluator) timeArg(args []Node, i int) (time.Time, Value) {
	s, err := e.serialArg(args, i)
	if err.Kind == Error {
		return time.Time{}, err
	}
	if s < 0 {
		return time.Time{}, errNum
	}
	return format.FromSerial(s, e.date1904()), Value{}
}

func (e *Evaluator) serial(t time.Time) float64 { return format.ToSerial(t, e.date1904()) }

func fnDate(e *Evaluator, args []Node) Value {
	y, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	m, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	d, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	if y < 0 || y >= 10000 {
		return errNum
	}
	if y < 1900 {
		y += 1900
	}
	t := time.Date(int(y), time.Month(int(m)), int(d), 0, 0, 0, 0, time.UTC)
	s := math.Floor(e.serial(t))
	if s < 0 {
		return errNum
	}
	return Num(s)
}

func fnTime(e *Evaluator, args []Node) Value {
	h, err := e.num(args, 0)
	if err.Kind == Error {
		return err
	}
	m, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	s, err := e.num(args, 2)
	if err.Kind == Error {
		return err
	}
	total := math.Trunc(h)*3600 + math.Trunc(m)*60 + math.Trunc(s)
	if total < 0 {
		return errNum
	}
	return Num(math.Mod(total, 86400) / 86400)
}

func fnToday(e *Evaluator, args []Node) Value {
	n := Now()
	return Num(math.Floor(e.serial(time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC))))
}

func fnNow(e *Evaluator, args []Node) Value {
	n := Now()
	return Num(e.serial(time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), 0, time.UTC)))
}

func datePart(f func(time.Time) float64) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
			var s float64
			var err Value
			if v.Kind == Text {
				f2, ok := parseDateText(v.Str)
				if !ok {
					return errValue
				}
				s = f2
			} else {
				s, err = toNumber(v)
				if err.Kind == Error {
					return err
				}
			}
			if s < 0 {
				return errNum
			}
			return Num(f(format.FromSerial(math.Floor(s), e.date1904())))
		})
	}
}

func timePart(unit float64) Func {
	return func(e *Evaluator, args []Node) Value {
		return e.mapScalar(e.Arg(args, 0), func(v Value) Value {
			var s float64
			if v.Kind == Text {
				f, ok := parseDateText(v.Str)
				if !ok {
					return errValue
				}
				s = f
			} else {
				var err Value
				s, err = toNumber(v)
				if err.Kind == Error {
					return err
				}
			}
			if s < 0 {
				return errNum
			}
			secs := math.Round((s - math.Floor(s)) * 86400)
			switch unit {
			case 3600:
				return Num(math.Floor(secs / 3600))
			case 60:
				return Num(math.Floor(math.Mod(secs, 3600) / 60))
			}
			return Num(math.Mod(secs, 60))
		})
	}
}

func fnWeekday(e *Evaluator, args []Node) Value {
	t, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	typ, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	wd := int(t.Weekday()) // Sunday = 0
	switch int(typ) {
	case 1, 17:
		return Num(float64(wd + 1))
	case 2, 11:
		return Num(float64((wd+6)%7 + 1))
	case 3:
		return Num(float64((wd + 6) % 7))
	case 12, 13, 14, 15, 16:
		start := int(typ) - 10 // 2=Tue ... 6=Sat
		return Num(float64((wd-start+7)%7 + 1))
	}
	return errNum
}

func fnWeekNum(e *Evaluator, args []Node) Value {
	t, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	typ, err := e.numDefault(args, 1, 1)
	if err.Kind == Error {
		return err
	}
	if typ == 21 {
		_, w := t.ISOWeek()
		return Num(float64(w))
	}
	jan1 := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	startDay := time.Sunday
	if typ == 2 || typ == 11 {
		startDay = time.Monday
	} else if typ >= 12 && typ <= 17 {
		startDay = time.Weekday((int(typ) - 10) % 7)
	}
	offset := (int(jan1.Weekday()) - int(startDay) + 7) % 7
	days := int(t.Sub(jan1).Hours() / 24)
	return Num(float64((days+offset)/7 + 1))
}

func fnISOWeekNum(e *Evaluator, args []Node) Value {
	t, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	_, w := t.ISOWeek()
	return Num(float64(w))
}

func fnDays(e *Evaluator, args []Node) Value {
	end, err := e.serialArg(args, 0)
	if err.Kind == Error {
		return err
	}
	start, err := e.serialArg(args, 1)
	if err.Kind == Error {
		return err
	}
	return Num(math.Trunc(end) - math.Trunc(start))
}

func fnDateDif(e *Evaluator, args []Node) Value {
	a, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	b, err := e.timeArg(args, 1)
	if err.Kind == Error {
		return err
	}
	unit, err := e.text(args, 2)
	if err.Kind == Error {
		return err
	}
	if b.Before(a) {
		return errNum
	}
	y, m, d := b.Year()-a.Year(), int(b.Month())-int(a.Month()), b.Day()-a.Day()
	if d < 0 {
		m--
		prev := time.Date(b.Year(), b.Month(), 0, 0, 0, 0, 0, time.UTC)
		d += prev.Day()
	}
	if m < 0 {
		y--
		m += 12
	}
	days := int(math.Floor(e.serial(b)) - math.Floor(e.serial(a)))
	switch strings.ToUpper(unit) {
	case "Y":
		return Num(float64(y))
	case "M":
		return Num(float64(y*12 + m))
	case "D":
		return Num(float64(days))
	case "MD":
		return Num(float64(d))
	case "YM":
		return Num(float64(m))
	case "YD":
		anniv := time.Date(b.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
		if anniv.After(b) {
			anniv = anniv.AddDate(-1, 0, 0)
		}
		return Num(math.Floor(b.Sub(anniv).Hours() / 24))
	}
	return errNum
}

func addMonths(t time.Time, n int) time.Time {
	y, m := t.Year(), int(t.Month())+n
	y += (m - 1) / 12
	m = (m-1)%12 + 1
	if m <= 0 {
		m += 12
		y--
	}
	last := time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	d := t.Day()
	if d > last {
		d = last
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func fnEDate(e *Evaluator, args []Node) Value {
	t, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	n, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	return Num(math.Floor(e.serial(addMonths(t, int(n)))))
}

func fnEOMonth(e *Evaluator, args []Node) Value {
	t, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	n, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	t = addMonths(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), int(n))
	t = time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return Num(math.Floor(e.serial(t)))
}

func fnDateValue(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	f, ok := parseDateText(s)
	if !ok {
		return errValue
	}
	return Num(math.Floor(f))
}

func fnTimeValue(e *Evaluator, args []Node) Value {
	s, err := e.text(args, 0)
	if err.Kind == Error {
		return err
	}
	f, ok := parseDateText(s)
	if !ok {
		return errValue
	}
	return Num(f - math.Floor(f))
}

// parseDateText recognises common date and time spellings and returns a
// serial number in the 1900 system.
func parseDateText(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	layouts := []string{
		"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02", "2006/01/02",
		"1/2/2006 15:04:05", "1/2/2006 15:04", "1/2/2006", "01/02/2006", "1/2/06", "2-Jan-2006", "2 Jan 2006", "Jan 2, 2006", "January 2, 2006",
		"2-Jan-06", "Jan-06", "2-Jan", "15:04:05", "15:04", "3:04 PM", "3:04:05 PM", "3 PM",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			if !strings.ContainsAny(l, "26") || l == "2-Jan" {
				// time-only (or day-month without year): current year for 2-Jan, 1899-12-30 for times
				if l == "2-Jan" {
					t = time.Date(Now().Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
					return format.ToSerial(t, false), true
				}
				return float64(t.Hour()*3600+t.Minute()*60+t.Second()) / 86400, true
			}
			return format.ToSerial(t, false), true
		}
	}
	return 0, false
}

func (e *Evaluator) holidays(args []Node, i int) map[int]bool {
	hol := map[int]bool{}
	if i >= len(args) {
		return hol
	}
	each(e.Arg(args, i), func(v Value) bool {
		if v.Kind == Number {
			hol[int(v.Num)] = true
		} else if v.Kind == Text {
			if f, ok := parseDateText(v.Str); ok {
				hol[int(f)] = true
			}
		}
		return true
	})
	return hol
}

func fnNetworkDays(e *Evaluator, args []Node) Value {
	start, err := e.serialArg(args, 0)
	if err.Kind == Error {
		return err
	}
	end, err := e.serialArg(args, 1)
	if err.Kind == Error {
		return err
	}
	hol := e.holidays(args, 2)
	a, b := int(start), int(end)
	signv := 1
	if b < a {
		a, b = b, a
		signv = -1
	}
	n := 0
	for d := a; d <= b; d++ {
		wd := format.FromSerial(float64(d), e.date1904()).Weekday()
		if wd == time.Saturday || wd == time.Sunday || hol[d] {
			continue
		}
		n++
	}
	return Num(float64(n * signv))
}

func fnWorkday(e *Evaluator, args []Node) Value {
	start, err := e.serialArg(args, 0)
	if err.Kind == Error {
		return err
	}
	days, err := e.num(args, 1)
	if err.Kind == Error {
		return err
	}
	hol := e.holidays(args, 2)
	d := int(start)
	step := 1
	if days < 0 {
		step = -1
		days = -days
	}
	for n := 0; n < int(days); {
		d += step
		wd := format.FromSerial(float64(d), e.date1904()).Weekday()
		if wd == time.Saturday || wd == time.Sunday || hol[d] {
			continue
		}
		n++
	}
	return Num(float64(d))
}

func fnYearFrac(e *Evaluator, args []Node) Value {
	a, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	b, err := e.timeArg(args, 1)
	if err.Kind == Error {
		return err
	}
	basis, _ := e.numDefault(args, 2, 0)
	if b.Before(a) {
		a, b = b, a
	}
	switch int(basis) {
	case 1:
		days := b.Sub(a).Hours() / 24
		// average year length over the span
		y1, y2 := a.Year(), b.Year()
		total := 0.0
		for y := y1; y <= y2; y++ {
			if isLeap(y) {
				total += 366
			} else {
				total += 365
			}
		}
		return Num(days / (total / float64(y2-y1+1)))
	case 2:
		return Num(b.Sub(a).Hours() / 24 / 360)
	case 3:
		return Num(b.Sub(a).Hours() / 24 / 365)
	case 4:
		return Num(days360(a, b, true) / 360)
	}
	return Num(days360(a, b, false) / 360)
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func days360(a, b time.Time, european bool) float64 {
	d1, d2 := a.Day(), b.Day()
	m1, m2 := int(a.Month()), int(b.Month())
	y1, y2 := a.Year(), b.Year()
	if european {
		if d1 == 31 {
			d1 = 30
		}
		if d2 == 31 {
			d2 = 30
		}
	} else {
		lastFeb := func(t time.Time) bool {
			return t.Month() == 2 && t.AddDate(0, 0, 1).Month() == 3
		}
		if lastFeb(a) && lastFeb(b) {
			d2 = 30
		}
		if lastFeb(a) {
			d1 = 30
		}
		if d2 == 31 && d1 >= 30 {
			d2 = 30
		}
		if d1 == 31 {
			d1 = 30
		}
	}
	return float64((y2-y1)*360 + (m2-m1)*30 + (d2 - d1))
}

func fnDays360(e *Evaluator, args []Node) Value {
	a, err := e.timeArg(args, 0)
	if err.Kind == Error {
		return err
	}
	b, err := e.timeArg(args, 1)
	if err.Kind == Error {
		return err
	}
	eu := false
	if len(args) > 2 {
		eu, _ = e.boolean(args, 2)
	}
	return Num(days360(a, b, eu))
}
