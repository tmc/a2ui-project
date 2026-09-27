package a2uistate_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

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

func ExampleEvaluator() {
	var m a2uistate.DataModel
	m.Set("/items", []map[string]any{{"name": "Ann", "count": 1}, {"name": "Bob", "count": 3}})
	e := &a2uistate.Evaluator{Data: &m}

	label := a2ui.FormatString(a2ui.StringLiteral(
		"${@index(offset: 1)}. ${name}: ${pluralize(value: ${count}, one: 'item', other: 'items')}"))
	for _, scope := range []string{"/items/0", "/items/1"} {
		s, err := e.ResolveString(label, scope)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(s)
	}
	// Output:
	// 1. Ann: item
	// 2. Bob: items
}

func ExampleEvaluator_ResolveString() {
	var m a2uistate.DataModel
	m.Set("/total", 1234.5)
	e := &a2uistate.Evaluator{Data: &m}

	fmt.Println(e.ResolveString(a2ui.StringFunc(a2ui.FunctionCall{
		Call: "formatCurrency",
		Args: map[string]any{"value": a2ui.NumberBinding("/total"), "currency": "USD"},
	}), ""))
	_, err := e.ResolveString(a2ui.StringBinding("/missing"), "")
	fmt.Println(errors.Is(err, a2uistate.ErrNoValue), err)
	// Output:
	// $1,234.50 <nil>
	// true a2uistate: no value at "/missing"
}

func ExampleEvaluator_ResolveBoolean() {
	var m a2uistate.DataModel
	m.Set("/agreed", true)
	m.Set("/email", "ann@example.com")
	e := &a2uistate.Evaluator{Data: &m}

	enabled := a2ui.And([]a2ui.DynamicBoolean{
		a2ui.BoolBinding("/agreed"),
		a2ui.BoolFunc(*a2ui.Email(a2ui.StringBinding("/email")).FunctionCall),
	})
	fmt.Println(e.ResolveBoolean(enabled, ""))
	// Output: true <nil>
}

func ExampleEvaluator_ResolveArgs() {
	var m a2uistate.DataModel
	m.Set("/id", 42)
	e := &a2uistate.Evaluator{Data: &m}

	// The evaluator does not perform actions. A renderer resolves
	// their arguments and performs them itself.
	action := a2ui.OpenURL(a2ui.FormatString(a2ui.StringLiteral("https://example.com/orders/${/id}")))
	_, err := e.ResolveValue(a2ui.ValueFunc(*action.FunctionCall), "")
	fmt.Println(errors.Is(err, a2uistate.ErrAction))
	args, err := e.ResolveArgs(*action.FunctionCall, "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(action.FunctionCall.Call, args["url"])
	// Output:
	// true
	// openUrl https://example.com/orders/42
}

func ExampleEvaluator_Check() {
	var m a2uistate.DataModel
	m.Set("/email", "ann@")
	e := &a2uistate.Evaluator{Data: &m}

	checks := []a2ui.CheckRule{
		{Condition: a2ui.Required(a2ui.ValueBinding("/email")), Message: "Email is required"},
		{Condition: a2ui.Email(a2ui.StringBinding("/email")), Message: "Enter a valid email address"},
	}
	failed, err := e.Check(checks, "")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range failed {
		fmt.Println(r.Severity, r.Message)
	}
	// Output: error Enter a valid email address
}

func ExampleBasicFunctions() {
	funcs := a2uistate.BasicFunctions()
	funcs["upper"] = func(e *a2uistate.Evaluator, scope string, args map[string]any) (any, error) {
		s, ok := args["value"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: upper: value must be a string", a2uistate.ErrInvalidArgs)
		}
		return strings.ToUpper(s), nil
	}
	var m a2uistate.DataModel
	m.Set("/name", "Ann")
	e := &a2uistate.Evaluator{Data: &m, Funcs: funcs}

	fmt.Println(e.ResolveString(a2ui.FormatString(a2ui.StringLiteral("Hello, ${upper(value: ${/name})}")), ""))
	// Output: Hello, ANN <nil>
}
