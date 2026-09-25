package a2uischema

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
	a2uiv10 "github.com/a2ui-project/a2ui/go/a2ui/v10"
	"github.com/a2ui-project/a2ui/go/a2uibuild"
	"github.com/a2ui-project/a2ui/go/a2uistream"
)

func TestSchemaManagerGenerateSystemPrompt(t *testing.T) {
	basic, err := BasicCatalogConfig(Version09)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version09, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := manager.GenerateSystemPrompt("role", "", "", nil, nil, nil, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, A2UISchemaBlockStart) {
		t.Fatal("expected schema block")
	}
	if !strings.Contains(prompt, "catalogs/basic/catalog.json") {
		t.Fatal("expected basic catalog schema in prompt")
	}
}

func TestSchemaManagerGenerateSystemPromptVersioned(t *testing.T) {
	basic, err := BasicCatalogConfig(Version10)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version10, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	caps := &a2uiv10.RendererCapabilities{V10: &a2uiv10.RendererCapabilitiesV10{
		SupportedCatalogIDs: []string{"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"},
	}}
	prompt, err := manager.GenerateSystemPrompt("role", "", "", caps, nil, nil, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, A2UISchemaBlockStart) {
		t.Fatal("expected schema block")
	}
	if !strings.Contains(prompt, "v1_0/catalogs/basic/catalog.json") {
		t.Fatal("expected v1.0 basic catalog schema in prompt")
	}
}

func TestSchemaManagerGenerateSystemPromptV091(t *testing.T) {
	basic, err := BasicCatalogConfig(Version091)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version091, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := manager.GenerateSystemPrompt("role", "", "", nil, nil, nil, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, A2UISchemaBlockStart) {
		t.Fatal("expected schema block")
	}
	if !strings.Contains(prompt, "v0_9/catalogs/basic/catalog.json") {
		t.Fatal("expected v0.9 wire catalog schema in prompt")
	}
}

func TestValidatorAcceptsV091WireVersion(t *testing.T) {
	validator := mustBasicValidatorV091(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID: "s1",
			CatalogID: "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json",
		},
	}
	if err := validator.ValidateMessages([]a2ui.ServerMessage{msg}); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorAcceptsValidSurfaceMessages(t *testing.T) {
	validator := mustBasicValidator(t)
	surface := a2uibuild.NewSurface("contact", "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json").
		Add(a2uibuild.Column("root", a2uibuild.Children("greeting"))).
		Add(a2uibuild.Text("greeting", a2ui.StringLiteral("Hello, world!")))
	if err := validator.ValidateMessages(surface.Messages()); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorAcceptsV10Examples(t *testing.T) {
	validator := mustBasicValidatorV10(t)
	paths, err := filepath.Glob("testdata/v1_0/basic/examples/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no v1.0 examples found")
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

func TestValidatorAcceptsV10AgentFunctionResponseNull(t *testing.T) {
	validator := mustBasicValidatorV10(t)
	msg := a2uiv10.AgentMessage{
		Version:               a2uiv10.Version,
		AgentFunctionResponse: ptr(a2uiv10.FunctionResponseValue("call-1", nil)),
	}
	if err := validator.ValidateVersionMessages([]a2uiv10.AgentMessage{msg}); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorRejectsDuplicateIDs(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s1",
			Components: []a2ui.Component{
				a2uibuild.Column("root", a2uibuild.Children("dup")),
				a2uibuild.Text("dup", a2ui.StringLiteral("one")),
				a2uibuild.Text("dup", a2ui.StringLiteral("two")),
			},
		},
	}
	err := validator.ValidateMessages([]a2ui.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationDuplicateComponent, "dup")
}

func TestValidatorRejectsOrphanedComponent(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s1",
			Components: []a2ui.Component{
				a2uibuild.Column("root", a2uibuild.Children("greeting")),
				a2uibuild.Text("greeting", a2ui.StringLiteral("hello")),
				a2uibuild.Text("extra", a2ui.StringLiteral("orphan")),
			},
		},
	}
	err := validator.ValidateMessages([]a2ui.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationOrphanedComponent, "")
}

func TestValidatorRejectsUnknownFunction(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s1",
			Components: []a2ui.Component{
				a2uibuild.Button("root",
					a2ui.Action{
						FunctionCall: &a2ui.FunctionCall{Call: "definitelyUnknown"},
					},
					"label",
				),
				a2uibuild.Text("label", a2ui.StringLiteral("Run")),
			},
		},
	}
	err := validator.ValidateMessages([]a2ui.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationUnknownFunction, "")
}

func TestValidatorReportsStructuredInvalidPath(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateDataModel: &a2ui.UpdateDataModel{
			SurfaceID: "s1",
			Path:      "/bad~path",
			Value:     "value",
		},
	}
	err := validator.ValidateMessages([]a2ui.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationInvalidPath, "")
}

func TestParseAndValidate(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s1",
			Components: []a2ui.Component{
				a2uibuild.Text("bad", a2ui.StringLiteral("missing root")),
			},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a2uistream.ParseAndValidate(string(data), validator); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func mustBasicValidator(t *testing.T) *Validator {
	t.Helper()
	basic, err := BasicCatalogConfig(Version09)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version09, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Validator()
}

func mustBasicValidatorV10(t *testing.T) *Validator {
	t.Helper()
	basic, err := BasicCatalogConfig(Version10)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version10, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Validator()
}

func mustBasicValidatorV091(t *testing.T) *Validator {
	t.Helper()
	basic, err := BasicCatalogConfig(Version091)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSchemaManager(Version091, []CatalogConfig{basic}, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Validator()
}

func ptr[T any](v T) *T {
	return &v
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

func TestValidatorV10ComponentRefs(t *testing.T) {
	validator := mustBasicValidatorV10(t)
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
