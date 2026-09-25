package a2ui

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFlightStatusMessages(t *testing.T) {
	data, err := os.ReadFile(basicExamplesDir + "/01_flight-status.json")
	if err != nil {
		t.Fatal(err)
	}

	var example struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Messages    []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}
	if len(example.Messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(example.Messages))
	}

	tests := []struct {
		name      string
		index     int
		wantField string
	}{
		{"CreateSurface", 0, "createSurface"},
		{"UpdateComponents", 1, "updateComponents"},
		{"UpdateDataModel", 2, "updateDataModel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg AgentMessage
			if err := json.Unmarshal(example.Messages[tt.index], &msg); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if msg.Version != Version {
				t.Fatalf("version = %q, want %q", msg.Version, Version)
			}
			switch tt.wantField {
			case "createSurface":
				if msg.CreateSurface == nil {
					t.Fatal("CreateSurface is nil")
				}
				if msg.CreateSurface.SurfaceID != "gallery-flight-status" {
					t.Fatalf("surfaceId = %q", msg.CreateSurface.SurfaceID)
				}
			case "updateComponents":
				if msg.UpdateComponents == nil {
					t.Fatal("UpdateComponents is nil")
				}
				if len(msg.UpdateComponents.Components) == 0 {
					t.Fatal("no components")
				}
			case "updateDataModel":
				if msg.UpdateDataModel == nil {
					t.Fatal("UpdateDataModel is nil")
				}
			}

			// Round-trip: marshal and compare JSON equivalence.
			jsonEquivalent(t, example.Messages[tt.index], msg)
		})
	}
}

func TestAgentMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  AgentMessage
	}{
		{
			name: "create_surface",
			msg: AgentMessage{
				Version: Version,
				CreateSurface: &CreateSurface{
					SurfaceID: "test-1",
					CatalogID: "https://example.com/catalog.json",
				},
			},
		},
		{
			name: "create_surface_without_catalog",
			msg: AgentMessage{
				Version: Version,
				CreateSurface: &CreateSurface{
					SurfaceID: "test-1",
					DataModel: map[string]any{"count": float64(1)},
					Metadata:  &Metadata{Extensions: map[string]any{"x": "y"}},
				},
			},
		},
		{
			name: "delete_surface",
			msg: AgentMessage{
				Version:       Version,
				DeleteSurface: &DeleteSurface{SurfaceID: "test-1"},
			},
		},
		{
			name: "update_data_model",
			msg: AgentMessage{
				Version: Version,
				UpdateDataModel: &UpdateDataModel{
					SurfaceID: "test-1",
					Path:      "/count",
					Value:     float64(42),
				},
			},
		},
		{
			name: "call_renderer_function",
			msg: AgentMessage{
				Version: Version,
				CallRendererFunction: &CallRendererFunction{
					FunctionCallID: "call-1",
					CallFunction: FunctionCall{
						Call:      "lookup",
						CatalogID: "https://example.com/catalog.json",
						Args:      map[string]any{"q": "x"},
					},
				},
			},
		},
		{
			name: "agent_function_response",
			msg: AgentMessage{
				Version:               Version,
				AgentFunctionResponse: ptr(FunctionResponseValue("call-1", "done")),
			},
		},
		{
			name: "agent_function_response_null",
			msg: AgentMessage{
				Version:               Version,
				AgentFunctionResponse: ptr(FunctionResponseValue("call-1", nil)),
			},
		},
		{
			name: "agent_function_response_error",
			msg: AgentMessage{
				Version: Version,
				AgentFunctionResponse: &FunctionResponse{
					FunctionCallID: "call-1",
					Error:          &FunctionError{Code: "FAILED", Message: "failed"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got AgentMessage
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(got, tt.msg) {
				t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", got, tt.msg)
			}
		})
	}
}

func TestUpdateDataModelNullValue(t *testing.T) {
	data := []byte(`{"version":"v1.0","updateDataModel":{"surfaceId":"s1","path":"/x","value":null}}`)
	var msg AgentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatal(err)
	}
	jsonEquivalent(t, data, msg)
}

func TestFunctionResponseRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name     string
		response FunctionResponse
	}{
		{"none", FunctionResponse{FunctionCallID: "call-1"}},
		{"both", FunctionResponse{FunctionCallID: "call-1", Value: "done", Error: &FunctionError{Code: "ERR", Message: "bad"}}},
		{"missing_id", FunctionResponseValue("", "done")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := json.Marshal(tt.response); err == nil {
				t.Fatal("expected marshal error, got nil")
			}
		})
	}
	for _, raw := range []string{
		`{"functionCallId":"call-1"}`,
		`{"value":"done"}`,
		`{"functionCallId":"call-1","value":"done","error":{"code":"ERR","message":"bad"}}`,
		`{"functionCallId":"call-1","value":"done","extra":true}`,
	} {
		var response FunctionResponse
		if err := json.Unmarshal([]byte(raw), &response); err == nil {
			t.Fatalf("expected unmarshal error for %s", raw)
		}
	}
}

func TestRendererMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  RendererMessage
	}{
		{
			name: "action",
			msg: RendererMessage{
				Version: Version,
				Action: &ActionEvent{
					Name:              "submit",
					UserMessage:       "Submit the form",
					SurfaceID:         "test-1",
					SourceComponentID: "btn-1",
					Timestamp:         "2025-01-01T00:00:00Z",
					Context:           map[string]any{"key": "value"},
				},
			},
		},
		{
			name: "call_agent_function",
			msg: RendererMessage{
				Version: Version,
				CallAgentFunction: &CallAgentFunction{
					SurfaceID:      "test-1",
					FunctionCallID: "call-1",
					CallFunction:   FunctionCall{Call: "lookup"},
				},
			},
		},
		{
			name: "renderer_function_response",
			msg: RendererMessage{
				Version:                  Version,
				RendererFunctionResponse: ptr(FunctionResponseValue("call-1", "ok")),
			},
		},
		{
			name: "error",
			msg: RendererMessage{
				Version: Version,
				Error: &RendererError{
					Code:           "INVALID_FUNCTION",
					FunctionCallID: "call-1",
					Message:        "function not found",
				},
			},
		},
		{
			name: "validation_error",
			msg: RendererMessage{
				Version: Version,
				Error: &RendererError{
					Code:      ErrorValidationFailed,
					SurfaceID: "test-1",
					Path:      "/components/0/text",
					Message:   "expected string",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got RendererMessage
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(got, tt.msg) {
				t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", got, tt.msg)
			}
		})
	}
}

func TestRendererErrorTargets(t *testing.T) {
	tests := []struct {
		name string
		err  RendererError
	}{
		{"neither", RendererError{Code: "ERR", Message: "bad"}},
		{"both", RendererError{Code: "ERR", Message: "bad", SurfaceID: "s1", FunctionCallID: "call-1"}},
		{"validation_missing_path", RendererError{Code: ErrorValidationFailed, Message: "bad", SurfaceID: "s1"}},
		{"unallowed_child_missing_surface", RendererError{Code: ErrorUnallowedChild, Message: "bad", Path: "/x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := RendererMessage{Version: Version, Error: &tt.err}
			if _, err := json.Marshal(msg); err == nil {
				t.Fatal("expected marshal error, got nil")
			}
		})
	}
	for _, raw := range []string{
		`{"version":"v1.0","error":{"code":"ERR","message":"bad"}}`,
		`{"version":"v1.0","error":{"code":"ERR","message":"bad","surfaceId":"s1","functionCallId":"call-1"}}`,
		`{"version":"v1.0","error":{"code":"VALIDATION_FAILED","message":"bad","surfaceId":"s1"}}`,
	} {
		var msg RendererMessage
		if err := json.Unmarshal([]byte(raw), &msg); err == nil {
			t.Fatalf("expected unmarshal error for %s", raw)
		}
	}
}

func TestAgentMessageRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name string
		msg  AgentMessage
	}{
		{
			name: "none",
			msg:  AgentMessage{Version: Version},
		},
		{
			name: "multiple",
			msg: AgentMessage{
				Version:       Version,
				CreateSurface: &CreateSurface{SurfaceID: "s1", CatalogID: "cat"},
				DeleteSurface: &DeleteSurface{SurfaceID: "s1"},
			},
		},
		{
			name: "call_renderer_function_missing_id",
			msg: AgentMessage{
				Version: Version,
				CallRendererFunction: &CallRendererFunction{
					CallFunction: FunctionCall{Call: "lookup", CatalogID: "cat"},
				},
			},
		},
		{
			name: "call_renderer_function_missing_catalog",
			msg: AgentMessage{
				Version: Version,
				CallRendererFunction: &CallRendererFunction{
					FunctionCallID: "call-1",
					CallFunction:   FunctionCall{Call: "lookup"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := json.Marshal(tt.msg); err == nil {
				t.Fatal("expected marshal error, got nil")
			}
		})
	}
	for _, raw := range []string{
		`{"version":"v1.0"}`,
		`{"version":"v1.0","createSurface":{"surfaceId":"s1","catalogId":"cat"},"deleteSurface":{"surfaceId":"s1"}}`,
		`{"version":"v1.0","callRendererFunction":{"callFunction":{"call":"lookup","catalogId":"cat"}}}`,
		`{"version":"v1.0","callRendererFunction":{"functionCallId":"call-1","callFunction":{"catalogId":"cat"}}}`,
		`{"version":"v1.0","callRendererFunction":{"functionCallId":"call-1","callFunction":{"call":"lookup"}}}`,
		`{"version":"v1.0","agentFunctionResponse":{"value":"done"}}`,
	} {
		var msg AgentMessage
		if err := json.Unmarshal([]byte(raw), &msg); err == nil {
			t.Fatalf("expected unmarshal error for %s", raw)
		}
	}
}

func TestRendererMessageRejectsInvalidPayloads(t *testing.T) {
	for _, raw := range []string{
		`{"version":"v1.0"}`,
		`{"version":"v1.0","action":{"name":"submit","surfaceId":"s1","sourceComponentId":"btn","timestamp":"2025-01-01T00:00:00Z","context":{}},"error":{"code":"ERR","surfaceId":"s1","message":"bad"}}`,
		`{"version":"v1.0","callAgentFunction":{"functionCallId":"call-1","callFunction":{"call":"lookup"}}}`,
		`{"version":"v1.0","rendererFunctionResponse":{"functionCallId":"call-1"}}`,
	} {
		var msg RendererMessage
		if err := json.Unmarshal([]byte(raw), &msg); err == nil {
			t.Fatalf("expected unmarshal error for %s", raw)
		}
	}
}

func TestAgentMessageListWrapper(t *testing.T) {
	wrapper := AgentMessageListWrapper{
		Messages: []AgentMessage{
			{
				Version:       Version,
				CreateSurface: &CreateSurface{SurfaceID: "s1", CatalogID: "cat"},
			},
			{
				Version:       Version,
				DeleteSurface: &DeleteSurface{SurfaceID: "s1"},
			},
		},
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AgentMessageListWrapper
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, wrapper) {
		t.Fatalf("round-trip mismatch\n  got:  %+v\n  want: %+v", got, wrapper)
	}
}

func TestRendererMessageListWrapperEmpty(t *testing.T) {
	data := []byte(`{"messages":[]}`)
	var w RendererMessageListWrapper
	if err := json.Unmarshal(data, &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(w.Messages) != 0 {
		t.Fatalf("got %d messages, want 0", len(w.Messages))
	}
}

// jsonEquivalent marshals v and checks that the result is semantically
// equivalent to the original JSON. Empty maps/objects and missing fields
// are treated as equivalent (omitempty normalization).
func jsonEquivalent(t *testing.T, original json.RawMessage, v any) {
	t.Helper()
	remarshaled, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(remarshaled, &got); err != nil {
		t.Fatalf("unmarshal re-marshaled: %v", err)
	}
	if err := json.Unmarshal(original, &want); err != nil {
		t.Fatalf("unmarshal original: %v", err)
	}
	normalizeJSON(got)
	normalizeJSON(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON not equivalent:\n  got:  %s\n  want: %s", remarshaled, original)
	}
}

// normalizeJSON removes empty maps and nil values in-place so that
// omitempty differences don't cause false mismatches.
func normalizeJSON(v any) {
	switch v := v.(type) {
	case map[string]any:
		for k, val := range v {
			normalizeJSON(val)
			// Remove keys whose value is an empty map (matches omitempty behavior).
			if m, ok := val.(map[string]any); ok && len(m) == 0 {
				delete(v, k)
			}
		}
	case []any:
		for _, elem := range v {
			normalizeJSON(elem)
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
