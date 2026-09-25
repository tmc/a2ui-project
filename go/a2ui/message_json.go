package a2ui

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON implements json.Marshaler for AgentMessage.
func (m AgentMessage) MarshalJSON() ([]byte, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	type alias AgentMessage
	return json.Marshal(alias(m))
}

// UnmarshalJSON implements json.Unmarshaler for AgentMessage.
func (m *AgentMessage) UnmarshalJSON(data []byte) error {
	type alias AgentMessage
	var am alias
	if err := json.Unmarshal(data, &am); err != nil {
		return fmt.Errorf("a2ui: unmarshal agent message: %w", err)
	}
	msg := AgentMessage(am)
	if err := msg.validate(); err != nil {
		return err
	}
	*m = msg
	return nil
}

func (m AgentMessage) validate() error {
	switch countSet(m.CreateSurface != nil, m.UpdateComponents != nil, m.UpdateDataModel != nil, m.DeleteSurface != nil, m.CallRendererFunction != nil, m.AgentFunctionResponse != nil) {
	case 1:
	case 0:
		return fmt.Errorf("a2ui: agent message has no payload set")
	default:
		return fmt.Errorf("a2ui: agent message has multiple payloads set")
	}
	if c := m.CallRendererFunction; c != nil {
		if c.FunctionCallID == "" {
			return fmt.Errorf("a2ui: callRendererFunction functionCallId is required")
		}
		if c.CallFunction.Call == "" {
			return fmt.Errorf("a2ui: callRendererFunction call is required")
		}
		if c.CallFunction.CatalogID == "" {
			return fmt.Errorf("a2ui: callRendererFunction catalogId is required")
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for FunctionResponse.
func (r FunctionResponse) MarshalJSON() ([]byte, error) {
	if r.FunctionCallID == "" {
		return nil, fmt.Errorf("a2ui: function response functionCallId is required")
	}
	if r.Error == nil {
		return json.Marshal(struct {
			FunctionCallID string `json:"functionCallId"`
			Value          any    `json:"value"`
		}{r.FunctionCallID, r.Value})
	}
	if r.Value != nil {
		return nil, fmt.Errorf("a2ui: function response has both value and error set")
	}
	return json.Marshal(struct {
		FunctionCallID string         `json:"functionCallId"`
		Error          *FunctionError `json:"error"`
	}{r.FunctionCallID, r.Error})
}

// UnmarshalJSON implements json.Unmarshaler for FunctionResponse.
func (r *FunctionResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("a2ui: unmarshal function response: %w", err)
	}
	var (
		resp     FunctionResponse
		hasValue bool
	)
	for key, raw := range fields {
		switch key {
		case "functionCallId":
			if err := json.Unmarshal(raw, &resp.FunctionCallID); err != nil {
				return fmt.Errorf("a2ui: unmarshal function response functionCallId: %w", err)
			}
		case "value":
			hasValue = true
			if string(bytes.TrimSpace(raw)) != "null" {
				if err := json.Unmarshal(raw, &resp.Value); err != nil {
					return fmt.Errorf("a2ui: unmarshal function response value: %w", err)
				}
			}
		case "error":
			resp.Error = new(FunctionError)
			if err := json.Unmarshal(raw, resp.Error); err != nil {
				return fmt.Errorf("a2ui: unmarshal function response error: %w", err)
			}
		default:
			return fmt.Errorf("a2ui: function response has unknown field %q", key)
		}
	}
	switch {
	case resp.FunctionCallID == "":
		return fmt.Errorf("a2ui: function response functionCallId is required")
	case hasValue && resp.Error != nil:
		return fmt.Errorf("a2ui: function response must not have both value and error")
	case !hasValue && resp.Error == nil:
		return fmt.Errorf("a2ui: function response must have value or error")
	}
	*r = resp
	return nil
}

// MarshalJSON implements json.Marshaler for RendererMessage.
func (m RendererMessage) MarshalJSON() ([]byte, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	type alias RendererMessage
	return json.Marshal(alias(m))
}

// UnmarshalJSON implements json.Unmarshaler for RendererMessage.
func (m *RendererMessage) UnmarshalJSON(data []byte) error {
	type alias RendererMessage
	var am alias
	if err := json.Unmarshal(data, &am); err != nil {
		return fmt.Errorf("a2ui: unmarshal renderer message: %w", err)
	}
	msg := RendererMessage(am)
	if err := msg.validate(); err != nil {
		return err
	}
	*m = msg
	return nil
}

func (m RendererMessage) validate() error {
	switch countSet(m.Action != nil, m.CallAgentFunction != nil, m.RendererFunctionResponse != nil, m.Error != nil) {
	case 1:
	case 0:
		return fmt.Errorf("a2ui: renderer message has no payload set")
	default:
		return fmt.Errorf("a2ui: renderer message has multiple payloads set")
	}
	if c := m.CallAgentFunction; c != nil {
		if c.SurfaceID == "" {
			return fmt.Errorf("a2ui: callAgentFunction surfaceId is required")
		}
		if c.FunctionCallID == "" {
			return fmt.Errorf("a2ui: callAgentFunction functionCallId is required")
		}
		if c.CallFunction.Call == "" {
			return fmt.Errorf("a2ui: callAgentFunction call is required")
		}
	}
	if e := m.Error; e != nil {
		if e.Code == "" {
			return fmt.Errorf("a2ui: renderer error code is required")
		}
		if e.Message == "" {
			return fmt.Errorf("a2ui: renderer error message is required")
		}
		if e.IsValidationError() {
			if e.SurfaceID == "" || e.Path == "" {
				return fmt.Errorf("a2ui: renderer error %s requires surfaceId and path", e.Code)
			}
			if e.FunctionCallID != "" {
				return fmt.Errorf("a2ui: renderer error %s must not have functionCallId", e.Code)
			}
			return nil
		}
		switch countSet(e.SurfaceID != "", e.FunctionCallID != "") {
		case 1:
		case 0:
			return fmt.Errorf("a2ui: renderer error must have surfaceId or functionCallId")
		default:
			return fmt.Errorf("a2ui: renderer error must not have both surfaceId and functionCallId")
		}
	}
	return nil
}
