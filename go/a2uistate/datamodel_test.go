package a2uistate

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDataModelSet(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		pointer string
		value   any
		want    string // JSON of the data model, or "" if Set fails
	}{
		{"root", `{"a":1}`, "", map[string]any{"b": 2.0}, `{"b":2}`},
		{"slash root", `{"a":1}`, "/", map[string]any{"b": 2.0}, `{"b":2}`},
		{"root nil", `{"a":1}`, "/", nil, `{}`},
		{"root not object", `{}`, "", "x", ""},
		{"add key", `{"a":1}`, "/b", "x", `{"a":1,"b":"x"}`},
		{"replace key", `{"a":1}`, "/a", true, `{"a":true}`},
		{"delete key", `{"a":1,"b":2}`, "/a", nil, `{"b":2}`},
		{"delete missing", `{"a":1}`, "/x/y", nil, `{"a":1}`},
		{"create objects", `{}`, "/a/b/c", 1, `{"a":{"b":{"c":1}}}`},
		{"create array", `{}`, "/a/1", "x", `{"a":[null,"x"]}`},
		{"create array in array", `{}`, "/a/0/0", "x", `{"a":[["x"]]}`},
		{"replace null", `{"a":null}`, "/a/b", 1, `{"a":{"b":1}}`},
		{"grow array", `{"a":[1]}`, "/a/2", 3, `{"a":[1,null,3]}`},
		{"delete element", `{"a":[1,2]}`, "/a/0", nil, `{"a":[null,2]}`},
		{"escaped", `{}`, "/a~1b/c~0d", 1, `{"a/b":{"c~d":1}}`},
		{"struct value", `{}`, "/u", struct {
			Name string `json:"name"`
		}{"Ann"}, `{"u":{"name":"Ann"}}`},
		{"int value", `{}`, "/n", 3, `{"n":3}`},
		{"through string", `{"a":"x"}`, "/a/b", 1, ""},
		{"non-index in array", `{"a":[]}`, "/a/x", 1, ""},
		{"leading zero index", `{"a":[]}`, "/a/01", 1, ""},
		{"index too large", `{"a":[]}`, "/a/10001", 1, ""},
		{"relative pointer", `{}`, "a", 1, ""},
		{"bad escape", `{}`, "/a~2", 1, ""},
		{"unencodable", `{}`, "/a", func() {}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m DataModel
			if err := m.Set("", decode(t, tt.initial)); err != nil {
				t.Fatal(err)
			}
			err := m.Set(tt.pointer, tt.value)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("Set(%q, %v) succeeded, want error", tt.pointer, tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("Set(%q, %v): %v", tt.pointer, tt.value, err)
			}
			got, _ := m.Get("")
			if !reflect.DeepEqual(got, decode(t, tt.want)) {
				data, _ := json.Marshal(got)
				t.Errorf("Set(%q, %v): data model is %s, want %s", tt.pointer, tt.value, data, tt.want)
			}
		})
	}
}

func TestDataModelGet(t *testing.T) {
	var m DataModel
	if err := m.Set("", decode(t, `{"a":{"b":[1,{"c":"x"}],"":2,"d/e":3,"f~g":4},"n":null}`)); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		pointer string
		want    any
		ok      bool
	}{
		{"/a/b/0", 1.0, true},
		{"/a/b/1/c", "x", true},
		{"/a/", 2.0, true},
		{"/a/d~1e", 3.0, true},
		{"/a/f~0g", 4.0, true},
		{"/n", nil, true},
		{"/a/b/2", nil, false},
		{"/a/b/-1", nil, false},
		{"/a/b/01", nil, false},
		{"/a/b/0/x", nil, false},
		{"/x", nil, false},
		{"a", nil, false},
		{"/a~", nil, false},
	}
	for _, tt := range tests {
		got, ok := m.Get(tt.pointer)
		if ok != tt.ok || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Get(%q) = %v, %v, want %v, %v", tt.pointer, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDataModelZero(t *testing.T) {
	var m DataModel
	got, ok := m.Get("/")
	if !ok || !reflect.DeepEqual(got, map[string]any{}) {
		t.Errorf("Get(/) = %v, %v, want map[], true", got, ok)
	}
	if _, ok := m.Get("/a"); ok {
		t.Errorf("Get(/a) reports a value in the zero DataModel")
	}
}

func TestDataModelSetCopies(t *testing.T) {
	var m DataModel
	v := map[string]any{"list": []any{"a"}}
	if err := m.Set("/v", v); err != nil {
		t.Fatal(err)
	}
	v["list"].([]any)[0] = "changed"
	if got, _ := m.Get("/v/list/0"); got != "a" {
		t.Errorf("Get(/v/list/0) = %v after changing the value set, want a", got)
	}
}

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
