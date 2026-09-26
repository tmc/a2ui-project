package a2ui

// DataBinding references a value in the data model by JSON Pointer path.
type DataBinding struct {
	Path string `json:"path"`
}

// FunctionCall invokes a named catalog function.
// CatalogID overrides the surface's default catalog.
type FunctionCall struct {
	Call      string         `json:"call"`
	CatalogID string         `json:"catalogId,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
}

// ChildList is either a static list of component IDs or a dynamic template.
// Exactly one of IDs or Template is set.
type ChildList struct {
	IDs      []string
	Template *ChildTemplate
}

// ChildTemplate generates a dynamic list of children from a data model list.
type ChildTemplate struct {
	ComponentID string `json:"componentId"`
	Path        string `json:"path"`
}

// CheckRule is a single validation check applied to an input component.
// Condition evaluates to a [ValidationResult]; Message is a fallback
// error message.
type CheckRule struct {
	Condition DynamicValidationResult `json:"condition"`
	Message   string                  `json:"message,omitempty"`
}

// AccessibilityAttributes enhance accessibility for assistive technologies.
type AccessibilityAttributes struct {
	Label       *DynamicString  `json:"label,omitempty"`
	Description *DynamicString  `json:"description,omitempty"`
	Live        AccessibleLive  `json:"live,omitempty"`
	Hidden      *DynamicBoolean `json:"hidden,omitempty"`
}

// AccessibleLive controls screen reader announcements for dynamic updates.
type AccessibleLive string

const (
	AccessibleLiveOff       AccessibleLive = "off"
	AccessibleLivePolite    AccessibleLive = "polite"
	AccessibleLiveAssertive AccessibleLive = "assertive"
)

// Metadata carries extension metadata on surfaces, components, and actions.
// Keys starting with "a2ui_" are reserved for official extensions.
type Metadata struct {
	Extensions map[string]any `json:"extensions,omitempty"`
}

// Action is an interaction handler that either triggers an agent-side event
// or executes a function. Exactly one field is non-nil.
type Action struct {
	Event        *EventAction  `json:"event,omitempty"`
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`
}

// EventAction triggers an agent-side event.
type EventAction struct {
	Name        string                  `json:"name"`
	UserMessage *DynamicString          `json:"userMessage,omitempty"`
	Context     map[string]DynamicValue `json:"context,omitempty"`
}

// IconNameOrPath is a well-known icon name, a custom SVG path, or a
// binding to an icon name in the data model. Exactly one field is non-nil.
type IconNameOrPath struct {
	Name    *IconName
	SVGPath *DynamicString
	Binding *DataBinding
}

// Index returns a call to the @index system function, the 0-based index
// of the current item in a template list, plus offset.
func Index(offset float64) DynamicNumber {
	call := FunctionCall{Call: "@index"}
	if offset != 0 {
		call.Args = map[string]any{"offset": offset}
	}
	return NumberFunc(call)
}
