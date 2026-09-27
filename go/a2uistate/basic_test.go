package a2uistate

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func evalModel(t *testing.T) *DataModel {
	t.Helper()
	var m DataModel
	err := m.Set("", decode(t, `{
		"name": "Ann", "email": "ann@example.com", "bad": "ann@", "empty": "",
		"n": 1234.5, "on": true, "off": false, "none": null, "list": [], "tags": ["a"],
		"when": "2026-01-16T14:30:05Z", "html": "<b>&</b>",
		"items": [{"name": "Bob", "count": 1}, {"name": "Cy", "count": 2}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return &m
}

// value decodes a dynamic value from JSON.
func value(t *testing.T, s string) a2ui.DynamicValue {
	t.Helper()
	var d a2ui.DynamicValue
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		t.Fatalf("decode %s: %v", s, err)
	}
	return d
}

func TestBasicFunctions(t *testing.T) {
	m := evalModel(t)
	invalid := map[string]any{"valid": false}
	ok := map[string]any{"valid": true}
	tests := []struct {
		call  string
		scope string
		want  any
	}{
		// @index
		{`{"call": "@index"}`, "/items/1", 1.0},
		{`{"call": "@index", "args": {"offset": 1}}`, "/items/0", 1.0},

		// Validation.
		{`{"call": "required", "args": {"value": {"path": "/name"}}}`, "", ok},
		{`{"call": "required", "args": {"value": {"path": "/empty"}}}`, "", invalid},
		{`{"call": "required", "args": {"value": {"path": "/none"}}}`, "", invalid},
		{`{"call": "required", "args": {"value": {"path": "/list"}}}`, "", invalid},
		{`{"call": "required", "args": {"value": {"path": "/missing"}}}`, "", invalid},
		{`{"call": "required", "args": {"value": {"path": "/off"}}}`, "", ok},
		{`{"call": "required", "args": {"value": 0}}`, "", ok},
		{`{"call": "regex", "args": {"value": "5551234567", "pattern": "^\\d{10}$"}}`, "", ok},
		{`{"call": "regex", "args": {"value": "555", "pattern": "^\\d{10}$"}}`, "", invalid},
		{`{"call": "regex", "args": {"value": {"path": "/missing"}, "pattern": "^$"}}`, "", ok},
		{`{"call": "length", "args": {"value": {"path": "/name"}, "min": 3}}`, "", ok},
		{`{"call": "length", "args": {"value": {"path": "/name"}, "min": 4}}`, "", invalid},
		{`{"call": "length", "args": {"value": {"path": "/name"}, "max": 2}}`, "", invalid},
		{`{"call": "length", "args": {"value": "😀", "max": 1}}`, "", invalid}, // UTF-16 length 2
		{`{"call": "numeric", "args": {"value": {"path": "/n"}, "min": 0, "max": 2000}}`, "", ok},
		{`{"call": "numeric", "args": {"value": {"path": "/n"}, "max": 1000}}`, "", invalid},
		{`{"call": "numeric", "args": {"value": " 42 ", "min": 42}}`, "", ok},
		{`{"call": "numeric", "args": {"value": "4x", "min": 0}}`, "", invalid},
		{`{"call": "numeric", "args": {"value": {"path": "/missing"}, "min": 0}}`, "", invalid},
		{`{"call": "email", "args": {"value": {"path": "/email"}}}`, "", ok},
		{`{"call": "email", "args": {"value": {"path": "/bad"}}}`, "", invalid},

		// Logic.
		{`{"call": "and", "args": {"values": [true, {"path": "/on"}]}}`, "", true},
		{`{"call": "and", "args": {"values": [true, {"path": "/off"}]}}`, "", false},
		{`{"call": "or", "args": {"values": [false, {"path": "/missing"}]}}`, "", false},
		{`{"call": "or", "args": {"values": [false, {"call": "not", "args": {"value": false}}]}}`, "", true},
		{`{"call": "not", "args": {"value": {"path": "/on"}}}`, "", false},
		{`{"call": "and", "args": {"values": [
			{"call": "required", "args": {"value": {"path": "/name"}}},
			{"call": "email", "args": {"value": {"path": "/bad"}}}
		]}}`, "", false},

		// Formatting.
		{`{"call": "formatString", "args": {"value": "Hello, ${/name}! ${n} ${/on} ${/none}${/missing}"}}`, "", "Hello, Ann! 1234.5 true "},
		{`{"call": "formatString", "args": {"value": "${name} is #${@index(offset: 1)}"}}`, "/items/1", "Cy is #2"},
		{`{"call": "formatString", "args": {"value": "${/tags} ${/items/0} ${/html}"}}`, "", `["a"] {"count":1,"name":"Bob"} <b>&</b>`},
		{`{"call": "formatString", "args": {"value": "${formatDate(value: ${/when}, format: 'MMM d')}"}}`, "", "Jan 16"},
		{`{"call": "formatString", "args": {"value": "\\${/name}"}}`, "", "${/name}"},
		{`{"call": "formatString", "args": {"value": {"path": "/name"}}}`, "", "Ann"},
		{`{"call": "formatNumber", "args": {"value": {"path": "/n"}}}`, "", "1,234.5"},
		{`{"call": "formatNumber", "args": {"value": 1234.5678, "decimals": 2, "grouping": false}}`, "", "1234.57"},
		{`{"call": "formatCurrency", "args": {"value": {"path": "/n"}, "currency": "usd"}}`, "", "$1,234.50"},
		{`{"call": "formatCurrency", "args": {"value": 3, "currency": "EUR", "decimals": 0}}`, "", "€3"},
		{`{"call": "formatDate", "args": {"value": {"path": "/when"}, "format": "EEEE, d MMMM yyyy h:mm a"}}`, "", "Friday, 16 January 2026 2:30 PM"},
		{`{"call": "formatDate", "args": {"value": {"path": "/n"}, "format": "yyyy"}}`, "", ""},
		{`{"call": "formatDate", "args": {"value": {"path": "/missing"}, "format": "yyyy"}}`, "", ""},
		{`{"call": "pluralize", "args": {"value": 1, "one": "item", "other": "items"}}`, "", "item"},
		{`{"call": "pluralize", "args": {"value": 2, "one": "item", "other": "items"}}`, "", "items"},
		{`{"call": "pluralize", "args": {"value": 0, "zero": "none", "other": "items"}}`, "", "none"},
		{`{"call": "pluralize", "args": {"value": 0, "one": "item", "other": "items"}}`, "", "items"},
		{`{"call": "pluralize", "args": {"value": -1, "one": "item", "other": "items"}}`, "", "item"},
		{`{"call": "pluralize", "args": {"value": 1.5, "one": "item", "other": "items"}}`, "", "items"},
		{`{"call": "pluralize", "args": {"value": 2, "two": "pair", "other": "items"}}`, "", "pair"},
		{`{"call": "pluralize", "args": {"value": 1, "other": "items"}}`, "", "items"},
		{`{"call": "pluralize", "args": {"value": {"path": "count"}, "one": "", "other": "items"}}`, "/items/0", ""},
	}
	var e Evaluator
	e.Data = m
	for _, tt := range tests {
		got, err := e.ResolveValue(value(t, tt.call), tt.scope)
		if err != nil {
			t.Errorf("%s: %v", tt.call, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %#v, want %#v", tt.call, got, tt.want)
		}
	}
}

func TestBasicFunctionErrors(t *testing.T) {
	m := evalModel(t)
	tests := []struct {
		call  string
		scope string
		want  error
	}{
		{`{"call": "trim", "args": {"value": "x"}}`, "", ErrUnknownFunction},
		{`{"call": "formatString", "args": {"value": "${trim(value: 'x')}"}}`, "", ErrUnknownFunction},
		{`{"call": "openUrl", "args": {"url": "https://a2ui.org"}}`, "", ErrAction},
		{`{"call": "@index"}`, "", ErrNoValue},
		{`{"call": "@index"}`, "/name", ErrNoValue},
		{`{"call": "@index", "args": {"offset": "1"}}`, "/items/0", ErrInvalidArgs},
		{`{"call": "regex", "args": {"value": "x", "pattern": "(?=x)"}}`, "", ErrInvalidArgs},
		{`{"call": "regex", "args": {"value": "x"}}`, "", ErrInvalidArgs},
		{`{"call": "regex", "args": {"value": 1, "pattern": "1"}}`, "", ErrInvalidArgs},
		{`{"call": "length", "args": {"value": "x", "min": 1.5}}`, "", ErrInvalidArgs},
		{`{"call": "numeric", "args": {"value": true, "min": 1}}`, "", ErrInvalidArgs},
		{`{"call": "email", "args": {"value": {"path": "/n"}}}`, "", ErrInvalidArgs},
		{`{"call": "and", "args": {"values": [true]}}`, "", ErrInvalidArgs},
		{`{"call": "and", "args": {"values": true}}`, "", ErrInvalidArgs},
		{`{"call": "or", "args": {"values": [true, "yes"]}}`, "", ErrInvalidArgs},
		{`{"call": "not", "args": {"value": 1}}`, "", ErrInvalidArgs},
		{`{"call": "formatString", "args": {"value": "${/name"}}`, "", ErrInvalidArgs},
		{`{"call": "formatString", "args": {}}`, "", ErrInvalidArgs},
		{`{"call": "formatNumber", "args": {"value": "1"}}`, "", ErrInvalidArgs},
		{`{"call": "formatNumber", "args": {"value": 1, "decimals": 101}}`, "", ErrInvalidArgs},
		{`{"call": "formatNumber", "args": {"value": 1, "decimals": -1}}`, "", ErrInvalidArgs},
		{`{"call": "formatNumber", "args": {"value": 1, "grouping": "no"}}`, "", ErrInvalidArgs},
		{`{"call": "formatNumber", "args": {"value": {"path": "/missing"}}}`, "", ErrInvalidArgs},
		{`{"call": "formatCurrency", "args": {"value": 1}}`, "", ErrInvalidArgs},
		{`{"call": "formatDate", "args": {"value": "2026-01-16"}}`, "", ErrInvalidArgs},
		{`{"call": "pluralize", "args": {"value": 1, "one": "item"}}`, "", ErrInvalidArgs},
		{`{"call": "pluralize", "args": {"value": 1, "one": 1, "other": "items"}}`, "", ErrInvalidArgs},
	}
	e := &Evaluator{Data: m}
	for _, tt := range tests {
		_, err := e.ResolveValue(value(t, tt.call), tt.scope)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s in %q: error = %v, want %v", tt.call, tt.scope, err, tt.want)
		}
	}
}

func TestBasicFunctionsClone(t *testing.T) {
	f := BasicFunctions()
	if len(f) != 15 {
		t.Errorf("len(BasicFunctions()) = %d, want 15", len(f))
	}
	delete(f, "required")
	if _, ok := BasicFunctions()["required"]; !ok {
		t.Error("BasicFunctions returned a shared map")
	}
}
