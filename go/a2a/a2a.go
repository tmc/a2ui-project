package a2a

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	A2UIExtensionBaseURI     = "https://a2ui.org/a2a-extension/a2ui"
	MIMETypeKey              = "mimeType"
	AcceptsInlineCatalogsKey = "acceptsInlineCatalogs"
	SupportedCatalogIDsKey   = "supportedCatalogIds"
)

// A2UIMIMEType is the MIME type of A2UI content,
// carried in the [MIMETypeKey] metadata of a [DataPart].
const A2UIMIMEType = "application/a2ui+json"

// defaultVersion is the extension version used when none is given.
const defaultVersion = "v1.0"

// DataPart is a transport-neutral A2A data part carrying A2UI JSON.
// Its shape matches the official A2A Go SDK's DataPart.
type DataPart struct {
	Data     map[string]any `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AgentExtension is a transport-neutral A2A agent extension descriptor.
// Its shape matches the official A2A Go SDK's AgentExtension.
type AgentExtension struct {
	Description string         `json:"description,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
	Required    bool           `json:"required,omitempty"`
	URI         string         `json:"uri"`
}

// Meta returns the part metadata.
func (p DataPart) Meta() map[string]any {
	return p.Metadata
}

// SetMeta sets a metadata entry.
func (p *DataPart) SetMeta(k string, v any) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}
	p.Metadata[k] = v
}

// MarshalA2UIData marshals payload into an A2A data-part payload.
// A2A data parts carry JSON objects, so payload must encode as a JSON object.
// The result shares no memory with payload.
func MarshalA2UIData(payload any) (map[string]any, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("a2a: marshal payload: %w", err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("a2a: decode payload object: %w", err)
	}
	if object == nil {
		return nil, fmt.Errorf("a2a: payload must encode as a JSON object")
	}
	return object, nil
}

// CreateDataPart marshals an A2UI payload into a transport-neutral A2A data part
// with MIME type [A2UIMIMEType].
func CreateDataPart(payload any) (DataPart, error) {
	data, err := MarshalA2UIData(payload)
	if err != nil {
		return DataPart{}, err
	}
	part := DataPart{Data: data}
	part.SetMeta(MIMETypeKey, A2UIMIMEType)
	return part, nil
}

// IsA2UIPart reports whether the part carries an A2UI MIME type.
func IsA2UIPart(part DataPart) bool {
	if part.Metadata == nil {
		return false
	}
	mimeType, _ := part.Metadata[MIMETypeKey].(string)
	return mimeType == A2UIMIMEType
}

// A2UIData returns the structured A2UI payload if the part carries A2UI data.
func A2UIData(part DataPart) (map[string]any, bool) {
	if !IsA2UIPart(part) {
		return nil, false
	}
	return part.Data, true
}

// AgentExtensionOptions configures an A2A agent extension descriptor.
type AgentExtensionOptions struct {
	Version               string // extension version, such as "v1.0"; empty means v1.0
	AcceptsInlineCatalogs bool
	SupportedCatalogIDs   []string
}

// NewAgentExtension constructs an A2UI extension descriptor.
// An empty opts.Version means v1.0.
func NewAgentExtension(opts AgentExtensionOptions) AgentExtension {
	params := make(map[string]any)
	if opts.AcceptsInlineCatalogs {
		params[AcceptsInlineCatalogsKey] = true
	}
	if len(opts.SupportedCatalogIDs) > 0 {
		params[SupportedCatalogIDsKey] = append([]string(nil), opts.SupportedCatalogIDs...)
	}
	if len(params) == 0 {
		params = nil
	}
	return AgentExtension{
		URI:         fmt.Sprintf("%s/%s", A2UIExtensionBaseURI, normalizeVersion(opts.Version)),
		Description: "Provides agent driven UI using the A2UI JSON format.",
		Params:      params,
	}
}

// SelectNewestRequestedExtension returns the newest requested extension also advertised by the agent.
func SelectNewestRequestedExtension(requested, advertised []string) (string, bool) {
	best := ""
	for _, candidate := range requested {
		if !slices.Contains(advertised, candidate) {
			continue
		}
		if best == "" || compareExtensionVersion(candidate, best) > 0 {
			best = candidate
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

// TryActivateExtension selects and activates the newest mutually supported extension.
func TryActivateExtension(requested, advertised []string) (activated, version string, ok bool) {
	activated, ok = SelectNewestRequestedExtension(requested, advertised)
	if !ok {
		return "", "", false
	}
	version = strings.TrimPrefix(activated, A2UIExtensionBaseURI+"/")
	version = strings.TrimPrefix(version, "v")
	return activated, version, true
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		return defaultVersion
	}
	return "v" + version
}

func compareExtensionVersion(a, b string) int {
	av := strings.TrimPrefix(strings.TrimPrefix(a, A2UIExtensionBaseURI+"/"), "v")
	bv := strings.TrimPrefix(strings.TrimPrefix(b, A2UIExtensionBaseURI+"/"), "v")
	aparts := parseVersionParts(av)
	bparts := parseVersionParts(bv)
	for i := 0; i < len(aparts) || i < len(bparts); i++ {
		var ai, bi int
		if i < len(aparts) {
			ai = aparts[i]
		}
		if i < len(bparts) {
			bi = bparts[i]
		}
		switch {
		case ai < bi:
			return -1
		case ai > bi:
			return 1
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	fields := strings.Split(version, ".")
	out := make([]int, 0, len(fields))
	for _, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			return []int{0}
		}
		out = append(out, n)
	}
	return out
}
