package a2uistate_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2uistate"
)

func Example() {
	var msgs []a2ui.AgentMessage
	err := json.Unmarshal([]byte(`[
		{"version": "v1.0", "createSurface": {
			"surfaceId": "main",
			"catalogId": "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json",
			"components": [{"id": "root", "component": "Text", "text": {"path": "/greeting"}}],
			"dataModel": {"greeting": "Hello"}
		}},
		{"version": "v1.0", "updateDataModel": {"surfaceId": "main", "path": "/greeting", "value": "Hello, world"}}
	]`), &msgs)
	if err != nil {
		log.Fatal(err)
	}

	var surfaces a2uistate.Surfaces
	if err := surfaces.Apply(msgs...); err != nil {
		log.Fatal(err)
	}
	s, _ := surfaces.Surface("main")
	root, _ := s.Root()
	text, _ := s.Data().ResolveString(root.Text.Text, "")
	fmt.Println(text)
	// Output: Hello, world
}

func ExampleSurface_Apply() {
	s := a2uistate.NewSurface("main")
	err := s.Apply(a2ui.AgentMessage{CreateSurface: &a2ui.CreateSurface{
		SurfaceID: "main",
		CatalogID: a2ui.BasicCatalogID,
		Components: []a2ui.Component{
			{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Hi")}},
		},
	}})
	if err != nil {
		log.Fatal(err)
	}
	root, _ := s.Root()
	fmt.Println(root.ComponentType(), s.CatalogID() == a2ui.BasicCatalogID)

	err = s.Apply(a2ui.AgentMessage{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "other"}})
	fmt.Println(err)
	// Output:
	// Text true
	// a2uistate: surface "main": deleteSurface for surface "other"
}

func ExampleSurfaces_Apply() {
	var surfaces a2uistate.Surfaces
	err := surfaces.Apply(
		a2ui.AgentMessage{CreateSurface: &a2ui.CreateSurface{SurfaceID: "a"}},
		a2ui.AgentMessage{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "a"}},
		a2ui.AgentMessage{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "a", Path: "/x", Value: 1}},
	)
	fmt.Println(err)
	// Output: a2uistate: updateDataModel for unknown surface "a"
}

func ExampleDataModel_Set() {
	var m a2uistate.DataModel
	m.Set("/user/name", "Ann")
	m.Set("/user/tags/1", "admin")
	m.Set("/draft", true)
	m.Set("/draft", nil)
	data, _ := m.Get("")
	out, _ := json.Marshal(data)
	fmt.Println(string(out))
	// Output: {"user":{"name":"Ann","tags":[null,"admin"]}}
}

func ExampleDataModel_Get() {
	var m a2uistate.DataModel
	m.Set("", map[string]any{"items": []any{"a", "b"}})
	fmt.Println(m.Get("/items/1"))
	fmt.Println(m.Get("/items/2"))
	// Output:
	// b true
	// <nil> false
}

func ExampleResolvePath() {
	fmt.Println(a2uistate.ResolvePath("name", "/users/0"))
	fmt.Println(a2uistate.ResolvePath("/title", "/users/0"))
	// Output:
	// /users/0/name
	// /title
}

func ExampleDataModel_ResolveString() {
	var m a2uistate.DataModel
	m.Set("/users", []map[string]any{{"name": "Ann", "age": 41}})

	// Inside a list template, bindings resolve relative to the item.
	for _, path := range []string{"name", "age", "/missing"} {
		fmt.Println(m.ResolveString(a2ui.StringBinding(path), "/users/0"))
	}
	// Output:
	// Ann true
	// 41 true
	//  false
}

func ExampleDataModel_ResolveNumber() {
	var m a2uistate.DataModel
	m.Set("/volume", 0.5)
	fmt.Println(m.ResolveNumber(a2ui.NumberBinding("/volume"), ""))
	// Output: 0.5 true
}

func ExampleDataModel_ResolveBoolean() {
	var m a2uistate.DataModel
	m.Set("/agreed", true)
	fmt.Println(m.ResolveBoolean(a2ui.BoolBinding("/agreed"), ""))
	// Output: true true
}

func ExampleDataModel_ResolveStringList() {
	var m a2uistate.DataModel
	m.Set("/selected", []string{"red", "blue"})
	fmt.Println(m.ResolveStringList(a2ui.StringListBinding("/selected"), ""))
	// Output: [red blue] true
}

func ExampleDataModel_ResolveValue() {
	var m a2uistate.DataModel
	m.Set("/count", 3)
	fmt.Println(m.ResolveValue(a2ui.ValueBinding("/count"), ""))
	// Output: 3 true
}
