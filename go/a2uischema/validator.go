package a2uischema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

var (
	jsonPointerPattern         = regexp.MustCompile(`^(?:/(?:[^~/]|~[01])*)*$`)
	relativeJSONPointerPattern = regexp.MustCompile(`^(?:[^~/]|~[01])+(?:/(?:[^~/]|~[01])*)*$`)
)

// Validator validates A2UI payloads against the selected catalog and protocol rules.
type Validator struct {
	catalog           *Catalog
	allowedComponents map[string]struct{}
	allowedFunctions  map[string]struct{}
}

// NewValidator constructs a validator for a catalog.
func NewValidator(catalog *Catalog) *Validator {
	v := &Validator{
		catalog:           catalog,
		allowedComponents: make(map[string]struct{}),
		allowedFunctions:  make(map[string]struct{}),
	}
	if catalog == nil {
		return v
	}
	if components, ok := catalog.CatalogSchema[CatalogComponentsKey].(map[string]any); ok {
		for name := range components {
			v.allowedComponents[name] = struct{}{}
		}
	}
	switch functions := catalog.CatalogSchema[CatalogFunctionsKey].(type) {
	case map[string]any:
		for name := range functions {
			v.allowedFunctions[name] = struct{}{}
		}
	case []any:
		for _, item := range functions {
			name, _ := item.(map[string]any)["name"].(string)
			if name != "" {
				v.allowedFunctions[name] = struct{}{}
			}
		}
	}
	return v
}

// ParseMessages parses a single A2UI message object or an array of them.
func (v *Validator) ParseMessages(data []byte) ([]a2ui.AgentMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, invalid(ErrInvalidMessage, "", "empty payload")
	}
	if data[0] == '[' {
		var msgs []a2ui.AgentMessage
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, within(err, "", "parse messages")
		}
		return msgs, nil
	}
	var msg a2ui.AgentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, within(err, "", "parse message")
	}
	return []a2ui.AgentMessage{msg}, nil
}

// ValidateMessages validates a batch of A2UI messages.
// A validation failure is reported as a *[ValidationError].
func (v *Validator) ValidateMessages(msgs []a2ui.AgentMessage) error {
	if len(msgs) == 0 {
		return invalid(ErrInvalidMessage, "", "no messages to validate")
	}
	// Components may reference children that arrive in later messages
	// (progressive rendering), so unknown references are only reported
	// if they remain unresolved after the last message.
	surfaceComponents := make(map[string]map[string]bool)
	pending := make(map[string][]componentRef)
	for i, msg := range msgs {
		if err := v.validateMessage(msg); err != nil {
			return within(err, pointer(i), fmt.Sprintf("message[%d]", i))
		}
		switch {
		case msg.CreateSurface != nil:
			id := msg.CreateSurface.SurfaceID
			known := make(map[string]bool)
			surfaceComponents[id] = known
			pending[id] = nil
			if len(msg.CreateSurface.Components) > 0 {
				refs, err := v.validateComponents(msg.CreateSurface.Components, nil)
				if err != nil {
					return within(err, pointer(i, "createSurface", "components"), fmt.Sprintf("message[%d]: createSurface", i))
				}
				for _, component := range msg.CreateSurface.Components {
					known[component.ID] = true
				}
				pending[id] = at(refs, pointer(i, "createSurface", "components"))
			}
		case msg.UpdateComponents != nil:
			id := msg.UpdateComponents.SurfaceID
			known := surfaceComponents[id]
			refs, err := v.validateComponents(msg.UpdateComponents.Components, known)
			if err != nil {
				return within(err, pointer(i, "updateComponents", "components"), fmt.Sprintf("message[%d]: updateComponents", i))
			}
			if known == nil {
				known = make(map[string]bool)
				surfaceComponents[id] = known
			}
			for _, component := range msg.UpdateComponents.Components {
				known[component.ID] = true
			}
			refs = at(refs, pointer(i, "updateComponents", "components"))
			pending[id] = slices.DeleteFunc(append(pending[id], refs...), func(r componentRef) bool { return known[r.to] })
		case msg.DeleteSurface != nil:
			delete(surfaceComponents, msg.DeleteSurface.SurfaceID)
			delete(pending, msg.DeleteSurface.SurfaceID)
		case msg.UpdateDataModel != nil:
			if err := validatePath(msg.UpdateDataModel.Path, true); err != nil {
				return within(err, pointer(i, "updateDataModel", "path"), fmt.Sprintf("message[%d]: updateDataModel.path", i))
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(pending)) {
		if refs := pending[id]; len(refs) > 0 {
			r := refs[0]
			return invalid(ErrInvalidTree, r.path, fmt.Sprintf("surface %q: component %q references unknown component %q", id, r.from, r.to))
		}
	}
	return nil
}

// A componentRef is a reference from one component to another.
type componentRef struct {
	from, to string
	path     string // JSON pointer of the reference
}

// at returns refs with path prefixed to their paths.
func at(refs []componentRef, path string) []componentRef {
	for i := range refs {
		refs[i].path = path + refs[i].path
	}
	return refs
}

func (v *Validator) validateMessage(msg a2ui.AgentMessage) error {
	if msg.Version != a2ui.Version {
		return invalid(ErrVersionMismatch, "/version", fmt.Sprintf("version = %q, want %q", msg.Version, a2ui.Version))
	}
	switch countSet(msg.CreateSurface != nil, msg.UpdateComponents != nil, msg.UpdateDataModel != nil, msg.DeleteSurface != nil, msg.CallRendererFunction != nil, msg.AgentFunctionResponse != nil) {
	case 1:
	case 0:
		return invalid(ErrInvalidMessage, "", "message has no payload")
	default:
		return invalid(ErrInvalidMessage, "", "message has multiple payloads")
	}
	switch {
	case msg.CreateSurface != nil:
		if msg.CreateSurface.SurfaceID == "" {
			return invalid(ErrInvalidMessage, "/createSurface/surfaceId", "createSurface.surfaceId is required")
		}
		if msg.CreateSurface.CatalogID != "" && v.catalog != nil {
			id, err := v.catalog.ID()
			if err == nil && len(v.allowedComponents) > 0 && msg.CreateSurface.CatalogID != id {
				return invalid(ErrInvalidMessage, "/createSurface/catalogId", fmt.Sprintf("createSurface.catalogId = %q, want %q", msg.CreateSurface.CatalogID, id))
			}
		}
	case msg.UpdateComponents != nil:
		if msg.UpdateComponents.SurfaceID == "" {
			return invalid(ErrInvalidMessage, "/updateComponents/surfaceId", "updateComponents.surfaceId is required")
		}
		if len(msg.UpdateComponents.Components) == 0 {
			return invalid(ErrInvalidMessage, "/updateComponents/components", "updateComponents.components must not be empty")
		}
	case msg.UpdateDataModel != nil:
		if msg.UpdateDataModel.SurfaceID == "" {
			return invalid(ErrInvalidMessage, "/updateDataModel/surfaceId", "updateDataModel.surfaceId is required")
		}
	case msg.DeleteSurface != nil:
		if msg.DeleteSurface.SurfaceID == "" {
			return invalid(ErrInvalidMessage, "/deleteSurface/surfaceId", "deleteSurface.surfaceId is required")
		}
	case msg.CallRendererFunction != nil:
		call := msg.CallRendererFunction
		if call.FunctionCallID == "" {
			return invalid(ErrInvalidMessage, "/callRendererFunction/functionCallId", "callRendererFunction.functionCallId is required")
		}
		if call.CallFunction.CatalogID == "" {
			return invalid(ErrInvalidMessage, "/callRendererFunction/callFunction/catalogId", "callRendererFunction.callFunction.catalogId is required")
		}
		if err := v.validateFunctionCall(call.CallFunction, 0); err != nil {
			return within(err, "/callRendererFunction/callFunction", "callRendererFunction.callFunction")
		}
	case msg.AgentFunctionResponse != nil:
		if err := validateFunctionResponse(*msg.AgentFunctionResponse); err != nil {
			return within(err, "/agentFunctionResponse", "agentFunctionResponse")
		}
	}
	return nil
}

// validateComponents validates components and returns the references
// to components that are neither in components nor in known.
// Error and reference paths are relative to the components array.
func (v *Validator) validateComponents(components []a2ui.Component, known map[string]bool) ([]componentRef, error) {
	ids := make(map[string]int, len(components))
	for i, component := range components {
		if err := v.validateComponent(component); err != nil {
			return nil, within(err, pointer(i), fmt.Sprintf("component[%d] (%s)", i, component.ID))
		}
		if _, ok := ids[component.ID]; ok {
			return nil, invalid(ErrInvalidTree, pointer(i, "id"), fmt.Sprintf("duplicate component id %q", component.ID))
		}
		ids[component.ID] = i
	}
	var unknown []componentRef
	graph := make(map[string][]string, len(components))
	for i, component := range components {
		graph[component.ID] = nil
		for _, ref := range componentRefs(component) {
			if _, ok := ids[ref.to]; ok {
				graph[component.ID] = append(graph[component.ID], ref.to)
				continue
			}
			if known != nil && known[ref.to] {
				continue
			}
			ref.from = component.ID
			ref.path = pointer(i) + ref.path
			unknown = append(unknown, ref)
		}
	}
	if _, ok := ids["root"]; !ok && len(known) == 0 {
		return nil, invalid(ErrInvalidTree, "", fmt.Sprintf("components must include id %q", "root"))
	}
	if _, ok := ids["root"]; ok {
		if err := validateTopology(graph, "root", ids); err != nil {
			return nil, err
		}
	}
	return unknown, nil
}

// validateComponent validates a component. Error paths are relative
// to the component.
func (v *Validator) validateComponent(component a2ui.Component) error {
	if component.ID == "" {
		return invalid(ErrInvalidMessage, "/id", "id is required")
	}
	componentType := component.ComponentType()
	if componentType == "" {
		return invalid(ErrInvalidMessage, "/component", "exactly one concrete component type must be set")
	}
	if len(v.allowedComponents) > 0 {
		if _, ok := v.allowedComponents[componentType]; !ok {
			return invalid(ErrUnknownComponent, "/component", fmt.Sprintf("component type %q is not allowed by the selected catalog", componentType))
		}
	}
	for i, check := range component.Checks {
		if err := v.validateDynamicValidationResult(check.Condition, 0); err != nil {
			return within(err, pointer("checks", i, "condition"), "check condition")
		}
	}
	if a := component.Accessibility; a != nil {
		if a.Label != nil {
			if err := v.validateDynamicString(*a.Label, 0); err != nil {
				return within(err, "/accessibility/label", "accessibility.label")
			}
		}
		if a.Description != nil {
			if err := v.validateDynamicString(*a.Description, 0); err != nil {
				return within(err, "/accessibility/description", "accessibility.description")
			}
		}
		switch a.Live {
		case "", a2ui.AccessibleLiveOff, a2ui.AccessibleLivePolite, a2ui.AccessibleLiveAssertive:
		default:
			return invalid(ErrInvalidMessage, "/accessibility/live", fmt.Sprintf("accessibility.live = %q is not allowed", a.Live))
		}
		if a.Hidden != nil {
			if err := v.validateDynamicBoolean(*a.Hidden, 0); err != nil {
				return within(err, "/accessibility/hidden", "accessibility.hidden")
			}
		}
	}
	switch {
	case component.Text != nil:
		return v.validateTextComponent(*component.Text)
	case component.Image != nil:
		return v.validateImageComponent(*component.Image)
	case component.Icon != nil:
		return v.validateIconComponent(*component.Icon)
	case component.Video != nil:
		return v.validateVideoComponent(*component.Video)
	case component.AudioPlayer != nil:
		return v.validateAudioPlayerComponent(*component.AudioPlayer)
	case component.Row != nil:
		return v.validateContainerChildren(component.Row.Children)
	case component.Column != nil:
		return v.validateContainerChildren(component.Column.Children)
	case component.List != nil:
		return v.validateContainerChildren(component.List.Children)
	case component.Card != nil:
		if component.Card.Child == "" {
			return invalid(ErrInvalidMessage, "/child", "card.child is required")
		}
	case component.Tabs != nil:
		if len(component.Tabs.Tabs) == 0 {
			return invalid(ErrInvalidMessage, "/tabs", "tabs.tabs must not be empty")
		}
		for i, tab := range component.Tabs.Tabs {
			if tab.Child == "" {
				return invalid(ErrInvalidMessage, pointer("tabs", i, "child"), "tabs.child is required")
			}
			if err := v.validateDynamicString(tab.Title, 0); err != nil {
				return within(err, pointer("tabs", i, "title"), "tabs.title")
			}
		}
	case component.Modal != nil:
		if component.Modal.Content == "" || component.Modal.Trigger == "" {
			path := "/content"
			if component.Modal.Trigger == "" {
				path = "/trigger"
			}
			return invalid(ErrInvalidMessage, path, "modal.content and modal.trigger are required")
		}
	case component.Divider != nil:
		return nil
	case component.Button != nil:
		if component.Button.Child == "" {
			return invalid(ErrInvalidMessage, "/child", "button.child is required")
		}
		if err := v.validateAction(component.Button.Action, 0); err != nil {
			return within(err, "/action", "button.action")
		}
	case component.TextField != nil:
		c := component.TextField
		if err := v.validateDynamicString(c.Label, 0); err != nil {
			return within(err, "/label", "textField.label")
		}
		if c.Value != nil {
			if err := v.validateDynamicString(*c.Value, 0); err != nil {
				return within(err, "/value", "textField.value")
			}
		}
		if c.Placeholder != nil {
			if err := v.validateDynamicString(*c.Placeholder, 0); err != nil {
				return within(err, "/placeholder", "textField.placeholder")
			}
		}
	case component.CheckBox != nil:
		if err := v.validateDynamicString(component.CheckBox.Label, 0); err != nil {
			return within(err, "/label", "checkBox.label")
		}
		if err := v.validateDynamicBoolean(component.CheckBox.Value, 0); err != nil {
			return within(err, "/value", "checkBox.value")
		}
	case component.ChoicePicker != nil:
		c := component.ChoicePicker
		if len(c.Options) == 0 {
			return invalid(ErrInvalidMessage, "/options", "choicePicker.options must not be empty")
		}
		if c.Label != nil {
			if err := v.validateDynamicString(*c.Label, 0); err != nil {
				return within(err, "/label", "choicePicker.label")
			}
		}
		for i, option := range c.Options {
			if option.Value == "" {
				return invalid(ErrInvalidMessage, pointer("options", i, "value"), "choicePicker option value is required")
			}
			if err := v.validateDynamicString(option.Label, 0); err != nil {
				return within(err, pointer("options", i, "label"), "choicePicker option label")
			}
		}
		if err := v.validateDynamicStringList(c.Value, 0); err != nil {
			return within(err, "/value", "choicePicker.value")
		}
	case component.Slider != nil:
		if err := v.validateDynamicNumber(component.Slider.Value, 0); err != nil {
			return within(err, "/value", "slider.value")
		}
		if component.Slider.Label != nil {
			if err := v.validateDynamicString(*component.Slider.Label, 0); err != nil {
				return within(err, "/label", "slider.label")
			}
		}
	case component.DateTimeInput != nil:
		c := component.DateTimeInput
		if err := v.validateDynamicString(c.Value, 0); err != nil {
			return within(err, "/value", "dateTimeInput.value")
		}
		for _, f := range []struct {
			name  string
			value *a2ui.DynamicString
		}{
			{"label", c.Label},
			{"max", c.Max},
			{"min", c.Min},
		} {
			if f.value != nil {
				if err := v.validateDynamicString(*f.value, 0); err != nil {
					return within(err, pointer(f.name), "dateTimeInput."+f.name)
				}
			}
		}
	}
	return nil
}

func (v *Validator) validateTextComponent(component a2ui.TextComponent) error {
	return within(v.validateDynamicString(component.Text, 0), "/text", "")
}

func (v *Validator) validateImageComponent(component a2ui.ImageComponent) error {
	if err := v.validateDynamicString(component.URL, 0); err != nil {
		return within(err, "/url", "")
	}
	if component.Description != nil {
		return within(v.validateDynamicString(*component.Description, 0), "/description", "")
	}
	return nil
}

func (v *Validator) validateIconComponent(component a2ui.IconComponent) error {
	name := component.Name
	switch {
	case name.SVGPath != nil:
		return within(v.validateDynamicString(*name.SVGPath, 0), "/name/path", "")
	case name.Name == nil && name.Binding == nil:
		return invalid(ErrInvalidMessage, "/name", "icon.name is required")
	}
	return nil
}

func (v *Validator) validateVideoComponent(component a2ui.VideoComponent) error {
	if err := v.validateDynamicString(component.URL, 0); err != nil {
		return within(err, "/url", "")
	}
	if component.PosterURL != nil {
		return within(v.validateDynamicString(*component.PosterURL, 0), "/posterUrl", "")
	}
	return nil
}

func (v *Validator) validateAudioPlayerComponent(component a2ui.AudioPlayerComponent) error {
	if err := v.validateDynamicString(component.URL, 0); err != nil {
		return within(err, "/url", "")
	}
	if component.Description != nil {
		return within(v.validateDynamicString(*component.Description, 0), "/description", "")
	}
	return nil
}

func (v *Validator) validateContainerChildren(children a2ui.ChildList) error {
	if len(children.IDs) == 0 && children.Template == nil {
		return invalid(ErrInvalidMessage, "/children", "children must not be empty")
	}
	if children.Template != nil {
		if children.Template.ComponentID == "" {
			return invalid(ErrInvalidMessage, "/children/componentId", "children.template.componentId is required")
		}
		if err := validatePath(children.Template.Path, false); err != nil {
			return within(err, "/children/path", "children.template.path")
		}
	}
	return nil
}

func (v *Validator) validateAction(action a2ui.Action, depth int) error {
	switch {
	case action.Event != nil && action.FunctionCall != nil:
		return invalid(ErrInvalidMessage, "", "action must not have both event and functionCall")
	case action.Event != nil:
		if action.Event.Name == "" {
			return invalid(ErrInvalidMessage, "/event/name", "event.name is required")
		}
		if action.Event.UserMessage != nil {
			if err := v.validateDynamicString(*action.Event.UserMessage, depth+1); err != nil {
				return within(err, "/event/userMessage", "event.userMessage")
			}
		}
		for _, key := range slices.Sorted(maps.Keys(action.Event.Context)) {
			if err := v.validateDynamicValue(action.Event.Context[key], depth+1); err != nil {
				return within(err, pointer("event", "context", key), fmt.Sprintf("event.context[%q]", key))
			}
		}
	case action.FunctionCall != nil:
		return within(v.validateFunctionCall(*action.FunctionCall, depth+1), "/functionCall", "")
	default:
		return invalid(ErrInvalidMessage, "", "action must have event or functionCall")
	}
	return nil
}

func (v *Validator) validateDynamicString(value a2ui.DynamicString, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "dynamic string has no value")
	}
}

func (v *Validator) validateDynamicNumber(value a2ui.DynamicNumber, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "dynamic number has no value")
	}
}

func (v *Validator) validateDynamicBoolean(value a2ui.DynamicBoolean, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "dynamic boolean has no value")
	}
}

func (v *Validator) validateDynamicStringList(value a2ui.DynamicStringList, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "dynamic string list has no value")
	}
}

func (v *Validator) validateDynamicValue(value a2ui.DynamicValue, depth int) error {
	switch {
	case value.String != nil, value.Number != nil, value.Bool != nil, value.Array != nil:
		return nil
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "dynamic value has no value")
	}
}

func (v *Validator) validateDynamicValidationResult(value a2ui.DynamicValidationResult, depth int) error {
	switch {
	case value.Binding != nil && value.FunctionCall != nil:
		return invalid(ErrInvalidMessage, "", "condition must not have both path and call")
	case value.Binding != nil:
		return within(validatePath(value.Binding.Path, false), "/path", "")
	case value.FunctionCall != nil:
		return v.validateFunctionCall(*value.FunctionCall, depth+1)
	default:
		return invalid(ErrInvalidMessage, "", "condition has no value")
	}
}

// indexFunction is the v1.0 system function available in list templates.
const indexFunction = "@index"

func (v *Validator) validateFunctionCall(call a2ui.FunctionCall, depth int) error {
	if depth > 32 {
		return invalid(ErrInvalidMessage, "", "function call recursion depth exceeded")
	}
	if call.Call == "" {
		return invalid(ErrInvalidMessage, "/call", "function call name is required")
	}
	if len(v.allowedFunctions) > 0 && call.Call != indexFunction {
		if _, ok := v.allowedFunctions[call.Call]; !ok {
			return invalid(ErrUnknownFunction, "/call", fmt.Sprintf("unknown function %q", call.Call))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(call.Args)) {
		if err := v.validateFunctionArg(call.Args[key], depth+1); err != nil {
			return within(err, pointer("args", key), fmt.Sprintf("function arg %q", key))
		}
	}
	return nil
}

func (v *Validator) validateFunctionArg(arg any, depth int) error {
	switch value := arg.(type) {
	case nil, string, bool, float64, int:
		return nil
	case []string:
		return nil
	case []any:
		for i, item := range value {
			if err := v.validateFunctionArg(item, depth+1); err != nil {
				return within(err, pointer(i), fmt.Sprintf("[%d]", i))
			}
		}
		return nil
	case map[string]any:
		if _, ok := value["path"]; ok {
			path, _ := value["path"].(string)
			return within(validatePath(path, false), "/path", "")
		}
		if _, ok := value["call"]; ok {
			data, err := json.Marshal(value)
			if err != nil {
				return within(err, "", "")
			}
			var call a2ui.FunctionCall
			if err := json.Unmarshal(data, &call); err != nil {
				return within(err, "", "")
			}
			return v.validateFunctionCall(call, depth+1)
		}
		for _, key := range slices.Sorted(maps.Keys(value)) {
			if err := v.validateFunctionArg(value[key], depth+1); err != nil {
				return within(err, pointer(key), key)
			}
		}
		return nil
	case a2ui.DynamicValue:
		return v.validateDynamicValue(value, depth+1)
	case a2ui.DynamicString:
		return v.validateDynamicString(value, depth+1)
	case a2ui.DynamicNumber:
		return v.validateDynamicNumber(value, depth+1)
	case a2ui.DynamicBoolean:
		return v.validateDynamicBoolean(value, depth+1)
	case a2ui.DynamicStringList:
		return v.validateDynamicStringList(value, depth+1)
	default:
		return nil
	}
}

func validateFunctionResponse(response a2ui.FunctionResponse) error {
	if response.FunctionCallID == "" {
		return invalid(ErrInvalidMessage, "/functionCallId", "functionCallId is required")
	}
	switch {
	case response.Error == nil:
		return nil
	case response.Value != nil:
		return invalid(ErrInvalidMessage, "", "must not have both value and error")
	case response.Error.Code == "":
		return invalid(ErrInvalidMessage, "/error/code", "error.code is required")
	case response.Error.Message == "":
		return invalid(ErrInvalidMessage, "/error/message", "error.message is required")
	}
	return nil
}

// componentRefs returns the references from component to other
// components, with paths relative to the component.
func componentRefs(component a2ui.Component) []componentRef {
	switch {
	case component.Button != nil:
		return []componentRef{{to: component.Button.Child, path: "/child"}}
	case component.Card != nil:
		return []componentRef{{to: component.Card.Child, path: "/child"}}
	case component.Column != nil:
		return childListRefs(component.Column.Children)
	case component.List != nil:
		return childListRefs(component.List.Children)
	case component.Row != nil:
		return childListRefs(component.Row.Children)
	case component.Modal != nil:
		return []componentRef{
			{to: component.Modal.Trigger, path: "/trigger"},
			{to: component.Modal.Content, path: "/content"},
		}
	case component.Tabs != nil:
		refs := make([]componentRef, 0, len(component.Tabs.Tabs))
		for i, tab := range component.Tabs.Tabs {
			refs = append(refs, componentRef{to: tab.Child, path: pointer("tabs", i, "child")})
		}
		return refs
	default:
		return nil
	}
}

func childListRefs(children a2ui.ChildList) []componentRef {
	if children.Template != nil {
		return []componentRef{{to: children.Template.ComponentID, path: "/children/componentId"}}
	}
	refs := make([]componentRef, 0, len(children.IDs))
	for i, id := range children.IDs {
		refs = append(refs, componentRef{to: id, path: pointer("children", i)})
	}
	return refs
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

// ValidateJSON parses and validates a raw JSON payload, a single
// message object or an array of them. A failure is reported as a
// *[ValidationError].
func (v *Validator) ValidateJSON(data []byte) error {
	msgs, err := v.ParseMessages(data)
	if err != nil {
		return err
	}
	err = v.ValidateMessages(msgs)
	if e, ok := err.(*ValidationError); ok && bytes.TrimSpace(data)[0] != '[' {
		// Make the path relative to the single message.
		e.Path = strings.TrimPrefix(e.Path, "/0")
	}
	return err
}

// ValidateExample validates either a raw message payload or an example file
// with a top-level messages array.
func (v *Validator) ValidateExample(data []byte) error {
	err := v.ValidateJSON(data)
	if err == nil {
		return nil
	}
	var example struct {
		Messages json.RawMessage `json:"messages"`
	}
	if json.Unmarshal(data, &example) != nil || len(bytes.TrimSpace(example.Messages)) == 0 {
		return err
	}
	return v.ValidateJSON(example.Messages)
}

// validateTopology checks that the components in graph form a tree
// from root. Ids maps component ids to their index in the components
// array, to which error paths are relative.
func validateTopology(graph map[string][]string, root string, ids map[string]int) error {
	seen := make(map[string]bool, len(graph))
	stack := make(map[string]bool, len(graph))
	var visit func(string) error
	visit = func(node string) error {
		if stack[node] {
			return invalid(ErrInvalidTree, pointer(ids[node]), fmt.Sprintf("cycle detected at component %q", node))
		}
		if seen[node] {
			return nil
		}
		seen[node] = true
		stack[node] = true
		for _, child := range graph[node] {
			if err := visit(child); err != nil {
				return err
			}
		}
		delete(stack, node)
		return nil
	}
	if err := visit(root); err != nil {
		return err
	}
	if len(seen) != len(graph) {
		first := -1
		for id, i := range ids {
			if !seen[id] && (first < 0 || i < first) {
				first = i
			}
		}
		return invalid(ErrInvalidTree, pointer(first), "orphaned components detected")
	}
	return nil
}

// validatePath checks that path is a JSON Pointer or a relative JSON
// Pointer. Error paths are relative to the path value.
func validatePath(path string, allowEmpty bool) error {
	if path == "" {
		if allowEmpty {
			return nil
		}
		return invalid(ErrInvalidMessage, "", "path is required")
	}
	if path == "/" {
		return nil
	}
	if !jsonPointerPattern.MatchString(path) && !relativeJSONPointerPattern.MatchString(path) {
		return invalid(ErrInvalidMessage, "", fmt.Sprintf("invalid JSON Pointer %q", path))
	}
	return nil
}
