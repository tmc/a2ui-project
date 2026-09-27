package a2uischema

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// checkFields reports a field of the JSON payload data that decoding it
// as msgs dropped, such as a misspelled field or one that A2UI 1.x
// removed, like the returnType of a function call. The root package
// decodes leniently, as encoding/json does, and the schemas forbid such
// fields, so the validator rejects them.
//
// A dropped field whose value is null, false, 0, "", [] or {} is not
// reported, since it may have been dropped only for being empty.
func checkFields(data []byte, msgs []a2ui.AgentMessage) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return within(err, "", "parse messages")
	}
	encoded, err := json.Marshal(msgs)
	if err != nil {
		return within(err, "", "encode messages")
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return within(err, "", "encode messages")
	}
	if _, ok := raw.([]any); !ok {
		decoded = decoded.([]any)[0]
	}
	return droppedField(raw, decoded, "")
}

// droppedField reports a field of the JSON value raw that is missing
// from decoded, the value re-encoded after decoding. Path is the JSON
// pointer of raw. Values of different shapes, such as a string literal
// and a union type's object form, are not compared.
func droppedField(raw, decoded any, path string) error {
	switch raw := raw.(type) {
	case map[string]any:
		decoded, ok := decoded.(map[string]any)
		if !ok {
			return nil
		}
		for _, key := range slices.Sorted(maps.Keys(raw)) {
			d, ok := decoded[key]
			if !ok {
				if isEmpty(raw[key]) {
					continue
				}
				return invalid(ErrInvalidMessage, path+pointer(key), fmt.Sprintf("unknown field %q", key))
			}
			if err := droppedField(raw[key], d, path+pointer(key)); err != nil {
				return err
			}
		}
	case []any:
		decoded, ok := decoded.([]any)
		if !ok || len(decoded) != len(raw) {
			return nil
		}
		for i := range raw {
			if err := droppedField(raw[i], decoded[i], path+pointer(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// isEmpty reports whether the JSON value v is null, false, 0, "", []
// or {}.
func isEmpty(v any) bool {
	switch v := v.(type) {
	case nil:
		return true
	case bool:
		return !v
	case float64:
		return v == 0
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}
