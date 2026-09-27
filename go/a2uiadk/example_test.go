package a2uiadk_test

import (
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2uiadk"
	"github.com/a2ui-project/a2ui/go/a2uischema"
)

func ExampleSendA2UIJSONToClientTool_Run() {
	manager, _ := a2uischema.NewSchemaManager([]a2uischema.CatalogConfig{a2uischema.BasicCatalogConfig()}, false)
	catalog, _ := manager.SelectedCatalog(nil, nil, nil)
	tool := a2uiadk.NewSendA2UIJSONToClientTool(catalog.Validator())

	result := tool.Run(map[string]any{
		a2uiadk.A2UIJSONArgName: `[
			{"version":"v1.0","createSurface":{"surfaceId":"demo","catalogId":"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"}},
			{"version":"v1.0","updateComponents":{"surfaceId":"demo","components":[{"component":"Text","id":"root","text":"hello"}]}}
		]`,
	}, nil)
	_, ok := result[a2uiadk.ValidatedJSONKey]
	fmt.Println(ok)
	// Output: true
}
