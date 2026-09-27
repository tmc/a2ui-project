package a2uistate

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// ResolvePath returns the JSON Pointer that the data binding path
// addresses in scope, the JSON Pointer of the current item of a list
// template, or "" outside templates.
//
// As the A2UI protocol specifies, a path that starts with "/" is
// absolute, and any other path is relative to scope: in the scope
// "/users/0", the path "name" addresses "/users/0/name". The empty path
// addresses scope itself.
func ResolvePath(path, scope string) string {
	switch {
	case strings.HasPrefix(path, "/"):
		return path
	case path == "":
		return scope
	case scope == "" || scope == "/":
		return "/" + path
	default:
		return strings.TrimSuffix(scope, "/") + "/" + path
	}
}

// ResolveValue returns the value of d, resolving data bindings in scope
// as [ResolvePath] does and evaluating calls to the functions of
// [BasicFunctions] only. It reports false if d is unset, is bound to a
// missing value or is a function call that fails, and drops the error.
// Use [Evaluator.ResolveValue] for other functions and for the error.
func (m *DataModel) ResolveValue(d a2ui.DynamicValue, scope string) (any, bool) {
	v, err := (&Evaluator{Data: m}).ResolveValue(d, scope)
	return v, err == nil
}

// ResolveString returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating calls to the functions of
// [BasicFunctions]. A value that is not a string is converted as the
// A2UI protocol specifies: numbers as JavaScript's String does (42,
// 1.5, 1e+21), booleans to "true" or "false", null to "", and objects
// and arrays to JSON.
// ResolveString evaluates calls to the basic functions only. It reports
// false if d is unset, is bound to a missing value or is a function call
// that fails, and drops the error. Use [Evaluator.ResolveString] for
// other functions and for the error.
func (m *DataModel) ResolveString(d a2ui.DynamicString, scope string) (string, bool) {
	s, err := (&Evaluator{Data: m}).ResolveString(d, scope)
	return s, err == nil
}

// ResolveNumber returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating calls to the functions of
// [BasicFunctions] only. It reports false if d is unset, is a function
// call that fails, or is not a number, and drops the error. Use
// [Evaluator.ResolveNumber] for other functions and for the error.
func (m *DataModel) ResolveNumber(d a2ui.DynamicNumber, scope string) (float64, bool) {
	n, err := (&Evaluator{Data: m}).ResolveNumber(d, scope)
	return n, err == nil
}

// ResolveBoolean returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating calls to the functions of
// [BasicFunctions] only. It reports false if d is unset, is a function
// call that fails, or is not a boolean, and drops the error. Use
// [Evaluator.ResolveBoolean] for other functions and for the error.
func (m *DataModel) ResolveBoolean(d a2ui.DynamicBoolean, scope string) (bool, bool) {
	b, err := (&Evaluator{Data: m}).ResolveBoolean(d, scope)
	return b, err == nil
}

// ResolveStringList returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating calls to the functions of
// [BasicFunctions] only. It reports false if d is unset, is a function
// call that fails, or is not an array of strings, and drops the error. Use
// [Evaluator.ResolveStringList] for other functions and for the error.
func (m *DataModel) ResolveStringList(d a2ui.DynamicStringList, scope string) ([]string, bool) {
	list, err := (&Evaluator{Data: m}).ResolveStringList(d, scope)
	return list, err == nil
}

// toString converts a JSON value to a string for display.
func toString(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return formatNumber(v)
	case bool:
		return strconv.FormatBool(v)
	}
	// Like JSON.stringify, do not escape <, > and &.
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}

// formatNumber formats x as JavaScript's String(x) does, which is the
// "standard string representation" the A2UI protocol specifies:
// 42, 1.5, 1e+21, 1e-7.
func formatNumber(x float64) string {
	switch {
	case x == 0:
		return "0" // including -0
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}
	// encoding/json formats numbers as ES6 does.
	data, _ := json.Marshal(x)
	return string(data)
}
