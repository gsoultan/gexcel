// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package format renders cell values with Excel number format codes and
// converts between time.Time and Excel's serial date numbers.
package format

import (
	"math"
	"time"
)

// Excel stores dates as days since an epoch, with the time of day as the
// fraction. The 1900 date system has the Lotus 1-2-3 bug: it believes 1900
// was a leap year, so serial 60 is the non-existent 29 February 1900 and
// serials below 61 are one day off from the proleptic calendar.
var (
	epoch1900 = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
	epoch1904 = time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)
)

// ToSerial converts a time to an Excel serial number. The time's wall clock
// is used as if it were UTC (Excel has no time zones).
func ToSerial(t time.Time, date1904 bool) float64 {
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	epoch := epoch1900
	if date1904 {
		epoch = epoch1904
	}
	days := t.Sub(epoch).Hours() / 24
	if !date1904 && t.Before(time.Date(1900, time.March, 1, 0, 0, 0, 0, time.UTC)) {
		days-- // skip the phantom 29 Feb 1900
	}
	// Round to the nearest millisecond: Excel keeps 15 significant digits.
	return math.Round(days*86400000) / 86400000
}

// FromSerial converts an Excel serial number to a time in UTC.
func FromSerial(serial float64, date1904 bool) time.Time {
	epoch := epoch1900
	if date1904 {
		epoch = epoch1904
	}
	if !date1904 && serial < 61 {
		if serial >= 60 {
			serial = 59 // 29 Feb 1900 shows as 28 Feb (Excel shows "2/29/1900"; we pick a real date)
		} else {
			serial++
		}
	}
	days := math.Floor(serial)
	frac := serial - days
	ns := math.Round(frac * 86400 * 1e3) // milliseconds
	return epoch.AddDate(0, 0, int(days)).Add(time.Duration(ns) * time.Millisecond)
}

// DurationToSerial converts a duration to days, for elapsed-time formats.
func DurationToSerial(d time.Duration) float64 { return d.Hours() / 24 }

// SerialToDuration converts days to a duration.
func SerialToDuration(serial float64) time.Duration {
	return time.Duration(math.Round(serial * 24 * float64(time.Hour)))
}
