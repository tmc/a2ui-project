package a2ui

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON implements json.Marshaler for ChildList.
func (c ChildList) MarshalJSON() ([]byte, error) {
	if c.Template != nil && c.IDs != nil {
		return nil, fmt.Errorf("a2ui: ChildList has both ids and template set")
	}
	switch {
	case c.Template != nil:
		return json.Marshal(c.Template)
	case c.IDs != nil:
		return json.Marshal(c.IDs)
	default:
		return []byte("[]"), nil
	}
}

// UnmarshalJSON implements json.Unmarshaler for ChildList.
// As usual, a JSON null leaves c unchanged.
func (c *ChildList) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}
	*c = ChildList{}
	var ids []string
	if err := json.Unmarshal(data, &ids); err == nil {
		c.IDs = ids
		return nil
	}
	var t ChildTemplate
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("a2ui: unmarshal child list: %w", err)
	}
	c.Template = &t
	return nil
}

// MarshalJSON implements json.Marshaler for Action.
func (a Action) MarshalJSON() ([]byte, error) {
	type actionAlias Action
	switch countSet(a.Event != nil, a.FunctionCall != nil) {
	case 1:
		return json.Marshal(actionAlias(a))
	case 0:
		return nil, fmt.Errorf("a2ui: Action has no value set")
	default:
		return nil, fmt.Errorf("a2ui: Action has multiple values set")
	}
}

// UnmarshalJSON implements json.Unmarshaler for Action.
// As usual, a JSON null leaves a unchanged.
func (a *Action) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}
	type actionAlias Action
	var aa actionAlias
	if err := json.Unmarshal(data, &aa); err != nil {
		return fmt.Errorf("a2ui: unmarshal action: %w", err)
	}
	switch countSet(aa.Event != nil, aa.FunctionCall != nil) {
	case 1:
		*a = Action(aa)
		return nil
	case 0:
		return fmt.Errorf("a2ui: action must have event or functionCall")
	default:
		return fmt.Errorf("a2ui: action must not have both event and functionCall")
	}
}

// MarshalJSON implements json.Marshaler for IconNameOrPath.
func (i IconNameOrPath) MarshalJSON() ([]byte, error) {
	switch countSet(i.Name != nil, i.SVGPath != nil, i.Binding != nil) {
	case 1:
		switch {
		case i.Name != nil:
			return json.Marshal(string(*i.Name))
		case i.SVGPath != nil:
			return json.Marshal(struct {
				SVGPath *DynamicString `json:"svgPath"`
			}{SVGPath: i.SVGPath})
		case i.Binding != nil:
			return json.Marshal(i.Binding)
		}
	case 0:
		return nil, fmt.Errorf("a2ui: IconNameOrPath has no value set")
	default:
		return nil, fmt.Errorf("a2ui: IconNameOrPath has multiple values set")
	}
	return nil, fmt.Errorf("a2ui: IconNameOrPath has no value set")
}

// UnmarshalJSON implements json.Unmarshaler for IconNameOrPath.
// As usual, a JSON null leaves i unchanged.
func (i *IconNameOrPath) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}
	*i = IconNameOrPath{}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		name := IconName(s)
		i.Name = &name
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("a2ui: unmarshal icon name or path: %w", err)
	}
	if len(obj) != 1 {
		return fmt.Errorf("a2ui: icon name object must have exactly one of svgPath or path")
	}
	if raw, ok := obj["svgPath"]; ok {
		i.SVGPath = new(DynamicString)
		if err := json.Unmarshal(raw, i.SVGPath); err != nil {
			return fmt.Errorf("a2ui: unmarshal icon svgPath: %w", err)
		}
		return nil
	}
	raw, ok := obj["path"]
	if !ok {
		return fmt.Errorf("a2ui: icon name object must have exactly one of svgPath or path")
	}
	var path string
	if err := json.Unmarshal(raw, &path); err != nil {
		return fmt.Errorf("a2ui: unmarshal icon path: %w", err)
	}
	if path == "" {
		return fmt.Errorf("a2ui: icon path must not be empty")
	}
	i.Binding = &DataBinding{Path: path}
	return nil
}

func countSet(values ...bool) int {
	var count int
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

// isNull reports whether data is the JSON null.
func isNull(data []byte) bool {
	return string(bytes.TrimSpace(data)) == "null"
}
