package a2uischema

import (
	"encoding/json"
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
	v09 "github.com/a2ui-project/a2ui/go/a2ui/v09"
	a2uiv091 "github.com/a2ui-project/a2ui/go/a2ui/v091"
)

// SchemaModifier can rewrite a decoded schema before it is used.
type SchemaModifier func(schema map[string]any) error

// SchemaManager manages schemas, catalogs, and prompt rendering.
type SchemaManager struct {
	version               Version
	acceptsInlineCatalogs bool
	messageSchema         map[string]any
	commonTypesSchema     map[string]any
	supportedCatalogs     []*Catalog
	catalogExamplePaths   map[string]string
	schemaModifiers       []SchemaModifier
}

// NewSchemaManager constructs a schema manager.
func NewSchemaManager(version Version, catalogs []CatalogConfig, acceptsInlineCatalogs bool, schemaModifiers ...SchemaModifier) (*SchemaManager, error) {
	messageSchema, commonSchema, err := embeddedSchemas(version)
	if err != nil {
		return nil, err
	}
	manager := &SchemaManager{
		version:               version,
		acceptsInlineCatalogs: acceptsInlineCatalogs,
		messageSchema:         messageSchema,
		commonTypesSchema:     commonSchema,
		catalogExamplePaths:   make(map[string]string),
		schemaModifiers:       schemaModifiers,
	}
	for _, cfg := range catalogs {
		data, err := cfg.Provider.Load()
		if err != nil {
			return nil, fmt.Errorf("a2uischema: load catalog %q: %w", cfg.Name, err)
		}
		messageSchemaData, err := marshalJSON(messageSchema)
		if err != nil {
			return nil, fmt.Errorf("a2uischema: encode message schema: %w", err)
		}
		commonSchemaData, err := marshalJSON(commonSchema)
		if err != nil {
			return nil, fmt.Errorf("a2uischema: encode common_types schema: %w", err)
		}
		catalog, err := newCatalog(version, cfg.Name, messageSchemaData, commonSchemaData, data)
		if err != nil {
			return nil, err
		}
		if err := manager.applyModifiers(catalog.MessageSchema); err != nil {
			return nil, err
		}
		if err := manager.applyModifiers(catalog.CommonTypesSchema); err != nil {
			return nil, err
		}
		if err := manager.applyModifiers(catalog.CatalogSchema); err != nil {
			return nil, err
		}
		manager.supportedCatalogs = append(manager.supportedCatalogs, catalog)
		if cfg.ExamplesPath != "" {
			id, err := catalog.ID()
			if err != nil {
				return nil, err
			}
			manager.catalogExamplePaths[id] = cfg.ExamplesPath
		}
	}
	return manager, nil
}

// SupportedCatalogIDs returns the agent-supported catalog identifiers.
func (m *SchemaManager) SupportedCatalogIDs() []string {
	ids := make([]string, 0, len(m.supportedCatalogs))
	for _, catalog := range m.supportedCatalogs {
		id, err := catalog.ID()
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// SelectedCatalog selects the catalog for A2UI 1.x renderer capabilities
// and prunes it to the allowed components and messages.
// A nil caps selects the default catalog for any manager version;
// otherwise the manager must be for version 1.x.
func (m *SchemaManager) SelectedCatalog(caps *a2ui.RendererCapabilities, allowedComponents, allowedMessages []string) (*Catalog, error) {
	var selected *Catalog
	var err error
	switch {
	case caps == nil:
		selected, err = m.defaultCatalog()
	case m.version != Version1:
		err = fmt.Errorf("a2uischema: manager version = %q, got 1.x capabilities", m.version)
	default:
		selected, err = m.selectCatalogV1(caps)
	}
	if err != nil {
		return nil, err
	}
	return selected.WithPruning(allowedComponents, allowedMessages)
}

// SelectedCatalogV09 is like [SchemaManager.SelectedCatalog] for A2UI v0.9
// client capabilities. The manager must be for version v0.9 or v0.9.1.
func (m *SchemaManager) SelectedCatalogV09(caps *v09.ClientCapabilities, allowedComponents, allowedMessages []string) (*Catalog, error) {
	if !isV09WireVersion(m.version) {
		return nil, fmt.Errorf("a2uischema: manager version = %q, got v0.9 capabilities", m.version)
	}
	selected, err := m.selectCatalog(caps)
	if err != nil {
		return nil, err
	}
	return selected.WithPruning(allowedComponents, allowedMessages)
}

// SelectedCatalogV091 is like [SchemaManager.SelectedCatalog] for A2UI v0.9.1
// client capabilities. The manager must be for version v0.9.1.
func (m *SchemaManager) SelectedCatalogV091(caps *a2uiv091.ClientCapabilities, allowedComponents, allowedMessages []string) (*Catalog, error) {
	selected, err := m.selectCatalogV091(caps)
	if err != nil {
		return nil, err
	}
	return selected.WithPruning(allowedComponents, allowedMessages)
}

// LoadExamples loads examples for a catalog if configured.
func (m *SchemaManager) LoadExamples(catalog *Catalog, validate bool) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("a2uischema: nil catalog")
	}
	id, err := catalog.ID()
	if err != nil {
		return "", err
	}
	path := m.catalogExamplePaths[id]
	return catalog.LoadExamples(path, validate)
}

// PromptOptions configures [SchemaManager.GenerateSystemPrompt].
type PromptOptions struct {
	RoleDescription     string
	WorkflowDescription string // appended to DefaultWorkflowRules
	UIDescription       string

	// At most one of the capabilities fields may be set, and it must match
	// the manager's version. If none is set, the default catalog is used.
	Capabilities     *a2ui.RendererCapabilities
	CapabilitiesV09  *v09.ClientCapabilities
	CapabilitiesV091 *a2uiv091.ClientCapabilities

	AllowedComponents []string // prune the catalog to these components
	AllowedMessages   []string // prune the schema to these messages

	IncludeSchema    bool
	IncludeExamples  bool
	ValidateExamples bool // validate examples before including them
}

// GenerateSystemPrompt assembles the system prompt.
func (m *SchemaManager) GenerateSystemPrompt(opts PromptOptions) (string, error) {
	var catalog *Catalog
	var err error
	switch {
	case opts.CapabilitiesV09 != nil && opts.CapabilitiesV091 != nil,
		opts.Capabilities != nil && (opts.CapabilitiesV09 != nil || opts.CapabilitiesV091 != nil):
		err = fmt.Errorf("a2uischema: more than one capabilities option set")
	case opts.CapabilitiesV09 != nil:
		catalog, err = m.SelectedCatalogV09(opts.CapabilitiesV09, opts.AllowedComponents, opts.AllowedMessages)
	case opts.CapabilitiesV091 != nil:
		catalog, err = m.SelectedCatalogV091(opts.CapabilitiesV091, opts.AllowedComponents, opts.AllowedMessages)
	default:
		catalog, err = m.SelectedCatalog(opts.Capabilities, opts.AllowedComponents, opts.AllowedMessages)
	}
	if err != nil {
		return "", err
	}
	return m.generateSystemPrompt(catalog, opts)
}

func (m *SchemaManager) generateSystemPrompt(catalog *Catalog, opts PromptOptions) (string, error) {
	parts := []string{opts.RoleDescription}
	workflow := DefaultWorkflowRules
	if opts.WorkflowDescription != "" {
		workflow += "\n" + opts.WorkflowDescription
	}
	parts = append(parts, "## Workflow Description:\n"+workflow)
	if opts.UIDescription != "" {
		parts = append(parts, "## UI Description:\n"+opts.UIDescription)
	}
	if opts.IncludeSchema {
		schemaBlock, err := catalog.RenderAsLLMInstructions()
		if err != nil {
			return "", err
		}
		parts = append(parts, schemaBlock)
	}
	if opts.IncludeExamples {
		examples, err := m.LoadExamples(catalog, opts.ValidateExamples)
		if err != nil {
			return "", err
		}
		if examples != "" {
			parts = append(parts, "### Examples:\n"+examples)
		}
	}
	return joinPromptParts(parts), nil
}

func (m *SchemaManager) applyModifiers(schema map[string]any) error {
	for _, modifier := range m.schemaModifiers {
		if err := modifier(schema); err != nil {
			return err
		}
	}
	return nil
}

func (m *SchemaManager) selectCatalog(clientCapabilities *v09.ClientCapabilities) (*Catalog, error) {
	if len(m.supportedCatalogs) == 0 {
		return nil, fmt.Errorf("a2uischema: no supported catalogs configured")
	}
	if clientCapabilities == nil || clientCapabilities.V09 == nil {
		return m.supportedCatalogs[0], nil
	}
	caps := clientCapabilities.V09
	if len(caps.InlineCatalogs) > 0 {
		if !m.acceptsInlineCatalogs {
			return nil, fmt.Errorf("a2uischema: inline catalogs provided but not accepted")
		}
		base := m.supportedCatalogs[0]
		if len(caps.SupportedCatalogIDs) > 0 {
			for _, id := range caps.SupportedCatalogIDs {
				for _, catalog := range m.supportedCatalogs {
					catalogID, err := catalog.ID()
					if err == nil && catalogID == id {
						base = catalog
						break
					}
				}
			}
		}
		return mergeInlineCatalogs(m.version, base, caps.InlineCatalogs)
	}
	if len(caps.SupportedCatalogIDs) == 0 {
		return m.supportedCatalogs[0], nil
	}
	for _, id := range caps.SupportedCatalogIDs {
		for _, catalog := range m.supportedCatalogs {
			catalogID, err := catalog.ID()
			if err == nil && catalogID == id {
				return catalog, nil
			}
		}
	}
	return nil, fmt.Errorf("a2uischema: no mutually supported catalog found")
}

func (m *SchemaManager) selectCatalogV1(rendererCapabilities *a2ui.RendererCapabilities) (*Catalog, error) {
	if m.version != Version1 {
		return nil, fmt.Errorf("a2uischema: manager version = %q, want %q", m.version, Version1)
	}
	if len(m.supportedCatalogs) == 0 {
		return nil, fmt.Errorf("a2uischema: no supported catalogs configured")
	}
	if rendererCapabilities == nil || rendererCapabilities.V1 == nil {
		return m.supportedCatalogs[0], nil
	}
	caps := rendererCapabilities.V1
	if len(caps.InlineCatalogs) > 0 {
		if !m.acceptsInlineCatalogs {
			return nil, fmt.Errorf("a2uischema: inline catalogs provided but not accepted")
		}
		base := m.supportedCatalogs[0]
		if len(caps.SupportedCatalogIDs) > 0 {
			for _, id := range caps.SupportedCatalogIDs {
				for _, catalog := range m.supportedCatalogs {
					catalogID, err := catalog.ID()
					if err == nil && catalogID == id {
						base = catalog
						break
					}
				}
			}
		}
		return mergeInlineCatalogsV1(m.version, base, caps.InlineCatalogs)
	}
	if len(caps.SupportedCatalogIDs) == 0 {
		return m.supportedCatalogs[0], nil
	}
	for _, id := range caps.SupportedCatalogIDs {
		for _, catalog := range m.supportedCatalogs {
			catalogID, err := catalog.ID()
			if err == nil && catalogID == id {
				return catalog, nil
			}
		}
	}
	return nil, fmt.Errorf("a2uischema: no mutually supported catalog found")
}

func (m *SchemaManager) defaultCatalog() (*Catalog, error) {
	if m.version == Version1 {
		return m.selectCatalogV1(nil)
	}
	return m.selectCatalog(nil)
}

func (m *SchemaManager) selectCatalogV091(clientCapabilities *a2uiv091.ClientCapabilities) (*Catalog, error) {
	if m.version != Version091 {
		return nil, fmt.Errorf("a2uischema: manager version = %q, want %q", m.version, Version091)
	}
	if len(m.supportedCatalogs) == 0 {
		return nil, fmt.Errorf("a2uischema: no supported catalogs configured")
	}
	if clientCapabilities == nil || clientCapabilities.V091 == nil {
		return m.supportedCatalogs[0], nil
	}
	caps := clientCapabilities.V091
	if len(caps.InlineCatalogs) > 0 {
		if !m.acceptsInlineCatalogs {
			return nil, fmt.Errorf("a2uischema: inline catalogs provided but not accepted")
		}
		base := m.supportedCatalogs[0]
		if len(caps.SupportedCatalogIDs) > 0 {
			for _, id := range caps.SupportedCatalogIDs {
				for _, catalog := range m.supportedCatalogs {
					catalogID, err := catalog.ID()
					if err == nil && catalogID == id {
						base = catalog
						break
					}
				}
			}
		}
		return mergeInlineCatalogsV091(m.version, base, caps.InlineCatalogs)
	}
	if len(caps.SupportedCatalogIDs) == 0 {
		return m.supportedCatalogs[0], nil
	}
	for _, id := range caps.SupportedCatalogIDs {
		for _, catalog := range m.supportedCatalogs {
			catalogID, err := catalog.ID()
			if err == nil && catalogID == id {
				return catalog, nil
			}
		}
	}
	return nil, fmt.Errorf("a2uischema: no mutually supported catalog found")
}

func mergeInlineCatalogs(version Version, base *Catalog, inlineCatalogs []v09.CatalogDef) (*Catalog, error) {
	messageSchema, commonSchema, catalogSchema, err := cloneCatalogSchemas(base)
	if err != nil {
		return nil, err
	}
	merged := &Catalog{
		Version:           version,
		Name:              InlineCatalogName,
		MessageSchema:     messageSchema,
		CommonTypesSchema: commonSchema,
		CatalogSchema:     catalogSchema,
	}
	for _, inline := range inlineCatalogs {
		if inline.CatalogID != "" {
			merged.CatalogSchema[CatalogIDKey] = inline.CatalogID
		}
		components, _ := merged.CatalogSchema[CatalogComponentsKey].(map[string]any)
		if components == nil {
			components = make(map[string]any)
			merged.CatalogSchema[CatalogComponentsKey] = components
		}
		for name, raw := range inline.Components {
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return nil, fmt.Errorf("a2uischema: decode inline component %q: %w", name, err)
			}
			components[name] = decoded
		}
		if len(inline.Theme) > 0 {
			theme, _ := merged.CatalogSchema[CatalogThemeKey].(map[string]any)
			if theme == nil {
				theme = make(map[string]any)
				merged.CatalogSchema[CatalogThemeKey] = theme
			}
			for name, raw := range inline.Theme {
				var decoded any
				if err := json.Unmarshal(raw, &decoded); err != nil {
					return nil, fmt.Errorf("a2uischema: decode inline theme %q: %w", name, err)
				}
				theme[name] = decoded
			}
		}
		if err := mergeInlineFunctions(merged.CatalogSchema, inline.Functions); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func mergeInlineCatalogsV091(version Version, base *Catalog, inlineCatalogs []a2uiv091.CatalogDef) (*Catalog, error) {
	messageSchema, commonSchema, catalogSchema, err := cloneCatalogSchemas(base)
	if err != nil {
		return nil, err
	}
	merged := &Catalog{
		Version:           version,
		Name:              InlineCatalogName,
		MessageSchema:     messageSchema,
		CommonTypesSchema: commonSchema,
		CatalogSchema:     catalogSchema,
	}
	for _, inline := range inlineCatalogs {
		if inline.CatalogID != "" {
			merged.CatalogSchema[CatalogIDKey] = inline.CatalogID
		}
		components, _ := merged.CatalogSchema[CatalogComponentsKey].(map[string]any)
		if components == nil {
			components = make(map[string]any)
			merged.CatalogSchema[CatalogComponentsKey] = components
		}
		for name, raw := range inline.Components {
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return nil, fmt.Errorf("a2uischema: decode inline component %q: %w", name, err)
			}
			components[name] = decoded
		}
		if len(inline.Theme) > 0 {
			theme, _ := merged.CatalogSchema[CatalogThemeKey].(map[string]any)
			if theme == nil {
				theme = make(map[string]any)
				merged.CatalogSchema[CatalogThemeKey] = theme
			}
			for name, raw := range inline.Theme {
				var decoded any
				if err := json.Unmarshal(raw, &decoded); err != nil {
					return nil, fmt.Errorf("a2uischema: decode inline theme %q: %w", name, err)
				}
				theme[name] = decoded
			}
		}
		if err := mergeInlineFunctions(merged.CatalogSchema, inline.Functions); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func mergeInlineCatalogsV1(version Version, base *Catalog, inlineCatalogs []a2ui.CatalogDef) (*Catalog, error) {
	messageSchema, commonSchema, catalogSchema, err := cloneCatalogSchemas(base)
	if err != nil {
		return nil, err
	}
	merged := &Catalog{
		Version:           version,
		Name:              InlineCatalogName,
		MessageSchema:     messageSchema,
		CommonTypesSchema: commonSchema,
		CatalogSchema:     catalogSchema,
	}
	for _, inline := range inlineCatalogs {
		if inline.CatalogID != "" {
			merged.CatalogSchema[CatalogIDKey] = inline.CatalogID
		}
		if err := mergeRawMap(merged.CatalogSchema, CatalogComponentsKey, "component", inline.Components); err != nil {
			return nil, err
		}
		if len(inline.Functions) == 0 {
			continue
		}
		functions, _ := merged.CatalogSchema[CatalogFunctionsKey].(map[string]any)
		if functions == nil {
			functions = make(map[string]any)
			merged.CatalogSchema[CatalogFunctionsKey] = functions
		}
		for name, def := range inline.Functions {
			data, err := json.Marshal(def)
			if err != nil {
				return nil, fmt.Errorf("a2uischema: encode inline function %q: %w", name, err)
			}
			var decoded any
			if err := json.Unmarshal(data, &decoded); err != nil {
				return nil, fmt.Errorf("a2uischema: decode inline function %q: %w", name, err)
			}
			functions[name] = decoded
		}
	}
	return merged, nil
}

// mergeRawMap decodes each entry of raw into schema[key], creating the map if needed.
func mergeRawMap(schema map[string]any, key, kind string, raw map[string]json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	m, _ := schema[key].(map[string]any)
	if m == nil {
		m = make(map[string]any)
		schema[key] = m
	}
	for name, data := range raw {
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return fmt.Errorf("a2uischema: decode inline %s %q: %w", kind, name, err)
		}
		m[name] = decoded
	}
	return nil
}

func mergeInlineFunctions(catalogSchema map[string]any, functions any) error {
	data, err := json.Marshal(functions)
	if err != nil {
		return fmt.Errorf("a2uischema: encode inline functions: %w", err)
	}
	var defs []map[string]any
	if err := json.Unmarshal(data, &defs); err != nil {
		return fmt.Errorf("a2uischema: decode inline functions: %w", err)
	}
	if len(defs) == 0 {
		return nil
	}
	functionsMap, _ := catalogSchema[CatalogFunctionsKey].(map[string]any)
	if functionsMap != nil {
		for _, def := range defs {
			name, _ := def["name"].(string)
			if name == "" {
				return fmt.Errorf("a2uischema: inline function missing name")
			}
			functionsMap[name] = def
		}
		return nil
	}
	functionsList, _ := catalogSchema[CatalogFunctionsKey].([]any)
	for _, def := range defs {
		functionsList = append(functionsList, def)
	}
	catalogSchema[CatalogFunctionsKey] = functionsList
	return nil
}

func embeddedSchemas(version Version) (map[string]any, map[string]any, error) {
	switch version {
	case Version09:
		messageMap, err := unmarshalJSONMap(serverToClientV09)
		if err != nil {
			return nil, nil, err
		}
		commonMap, err := unmarshalJSONMap(commonTypesV09)
		if err != nil {
			return nil, nil, err
		}
		return messageMap, commonMap, nil
	case Version091:
		messageMap, err := unmarshalJSONMap(serverToClientV091)
		if err != nil {
			return nil, nil, err
		}
		commonMap, err := unmarshalJSONMap(commonTypesV091)
		if err != nil {
			return nil, nil, err
		}
		return messageMap, commonMap, nil
	case Version1:
		messageMap, err := unmarshalJSONMap(agentToRendererV1)
		if err != nil {
			return nil, nil, err
		}
		commonMap, err := unmarshalJSONMap(commonTypesV1)
		if err != nil {
			return nil, nil, err
		}
		return messageMap, commonMap, nil
	default:
		return nil, nil, fmt.Errorf("a2uischema: unsupported version %q", version)
	}
}

func marshalJSON(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func joinPromptParts(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return string(bytesJoinWithDoubleNewline(out))
}

func bytesJoinWithDoubleNewline(parts []string) []byte {
	if len(parts) == 0 {
		return nil
	}
	var out []byte
	for i, part := range parts {
		if i > 0 {
			out = append(out, '\n', '\n')
		}
		out = append(out, part...)
	}
	return out
}
