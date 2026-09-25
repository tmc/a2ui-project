package a2uischema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func (v *Validator) parseMessagesV1(data []byte) ([]a2ui.AgentMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("a2uischema: empty payload")
	}
	if data[0] == '[' {
		var msgs []a2ui.AgentMessage
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, fmt.Errorf("a2uischema: parse messages: %w", err)
		}
		return msgs, nil
	}
	var msg a2ui.AgentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("a2uischema: parse message: %w", err)
	}
	return []a2ui.AgentMessage{msg}, nil
}

func (v *Validator) validateMessagesV1(msgs []a2ui.AgentMessage) error {
	if len(msgs) == 0 {
		return fmt.Errorf("a2uischema: no messages to validate")
	}
	// Components may reference children that arrive in later messages
	// (progressive rendering), so unknown references are only reported
	// if they remain unresolved after the last message.
	surfaceComponents := make(map[string]map[string]bool)
	pending := make(map[string][]componentRef)
	for i, msg := range msgs {
		if err := v.validateMessageV1(msg); err != nil {
			return fmt.Errorf("a2uischema: message[%d]: %w", i, err)
		}
		switch {
		case msg.CreateSurface != nil:
			id := msg.CreateSurface.SurfaceID
			known := make(map[string]bool)
			surfaceComponents[id] = known
			pending[id] = nil
			if len(msg.CreateSurface.Components) > 0 {
				refs, err := v.validateComponentsV1(msg.CreateSurface.Components, nil)
				if err != nil {
					return fmt.Errorf("createSurface: %w", err)
				}
				for _, component := range msg.CreateSurface.Components {
					known[component.ID] = true
				}
				pending[id] = refs
			}
		case msg.UpdateComponents != nil:
			id := msg.UpdateComponents.SurfaceID
			known := surfaceComponents[id]
			refs, err := v.validateComponentsV1(msg.UpdateComponents.Components, known)
			if err != nil {
				return fmt.Errorf("updateComponents: %w", err)
			}
			if known == nil {
				known = make(map[string]bool)
				surfaceComponents[id] = known
			}
			for _, component := range msg.UpdateComponents.Components {
				known[component.ID] = true
			}
			pending[id] = slices.DeleteFunc(append(pending[id], refs...), func(r componentRef) bool { return known[r.to] })
		case msg.DeleteSurface != nil:
			delete(surfaceComponents, msg.DeleteSurface.SurfaceID)
			delete(pending, msg.DeleteSurface.SurfaceID)
		case msg.UpdateDataModel != nil:
			if err := validatePath(msg.UpdateDataModel.Path, true); err != nil {
				return fmt.Errorf("updateDataModel.path: %w", err)
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(pending)) {
		if refs := pending[id]; len(refs) > 0 {
			r := refs[0]
			return fmt.Errorf("a2uischema: surface %q: %w", id, validationError(ValidationUnknownComponentRef, "", r.from, r.to, "", fmt.Sprintf("component %q references unknown component %q", r.from, r.to)))
		}
	}
	return nil
}

// A componentRef is a reference from one component to another.
type componentRef struct {
	from, to string
}

func (v *Validator) validateMessageV1(msg a2ui.AgentMessage) error {
	wantVersion := Version1
	if v.catalog != nil {
		wantVersion = v.catalog.Version
	}
	if msg.Version != string(wantVersion) {
		return fmt.Errorf("version = %q, want %q", msg.Version, wantVersion)
	}
	switch countSetV1(msg.CreateSurface != nil, msg.UpdateComponents != nil, msg.UpdateDataModel != nil, msg.DeleteSurface != nil, msg.CallRendererFunction != nil, msg.AgentFunctionResponse != nil) {
	case 1:
	case 0:
		return fmt.Errorf("message has no payload")
	default:
		return fmt.Errorf("message has multiple payloads")
	}
	switch {
	case msg.CreateSurface != nil:
		if msg.CreateSurface.SurfaceID == "" {
			return fmt.Errorf("createSurface.surfaceId is required")
		}
		if msg.CreateSurface.CatalogID != "" && v.catalog != nil {
			id, err := v.catalog.ID()
			if err == nil && len(v.allowedComponents) > 0 && msg.CreateSurface.CatalogID != id {
				return fmt.Errorf("createSurface.catalogId = %q, want %q", msg.CreateSurface.CatalogID, id)
			}
		}
	case msg.UpdateComponents != nil:
		if msg.UpdateComponents.SurfaceID == "" {
			return fmt.Errorf("updateComponents.surfaceId is required")
		}
		if len(msg.UpdateComponents.Components) == 0 {
			return fmt.Errorf("updateComponents.components must not be empty")
		}
	case msg.UpdateDataModel != nil:
		if msg.UpdateDataModel.SurfaceID == "" {
			return fmt.Errorf("updateDataModel.surfaceId is required")
		}
	case msg.DeleteSurface != nil:
		if msg.DeleteSurface.SurfaceID == "" {
			return fmt.Errorf("deleteSurface.surfaceId is required")
		}
	case msg.CallRendererFunction != nil:
		call := msg.CallRendererFunction
		if call.FunctionCallID == "" {
			return fmt.Errorf("callRendererFunction.functionCallId is required")
		}
		if call.CallFunction.CatalogID == "" {
			return fmt.Errorf("callRendererFunction.callFunction.catalogId is required")
		}
		if err := v.validateFunctionCallV1(call.CallFunction, 0); err != nil {
			return fmt.Errorf("callRendererFunction.callFunction: %w", err)
		}
	case msg.AgentFunctionResponse != nil:
		if err := validateFunctionResponseV1(*msg.AgentFunctionResponse); err != nil {
			return fmt.Errorf("agentFunctionResponse: %w", err)
		}
	}
	return nil
}

// validateComponentsV1 validates components and returns the references
// to components that are neither in components nor in known.
func (v *Validator) validateComponentsV1(components []a2ui.Component, known map[string]bool) ([]componentRef, error) {
	ids := make(map[string]int, len(components))
	for i, component := range components {
		if err := v.validateComponentV1(component); err != nil {
			return nil, fmt.Errorf("component[%d] (%s): %w", i, component.ID, err)
		}
		if _, ok := ids[component.ID]; ok {
			return nil, validationError(ValidationDuplicateComponent, "", component.ID, "", "", fmt.Sprintf("duplicate component id %q", component.ID))
		}
		ids[component.ID] = i
	}
	var unknown []componentRef
	graph := make(map[string][]string, len(components))
	for _, component := range components {
		refs, err := componentRefsV1(component)
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", component.ID, err)
		}
		graph[component.ID] = nil
		for _, ref := range refs {
			if _, ok := ids[ref]; ok {
				graph[component.ID] = append(graph[component.ID], ref)
				continue
			}
			if known != nil && known[ref] {
				continue
			}
			unknown = append(unknown, componentRef{component.ID, ref})
		}
	}
	if _, ok := ids["root"]; !ok && len(known) == 0 {
		return nil, validationError(ValidationMissingRootComponent, "", "root", "", "", fmt.Sprintf("components must include id %q", "root"))
	}
	if _, ok := ids["root"]; ok {
		if err := validateTopology(graph, "root"); err != nil {
			return nil, err
		}
	}
	return unknown, nil
}

func (v *Validator) validateComponentV1(component a2ui.Component) error {
	if component.ID == "" {
		return fmt.Errorf("id is required")
	}
	componentType := component.ComponentType()
	if componentType == "" {
		return fmt.Errorf("exactly one concrete component type must be set")
	}
	if len(v.allowedComponents) > 0 {
		if _, ok := v.allowedComponents[componentType]; !ok {
			return validationError(ValidationUnknownComponentType, "", component.ID, "", "", fmt.Sprintf("component type %q is not allowed by the selected catalog", componentType))
		}
	}
	for _, check := range component.Checks {
		if err := v.validateDynamicValidationResultV1(check.Condition, 0); err != nil {
			return fmt.Errorf("check condition: %w", err)
		}
	}
	if component.Accessibility != nil {
		if component.Accessibility.Label != nil {
			if err := v.validateDynamicStringV1(*component.Accessibility.Label, 0); err != nil {
				return fmt.Errorf("accessibility.label: %w", err)
			}
		}
		if component.Accessibility.Description != nil {
			if err := v.validateDynamicStringV1(*component.Accessibility.Description, 0); err != nil {
				return fmt.Errorf("accessibility.description: %w", err)
			}
		}
		switch component.Accessibility.Live {
		case "", a2ui.AccessibleLiveOff, a2ui.AccessibleLivePolite, a2ui.AccessibleLiveAssertive:
		default:
			return fmt.Errorf("accessibility.live = %q is not allowed", component.Accessibility.Live)
		}
		if component.Accessibility.Hidden != nil {
			if err := v.validateDynamicBooleanV1(*component.Accessibility.Hidden, 0); err != nil {
				return fmt.Errorf("accessibility.hidden: %w", err)
			}
		}
	}
	switch {
	case component.Text != nil:
		return v.validateTextComponentV1(*component.Text)
	case component.Image != nil:
		return v.validateImageComponentV1(*component.Image)
	case component.Icon != nil:
		return v.validateIconComponentV1(*component.Icon)
	case component.Video != nil:
		return v.validateVideoComponentV1(*component.Video)
	case component.AudioPlayer != nil:
		return v.validateAudioPlayerComponentV1(*component.AudioPlayer)
	case component.Row != nil:
		return v.validateContainerChildrenV1(component.Row.Children)
	case component.Column != nil:
		return v.validateContainerChildrenV1(component.Column.Children)
	case component.List != nil:
		return v.validateContainerChildrenV1(component.List.Children)
	case component.Card != nil:
		if component.Card.Child == "" {
			return fmt.Errorf("card.child is required")
		}
	case component.Tabs != nil:
		if len(component.Tabs.Tabs) == 0 {
			return fmt.Errorf("tabs.tabs must not be empty")
		}
		for _, tab := range component.Tabs.Tabs {
			if tab.Child == "" {
				return fmt.Errorf("tabs.child is required")
			}
			if err := v.validateDynamicStringV1(tab.Title, 0); err != nil {
				return fmt.Errorf("tabs.title: %w", err)
			}
		}
	case component.Modal != nil:
		if component.Modal.Content == "" || component.Modal.Trigger == "" {
			return fmt.Errorf("modal.content and modal.trigger are required")
		}
	case component.Divider != nil:
		return nil
	case component.Button != nil:
		if component.Button.Child == "" {
			return fmt.Errorf("button.child is required")
		}
		if err := v.validateActionV1(component.Button.Action, 0); err != nil {
			return fmt.Errorf("button.action: %w", err)
		}
	case component.TextField != nil:
		if err := v.validateDynamicStringV1(component.TextField.Label, 0); err != nil {
			return fmt.Errorf("textField.label: %w", err)
		}
		if component.TextField.Value != nil {
			if err := v.validateDynamicStringV1(*component.TextField.Value, 0); err != nil {
				return fmt.Errorf("textField.value: %w", err)
			}
		}
		if component.TextField.Placeholder != nil {
			if err := v.validateDynamicStringV1(*component.TextField.Placeholder, 0); err != nil {
				return fmt.Errorf("textField.placeholder: %w", err)
			}
		}
	case component.CheckBox != nil:
		if err := v.validateDynamicStringV1(component.CheckBox.Label, 0); err != nil {
			return fmt.Errorf("checkBox.label: %w", err)
		}
		if err := v.validateDynamicBooleanV1(component.CheckBox.Value, 0); err != nil {
			return fmt.Errorf("checkBox.value: %w", err)
		}
	case component.ChoicePicker != nil:
		if len(component.ChoicePicker.Options) == 0 {
			return fmt.Errorf("choicePicker.options must not be empty")
		}
		if component.ChoicePicker.Label != nil {
			if err := v.validateDynamicStringV1(*component.ChoicePicker.Label, 0); err != nil {
				return fmt.Errorf("choicePicker.label: %w", err)
			}
		}
		for _, option := range component.ChoicePicker.Options {
			if option.Value == "" {
				return fmt.Errorf("choicePicker option value is required")
			}
			if err := v.validateDynamicStringV1(option.Label, 0); err != nil {
				return fmt.Errorf("choicePicker option label: %w", err)
			}
		}
		if err := v.validateDynamicStringListV1(component.ChoicePicker.Value, 0); err != nil {
			return fmt.Errorf("choicePicker.value: %w", err)
		}
	case component.Slider != nil:
		if err := v.validateDynamicNumberV1(component.Slider.Value, 0); err != nil {
			return fmt.Errorf("slider.value: %w", err)
		}
		if component.Slider.Label != nil {
			if err := v.validateDynamicStringV1(*component.Slider.Label, 0); err != nil {
				return fmt.Errorf("slider.label: %w", err)
			}
		}
	case component.DateTimeInput != nil:
		if err := v.validateDynamicStringV1(component.DateTimeInput.Value, 0); err != nil {
			return fmt.Errorf("dateTimeInput.value: %w", err)
		}
		for name, value := range map[string]*a2ui.DynamicString{
			"label": component.DateTimeInput.Label,
			"max":   component.DateTimeInput.Max,
			"min":   component.DateTimeInput.Min,
		} {
			if value != nil {
				if err := v.validateDynamicStringV1(*value, 0); err != nil {
					return fmt.Errorf("dateTimeInput.%s: %w", name, err)
				}
			}
		}
	}
	return nil
}

func (v *Validator) validateTextComponentV1(component a2ui.TextComponent) error {
	return v.validateDynamicStringV1(component.Text, 0)
}

func (v *Validator) validateImageComponentV1(component a2ui.ImageComponent) error {
	if err := v.validateDynamicStringV1(component.URL, 0); err != nil {
		return err
	}
	if component.Description != nil {
		if err := v.validateDynamicStringV1(*component.Description, 0); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateIconComponentV1(component a2ui.IconComponent) error {
	if component.Name.Name == nil && component.Name.Path == nil {
		return fmt.Errorf("icon.name is required")
	}
	return nil
}

func (v *Validator) validateVideoComponentV1(component a2ui.VideoComponent) error {
	if err := v.validateDynamicStringV1(component.URL, 0); err != nil {
		return err
	}
	if component.PosterURL != nil {
		return v.validateDynamicStringV1(*component.PosterURL, 0)
	}
	return nil
}

func (v *Validator) validateAudioPlayerComponentV1(component a2ui.AudioPlayerComponent) error {
	if err := v.validateDynamicStringV1(component.URL, 0); err != nil {
		return err
	}
	if component.Description != nil {
		return v.validateDynamicStringV1(*component.Description, 0)
	}
	return nil
}

func (v *Validator) validateContainerChildrenV1(children a2ui.ChildList) error {
	if len(children.IDs) == 0 && children.Template == nil {
		return fmt.Errorf("children must not be empty")
	}
	if children.Template != nil {
		if children.Template.ComponentID == "" {
			return fmt.Errorf("children.template.componentId is required")
		}
		if err := validatePath(children.Template.Path, false); err != nil {
			return fmt.Errorf("children.template.path: %w", err)
		}
	}
	return nil
}

func (v *Validator) validateActionV1(action a2ui.Action, depth int) error {
	switch {
	case action.Event != nil && action.FunctionCall != nil:
		return fmt.Errorf("action must not have both event and functionCall")
	case action.Event != nil:
		if action.Event.Name == "" {
			return fmt.Errorf("event.name is required")
		}
		if action.Event.UserMessage != nil {
			if err := v.validateDynamicStringV1(*action.Event.UserMessage, depth+1); err != nil {
				return fmt.Errorf("event.userMessage: %w", err)
			}
		}
		for key, value := range action.Event.Context {
			if err := v.validateDynamicValueV1(value, depth+1); err != nil {
				return fmt.Errorf("event.context[%q]: %w", key, err)
			}
		}
	case action.FunctionCall != nil:
		if err := v.validateFunctionCallV1(*action.FunctionCall, depth+1); err != nil {
			return err
		}
	default:
		return fmt.Errorf("action must have event or functionCall")
	}
	return nil
}

func (v *Validator) validateDynamicStringV1(value a2ui.DynamicString, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("dynamic string has no value")
	}
}

func (v *Validator) validateDynamicNumberV1(value a2ui.DynamicNumber, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("dynamic number has no value")
	}
}

func (v *Validator) validateDynamicBooleanV1(value a2ui.DynamicBoolean, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("dynamic boolean has no value")
	}
}

func (v *Validator) validateDynamicStringListV1(value a2ui.DynamicStringList, depth int) error {
	switch {
	case value.Literal != nil:
		return nil
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("dynamic string list has no value")
	}
}

func (v *Validator) validateDynamicValueV1(value a2ui.DynamicValue, depth int) error {
	switch {
	case value.String != nil, value.Number != nil, value.Bool != nil, value.Array != nil:
		return nil
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("dynamic value has no value")
	}
}

func (v *Validator) validateDynamicValidationResultV1(value a2ui.DynamicValidationResult, depth int) error {
	switch {
	case value.Binding != nil && value.FunctionCall != nil:
		return fmt.Errorf("condition must not have both path and call")
	case value.Binding != nil:
		return validatePath(value.Binding.Path, false)
	case value.FunctionCall != nil:
		return v.validateFunctionCallV1(*value.FunctionCall, depth+1)
	default:
		return fmt.Errorf("condition has no value")
	}
}

// indexFunction is the v1.0 system function available in list templates.
const indexFunction = "@index"

func (v *Validator) validateFunctionCallV1(call a2ui.FunctionCall, depth int) error {
	if depth > 32 {
		return fmt.Errorf("function call recursion depth exceeded")
	}
	if call.Call == "" {
		return fmt.Errorf("function call name is required")
	}
	if len(v.allowedFunctions) > 0 && call.Call != indexFunction {
		if _, ok := v.allowedFunctions[call.Call]; !ok {
			return validationError(ValidationUnknownFunction, "", "", "", call.Call, fmt.Sprintf("unknown function %q", call.Call))
		}
	}
	for key, arg := range call.Args {
		if err := v.validateFunctionArgV1(arg, depth+1); err != nil {
			return fmt.Errorf("function arg %q: %w", key, err)
		}
	}
	return nil
}

func (v *Validator) validateFunctionArgV1(arg any, depth int) error {
	switch value := arg.(type) {
	case nil, string, bool, float64, int:
		return nil
	case []string:
		return nil
	case []any:
		for i, item := range value {
			if err := v.validateFunctionArgV1(item, depth+1); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		return nil
	case map[string]any:
		if _, ok := value["path"]; ok {
			path, _ := value["path"].(string)
			return validatePath(path, false)
		}
		if _, ok := value["call"]; ok {
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var call a2ui.FunctionCall
			if err := json.Unmarshal(data, &call); err != nil {
				return err
			}
			return v.validateFunctionCallV1(call, depth+1)
		}
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			if err := v.validateFunctionArgV1(value[key], depth+1); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
		return nil
	case a2ui.DynamicValue:
		return v.validateDynamicValueV1(value, depth+1)
	case a2ui.DynamicString:
		return v.validateDynamicStringV1(value, depth+1)
	case a2ui.DynamicNumber:
		return v.validateDynamicNumberV1(value, depth+1)
	case a2ui.DynamicBoolean:
		return v.validateDynamicBooleanV1(value, depth+1)
	case a2ui.DynamicStringList:
		return v.validateDynamicStringListV1(value, depth+1)
	default:
		return nil
	}
}

func validateFunctionResponseV1(response a2ui.FunctionResponse) error {
	if response.FunctionCallID == "" {
		return fmt.Errorf("functionCallId is required")
	}
	switch {
	case response.Error == nil:
		return nil
	case response.Value != nil:
		return fmt.Errorf("must not have both value and error")
	case response.Error.Code == "":
		return fmt.Errorf("error.code is required")
	case response.Error.Message == "":
		return fmt.Errorf("error.message is required")
	}
	return nil
}

func componentRefsV1(component a2ui.Component) ([]string, error) {
	switch {
	case component.Button != nil:
		return []string{component.Button.Child}, nil
	case component.Card != nil:
		return []string{component.Card.Child}, nil
	case component.Column != nil:
		return childListRefsV1(component.Column.Children)
	case component.List != nil:
		return childListRefsV1(component.List.Children)
	case component.Row != nil:
		return childListRefsV1(component.Row.Children)
	case component.Modal != nil:
		return []string{component.Modal.Trigger, component.Modal.Content}, nil
	case component.Tabs != nil:
		refs := make([]string, 0, len(component.Tabs.Tabs))
		for _, tab := range component.Tabs.Tabs {
			refs = append(refs, tab.Child)
		}
		return refs, nil
	default:
		return nil, nil
	}
}

func childListRefsV1(children a2ui.ChildList) ([]string, error) {
	if children.Template != nil {
		return []string{children.Template.ComponentID}, nil
	}
	return append([]string(nil), children.IDs...), nil
}

func countSetV1(values ...bool) int {
	var count int
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
