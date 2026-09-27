package a2uistream

import (
	"encoding/json"
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// MessageValidator validates a batch of A2UI messages.
//
// Decoding drops fields that [a2ui.AgentMessage] does not define. If a
// MessageValidator also has the method
//
//	ValidateJSON(data []byte) error
//
// [ParseAndValidate] calls it instead of ValidateMessages, passing the
// JSON array of the batch's payloads, so that the validator can check
// those fields too. The Validator of package a2uischema has both.
type MessageValidator interface {
	ValidateMessages([]a2ui.AgentMessage) error
}

type jsonValidator interface {
	ValidateJSON([]byte) error
}

// ParseAndValidate parses a complete response and validates each discovered
// batch of 1.x messages. It reports an [ErrInvalidPayload] for A2UI messages
// of any other version, or 1.x messages that do not decode, rather than
// skipping them, and returns the validator's errors unchanged.
// A nil validator only checks versions and decoding: it accepts, for
// example, unknown fields and a Button with no child.
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
		if validator == nil || len(part.Messages) == 0 {
			continue
		}
		if err := validate(validator, part); err != nil {
			return nil, err
		}
	}
	return parts, nil
}

// validate validates the messages of part.
func validate(validator MessageValidator, part ResponsePart) error {
	v, ok := validator.(jsonValidator)
	if !ok {
		return validator.ValidateMessages(part.Messages)
	}
	data, err := json.Marshal(part.Payload)
	if err != nil {
		return &payloadError{msg: "a2uistream: encode payload", err: err}
	}
	return v.ValidateJSON(data)
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
