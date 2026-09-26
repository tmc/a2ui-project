package a2ui

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestActionRejectsInvalidStates(t *testing.T) {
	action := Action{
		Event:        &EventAction{Name: "submit"},
		FunctionCall: &FunctionCall{Call: "openUrl"},
	}
	if _, err := json.Marshal(action); err == nil {
		t.Fatal("expected marshal error, got nil")
	}

	var decoded Action
	if err := json.Unmarshal([]byte(`{"event":{"name":"submit"},"functionCall":{"call":"openUrl"}}`), &decoded); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func TestIconNameOrPathRejectsInvalidStates(t *testing.T) {
	name := IconSearch
	for _, icon := range []IconNameOrPath{
		{},
		{Name: &name, Binding: &DataBinding{Path: "/icon"}},
		{SVGPath: ptr(StringLiteral("M0 0h1v1z")), Binding: &DataBinding{Path: "/icon"}},
	} {
		if _, err := json.Marshal(icon); err == nil {
			t.Fatalf("marshal %+v: expected error, got nil", icon)
		}
	}

	for _, raw := range []string{
		`{"path":""}`,
		`{}`,
		`{"svgPath":"M0 0","path":"/icon"}`,
		`{"href":"icon.svg"}`,
		`{"svgPath":42}`,
	} {
		var decoded IconNameOrPath
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			t.Fatalf("unmarshal %s: expected error, got nil", raw)
		}
	}
}

func TestIconNameOrPathRoundTrip(t *testing.T) {
	name := IconSearch
	tests := []struct {
		name string
		icon IconNameOrPath
		json string
	}{
		{"name", IconNameOrPath{Name: &name}, `"search"`},
		{"svg_path", IconNameOrPath{SVGPath: ptr(StringLiteral("M0 0h1v1z"))}, `{"svgPath":"M0 0h1v1z"}`},
		{"svg_path_binding", IconNameOrPath{SVGPath: ptr(StringBinding("/icons/custom"))}, `{"svgPath":{"path":"/icons/custom"}}`},
		{"binding", IconNameOrPath{Binding: &DataBinding{Path: "/icon"}}, `{"path":"/icon"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.icon)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(data) != tt.json {
				t.Fatalf("marshal = %s, want %s", data, tt.json)
			}
			var got IconNameOrPath
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(got, tt.icon) {
				t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", got, tt.icon)
			}
		})
	}
}

func TestChildListRejectsMultipleRepresentations(t *testing.T) {
	children := ChildList{
		IDs:      []string{"a"},
		Template: &ChildTemplate{ComponentID: "child", Path: "/items"},
	}
	if _, err := json.Marshal(children); err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}
