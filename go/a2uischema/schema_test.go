package a2uischema

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2uibuild"
	"github.com/a2ui-project/a2ui/go/a2uistream"
)

const basicCatalogID = "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"

func TestSchemaManagerGenerateSystemPrompt(t *testing.T) {
	manager := mustBasicManager(t)
	caps := &a2ui.RendererCapabilities{V1: &a2ui.RendererCapabilitiesV1{
		SupportedCatalogIDs: []string{basicCatalogID},
	}}
	for _, opts := range []PromptOptions{
		{RoleDescription: "role", IncludeSchema: true},
		{RoleDescription: "role", Capabilities: caps, IncludeSchema: true},
	} {
		prompt, err := manager.GenerateSystemPrompt(opts)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(prompt, A2UISchemaBlockStart) {
			t.Fatal("expected schema block")
		}
		if !strings.Contains(prompt, "v1_0/catalogs/basic/catalog.json") {
			t.Fatal("expected basic catalog schema in prompt")
		}
	}
}

func TestSchemaManagerRejectsUnsupportedCatalog(t *testing.T) {
	caps := &a2ui.RendererCapabilities{V1: &a2ui.RendererCapabilitiesV1{
		SupportedCatalogIDs: []string{"https://example.com/other.json"},
	}}
	if _, err := mustBasicManager(t).SelectedCatalog(caps, nil, nil); err == nil {
		t.Fatal("SelectedCatalog with no common catalog succeeded")
	}
}

func TestValidatorAcceptsValidSurfaceMessages(t *testing.T) {
	validator := mustBasicValidator(t)
	surface := a2uibuild.NewSurface("contact", basicCatalogID).
		Add(a2uibuild.Column("root", a2uibuild.Children("greeting"))).
		Add(a2uibuild.Text("greeting", a2ui.StringLiteral("Hello, world!")))
	if err := validator.ValidateMessages(surface.Messages()); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorAcceptsExamples(t *testing.T) {
	validator := mustBasicValidator(t)
	paths, err := filepath.Glob("testdata/v1_0/basic/examples/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no examples found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := validator.ValidateExample(data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidatorAcceptsAgentFunctionResponseNull(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.AgentMessage{
		Version:               a2ui.Version,
		AgentFunctionResponse: &a2ui.FunctionResponse{FunctionCallID: "call-1"},
	}
	if err := validator.ValidateMessages([]a2ui.AgentMessage{msg}); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorRejectsOtherVersions(t *testing.T) {
	err := mustBasicValidator(t).ValidateJSON([]byte(`{"version":"v0.9","deleteSurface":{"surfaceId":"s1"}}`))
	if err == nil {
		t.Fatal("v0.9 message accepted")
	}
}

func TestValidatorStructuredErrors(t *testing.T) {
	tests := []struct {
		name      string
		msgs      string
		code      ValidationCode
		component string
	}{
		{
			name:      "duplicate id",
			msgs:      `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["dup"]},{"id":"dup","component":"Text","text":"one"},{"id":"dup","component":"Text","text":"two"}]}}`,
			code:      ValidationDuplicateComponent,
			component: "dup",
		},
		{
			name: "unknown function",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Button","child":"label","action":{"functionCall":{"call":"definitelyUnknown"}}},{"id":"label","component":"Text","text":"Run"}]}}`,
			code: ValidationUnknownFunction,
		},
		{
			name: "invalid path",
			msgs: `{"version":"v1.0","updateDataModel":{"surfaceId":"s1","path":"/bad~path","value":"value"}}`,
			code: ValidationInvalidPath,
		},
		{
			name: "orphan",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["greeting"]},{"id":"greeting","component":"Text","text":"hello"},{"id":"extra","component":"Text","text":"orphan"}]}}`,
			code: ValidationOrphanedComponent,
		},
	}
	validator := mustBasicValidator(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte(tt.msgs))
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			assertValidationError(t, err, tt.code, tt.component)
		})
	}
}

func TestParseAndValidate(t *testing.T) {
	const bad = `{"version":"v1.0","createSurface":{"surfaceId":"s1","catalogId":"https://example.com/other.json"}}`
	if _, err := a2uistream.ParseAndValidate(bad, mustBasicValidator(t)); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestValidatorComponentRefs(t *testing.T) {
	validator := mustBasicValidator(t)
	const (
		create  = `{"version":"v1.0","createSurface":{"surfaceId":"s1"}}`
		root    = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Card","child":"body"}]}}`
		body    = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"body","component":"Text","text":"hi"}]}}`
		deleteS = `{"version":"v1.0","deleteSurface":{"surfaceId":"s1"}}`
	)
	tests := []struct {
		name    string
		msgs    []string
		wantErr bool
	}{
		{"forward_ref_resolved", []string{create, root, body}, false},
		{"forward_ref_unresolved", []string{create, root}, true},
		{"deleted_surface", []string{create, root, deleteS}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := "[" + strings.Join(tt.msgs, ",") + "]"
			err := validator.ValidateJSON([]byte(data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateJSON() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mustBasicManager(t *testing.T) *SchemaManager {
	t.Helper()
	manager, err := NewSchemaManager([]CatalogConfig{BasicCatalogConfig()}, false)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func mustBasicValidator(t *testing.T) *Validator {
	t.Helper()
	catalog, err := mustBasicManager(t).SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Validator()
}

func assertValidationError(t *testing.T, err error, code ValidationCode, component string) {
	t.Helper()
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("errors.As(*ValidationError) = false for %v", err)
	}
	if validationErr.Code != code {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, code)
	}
	if component != "" && validationErr.Component != component {
		t.Fatalf("ValidationError.Component = %q, want %q", validationErr.Component, component)
	}
}
