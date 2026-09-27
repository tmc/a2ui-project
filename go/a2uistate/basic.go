package a2uistate

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// basicFuncs holds the functions of BasicFunctions. It is set by init,
// since formatString refers to it through Evaluator.funcs.
var basicFuncs map[string]Func

func init() {
	basicFuncs = map[string]Func{
		"@index":         index,
		"required":       required,
		"regex":          regexCheck,
		"length":         length,
		"numeric":        numeric,
		"email":          email,
		"formatString":   formatString,
		"formatNumber":   formatNumberFunc,
		"formatCurrency": formatCurrency,
		"formatDate":     formatDate,
		"pluralize":      pluralize,
		"openUrl":        openURL,
		"and":            and,
		"or":             or,
		"not":            not,
	}
}

// BasicFunctions returns the functions of the A2UI basic catalog, plus
// the @index system function, in a new map that the caller may extend
// with the functions of other catalogs.
//
// The functions behave as those of the web_core renderer in the A2UI
// repository, with the en-US locale, except as follows.
//
//   - Formatting uses the en-US locale only. The currency symbols are
//     those of CLDR as shipped with Node.js 24, for the 22 currencies
//     whose en-US symbol is not their ISO code.
//   - formatDate accepts ISO 8601 timestamps only (such as 2026-01-16,
//     2026-01-16T14:30Z or 2026-01-16 14:30:05.5+05:30) and yields ""
//     for any other value, including impossible dates. It reads its
//     pattern in runs of the same letter, as Unicode TR35 does, so EEE
//     is the short weekday name once (web_core repeats it) and MMMMM is
//     the long month name. Only the letters y, M, E, d, H, h, m, s and
//     a are fields; y, m and s alone, and any other letters, such as
//     YYYY, are copied as they are.
//   - regex uses Go regular expressions (package [regexp]), not
//     JavaScript ones: lookaround and backreferences are invalid.
//   - required, regex, length and email check a missing or null value
//     as empty instead of failing. numeric fails such a value, and it
//     accepts numeric strings, as the Python renderer does.
//   - The validation functions return {"valid": false} with no message
//     when they fail, so that the check's message is shown. (web_core
//     returns a default message, which takes precedence.)
//   - and, or and not take booleans, null (false) and validation result
//     objects, which count as their valid field. (web_core rejects
//     validation results, so that and cannot combine checks.)
//   - formatString accepts @ in function names, as in ${@index()},
//     and returns an error if an interpolated expression fails, instead
//     of interpolating "".
//   - openUrl returns an error wrapping [ErrAction]; see
//     [Evaluator.ResolveArgs].
//   - Unknown arguments are ignored.
func BasicFunctions() map[string]Func {
	return maps.Clone(basicFuncs)
}

// index implements @index: the index of the current item of a list
// template, the last element of the scope, plus offset.
func index(_ *Evaluator, scope string, args map[string]any) (any, error) {
	offset, _, err := numberArg("@index", args, "offset")
	if err != nil {
		return nil, err
	}
	if math.IsInf(offset, 0) {
		offset = 0
	}
	i := strings.LastIndexByte(scope, '/')
	n, ok := arrayIndex(scope[i+1:])
	if i < 0 || !ok {
		return nil, fmt.Errorf("%w: @index outside a list template", ErrNoValue)
	}
	return float64(n) + offset, nil
}

// valid returns a validation result object.
func valid(ok bool) map[string]any {
	return map[string]any{"valid": ok}
}

func required(_ *Evaluator, _ string, args map[string]any) (any, error) {
	switch v := args["value"].(type) {
	case nil:
		return valid(false), nil
	case string:
		return valid(v != ""), nil
	case []any:
		return valid(len(v) > 0), nil
	}
	return valid(true), nil
}

func regexCheck(_ *Evaluator, _ string, args map[string]any) (any, error) {
	pattern, err := requiredString("regex", args, "pattern")
	if err != nil {
		return nil, err
	}
	s, _, err := stringArg("regex", args, "value")
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: regex: %v", ErrInvalidArgs, err)
	}
	return valid(re.MatchString(s)), nil
}

func length(_ *Evaluator, _ string, args map[string]any) (any, error) {
	s, _, err := stringArg("length", args, "value")
	if err != nil {
		return nil, err
	}
	n := float64(utf16Len(s))
	for _, key := range []string{"min", "max"} {
		limit, ok, err := numberArg("length", args, key)
		if err != nil {
			return nil, err
		}
		if ok && limit != math.Trunc(limit) {
			return nil, argError("length", "%s is not an integer", key)
		}
		if ok && (key == "min" && n < limit || key == "max" && n > limit) {
			return valid(false), nil
		}
	}
	return valid(true), nil
}

func numeric(_ *Evaluator, _ string, args map[string]any) (any, error) {
	var n float64
	switch v := args["value"].(type) {
	case nil:
		return valid(false), nil
	case float64:
		n = v
	case string:
		var err error
		if n, err = strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
			return valid(false), nil
		}
	default:
		return nil, argError("numeric", "value is %s, not a number", jsonType(v))
	}
	for _, key := range []string{"min", "max"} {
		limit, ok, err := numberArg("numeric", args, key)
		if err != nil {
			return nil, err
		}
		if ok && (key == "min" && n < limit || key == "max" && n > limit) {
			return valid(false), nil
		}
	}
	return valid(true), nil
}

// emailPattern is the email address syntax of the web renderers.
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func email(_ *Evaluator, _ string, args map[string]any) (any, error) {
	s, _, err := stringArg("email", args, "value")
	if err != nil {
		return nil, err
	}
	return valid(emailPattern.MatchString(s)), nil
}

func formatString(e *Evaluator, scope string, args map[string]any) (any, error) {
	tmpl, err := requiredString("formatString", args, "value")
	if err != nil {
		return nil, err
	}
	parts, err := parseTemplate(tmpl, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: formatString: %v", ErrInvalidArgs, err)
	}
	var b strings.Builder
	for _, p := range parts {
		v, _, err := e.eval(p, scope, 1)
		if err != nil {
			return nil, err
		}
		b.WriteString(toString(v))
	}
	return b.String(), nil
}

func formatNumberFunc(_ *Evaluator, _ string, args map[string]any) (any, error) {
	x, err := requiredNumber("formatNumber", args, "value")
	if err != nil {
		return nil, err
	}
	minFrac, maxFrac, grouping, err := numberOptions("formatNumber", args, 0, 3)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(x) {
		return "", nil
	}
	return formatDecimal(x, minFrac, maxFrac, grouping), nil
}

func formatCurrency(_ *Evaluator, _ string, args map[string]any) (any, error) {
	x, err := requiredNumber("formatCurrency", args, "value")
	if err != nil {
		return nil, err
	}
	code, err := requiredString("formatCurrency", args, "currency")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, argError("formatCurrency", "currency is empty")
	}
	minFrac, maxFrac, grouping, err := numberOptions("formatCurrency", args, 2, 2)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(x) {
		return "", nil
	}
	return formatMoney(x, strings.ToUpper(code), minFrac, maxFrac, grouping), nil
}

// numberOptions returns the fraction digits and grouping that the
// decimals and grouping arguments of fn ask for, with the fraction
// digits minFrac and maxFrac if decimals is absent.
func numberOptions(fn string, args map[string]any, minFrac, maxFrac int) (int, int, bool, error) {
	d, ok, err := numberArg(fn, args, "decimals")
	if err != nil {
		return 0, 0, false, err
	}
	if ok {
		// Intl.NumberFormat truncates fraction digits and accepts 0 to 100.
		if !(d >= 0 && d < 101) {
			return 0, 0, false, argError(fn, "decimals %v is out of range [0, 100]", d)
		}
		minFrac, maxFrac = int(d), int(d)
	}
	grouping, ok, err := boolArg(fn, args, "grouping")
	if err != nil {
		return 0, 0, false, err
	}
	return minFrac, maxFrac, grouping || !ok, nil
}

func formatDate(_ *Evaluator, _ string, args map[string]any) (any, error) {
	pattern, err := requiredString("formatDate", args, "format")
	if err != nil {
		return nil, err
	}
	s, ok := args["value"].(string)
	if !ok {
		return "", nil
	}
	return formatTimestamp(s, pattern), nil
}

func pluralize(_ *Evaluator, _ string, args map[string]any) (any, error) {
	n, err := requiredNumber("pluralize", args, "value")
	if err != nil {
		return nil, err
	}
	forms := make(map[string]string)
	for _, key := range []string{"zero", "one", "two", "few", "many", "other"} {
		s, ok, err := stringArg("pluralize", args, key)
		if err != nil {
			return nil, err
		}
		if ok {
			forms[key] = s
		}
	}
	if _, ok := forms["other"]; !ok {
		return nil, argError("pluralize", "missing other")
	}
	category := "other"
	if _, ok := forms["zero"]; ok && n == 0 {
		category = "zero"
	} else if _, ok := forms["one"]; ok && n == 1 {
		category = "one"
	} else if _, ok := forms["two"]; ok && n == 2 {
		category = "two"
	} else if math.Abs(n) == 1 {
		// The CLDR plural rule for English: "one" for the integer 1.
		category = "one"
	}
	if s, ok := forms[category]; ok {
		return s, nil
	}
	return forms["other"], nil
}

func openURL(_ *Evaluator, _ string, _ map[string]any) (any, error) {
	return nil, fmt.Errorf("%w: openUrl", ErrAction)
}

func and(_ *Evaluator, _ string, args map[string]any) (any, error) {
	values, err := boolValues("and", args)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		if !v {
			return false, nil
		}
	}
	return true, nil
}

func or(_ *Evaluator, _ string, args map[string]any) (any, error) {
	values, err := boolValues("or", args)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		if v {
			return true, nil
		}
	}
	return false, nil
}

func not(_ *Evaluator, _ string, args map[string]any) (any, error) {
	b, err := toBool("not", "value", args["value"])
	if err != nil {
		return nil, err
	}
	return !b, nil
}

// boolValues returns the values argument of the logical function fn.
func boolValues(fn string, args map[string]any) ([]bool, error) {
	list, ok := args["values"].([]any)
	if !ok {
		return nil, argError(fn, "values is %s, not an array", jsonType(args["values"]))
	}
	if len(list) < 2 {
		return nil, argError(fn, "values has %d elements, fewer than 2", len(list))
	}
	out := make([]bool, len(list))
	for i, v := range list {
		b, err := toBool(fn, fmt.Sprintf("values[%d]", i), v)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

// toBool returns the argument v of fn, named name, as a boolean.
func toBool(fn, name string, v any) (bool, error) {
	switch v := v.(type) {
	case nil:
		return false, nil
	case bool:
		return v, nil
	case map[string]any:
		if _, ok := v["valid"]; ok {
			return truthy(v), nil
		}
	}
	return false, argError(fn, "%s is %s, not a boolean", name, jsonType(v))
}

// argError returns an error wrapping ErrInvalidArgs for a call to fn.
func argError(fn, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidArgs, fn, fmt.Sprintf(format, args...))
}

// stringArg returns the string argument key of fn, reporting whether
// it is present and not null.
func stringArg(fn string, args map[string]any, key string) (string, bool, error) {
	v := args[key]
	if v == nil {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", false, argError(fn, "%s is %s, not a string", key, jsonType(v))
	}
	return s, true, nil
}

// numberArg returns the number argument key of fn, reporting whether
// it is present and not null.
func numberArg(fn string, args map[string]any, key string) (float64, bool, error) {
	v := args[key]
	if v == nil {
		return 0, false, nil
	}
	n, ok := v.(float64)
	if !ok {
		return 0, false, argError(fn, "%s is %s, not a number", key, jsonType(v))
	}
	return n, true, nil
}

// boolArg returns the boolean argument key of fn, reporting whether it
// is present and not null.
func boolArg(fn string, args map[string]any, key string) (bool, bool, error) {
	v := args[key]
	if v == nil {
		return false, false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, false, argError(fn, "%s is %s, not a boolean", key, jsonType(v))
	}
	return b, true, nil
}

// requiredString returns the string argument key of fn, which must be
// present.
func requiredString(fn string, args map[string]any, key string) (string, error) {
	s, ok, err := stringArg(fn, args, key)
	if err == nil && !ok {
		err = argError(fn, "missing %s", key)
	}
	return s, err
}

// requiredNumber returns the number argument key of fn, which must be
// present.
func requiredNumber(fn string, args map[string]any, key string) (float64, error) {
	n, ok, err := numberArg(fn, args, key)
	if err == nil && !ok {
		err = argError(fn, "missing %s", key)
	}
	return n, err
}

// utf16Len returns the length of s in UTF-16 code units, as
// JavaScript's String.length does.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}
