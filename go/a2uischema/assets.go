package a2uischema

import _ "embed"

var (
	//go:embed schemas/v0_9/server_to_client.json
	serverToClientV09 []byte

	//go:embed schemas/v0_9/common_types.json
	commonTypesV09 []byte

	//go:embed schemas/v0_9/basic_catalog.json
	basicCatalogV09 []byte

	//go:embed schemas/v0_9/basic_catalog_rules.txt
	basicCatalogRulesV09 string

	//go:embed schemas/v0_9_1/server_to_client.json
	serverToClientV091 []byte

	//go:embed schemas/v0_9_1/common_types.json
	commonTypesV091 []byte

	//go:embed schemas/v0_9_1/basic_catalog.json
	basicCatalogV091 []byte

	//go:embed schemas/v0_9_1/basic_catalog_rules.txt
	basicCatalogRulesV091 string

	//go:embed schemas/v1_0/agent_to_renderer.json
	agentToRendererV1 []byte

	//go:embed schemas/v1_0/common_types.json
	commonTypesV1 []byte

	//go:embed schemas/v1_0/basic_catalog.json
	basicCatalogV1 []byte
)
