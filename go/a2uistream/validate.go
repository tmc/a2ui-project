package a2uistream

import (
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// MessageValidator validates a batch of A2UI messages.
type MessageValidator interface {
	ValidateMessages([]a2ui.AgentMessage) error
}

// ParseAndValidate parses a complete response and validates each discovered
// batch of 1.x messages. It reports an [ErrInvalidPayload] for A2UI messages
// of any other version, or 1.x messages that do not decode, rather than
// skipping them, and returns the validator's errors unchanged.
// A nil validator only checks versions and decoding.
func ParseAndValidate(content string, validator MessageValidator) ([]ResponsePart, error) {
	parts, err := parseAll(content)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		for _, payload := range part.Payload {
			if version := payloadVersion(payload); !isV1(version) {
				return nil, &payloadError{msg: fmt.Sprintf("a2uistream: message version %q is not 1.x", version)}
			}
		}
		if len(part.Messages) != len(part.Payload) {
			return nil, &payloadError{msg: "a2uistream: invalid 1.x message"}
		}
		if validator != nil && len(part.Messages) > 0 {
			if err := validator.ValidateMessages(part.Messages); err != nil {
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
