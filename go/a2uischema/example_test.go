package a2uischema_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2uischema"
)

func newManager() *a2uischema.SchemaManager {
	manager, err := a2uischema.NewSchemaManager([]a2uischema.CatalogConfig{a2uischema.BasicCatalogConfig()}, false)
	if err != nil {
		panic(err)
	}
	return manager
}

func ExampleSchemaManager_GenerateSystemPrompt() {
	manager := newManager()
	prompt, err := manager.GenerateSystemPrompt(a2uischema.PromptOptions{
		RoleDescription: "You are a helpful assistant.",
		UIDescription:   "Show contact cards.",
		IncludeSchema:   true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.HasPrefix(prompt, "You are a helpful assistant."))
	fmt.Println(strings.Contains(prompt, a2uischema.A2UISchemaBlockStart))
	// Output:
	// true
	// true
}

func ExampleSchemaManager_SelectedCatalog() {
	manager := newManager()
	caps := &a2ui.RendererCapabilities{V1: &a2ui.RendererCapabilitiesV1{
		SupportedCatalogIDs: []string{"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"},
	}}
	catalog, err := manager.SelectedCatalog(caps, []string{"Text"}, nil)
	if err != nil {
		panic(err)
	}
	id, err := catalog.ID()
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	// Output:
	// https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json
}

func ExampleValidator_ValidateMessages() {
	catalog, err := newManager().SelectedCatalog(nil, nil, nil)
	if err != nil {
		panic(err)
	}
	validator := catalog.Validator()
	msgs, err := validator.ParseMessages([]byte(`{"version":"v1.0","deleteSurface":{"surfaceId":"s1"}}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(validator.ValidateMessages(msgs))
	fmt.Println(validator.ValidateMessages([]a2ui.AgentMessage{{Version: a2ui.Version, DeleteSurface: &a2ui.DeleteSurface{}}}))
	// Output:
	// <nil>
	// a2uischema: message[0]: deleteSurface.surfaceId is required
}

func ExampleValidationError() {
	catalog, err := newManager().SelectedCatalog(nil, nil, nil)
	if err != nil {
		panic(err)
	}
	err = catalog.Validator().ValidateJSON([]byte(`[
		{"version": "v1.0", "createSurface": {"surfaceId": "s1"}},
		{"version": "v1.0", "updateComponents": {"surfaceId": "s1", "components": [
			{"id": "root", "component": "Card"}
		]}}
	]`))
	var ve *a2uischema.ValidationError
	if errors.As(err, &ve) {
		fmt.Println(ve.Path)
		fmt.Println(ve.Err)
		fmt.Println(ve)
	}
	// Output:
	// /1/updateComponents/components/0/child
	// a2uischema: invalid message
	// a2uischema: message[1]: updateComponents: component[0] (root): card.child is required
}

func ExampleErrInvalidTree() {
	catalog, err := newManager().SelectedCatalog(nil, nil, nil)
	if err != nil {
		panic(err)
	}
	err = catalog.Validator().ValidateJSON([]byte(`[
		{"version": "v1.0", "createSurface": {"surfaceId": "s1"}},
		{"version": "v1.0", "updateComponents": {"surfaceId": "s1", "components": [
			{"id": "root", "component": "Card", "child": "body"}
		]}}
	]`))
	fmt.Println(errors.Is(err, a2uischema.ErrInvalidTree))
	fmt.Println(err)
	// Output:
	// true
	// a2uischema: surface "s1": component "root" references unknown component "body"
}
