package a2uistate

import (
	"math"
	"reflect"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func TestResolvePath(t *testing.T) {
	tests := []struct {
		path, scope, want string
	}{
		{"/a/b", "/items/0", "/a/b"},
		{"name", "/items/0", "/items/0/name"},
		{"name", "/items/0/", "/items/0/name"},
		{"name", "", "/name"},
		{"name", "/", "/name"},
		{"", "/items/0", "/items/0"},
		{"", "", ""},
	}
	for _, tt := range tests {
		if got := ResolvePath(tt.path, tt.scope); got != tt.want {
			t.Errorf("ResolvePath(%q, %q) = %q, want %q", tt.path, tt.scope, got, tt.want)
		}
	}
}

func testModel(t *testing.T) *DataModel {
	t.Helper()
	var m DataModel
	err := m.Set("", decode(t, `{
		"name": "Ann", "n": 1.5, "on": true, "null": null,
		"tags": ["a", "b"], "mixed": ["a", 1], "obj": {"k": 1},
		"items": [{"name": "Bob", "age": 30, "ok": false, "tags": ["c"]}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestResolveString(t *testing.T) {
	m := testModel(t)
	call := a2ui.DynamicString{FunctionCall: &a2ui.FunctionCall{Call: "now"}}
	tests := []struct {
		name  string
		d     a2ui.DynamicString
		scope string
		want  string
		ok    bool
	}{
		{"literal", a2ui.StringLiteral("x"), "", "x", true},
		{"string", a2ui.StringBinding("/name"), "", "Ann", true},
		{"number", a2ui.StringBinding("/n"), "", "1.5", true},
		{"bool", a2ui.StringBinding("/on"), "", "true", true},
		{"null", a2ui.StringBinding("/null"), "", "", true},
		{"array", a2ui.StringBinding("/tags"), "", `["a","b"]`, true},
		{"object", a2ui.StringBinding("/obj"), "", `{"k":1}`, true},
		{"relative", a2ui.StringBinding("name"), "/items/0", "Bob", true},
		{"relative number", a2ui.StringBinding("age"), "/items/0", "30", true},
		{"missing", a2ui.StringBinding("/x"), "", "", false},
		{"function call", call, "", "", false},
		{"unset", a2ui.DynamicString{}, "", "", false},
	}
	for _, tt := range tests {
		got, ok := m.ResolveString(tt.d, tt.scope)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: ResolveString = %q, %v, want %q, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveNumber(t *testing.T) {
	m := testModel(t)
	tests := []struct {
		name  string
		d     a2ui.DynamicNumber
		scope string
		want  float64
		ok    bool
	}{
		{"literal", a2ui.NumberLiteral(2), "", 2, true},
		{"binding", a2ui.NumberBinding("/n"), "", 1.5, true},
		{"relative", a2ui.NumberBinding("age"), "/items/0", 30, true},
		{"not number", a2ui.NumberBinding("/name"), "", 0, false},
		{"missing", a2ui.NumberBinding("/x"), "", 0, false},
		{"unset", a2ui.DynamicNumber{}, "", 0, false},
	}
	for _, tt := range tests {
		got, ok := m.ResolveNumber(tt.d, tt.scope)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: ResolveNumber = %v, %v, want %v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveBoolean(t *testing.T) {
	m := testModel(t)
	tests := []struct {
		name  string
		d     a2ui.DynamicBoolean
		scope string
		want  bool
		ok    bool
	}{
		{"literal", a2ui.BoolLiteral(true), "", true, true},
		{"binding", a2ui.BoolBinding("/on"), "", true, true},
		{"relative", a2ui.BoolBinding("ok"), "/items/0", false, true},
		{"not bool", a2ui.BoolBinding("/n"), "", false, false},
		{"missing", a2ui.BoolBinding("/x"), "", false, false},
		{"unset", a2ui.DynamicBoolean{}, "", false, false},
	}
	for _, tt := range tests {
		got, ok := m.ResolveBoolean(tt.d, tt.scope)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: ResolveBoolean = %v, %v, want %v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveStringList(t *testing.T) {
	m := testModel(t)
	tests := []struct {
		name  string
		d     a2ui.DynamicStringList
		scope string
		want  []string
		ok    bool
	}{
		{"literal", a2ui.StringListLiteral([]string{"x"}), "", []string{"x"}, true},
		{"binding", a2ui.StringListBinding("/tags"), "", []string{"a", "b"}, true},
		{"relative", a2ui.StringListBinding("tags"), "/items/0", []string{"c"}, true},
		{"mixed", a2ui.StringListBinding("/mixed"), "", nil, false},
		{"not list", a2ui.StringListBinding("/name"), "", nil, false},
		{"unset", a2ui.DynamicStringList{}, "", nil, false},
	}
	for _, tt := range tests {
		got, ok := m.ResolveStringList(tt.d, tt.scope)
		if !reflect.DeepEqual(got, tt.want) || ok != tt.ok {
			t.Errorf("%s: ResolveStringList = %v, %v, want %v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveValue(t *testing.T) {
	m := testModel(t)
	s, n, b := "x", 2.0, true
	tests := []struct {
		name  string
		d     a2ui.DynamicValue
		scope string
		want  any
		ok    bool
	}{
		{"string", a2ui.DynamicValue{String: &s}, "", "x", true},
		{"number", a2ui.DynamicValue{Number: &n}, "", 2.0, true},
		{"bool", a2ui.DynamicValue{Bool: &b}, "", true, true},
		{"array", a2ui.DynamicValue{Array: []any{"a"}}, "", []any{"a"}, true},
		{"binding", a2ui.DynamicValue{Binding: &a2ui.DataBinding{Path: "name"}}, "/items/0", "Bob", true},
		{"missing", a2ui.DynamicValue{Binding: &a2ui.DataBinding{Path: "/x"}}, "", nil, false},
		{"function call", a2ui.DynamicValue{FunctionCall: &a2ui.FunctionCall{Call: "now"}}, "", nil, false},
		{"unset", a2ui.DynamicValue{}, "", nil, false},
	}
	for _, tt := range tests {
		got, ok := m.ResolveValue(tt.d, tt.scope)
		if !reflect.DeepEqual(got, tt.want) || ok != tt.ok {
			t.Errorf("%s: ResolveValue = %v, %v, want %v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		x    float64
		want string
	}{
		{42, "42"},
		{1.5, "1.5"},
		{-1.5, "-1.5"},
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{0.1, "0.1"},
		{0.000001, "0.000001"},
		{1e-7, "1e-7"},
		{1.5e-7, "1.5e-7"},
		{123456789012345680000, "123456789012345680000"},
		{1e21, "1e+21"},
		{1.5e300, "1.5e+300"},
		{-1e21, "-1e+21"},
		{math.MaxFloat64, "1.7976931348623157e+308"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, tt := range tests {
		if got := formatNumber(tt.x); got != tt.want {
			t.Errorf("formatNumber(%v) = %q, want %q", tt.x, got, tt.want)
		}
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{"<b>&</b>", "<b>&</b>"},
		{[]any{"<a>", 1.5}, `["<a>",1.5]`},
		{map[string]any{"k": "a&b"}, `{"k":"a&b"}`},
		{map[string]any{}, "{}"},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := toString(tt.v); got != tt.want {
			t.Errorf("toString(%#v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
