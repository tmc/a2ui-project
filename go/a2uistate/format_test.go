package a2uistate

import (
	"math"
	"testing"
)

// The wanted results in this file are those of Intl.NumberFormat and
// the formatDate function of web_core, run in Node.js 24 with the
// en-US locale.

func TestFormatDecimal(t *testing.T) {
	tests := []struct {
		x                float64
		minFrac, maxFrac int
		grouping         bool
		want             string
	}{
		{0, 0, 3, true, "0"},
		{math.Copysign(0, -1), 0, 3, true, "-0"},
		{1234.5678, 0, 3, true, "1,234.568"},
		{1.0005, 0, 3, true, "1.001"},
		{1.005, 2, 2, true, "1.01"},
		{2.5, 0, 0, true, "3"},
		{-2.5, 0, 0, true, "-3"},
		{-0.0001, 0, 3, true, "-0"},
		{0.9995, 0, 3, true, "1"},
		{0.9995, 2, 2, true, "1.00"},
		{999.9995, 0, 3, true, "1,000"},
		{1e21, 0, 3, true, "1,000,000,000,000,000,000,000"},
		{1.5e-7, 0, 3, true, "0"},
		{123456789.123, 0, 3, true, "123,456,789.123"},
		{1234567.891, 0, 3, false, "1234567.891"},
		{0.1, 20, 20, true, "0.10000000000000000000"},
		{42, 2, 2, true, "42.00"},
		{-1234.5, 1, 1, true, "-1,234.5"},
		{1e-7, 10, 10, true, "0.0000001000"},
		{5e-324, 3, 3, true, "0.000"},
		{0.125, 2, 2, true, "0.13"},
		{0.375, 2, 2, true, "0.38"},
		{100, 0, 0, false, "100"},
		{math.Inf(1), 0, 3, true, "∞"},
		{math.Inf(-1), 0, 3, true, "-∞"},
		{math.MaxFloat64, 0, 0, false, "179769313486231570" + zeros(291)},
	}
	for _, tt := range tests {
		if got := formatDecimal(tt.x, tt.minFrac, tt.maxFrac, tt.grouping); got != tt.want {
			t.Errorf("formatDecimal(%v, %d, %d, %v) = %q, want %q", tt.x, tt.minFrac, tt.maxFrac, tt.grouping, got, tt.want)
		}
	}
}

func zeros(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
	}
	return string(b)
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		x        float64
		code     string
		frac     int
		grouping bool
		want     string
	}{
		{1234.5, "USD", 2, true, "$1,234.50"},
		{-1234.5, "USD", 2, true, "-$1,234.50"},
		{1, "CHF", 2, true, "CHF\u00a01.00"},
		{-1, "CHF", 2, true, "-CHF\u00a01.00"},
		{1234.5, "EUR", 0, true, "€1,235"},
		{1234.5, "JPY", 2, true, "¥1,234.50"},
		{1, "XOF", 2, true, "F\u202fCFA\u00a01.00"},
		{1, "XCG", 2, true, "Cg.\u00a01.00"},
		{1, "ZZZ", 2, true, "ZZZ\u00a01.00"},
		{1, "US", 2, true, "US\u00a01.00"},
		{1, "", 2, true, "\u00a01.00"},
		{1234567.891, "GBP", 3, false, "£1234567.891"},
		{math.Copysign(0, -1), "USD", 2, true, "-$0.00"},
		{0.005, "USD", 2, true, "$0.01"},
	}
	for _, tt := range tests {
		if got := formatMoney(tt.x, tt.code, tt.frac, tt.frac, tt.grouping); got != tt.want {
			t.Errorf("formatMoney(%v, %q, %d, %v) = %q, want %q", tt.x, tt.code, tt.frac, tt.grouping, got, tt.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		s, pattern, want string
	}{
		{"2026-01-16T14:30:05Z", "MMM dd, yyyy", "Jan 16, 2026"},
		{"2026-01-16T14:30:05Z", "HH:mm", "14:30"},
		{"2026-01-16T14:30:05Z", "h:mm a", "2:30 PM"},
		{"2026-01-16T14:30:05Z", "EEEE, d MMMM", "Friday, 16 January"},
		{"2026-01-16T00:05:00Z", "hh:mm a", "12:05 AM"},
		{"2026-01-16T12:00:00Z", "h a", "12 PM"},
		{"2026-01-16", "yyyy-MM-dd", "2026-01-16"},
		{"2026-01-16T14:30", "yy/M/d H:mm:ss", "26/1/16 14:30:00"},
		{"2026-01-16T14:30:05.5+05:30", "ISO", "2026-01-16T09:00:05.500Z"},
		{"2026-01-16T14:30:05.5+05:30", "HH:mm", "14:30"},
		{"2026-01-16T23:30:00-0800", "E yyyy-MM-dd HH:mm", "Fri 2026-01-16 23:30"},
		{"2026-01-16 14:30:05Z", "dd.MM.yyyy", "16.01.2026"},
		{"2026-02-02T15:17:00Z", "E MMM d, YYYY h:mm a", "Mon Feb 2, YYYY 3:17 PM"},
		{"2026-01-16T14:30:05Z", "", "2026-01-16"},
		{"not a date", "yyyy", ""},
		{"", "yyyy", ""},
		{"0999-01-01T00:00:00Z", "yyyy yy", "999 99"},
		// Divergences: web_core gives "MonMonMon Sep 7" and "2026-January1-16".
		{"2026-09-07T08:09:03Z", "EEE MMM d", "Mon Sep 7"},
		{"2026-01-16T14:30:05Z", "yyyy-MMMMM-dd", "2026-January-16"},
		// Divergences: V8 accepts these.
		{"2026-02-30T00:00Z", "yyyy-MM-dd", ""},
		{"Jan 16 2026", "yyyy-MM-dd", ""},
	}
	for _, tt := range tests {
		if got := formatTimestamp(tt.s, tt.pattern); got != tt.want {
			t.Errorf("formatTimestamp(%q, %q) = %q, want %q", tt.s, tt.pattern, got, tt.want)
		}
	}
}

func TestFormatTimestampRuns(t *testing.T) {
	const ts = "2026-09-07T08:09:03Z" // a Monday
	tests := []struct {
		pattern, want string
	}{
		{"y", "y"},
		{"yy", "26"},
		{"yyy", "2026"},
		{"yyyy", "2026"},
		{"yyyyy", "2026"},
		{"YYYY", "YYYY"},
		{"M", "9"},
		{"MM", "09"},
		{"MMM", "Sep"},
		{"MMMM", "September"},
		{"E", "Mon"},
		{"EE", "Mon"},
		{"EEE", "Mon"},
		{"EEEE", "Monday"},
		{"EEEEE", "Monday"},
		{"d", "7"},
		{"dd", "07"},
		{"ddd", "07"},
		{"H", "8"},
		{"HH", "08"},
		{"h", "8"},
		{"hh", "08"},
		{"m", "m"},
		{"mm", "09"},
		{"mmm", "09"},
		{"s", "s"},
		{"ss", "03"},
		{"a", "AM"},
		{"aa", "AM"},
		{"D Q G", "D Q G"},
		{"h:mm a, EEEE", "8:09 AM, Monday"},
		{"dd.MM.yy — é", "07.09.26 — é"},
	}
	for _, tt := range tests {
		if got := formatTimestamp(ts, tt.pattern); got != tt.want {
			t.Errorf("formatTimestamp(%q, %q) = %q, want %q", ts, tt.pattern, got, tt.want)
		}
	}
}
