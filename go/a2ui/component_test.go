package a2ui

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const basicExamplesDir = "testdata/v1_0/catalogs/basic/examples"

func TestLoginFormComponents(t *testing.T) {
	data, err := os.ReadFile(basicExamplesDir + "/09_login-form.json")
	if err != nil {
		t.Fatal(err)
	}

	var example struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}

	// Second message is updateComponents.
	var msg AgentMessage
	if err := json.Unmarshal(example.Messages[1], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.UpdateComponents == nil {
		t.Fatal("expected updateComponents")
	}

	components := msg.UpdateComponents.Components
	byID := make(map[string]*Component, len(components))
	for i := range components {
		byID[components[i].ID] = &components[i]
	}

	t.Run("TextField", func(t *testing.T) {
		c, ok := byID["email_field"]
		if !ok {
			t.Fatal("missing email_field")
		}
		if c.ComponentType() != "TextField" {
			t.Fatalf("type = %q, want TextField", c.ComponentType())
		}
		if c.TextField == nil {
			t.Fatal("TextField is nil")
		}
		if c.TextField.Label.Literal == nil || *c.TextField.Label.Literal != "Email" {
			t.Fatalf("label = %+v, want literal Email", c.TextField.Label)
		}
		if c.TextField.Value == nil || c.TextField.Value.Binding == nil || c.TextField.Value.Binding.Path != "/email" {
			t.Fatal("value should bind to /email")
		}
	})

	t.Run("Button", func(t *testing.T) {
		c, ok := byID["login_btn"]
		if !ok {
			t.Fatal("missing login_btn")
		}
		if c.ComponentType() != "Button" {
			t.Fatalf("type = %q, want Button", c.ComponentType())
		}
		if c.Button.Child != "login_btn_text" {
			t.Fatalf("child = %q", c.Button.Child)
		}
		if c.Button.Action.Event == nil {
			t.Fatal("expected event action")
		}
		if c.Button.Action.Event.Name != "login" {
			t.Fatalf("event name = %q", c.Button.Action.Event.Name)
		}
	})

	t.Run("Column", func(t *testing.T) {
		c, ok := byID["main_column"]
		if !ok {
			t.Fatal("missing main_column")
		}
		if c.ComponentType() != "Column" {
			t.Fatalf("type = %q, want Column", c.ComponentType())
		}
		if len(c.Column.Children.IDs) != 6 {
			t.Fatalf("children = %d, want 6", len(c.Column.Children.IDs))
		}
	})

	t.Run("CheckRule", func(t *testing.T) {
		c := byID["email_field"]
		if len(c.Checks) != 2 {
			t.Fatalf("checks = %d, want 2", len(c.Checks))
		}
		if c.Checks[0].Message != "Email is required" {
			t.Fatalf("message = %q", c.Checks[0].Message)
		}
		if c.Checks[0].Condition.FunctionCall == nil {
			t.Fatal("expected function call condition")
		}
		if c.Checks[0].Condition.FunctionCall.Call != "required" {
			t.Fatalf("call = %q", c.Checks[0].Condition.FunctionCall.Call)
		}
	})

	t.Run("Card", func(t *testing.T) {
		c, ok := byID["root"]
		if !ok {
			t.Fatal("missing root")
		}
		if c.ComponentType() != "Card" {
			t.Fatalf("type = %q, want Card", c.ComponentType())
		}
		if c.Card.Child != "main_column" {
			t.Fatalf("child = %q", c.Card.Child)
		}
	})
}

func TestComponentTypeDiscriminator(t *testing.T) {
	tests := []struct {
		name string
		comp Component
		want string
	}{
		{"Text", Component{Text: &TextComponent{}}, "Text"},
		{"Button", Component{Button: &ButtonComponent{}}, "Button"},
		{"Column", Component{Column: &ColumnComponent{}}, "Column"},
		{"Row", Component{Row: &RowComponent{}}, "Row"},
		{"Card", Component{Card: &CardComponent{}}, "Card"},
		{"Image", Component{Image: &ImageComponent{}}, "Image"},
		{"Icon", Component{Icon: &IconComponent{}}, "Icon"},
		{"TextField", Component{TextField: &TextFieldComponent{}}, "TextField"},
		{"CheckBox", Component{CheckBox: &CheckBoxComponent{}}, "CheckBox"},
		{"Divider", Component{Divider: &DividerComponent{}}, "Divider"},
		{"Slider", Component{Slider: &SliderComponent{}}, "Slider"},
		{"Tabs", Component{Tabs: &TabsComponent{}}, "Tabs"},
		{"Modal", Component{Modal: &ModalComponent{}}, "Modal"},
		{"List", Component{List: &ListComponent{}}, "List"},
		{"multiple", Component{Text: &TextComponent{}, Button: &ButtonComponent{}}, ""},
		{"empty", Component{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.comp.ComponentType(); got != tt.want {
				t.Fatalf("ComponentType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComponentRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "text",
			json: `{"component":"Text","id":"t1","text":"hello","variant":"h1"}`,
		},
		{
			name: "button_with_event",
			json: `{"component":"Button","id":"b1","child":"b1-text","action":{"event":{"name":"click"}}}`,
		},
		{
			name: "column",
			json: `{"component":"Column","id":"c1","children":["a","b","c"],"align":"center"}`,
		},
		{
			name: "text_with_binding",
			json: `{"component":"Text","id":"t2","text":{"path":"/name"}}`,
		},
		{
			name: "custom",
			json: `{"component":"Gauge","id":"g1","catalogId":"https://example.com/catalog.json","value":{"path":"/level"},"max":10,"weight":1}`,
		},
		{
			name: "custom_no_properties",
			json: `{"component":"Spacer","id":"s1"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Component
			if err := json.Unmarshal([]byte(tt.json), &c); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			roundTrip(t, c, tt.json)
		})
	}
}

func TestComponentCustom(t *testing.T) {
	var c Component
	if err := json.Unmarshal([]byte(`{"id":"g1","component":"Gauge","value":3}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Custom == nil || c.Custom.Type != "Gauge" || string(c.Custom.Properties["value"]) != "3" || len(c.Custom.Properties) != 1 {
		t.Fatalf("Custom = %+v, want Gauge with value 3", c.Custom)
	}
	if got := c.ComponentType(); got != "Gauge" {
		t.Errorf("ComponentType() = %q, want Gauge", got)
	}
}

func TestComponentUnmarshalNoType(t *testing.T) {
	var c Component
	err := json.Unmarshal([]byte(`{"id":"x"}`), &c)
	if err == nil || !strings.HasPrefix(err.Error(), "a2ui: ") {
		t.Fatalf("Unmarshal = %v, want a2ui: error", err)
	}
}

func TestComponentMarshalRejectsInvalidConcreteTypes(t *testing.T) {
	tests := []struct {
		name string
		comp Component
	}{
		{
			name: "none",
			comp: Component{ID: "empty"},
		},
		{
			name: "multiple",
			comp: Component{
				ID:     "bad",
				Text:   &TextComponent{Text: StringLiteral("hello")},
				Button: &ButtonComponent{Action: Action{Event: &EventAction{Name: "click"}}, Child: "child"},
			},
		},
		{
			name: "custom_and_text",
			comp: Component{
				ID:     "bad",
				Text:   &TextComponent{Text: StringLiteral("hello")},
				Custom: &CustomComponent{Type: "Gauge"},
			},
		},
		{
			name: "custom_no_type",
			comp: Component{ID: "bad", Custom: &CustomComponent{}},
		},
		{
			name: "custom_defined_type",
			comp: Component{ID: "bad", Custom: &CustomComponent{Type: "Text"}},
		},
		{
			name: "custom_reserved_property",
			comp: Component{ID: "bad", Custom: &CustomComponent{
				Type:       "Gauge",
				Properties: map[string]json.RawMessage{"id": json.RawMessage(`"other"`)},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := json.Marshal(tt.comp); err == nil {
				t.Fatal("expected marshal error, got nil")
			}
		})
	}
}

func TestAllExamplesUnmarshal(t *testing.T) {
	entries, err := os.ReadDir(basicExamplesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			data, err := os.ReadFile(basicExamplesDir + "/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			var example struct {
				Messages []json.RawMessage `json:"messages"`
			}
			if err := json.Unmarshal(data, &example); err != nil {
				t.Fatal(err)
			}
			for i, raw := range example.Messages {
				var msg AgentMessage
				if err := json.Unmarshal(raw, &msg); err != nil {
					t.Fatalf("message[%d]: unmarshal: %v", i, err)
				}
				remarshaled, err := json.Marshal(msg)
				if err != nil {
					t.Fatalf("message[%d]: re-marshal: %v", i, err)
				}
				var got, want any
				if err := json.Unmarshal(remarshaled, &got); err != nil {
					t.Fatalf("message[%d]: unmarshal re-marshaled: %v", i, err)
				}
				if err := json.Unmarshal(raw, &want); err != nil {
					t.Fatalf("message[%d]: unmarshal original: %v", i, err)
				}
				normalizeJSON(got)
				normalizeJSON(want)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("message[%d]: round-trip mismatch", i)
				}
			}
		})
	}
}
