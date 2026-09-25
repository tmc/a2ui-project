package a2ui

// Version is the A2UI protocol version implemented by this package.
const Version = "v1.0"

// AgentMessage is a message sent from the agent to the renderer.
// Exactly one of the payload fields is non-nil.
type AgentMessage struct {
	Version               string                `json:"version"`
	CreateSurface         *CreateSurface        `json:"createSurface,omitempty"`
	UpdateComponents      *UpdateComponents     `json:"updateComponents,omitempty"`
	UpdateDataModel       *UpdateDataModel      `json:"updateDataModel,omitempty"`
	DeleteSurface         *DeleteSurface        `json:"deleteSurface,omitempty"`
	CallRendererFunction  *CallRendererFunction `json:"callRendererFunction,omitempty"`
	AgentFunctionResponse *FunctionResponse     `json:"agentFunctionResponse,omitempty"`
}

// VersionString returns the A2UI protocol version carried by m.
func (m AgentMessage) VersionString() string { return m.Version }

// CreateSurface signals the renderer to create a new surface.
type CreateSurface struct {
	SurfaceID     string         `json:"surfaceId"`
	CatalogID     string         `json:"catalogId,omitempty"`
	SendDataModel bool           `json:"sendDataModel,omitempty"`
	Components    []Component    `json:"components,omitempty"`
	DataModel     map[string]any `json:"dataModel,omitempty"`
	Metadata      *Metadata      `json:"metadata,omitempty"`
}

// UpdateComponents updates a surface with a new set of components.
type UpdateComponents struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

// UpdateDataModel updates the data model for a surface.
// A nil Value deletes the key at Path.
type UpdateDataModel struct {
	SurfaceID string `json:"surfaceId"`
	Path      string `json:"path,omitempty"`
	Value     any    `json:"value"`
}

// DeleteSurface signals the renderer to delete a surface.
type DeleteSurface struct {
	SurfaceID string `json:"surfaceId"`
}

// CallRendererFunction asks the renderer to execute a function on behalf
// of the agent. CallFunction.CatalogID is required.
type CallRendererFunction struct {
	FunctionCallID string       `json:"functionCallId"`
	CallFunction   FunctionCall `json:"callFunction"`
}

// FunctionResponse is the result of a callRendererFunction or
// callAgentFunction invocation. Exactly one of a value or Error is set.
type FunctionResponse struct {
	FunctionCallID string         `json:"-"`
	Value          any            `json:"-"`
	HasValue       bool           `json:"-"`
	Error          *FunctionError `json:"-"`
}

// FunctionResponseValue returns a response carrying value, which may be nil.
func FunctionResponseValue(functionCallID string, value any) FunctionResponse {
	return FunctionResponse{FunctionCallID: functionCallID, Value: value, HasValue: true}
}

// FunctionError reports a failed function execution.
type FunctionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RendererMessage is a message sent from the renderer to the agent.
// Exactly one of the payload fields is non-nil.
type RendererMessage struct {
	Version                  string             `json:"version"`
	Action                   *ActionEvent       `json:"action,omitempty"`
	CallAgentFunction        *CallAgentFunction `json:"callAgentFunction,omitempty"`
	RendererFunctionResponse *FunctionResponse  `json:"rendererFunctionResponse,omitempty"`
	Error                    *RendererError     `json:"error,omitempty"`
}

// VersionString returns the A2UI protocol version carried by m.
func (m RendererMessage) VersionString() string { return m.Version }

// ActionEvent reports a user-initiated action from a component.
type ActionEvent struct {
	Name              string         `json:"name"`
	UserMessage       string         `json:"userMessage,omitempty"`
	SurfaceID         string         `json:"surfaceId"`
	SourceComponentID string         `json:"sourceComponentId"`
	Timestamp         string         `json:"timestamp"`
	Context           map[string]any `json:"context"`
	Metadata          *Metadata      `json:"metadata,omitempty"`
}

// CallAgentFunction asks the agent to execute a function on behalf of
// the renderer.
type CallAgentFunction struct {
	SurfaceID      string       `json:"surfaceId"`
	FunctionCallID string       `json:"functionCallId"`
	CallFunction   FunctionCall `json:"callFunction"`
}

// Renderer error codes that report a validation failure.
// These codes require Path and SurfaceID.
const (
	ErrorValidationFailed = "VALIDATION_FAILED"
	ErrorUnallowedParent  = "UNALLOWED_PARENT"
	ErrorUnallowedChild   = "UNALLOWED_CHILD"
)

// RendererError reports a renderer-side error.
type RendererError struct {
	Code           string `json:"code"`
	SurfaceID      string `json:"surfaceId,omitempty"`
	FunctionCallID string `json:"functionCallId,omitempty"`
	Message        string `json:"message"`
	Path           string `json:"path,omitempty"`
}

// IsValidationError reports whether e has a validation failure code.
func (e RendererError) IsValidationError() bool {
	switch e.Code {
	case ErrorValidationFailed, ErrorUnallowedParent, ErrorUnallowedChild:
		return true
	}
	return false
}
