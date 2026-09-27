package a2uischema

import _ "embed"

var (
	//go:embed schemas/v1_0/agent_to_renderer.json
	agentToRenderer []byte

	//go:embed schemas/v1_0/common_types.json
	commonTypes []byte

	//go:embed schemas/v1_0/basic_catalog.json
	basicCatalog []byte
)
