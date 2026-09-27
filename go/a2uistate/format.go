package a2uistate

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// formatDecimal formats x as Intl.NumberFormat does for the en-US
// locale, with between minFrac and maxFrac fraction digits, rounding
// half away from zero, and with or without grouping separators.
// Like Intl.NumberFormat, it rounds the shortest decimal representation
// of x, so that 1.005 rounds to 1.01, and it keeps the sign of a
// negative number that rounds to zero.
func formatDecimal(x float64, minFrac, maxFrac int, grouping bool) string {
	var b strings.Builder
	if math.Signbit(x) {
		b.WriteByte('-')
	}
	x = math.Abs(x)
	if math.IsInf(x, 0) {
		b.WriteString("∞")
		return b.String()
	}
	intPart, frac := roundDecimal(x, maxFrac)
	for len(frac) > minFrac && frac[len(frac)-1] == '0' {
		frac = frac[:len(frac)-1]
	}
	if len(frac) < minFrac {
		frac += strings.Repeat("0", minFrac-len(frac))
	}
	for i, c := range intPart {
		if grouping && i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if frac != "" {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return b.String()
}

// roundDecimal returns the integer and fraction digits of x, which is
// finite and not negative, rounded half away from zero to at most
// maxFrac fraction digits.
func roundDecimal(x float64, maxFrac int) (intPart, frac string) {
	// The shortest representation: d.ddde±XX.
	s := strconv.FormatFloat(x, 'e', -1, 64)
	mant, exp, _ := strings.Cut(s, "e")
	mant = strings.Replace(mant, ".", "", 1)
	e, _ := strconv.Atoi(exp)
	switch point := e + 1; {
	case point <= 0:
		intPart, frac = "0", strings.Repeat("0", -point)+mant
	case point >= len(mant):
		intPart = mant + strings.Repeat("0", point-len(mant))
	default:
		intPart, frac = mant[:point], mant[point:]
	}
	if len(frac) <= maxFrac {
		return intPart, frac
	}
	up := frac[maxFrac] >= '5'
	frac = frac[:maxFrac]
	if !up {
		return intPart, frac
	}
	digits := []byte(intPart + frac)
	i := len(digits) - 1
	for ; i >= 0 && digits[i] == '9'; i-- {
		digits[i] = '0'
	}
	if i < 0 {
		digits = append([]byte{'1'}, digits...)
	} else {
		digits[i]++
	}
	n := len(digits) - len(frac)
	return string(digits[:n]), string(digits[n:])
}

// currencySymbols maps ISO 4217 codes to their en-US symbols in CLDR,
// as Node.js 24 formats them, for the currencies whose symbol is not
// their code.
var currencySymbols = map[string]string{
	"AUD": "A$",
	"BRL": "R$",
	"CAD": "CA$",
	"CNY": "CN¥",
	"EUR": "€",
	"GBP": "£",
	"HKD": "HK$",
	"ILS": "₪",
	"INR": "₹",
	"JPY": "¥",
	"KRW": "₩",
	"MXN": "MX$",
	"NZD": "NZ$",
	"PHP": "₱",
	"TWD": "NT$",
	"USD": "$",
	"VND": "₫",
	"XAF": "FCFA",
	"XCD": "EC$",
	"XCG": "Cg.",
	"XOF": "F\u202fCFA",
	"XPF": "CFPF",
}

// formatMoney formats x in the currency code as Intl.NumberFormat does
// for the en-US locale: the sign, the currency symbol, a no-break space
// if the symbol does not end in a symbol character (as in "CHF 1.00"),
// and the number.
func formatMoney(x float64, code string, minFrac, maxFrac int, grouping bool) string {
	sym, ok := currencySymbols[code]
	if !ok {
		sym = code
	}
	if r, _ := utf8.DecodeLastRuneInString(sym); sym == "" || !unicode.IsSymbol(r) {
		sym += "\u00a0"
	}
	num := formatDecimal(x, minFrac, maxFrac, grouping)
	if rest, ok := strings.CutPrefix(num, "-"); ok {
		return "-" + sym + rest
	}
	return sym + num
}

// timestampLayouts are the ISO 8601 forms that formatDate accepts,
// after an offset of Z is added to a timestamp that has none.
var timestampLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05Z0700",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15:04Z0700",
	"2006-01-02Z07:00",
}

// hasOffset matches a timestamp that ends in a UTC offset.
var hasOffset = regexp.MustCompile(`(?:Z|[+-]\d{2}:?\d{2})$`)

// parseTimestamp parses an ISO 8601 timestamp. A timestamp without an
// offset is in UTC. The result is in the offset of the timestamp.
func parseTimestamp(s string) (time.Time, bool) {
	if !hasOffset.MatchString(s) {
		s += "Z"
	}
	if len(s) > 10 && s[10] == ' ' {
		s = s[:10] + "T" + s[11:]
	}
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatTimestamp formats the ISO 8601 timestamp s as the formatDate
// function does: with the wall clock time of its own offset, in the
// Unicode TR35 pattern, with en-US names. The pattern "ISO" formats the
// time in UTC as JavaScript's Date.toISOString does, and an empty
// pattern means "yyyy-MM-dd". It returns "" if s is not a timestamp.
//
// The pattern is read in runs of the same letter, as TR35 specifies:
//
//	yy          two-digit year; yyy or longer, the full year
//	M, MM       month number, unpadded or padded to two digits
//	MMM, MMMM   short and long month name (MMMM or longer)
//	E to EEE    short weekday name; EEEE or longer, the long name
//	d, dd       day of the month (dd or longer is padded)
//	H, HH       hour 0-23
//	h, hh       hour 1-12
//	mm, ss      minute and second, padded
//	a           AM or PM
//
// Any other run, such as YYYY, y, m or s, is copied as it is. Like
// web_core, the pattern has no quoting. Unlike web_core, a run is one
// field, so EEE is "Mon", not "MonMonMon".
func formatTimestamp(s, pattern string) string {
	t, ok := parseTimestamp(s)
	if !ok {
		return ""
	}
	switch pattern {
	case "":
		pattern = "yyyy-MM-dd"
	case "ISO":
		return t.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	num := func(n, width int) string {
		s := strconv.Itoa(n)
		if len(s) < width {
			s = strings.Repeat("0", width-len(s)) + s
		}
		return s
	}
	hour12 := t.Hour() % 12
	if hour12 == 0 {
		hour12 = 12
	}
	var b strings.Builder
	for i := 0; i < len(pattern); {
		c := pattern[i]
		n := 1
		for i+n < len(pattern) && pattern[i+n] == c {
			n++
		}
		run := pattern[i : i+n]
		i += n
		switch {
		case c == 'y' && n == 2:
			y := strconv.Itoa(t.Year())
			b.WriteString(y[max(len(y)-2, 0):])
		case c == 'y' && n > 2:
			b.WriteString(strconv.Itoa(t.Year()))
		case c == 'M' && n <= 2:
			b.WriteString(num(int(t.Month()), n))
		case c == 'M' && n == 3:
			b.WriteString(t.Month().String()[:3])
		case c == 'M':
			b.WriteString(t.Month().String())
		case c == 'E' && n <= 3:
			b.WriteString(t.Weekday().String()[:3])
		case c == 'E':
			b.WriteString(t.Weekday().String())
		case c == 'd':
			b.WriteString(num(t.Day(), min(n, 2)))
		case c == 'H':
			b.WriteString(num(t.Hour(), min(n, 2)))
		case c == 'h':
			b.WriteString(num(hour12, min(n, 2)))
		case c == 'm' && n >= 2:
			b.WriteString(num(t.Minute(), 2))
		case c == 's' && n >= 2:
			b.WriteString(num(t.Second(), 2))
		case c == 'a' && t.Hour() < 12:
			b.WriteString("AM")
		case c == 'a':
			b.WriteString("PM")
		default:
			b.WriteString(run)
		}
	}
	return b.String()
}
