package a2uischema

// The embedded schemas are copies of the specification and basic catalogs.
//go:generate cp ../../specification/v0_9/json/server_to_client.json ../../specification/v0_9/json/common_types.json schemas/v0_9/
//go:generate cp ../../specification/v0_9/catalogs/basic/catalog.json schemas/v0_9/basic_catalog.json
//go:generate cp ../../specification/v0_9/catalogs/basic/rules.txt schemas/v0_9/basic_catalog_rules.txt
//go:generate cp ../../specification/v0_9_1/json/server_to_client.json ../../specification/v0_9_1/json/common_types.json schemas/v0_9_1/
//go:generate cp ../../specification/v0_9_1/catalogs/basic/catalog.json schemas/v0_9_1/basic_catalog.json
//go:generate cp ../../specification/v0_9_1/catalogs/basic/rules.txt schemas/v0_9_1/basic_catalog_rules.txt
//go:generate cp ../../specification/v1_0/json/agent_to_renderer.json ../../specification/v1_0/json/common_types.json schemas/v1_0/
//go:generate cp ../../catalogs/basic/v1/catalog.json schemas/v1_0/basic_catalog.json
//go:generate rm -rf testdata
//go:generate mkdir -p testdata/v1_0/basic
//go:generate cp -R ../../catalogs/basic/v1/examples testdata/v1_0/basic/examples
