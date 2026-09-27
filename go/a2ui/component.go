package a2ui

import (
	"encoding/json"
	"fmt"
)

// Component represents any A2UI component in the component tree.
// Exactly one of the concrete type fields is non-nil. A component whose
// type this package does not define, such as one from a custom or
// inline catalog, is held in Custom.
//
// MarshalJSON/UnmarshalJSON in zz_component_marshal.go handle
// serialization, using the "component" field as a discriminator.
type Component struct {
	ID            string                   `json:"id"`
	CatalogID     string                   `json:"catalogId,omitempty"`
	Accessibility *AccessibilityAttributes `json:"accessibility,omitempty"`
	Metadata      *Metadata                `json:"metadata,omitempty"`
	Weight        *float64                 `json:"weight,omitempty"`
	Checks        []CheckRule              `json:"checks,omitempty"`

	// Concrete type fields (exactly one non-nil).
	Text          *TextComponent          `json:"-"`
	Image         *ImageComponent         `json:"-"`
	Icon          *IconComponent          `json:"-"`
	Video         *VideoComponent         `json:"-"`
	AudioPlayer   *AudioPlayerComponent   `json:"-"`
	Row           *RowComponent           `json:"-"`
	Column        *ColumnComponent        `json:"-"`
	List          *ListComponent          `json:"-"`
	Card          *CardComponent          `json:"-"`
	Tabs          *TabsComponent          `json:"-"`
	Modal         *ModalComponent         `json:"-"`
	Divider       *DividerComponent       `json:"-"`
	Button        *ButtonComponent        `json:"-"`
	TextField     *TextFieldComponent     `json:"-"`
	CheckBox      *CheckBoxComponent      `json:"-"`
	ChoicePicker  *ChoicePickerComponent  `json:"-"`
	Slider        *SliderComponent        `json:"-"`
	DateTimeInput *DateTimeInputComponent `json:"-"`
	Custom        *CustomComponent        `json:"-"`
}

// A CustomComponent holds a component whose type this package does not
// define, such as one from a custom or inline catalog.
type CustomComponent struct {
	// Type is the component type, the "component" field on the wire.
	// It must not be a type that has its own field in [Component].
	Type string

	// Properties holds the component-specific fields, those other than
	// the common fields of [Component] and "component".
	Properties map[string]json.RawMessage
}

// MarshalJSON encodes the properties of c as a JSON object.
// It reports an error if c's Type is empty or has its own field in
// [Component], or if a property has the name of a common field.
func (c *CustomComponent) MarshalJSON() ([]byte, error) {
	if c.Type == "" {
		return nil, fmt.Errorf("a2ui: custom component has no type")
	}
	if isDefinedComponentType(c.Type) {
		return nil, fmt.Errorf("a2ui: custom component has defined type %q", c.Type)
	}
	for _, key := range commonKeys {
		if _, ok := c.Properties[key]; ok {
			return nil, fmt.Errorf("a2ui: custom component %s: property %q is reserved", c.Type, key)
		}
	}
	if c.Properties == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(c.Properties)
}

// commonKeys lists the JSON names of the fields shared by all components.
var commonKeys = []string{"id", "component", "catalogId", "accessibility", "metadata", "weight", "checks"}

func (c Component) componentData() (string, any, int) {
	var (
		componentType string
		specific      any
		count         int
	)
	set := func(typ string, value any) {
		componentType = typ
		specific = value
		count++
	}
	if c.Text != nil {
		set("Text", c.Text)
	}
	if c.Image != nil {
		set("Image", c.Image)
	}
	if c.Icon != nil {
		set("Icon", c.Icon)
	}
	if c.Video != nil {
		set("Video", c.Video)
	}
	if c.AudioPlayer != nil {
		set("AudioPlayer", c.AudioPlayer)
	}
	if c.Row != nil {
		set("Row", c.Row)
	}
	if c.Column != nil {
		set("Column", c.Column)
	}
	if c.List != nil {
		set("List", c.List)
	}
	if c.Card != nil {
		set("Card", c.Card)
	}
	if c.Tabs != nil {
		set("Tabs", c.Tabs)
	}
	if c.Modal != nil {
		set("Modal", c.Modal)
	}
	if c.Divider != nil {
		set("Divider", c.Divider)
	}
	if c.Button != nil {
		set("Button", c.Button)
	}
	if c.TextField != nil {
		set("TextField", c.TextField)
	}
	if c.CheckBox != nil {
		set("CheckBox", c.CheckBox)
	}
	if c.ChoicePicker != nil {
		set("ChoicePicker", c.ChoicePicker)
	}
	if c.Slider != nil {
		set("Slider", c.Slider)
	}
	if c.DateTimeInput != nil {
		set("DateTimeInput", c.DateTimeInput)
	}
	if c.Custom != nil {
		set(c.Custom.Type, c.Custom)
	}
	return componentType, specific, count
}

// ComponentType returns the discriminator string (e.g. "Text", "Button").
func (c Component) ComponentType() string {
	componentType, _, count := c.componentData()
	if count != 1 {
		return ""
	}
	return componentType
}

// componentCommon holds the fields shared by all components.
type componentCommon struct {
	ID            string                   `json:"id"`
	CatalogID     string                   `json:"catalogId,omitempty"`
	Accessibility *AccessibilityAttributes `json:"accessibility,omitempty"`
	Metadata      *Metadata                `json:"metadata,omitempty"`
	Weight        *float64                 `json:"weight,omitempty"`
	Checks        []CheckRule              `json:"checks,omitempty"`
}

func (c Component) common() componentCommon {
	return componentCommon{
		ID:            c.ID,
		CatalogID:     c.CatalogID,
		Accessibility: c.Accessibility,
		Metadata:      c.Metadata,
		Weight:        c.Weight,
		Checks:        c.Checks,
	}
}

func (c *Component) setCommon(cm componentCommon) {
	c.ID = cm.ID
	c.CatalogID = cm.CatalogID
	c.Accessibility = cm.Accessibility
	c.Metadata = cm.Metadata
	c.Weight = cm.Weight
	c.Checks = cm.Checks
}
