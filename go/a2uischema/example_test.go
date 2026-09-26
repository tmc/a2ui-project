package a2uischema_test

import (
	"fmt"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2ui/v09"
	"github.com/a2ui-project/a2ui/go/a2uischema"
)

func newManager(version a2uischema.Version) *a2uischema.SchemaManager {
	basic, err := a2uischema.BasicCatalogConfig(version)
	if err != nil {
		panic(err)
	}
	manager, err := a2uischema.NewSchemaManager(version, []a2uischema.CatalogConfig{basic}, false)
	if err != nil {
		panic(err)
	}
	return manager
}

func ExampleSchemaManager_GenerateSystemPrompt() {
	manager := newManager(a2uischema.Version1)
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
	manager := newManager(a2uischema.Version1)
	caps := &a2ui.RendererCapabilities{V1: &a2ui.RendererCapabilitiesV1{
		SupportedCatalogIDs: []string{"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"},
	}}
	catalog, err := manager.SelectedCatalog(caps, []string{"Text"}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(catalog.Version)
	// Output:
	// v1.0
}

func ExampleSchemaManager_SelectedCatalogV09() {
	manager := newManager(a2uischema.Version09)
	catalog, err := manager.SelectedCatalogV09(&v09.ClientCapabilities{}, nil, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(catalog.Version)
	// Output:
	// v0.9
}

func ExampleValidator_ValidateMessages() {
	catalog, err := newManager(a2uischema.Version1).SelectedCatalog(nil, nil, nil)
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

func ExampleValidator_ValidateMessagesV09() {
	catalog, err := newManager(a2uischema.Version09).SelectedCatalog(nil, nil, nil)
	if err != nil {
		panic(err)
	}
	validator := catalog.Validator()
	msgs, err := validator.ParseMessagesV09([]byte(`{"version":"v0.9","deleteSurface":{"surfaceId":"s1"}}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(validator.ValidateMessagesV09(msgs))
	// Output:
	// <nil>
}
