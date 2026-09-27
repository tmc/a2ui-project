package a2uischema

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

var (
	componentType        = reflect.TypeFor[a2ui.Component]()
	customComponentType  = reflect.TypeFor[a2ui.CustomComponent]()
	functionResponseType = reflect.TypeFor[a2ui.FunctionResponse]()
	iconNameOrPathType   = reflect.TypeFor[a2ui.IconNameOrPath]()
	rawMessageType       = reflect.TypeFor[json.RawMessage]()
)

// checkFields reports a field of the JSON payload data that msgs, the
// messages decoded from it, do not define, such as a misspelled field
// or one that A2UI 1.x removed, like the returnType of a function call.
// The a2ui package decodes leniently, as encoding/json does, and the
// schemas forbid such fields, so the validator rejects them.
//
// The fields defined at each position are those of the Go type decoded
// there: the JSON names of a struct's fields, the fields of the set
// variant of a union type such as [a2ui.DynamicString], and for a
// component, its common fields, "component" and the fields of its
// type. Values of type any, such as a data model, and the properties
// of a custom component may hold any fields.
func checkFields(data []byte, msgs []a2ui.AgentMessage) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return within(err, "", "parse messages")
	}
	v := reflect.ValueOf(msgs)
	if _, ok := raw.([]any); !ok {
		v = v.Index(0)
	}
	return unknownField(raw, v, "")
}

// unknownField reports a field of the JSON value raw that v, the value
// decoded from it, does not define. Path is the JSON pointer of raw.
func unknownField(raw any, v reflect.Value, path string) error {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		obj, ok := raw.(map[string]any)
		if !ok || v.Type().Key().Kind() != reflect.String {
			return nil
		}
		for _, key := range slices.Sorted(maps.Keys(obj)) {
			elem := v.MapIndex(reflect.ValueOf(key).Convert(v.Type().Key()))
			if !elem.IsValid() {
				continue
			}
			if err := unknownField(obj[key], elem, path+pointer(key)); err != nil {
				return err
			}
		}
	case reflect.Slice:
		list, ok := raw.([]any)
		if !ok || v.Type() == rawMessageType || len(list) != v.Len() {
			return nil
		}
		for i := range list {
			if err := unknownField(list[i], v.Index(i), path+pointer(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		if obj, ok := raw.(map[string]any); ok {
			return unknownStructField(obj, v, path)
		}
	}
	// Anything else, including interface values, holds any JSON value.
	return nil
}

func unknownStructField(obj map[string]any, v reflect.Value, path string) error {
	switch v.Type() {
	case iconNameOrPathType:
		// Its decoding rejects unknown fields.
		return nil
	case functionResponseType:
		// Its decoding rejects unknown fields; check the error object.
		return unknownField(obj["error"], v.FieldByName("Error"), path+pointer("error"))
	}
	fields := jsonFields(v)
	if fields == nil {
		// A union type: the object form is its set variant.
		for i := range v.NumField() {
			if f := v.Field(i); f.Kind() == reflect.Pointer && !f.IsNil() {
				return unknownField(obj, f, path)
			}
		}
		return nil
	}
	anyField := false
	if v.Type() == componentType {
		fields["component"] = reflect.Value{}
		for i := range v.NumField() {
			f := v.Field(i)
			if v.Type().Field(i).Tag.Get("json") != "-" || f.IsNil() {
				continue
			}
			if f.Elem().Type() == customComponentType {
				anyField = true
				continue
			}
			maps.Copy(fields, jsonFields(f.Elem()))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(obj)) {
		f, ok := fields[key]
		if !ok {
			if anyField {
				continue
			}
			return invalid(ErrInvalidMessage, path+pointer(key), fmt.Sprintf("unknown field %q", key))
		}
		if !f.IsValid() {
			continue
		}
		if err := unknownField(obj[key], f, path+pointer(key)); err != nil {
			return err
		}
	}
	return nil
}

// jsonFields returns the fields of the struct v by JSON name, or nil if
// none of its fields has a JSON name in its tag, as in union types,
// which encode themselves.
func jsonFields(v reflect.Value) map[string]reflect.Value {
	var fields map[string]reflect.Value
	for i := range v.NumField() {
		name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		if fields == nil {
			fields = make(map[string]reflect.Value)
		}
		fields[name] = v.Field(i)
	}
	return fields
}
