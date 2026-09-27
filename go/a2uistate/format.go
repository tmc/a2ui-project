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

// dateTokens are the pattern letters of formatDate, as in web_core.
var dateTokens = regexp.MustCompile(`yyyy|yy|MMMM|MMM|MM|M|EEEE|E|dd|d|HH|H|hh|h|mm|ss|a`)

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
	pad := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	hour12 := t.Hour() % 12
	if hour12 == 0 {
		hour12 = 12
	}
	return dateTokens.ReplaceAllStringFunc(pattern, func(tok string) string {
		switch tok {
		case "yyyy":
			return strconv.Itoa(t.Year())
		case "yy":
			y := strconv.Itoa(t.Year())
			return y[max(len(y)-2, 0):]
		case "MMMM":
			return t.Month().String()
		case "MMM":
			return t.Month().String()[:3]
		case "MM":
			return pad(int(t.Month()))
		case "M":
			return strconv.Itoa(int(t.Month()))
		case "EEEE":
			return t.Weekday().String()
		case "E":
			return t.Weekday().String()[:3]
		case "dd":
			return pad(t.Day())
		case "d":
			return strconv.Itoa(t.Day())
		case "HH":
			return pad(t.Hour())
		case "H":
			return strconv.Itoa(t.Hour())
		case "hh":
			return pad(hour12)
		case "h":
			return strconv.Itoa(hour12)
		case "mm":
			return pad(t.Minute())
		case "ss":
			return pad(t.Second())
		case "a":
			if t.Hour() < 12 {
				return "AM"
			}
			return "PM"
		}
		return tok
	})
}
