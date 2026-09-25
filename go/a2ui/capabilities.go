package a2ui

import "encoding/json"

// AllowedCallers describes which peers may invoke a catalog function.
type AllowedCallers string

const (
	AllowedCallersRendererOnly    AllowedCallers = "rendererOnly"
	AllowedCallersAgentOnly       AllowedCallers = "agentOnly"
	AllowedCallersRendererOrAgent AllowedCallers = "rendererOrAgent"
)

// RendererCapabilities describes a renderer's UI rendering capabilities,
// sent as part of A2A metadata.
type RendererCapabilities struct {
	V1 *RendererCapabilitiesV1 `json:"v1.0,omitempty"`
}

// RendererCapabilitiesV1 is the v1.0 renderer capabilities structure.
type RendererCapabilitiesV1 struct {
	SupportedCatalogIDs []string     `json:"supportedCatalogIds"`
	InlineCatalogs      []CatalogDef `json:"inlineCatalogs,omitempty"`
}

// AgentCapabilities describes an agent's supported UI features,
// advertised via agent card or other discovery.
type AgentCapabilities struct {
	V1 *AgentCapabilitiesV1 `json:"v1.0,omitempty"`
}

// AgentCapabilitiesV1 is the v1.0 agent capabilities structure.
type AgentCapabilitiesV1 struct {
	SupportedCatalogIDs   []string `json:"supportedCatalogIds,omitempty"`
	AcceptsInlineCatalogs bool     `json:"acceptsInlineCatalogs,omitempty"`
}

// CatalogDef is a catalog definition containing component schemas
// and function definitions.
type CatalogDef struct {
	CatalogID       string                        `json:"catalogId"`
	ProtocolVersion string                        `json:"protocolVersion,omitempty"`
	Title           string                        `json:"title,omitempty"`
	Description     string                        `json:"description,omitempty"`
	Instructions    string                        `json:"instructions,omitempty"`
	Components      map[string]json.RawMessage    `json:"components,omitempty"`
	Functions       map[string]FunctionDefinition `json:"functions,omitempty"`
}

// FunctionDefinition is the JSON Schema of a catalog function together
// with its interface metadata.
type FunctionDefinition struct {
	Type                   string          `json:"type"`
	Description            string          `json:"description,omitempty"`
	Properties             json.RawMessage `json:"properties"`
	Required               []string        `json:"required"`
	ReturnType             ReturnType      `json:"returnType"`
	AllowedCallers         AllowedCallers  `json:"allowedCallers,omitempty"`
	RequiresUserActivation bool            `json:"requiresUserActivation,omitempty"`
}

// ValidationResult is the value of a check condition.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Code     string   `json:"code,omitempty"`
	Message  string   `json:"message,omitempty"`
	Severity Severity `json:"severity,omitempty"`
}

// Severity is the severity of a [ValidationResult].
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// RendererDataModel carries the renderer data model in A2A message metadata.
type RendererDataModel struct {
	Version  string                    `json:"version"`
	Surfaces map[string]map[string]any `json:"surfaces"`
}
