package v10_test

import (
	"encoding/json"
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui/v10"
)

func Example() {
	msg := v10.AgentMessage{
		Version: v10.Version,
		CreateSurface: &v10.CreateSurface{
			SurfaceID: "demo",
			CatalogID: "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json",
		},
	}
	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
	// Output: {"version":"v1.0","createSurface":{"surfaceId":"demo","catalogId":"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"}}
}

func ExampleComponent() {
	comp := v10.Component{
		ID: "greeting",
		Text: &v10.TextComponent{
			Text:    v10.StringLiteral("Hello, world!"),
			Variant: v10.TextVariantCaption,
		},
	}
	data, _ := json.Marshal(comp)
	fmt.Println(string(data))
	// Output: {"component":"Text","id":"greeting","text":"Hello, world!","variant":"caption"}
}

func ExampleDynamicString() {
	// Literal string.
	lit := v10.StringLiteral("hello")
	data, _ := json.Marshal(lit)
	fmt.Println(string(data))

	// Data binding.
	bind := v10.StringBinding("/user/name")
	data, _ = json.Marshal(bind)
	fmt.Println(string(data))
	// Output:
	// "hello"
	// {"path":"/user/name"}
}

func ExampleDynamicNumber() {
	n := v10.NumberLiteral(42)
	data, _ := json.Marshal(n)
	fmt.Println(string(data))
	// Output: 42
}

func ExampleDynamicBoolean() {
	b := v10.BoolBinding("/settings/enabled")
	data, _ := json.Marshal(b)
	fmt.Println(string(data))
	// Output: {"path":"/settings/enabled"}
}

func ExampleIndex() {
	data, _ := json.Marshal(v10.Index(1))
	fmt.Println(string(data))
	// Output: {"call":"@index","args":{"offset":1}}
}
