package a2uistate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// maxArrayIndex bounds the array indexes that [DataModel.Set] accepts,
// so that a pointer cannot allocate an arbitrarily large array. It
// matches the web renderers.
const maxArrayIndex = 10000

// A DataModel is the data model of a surface: a JSON object whose
// values are addressed by JSON Pointers (RFC 6901).
//
// The data model holds JSON values: map[string]any, []any, string,
// float64, bool and nil. The zero DataModel is an empty object.
type DataModel struct {
	root map[string]any
}

// Get returns the value at the JSON Pointer pointer, reporting whether
// it exists. The pointer "" or "/" addresses the whole data model.
// The result shares memory with the data model and must not be modified.
func (m *DataModel) Get(pointer string) (any, bool) {
	tokens, err := parsePointer(pointer)
	if err != nil {
		return nil, false
	}
	var v any = m.root
	if m.root == nil {
		v = map[string]any{}
	}
	for _, tok := range tokens {
		switch node := v.(type) {
		case map[string]any:
			var ok bool
			if v, ok = node[tok]; !ok {
				return nil, false
			}
		case []any:
			i, ok := arrayIndex(tok)
			if !ok || i >= len(node) {
				return nil, false
			}
			v = node[i]
		default:
			return nil, false
		}
	}
	return v, true
}

// Set sets the value at the JSON Pointer pointer, as an updateDataModel
// message does: it replaces an existing value, creates a missing one
// along with any missing objects and arrays on the way, and a nil value
// (JSON null) deletes it. Deleting an array element sets it to nil.
//
// The pointer "" or "/" addresses the whole data model, which must be
// set to an object or to nil, which empties it.
//
// Set stores a copy of value, converted to JSON values as by
// [encoding/json], so that values set at /user can later be addressed
// at /user/name.
func (m *DataModel) Set(pointer string, value any) error {
	tokens, err := parsePointer(pointer)
	if err != nil {
		return err
	}
	value, err = jsonValue(value)
	if err != nil {
		return fmt.Errorf("a2uistate: set %s: %w", pointer, err)
	}
	if len(tokens) == 0 {
		switch value := value.(type) {
		case nil:
			m.root = nil
		case map[string]any:
			m.root = value
		default:
			return fmt.Errorf("a2uistate: set %s: data model must be an object, not %T", pointer, value)
		}
		return nil
	}
	if value == nil {
		if _, ok := m.Get(pointer); !ok {
			return nil
		}
	}
	if m.root == nil {
		m.root = make(map[string]any)
	}
	root, err := set(m.root, tokens, value)
	if err != nil {
		return fmt.Errorf("a2uistate: set %s: %w", pointer, err)
	}
	m.root = root.(map[string]any)
	return nil
}

// set sets the value at tokens in node, an object or array, and returns
// the node, which differs from the one passed in if an array grew.
func set(node any, tokens []string, value any) (any, error) {
	tok, rest := tokens[0], tokens[1:]
	switch node := node.(type) {
	case map[string]any:
		if len(rest) == 0 {
			if value == nil {
				delete(node, tok)
			} else {
				node[tok] = value
			}
			return node, nil
		}
		child, err := container(node[tok], tok, rest[0])
		if err != nil {
			return nil, err
		}
		if node[tok], err = set(child, rest, value); err != nil {
			return nil, err
		}
		return node, nil
	case []any:
		i, ok := arrayIndex(tok)
		if !ok {
			return nil, fmt.Errorf("%q is not an array index", tok)
		}
		if i > maxArrayIndex {
			return nil, fmt.Errorf("array index %d exceeds %d", i, maxArrayIndex)
		}
		if i >= len(node) {
			node = append(node, make([]any, i+1-len(node))...)
		}
		if len(rest) == 0 {
			node[i] = value
			return node, nil
		}
		child, err := container(node[i], tok, rest[0])
		if err != nil {
			return nil, err
		}
		if node[i], err = set(child, rest, value); err != nil {
			return nil, err
		}
		return node, nil
	}
	panic("unreachable")
}

// container returns v, the value at token tok, for descending into it
// with the token next. A missing or null value becomes an empty array
// if next is an array index and an empty object otherwise.
func container(v any, tok, next string) (any, error) {
	switch v.(type) {
	case map[string]any, []any:
		return v, nil
	case nil:
		if _, ok := arrayIndex(next); ok {
			return []any{}, nil
		}
		return map[string]any{}, nil
	default:
		return nil, fmt.Errorf("value at %q is a %T, not an object or array", tok, v)
	}
}

// parsePointer returns the unescaped reference tokens of a JSON
// Pointer. As in the A2UI protocol, "/" addresses the root, like "".
func parsePointer(pointer string) ([]string, error) {
	if pointer == "" || pointer == "/" {
		return nil, nil
	}
	if pointer[0] != '/' {
		return nil, fmt.Errorf("a2uistate: invalid JSON Pointer %q", pointer)
	}
	tokens := strings.Split(pointer[1:], "/")
	for i, tok := range tokens {
		if !strings.Contains(tok, "~") {
			continue
		}
		for j := 0; j < len(tok); j++ {
			if tok[j] == '~' && (j+1 == len(tok) || tok[j+1] != '0' && tok[j+1] != '1') {
				return nil, fmt.Errorf("a2uistate: invalid JSON Pointer %q", pointer)
			}
		}
		tokens[i] = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
	}
	return tokens, nil
}

// arrayIndex returns tok as an array index: a decimal number without
// leading zeros.
func arrayIndex(tok string) (int, bool) {
	if tok == "" || len(tok) > 1 && tok[0] == '0' {
		return 0, false
	}
	for _, c := range tok {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	i, err := strconv.Atoi(tok)
	return i, err == nil
}

// jsonValue returns a copy of v made of JSON values.
func jsonValue(v any) (any, error) {
	switch v := v.(type) {
	case nil, string, float64, bool:
		return v, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, x := range v {
			x, err := jsonValue(x)
			if err != nil {
				return nil, err
			}
			out[k] = x
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			x, err := jsonValue(x)
			if err != nil {
				return nil, err
			}
			out[i] = x
		}
		return out, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
