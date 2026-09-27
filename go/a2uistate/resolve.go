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

// ResolveValue returns the value of d, resolving a data binding in scope
// as [ResolvePath] does. It reports false if d is unset, is a function
// call or is bound to a missing value.
func (m *DataModel) ResolveValue(d a2ui.DynamicValue, scope string) (any, bool) {
	switch {
	case d.String != nil:
		return *d.String, true
	case d.Number != nil:
		return *d.Number, true
	case d.Bool != nil:
		return *d.Bool, true
	case d.Array != nil:
		return d.Array, true
	case d.Binding != nil:
		return m.Get(ResolvePath(d.Binding.Path, scope))
	}
	return nil, false
}

// ResolveString returns the value of d, resolving a data binding in
// scope as [ResolvePath] does. A bound value that is not a string is
// converted as the A2UI protocol specifies: numbers as JavaScript's
// String does (42, 1.5, 1e+21), booleans to "true" or "false", null to
// "", and objects and arrays to JSON.
// ResolveString reports false if d is unset, is a function call or is
// bound to a missing value.
func (m *DataModel) ResolveString(d a2ui.DynamicString, scope string) (string, bool) {
	switch {
	case d.Literal != nil:
		return *d.Literal, true
	case d.Binding != nil:
		v, ok := m.Get(ResolvePath(d.Binding.Path, scope))
		if !ok {
			return "", false
		}
		return toString(v), true
	}
	return "", false
}

// ResolveNumber returns the value of d, resolving a data binding in
// scope as [ResolvePath] does. It reports false if d is unset, is a
// function call or is bound to a value that is missing or not a number.
func (m *DataModel) ResolveNumber(d a2ui.DynamicNumber, scope string) (float64, bool) {
	switch {
	case d.Literal != nil:
		return *d.Literal, true
	case d.Binding != nil:
		v, _ := m.Get(ResolvePath(d.Binding.Path, scope))
		n, ok := v.(float64)
		return n, ok
	}
	return 0, false
}

// ResolveBoolean returns the value of d, resolving a data binding in
// scope as [ResolvePath] does. It reports false if d is unset, is a
// function call or is bound to a value that is missing or not a boolean.
func (m *DataModel) ResolveBoolean(d a2ui.DynamicBoolean, scope string) (bool, bool) {
	switch {
	case d.Literal != nil:
		return *d.Literal, true
	case d.Binding != nil:
		v, _ := m.Get(ResolvePath(d.Binding.Path, scope))
		b, ok := v.(bool)
		return b, ok
	}
	return false, false
}

// ResolveStringList returns the value of d, resolving a data binding in
// scope as [ResolvePath] does. It reports false if d is unset, is a
// function call or is bound to a value that is missing or not an array
// of strings.
func (m *DataModel) ResolveStringList(d a2ui.DynamicStringList, scope string) ([]string, bool) {
	switch {
	case d.Literal != nil:
		return d.Literal, true
	case d.Binding != nil:
		v, _ := m.Get(ResolvePath(d.Binding.Path, scope))
		list, ok := v.([]any)
		if !ok {
			return nil, false
		}
		out := make([]string, len(list))
		for i, x := range list {
			if out[i], ok = x.(string); !ok {
				return nil, false
			}
		}
		return out, true
	}
	return nil, false
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
