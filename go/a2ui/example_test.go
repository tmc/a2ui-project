package a2ui_test

import (
	"encoding/json"
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func Example() {
	msg := a2ui.AgentMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID: "demo",
			CatalogID: a2ui.BasicCatalogID,
		},
	}
	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
	// Output: {"version":"v1.0","createSurface":{"surfaceId":"demo","catalogId":"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"}}
}

func ExampleComponent() {
	comp := a2ui.Component{
		ID: "greeting",
		Text: &a2ui.TextComponent{
			Text:    a2ui.StringLiteral("Hello, world!"),
			Variant: a2ui.TextVariantCaption,
		},
	}
	data, _ := json.Marshal(comp)
	fmt.Println(string(data))
	// Output: {"component":"Text","id":"greeting","text":"Hello, world!","variant":"caption"}
}

func ExampleDynamicString() {
	// Literal string.
	lit := a2ui.StringLiteral("hello")
	data, _ := json.Marshal(lit)
	fmt.Println(string(data))

	// Data binding.
	bind := a2ui.StringBinding("/user/name")
	data, _ = json.Marshal(bind)
	fmt.Println(string(data))
	// Output:
	// "hello"
	// {"path":"/user/name"}
}

func ExampleDynamicNumber() {
	n := a2ui.NumberLiteral(42)
	data, _ := json.Marshal(n)
	fmt.Println(string(data))
	// Output: 42
}

func ExampleDynamicBoolean() {
	b := a2ui.BoolBinding("/settings/enabled")
	data, _ := json.Marshal(b)
	fmt.Println(string(data))
	// Output: {"path":"/settings/enabled"}
}

func ExampleIndex() {
	data, _ := json.Marshal(a2ui.Index(1))
	fmt.Println(string(data))
	// Output: {"call":"@index","args":{"offset":1}}
}

func ExampleFunctionResponse() {
	for _, r := range []a2ui.FunctionResponse{
		{FunctionCallID: "call-1", Value: 42},
		{FunctionCallID: "call-2"}, // a nil Value is a null result
		{FunctionCallID: "call-3", Error: &a2ui.FunctionError{Code: "FAILED", Message: "no network"}},
	} {
		data, _ := json.Marshal(r)
		fmt.Println(string(data))
	}
	// Output:
	// {"functionCallId":"call-1","value":42}
	// {"functionCallId":"call-2","value":null}
	// {"functionCallId":"call-3","error":{"code":"FAILED","message":"no network"}}
}

func ExampleIconNameOrPath() {
	name := a2ui.IconSearch
	svg := a2ui.StringLiteral("M0 0h24v24H0z")
	for _, icon := range []a2ui.IconNameOrPath{
		{Name: &name},
		{SVGPath: &svg},
		{Binding: &a2ui.DataBinding{Path: "/icon"}},
	} {
		data, _ := json.Marshal(icon)
		fmt.Println(string(data))
	}
	// Output:
	// "search"
	// {"svgPath":"M0 0h24v24H0z"}
	// {"path":"/icon"}
}

// Optional fields such as Placeholder are pointers. Since Go 1.26,
// new(expr) makes a pointer to a literal without a helper.
func ExampleTextFieldComponent() {
	c := a2ui.Component{
		ID: "name",
		TextField: &a2ui.TextFieldComponent{
			Label:       a2ui.StringLiteral("Name"),
			Placeholder: new(a2ui.StringLiteral("Ann Smith")),
			Value:       new(a2ui.StringBinding("/user/name")),
		},
	}
	data, _ := json.Marshal(c)
	fmt.Println(string(data))
	// Output: {"component":"TextField","id":"name","label":"Name","placeholder":"Ann Smith","value":{"path":"/user/name"}}
}
