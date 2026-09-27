package a2uischema

import (
	"encoding/json"
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// SchemaModifier can rewrite a decoded schema before it is used.
type SchemaModifier func(schema map[string]any) error

// SchemaManager manages schemas, catalogs, and prompt rendering.
type SchemaManager struct {
	acceptsInlineCatalogs bool
	messageSchema         map[string]any
	commonTypesSchema     map[string]any
	supportedCatalogs     []*Catalog
	catalogExamplePaths   map[string]string
	schemaModifiers       []SchemaModifier
}

// NewSchemaManager constructs a schema manager.
func NewSchemaManager(catalogs []CatalogConfig, acceptsInlineCatalogs bool, schemaModifiers ...SchemaModifier) (*SchemaManager, error) {
	messageSchema, commonSchema, err := embeddedSchemas()
	if err != nil {
		return nil, err
	}
	manager := &SchemaManager{
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
		catalog, err := newCatalog(cfg.Name, messageSchemaData, commonSchemaData, data)
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

// SelectedCatalog selects the catalog for the renderer capabilities
// and prunes it to the allowed components and messages.
// A nil caps selects the first configured catalog.
func (m *SchemaManager) SelectedCatalog(caps *a2ui.RendererCapabilities, allowedComponents, allowedMessages []string) (*Catalog, error) {
	selected, err := m.selectCatalog(caps)
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

	// Capabilities selects the catalog, as in [SchemaManager.SelectedCatalog].
	Capabilities *a2ui.RendererCapabilities

	AllowedComponents []string // prune the catalog to these components
	AllowedMessages   []string // prune the schema to these messages

	IncludeSchema    bool
	IncludeExamples  bool
	ValidateExamples bool // validate examples before including them
}

// GenerateSystemPrompt assembles the system prompt.
func (m *SchemaManager) GenerateSystemPrompt(opts PromptOptions) (string, error) {
	catalog, err := m.SelectedCatalog(opts.Capabilities, opts.AllowedComponents, opts.AllowedMessages)
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

func (m *SchemaManager) selectCatalog(rendererCapabilities *a2ui.RendererCapabilities) (*Catalog, error) {
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
		return mergeInlineCatalogs(base, caps.InlineCatalogs)
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

func mergeInlineCatalogs(base *Catalog, inlineCatalogs []a2ui.CatalogDef) (*Catalog, error) {
	messageSchema, commonSchema, catalogSchema, err := cloneCatalogSchemas(base)
	if err != nil {
		return nil, err
	}
	merged := &Catalog{
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

func embeddedSchemas() (map[string]any, map[string]any, error) {
	messageMap, err := unmarshalJSONMap(agentToRenderer)
	if err != nil {
		return nil, nil, err
	}
	commonMap, err := unmarshalJSONMap(commonTypes)
	if err != nil {
		return nil, nil, err
	}
	return messageMap, commonMap, nil
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
