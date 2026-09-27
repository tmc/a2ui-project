package a2a_test

import (
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2a"
)

func ExampleCreateDataPart() {
	part, err := a2a.CreateDataPart(map[string]any{
		"version": "v1.0",
		"updateDataModel": map[string]any{
			"surfaceId": "dashboard",
			"value":     map[string]any{"status": "ready"},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(part.Metadata[a2a.MIMETypeKey])
	fmt.Println(a2a.IsA2UIPart(part))
	// Output:
	// application/a2ui+json
	// true
}

func ExampleA2UIData() {
	part, err := a2a.CreateDataPart(map[string]any{"version": "v1.0", "deleteSurface": map[string]any{"surfaceId": "s1"}})
	if err != nil {
		panic(err)
	}
	data, ok := a2a.A2UIData(part)
	fmt.Println(part.Metadata[a2a.MIMETypeKey], ok, data["deleteSurface"])
	// Output:
	// application/a2ui+json true map[surfaceId:s1]
}

func ExampleNewAgentExtension() {
	ext := a2a.NewAgentExtension(a2a.AgentExtensionOptions{
		SupportedCatalogIDs: []string{"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"},
	})
	fmt.Println(ext.URI)
	fmt.Println(ext.Params[a2a.SupportedCatalogIDsKey])
	// Output:
	// https://a2ui.org/a2a-extension/a2ui/v1.0
	// [https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json]
}

func ExampleTryActivateExtension() {
	activated, version, ok := a2a.TryActivateExtension(
		[]string{"https://a2ui.org/a2a-extension/a2ui/v1.0"},
		[]string{"https://a2ui.org/a2a-extension/a2ui/v1.0"},
	)
	fmt.Println(activated)
	fmt.Println(version)
	fmt.Println(ok)
	// Output:
	// https://a2ui.org/a2a-extension/a2ui/v1.0
	// 1.0
	// true
}
