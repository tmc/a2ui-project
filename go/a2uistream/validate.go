package a2uistream

import (
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2ui/v09"
)

// MessageValidator validates a batch of A2UI 1.x messages.
type MessageValidator interface {
	ValidateMessages([]a2ui.AgentMessage) error
}

// MessageValidatorV09 validates a batch of A2UI v0.9 messages.
type MessageValidatorV09 interface {
	ValidateMessagesV09([]v09.ServerMessage) error
}

// ParseAndValidate parses a complete response and validates each discovered
// batch of 1.x messages. It reports an error for A2UI messages of any other
// version, or 1.x messages that do not decode, rather than skipping them.
// A nil validator only checks versions and decoding.
func ParseAndValidate(content string, validator MessageValidator) ([]ResponsePart, error) {
	parts, err := parseAll(content)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		for _, payload := range part.Payload {
			if version := payloadVersion(payload); !isV1(version) {
				return nil, fmt.Errorf("a2uistream: message version %q is not 1.x", version)
			}
		}
		if len(part.Messages) != len(part.Payload) {
			return nil, fmt.Errorf("a2uistream: invalid 1.x message")
		}
		if validator != nil && len(part.Messages) > 0 {
			if err := validator.ValidateMessages(part.Messages); err != nil {
				return nil, err
			}
		}
	}
	return parts, nil
}

// ParseAndValidateV09 is like [ParseAndValidate] for v0.9 messages.
// It accepts the v0.9.x revisions, which share the v0.9 message types.
func ParseAndValidateV09(content string, validator MessageValidatorV09) ([]ResponsePartV09, error) {
	all, err := parseAll(content)
	if err != nil {
		return nil, err
	}
	parts := partsV09(all)
	for _, part := range parts {
		for _, payload := range part.Payload {
			if version := payloadVersion(payload); !isV09(version) {
				return nil, fmt.Errorf("a2uistream: message version %q is not v0.9", version)
			}
		}
		if len(part.Messages) != len(part.Payload) {
			return nil, fmt.Errorf("a2uistream: invalid v0.9 message")
		}
		if validator != nil && len(part.Messages) > 0 {
			if err := validator.ValidateMessagesV09(part.Messages); err != nil {
				return nil, err
			}
		}
	}
	return parts, nil
}

func payloadVersion(payload map[string]any) string {
	version, _ := payload["version"].(string)
	return version
}

func parseAll(content string) ([]ResponsePart, error) {
	parser := NewParser()
	parts, err := parser.ProcessChunk(content)
	if err != nil {
		return nil, err
	}
	flush, err := parser.Flush()
	if err != nil {
		return nil, err
	}
	return append(parts, flush...), nil
}
