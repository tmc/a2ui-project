package a2ui

import (
	"encoding/json"
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
	path := "/tmp/icon.svg"
	name := IconSearch
	icon := IconNameOrPath{Name: &name, Path: &path}
	if _, err := json.Marshal(icon); err == nil {
		t.Fatal("expected marshal error, got nil")
	}

	var decoded IconNameOrPath
	if err := json.Unmarshal([]byte(`{"path":""}`), &decoded); err == nil {
		t.Fatal("expected unmarshal error, got nil")
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
