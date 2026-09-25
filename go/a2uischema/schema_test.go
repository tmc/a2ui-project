package a2uischema

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	v09 "github.com/a2ui-project/a2ui/go/a2ui/v09"
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
	msg := v09.ServerMessage{
		Version: v09.Version,
		CreateSurface: &v09.CreateSurface{
			SurfaceID: "s1",
			CatalogID: "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json",
		},
	}
	if err := validator.ValidateMessages([]v09.ServerMessage{msg}); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorAcceptsValidSurfaceMessages(t *testing.T) {
	validator := mustBasicValidator(t)
	surface := a2uibuild.NewSurface("contact", "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json").
		Add(a2uibuild.Column("root", a2uibuild.Children("greeting"))).
		Add(a2uibuild.Text("greeting", v09.StringLiteral("Hello, world!")))
	if err := validator.ValidateMessages(surface.Messages()); err != nil {
		t.Fatal(err)
	}
}

func TestValidatorRejectsDuplicateIDs(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := v09.ServerMessage{
		Version: v09.Version,
		UpdateComponents: &v09.UpdateComponents{
			SurfaceID: "s1",
			Components: []v09.Component{
				a2uibuild.Column("root", a2uibuild.Children("dup")),
				a2uibuild.Text("dup", v09.StringLiteral("one")),
				a2uibuild.Text("dup", v09.StringLiteral("two")),
			},
		},
	}
	err := validator.ValidateMessages([]v09.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationDuplicateComponent, "dup")
}

func TestValidatorRejectsOrphanedComponent(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := v09.ServerMessage{
		Version: v09.Version,
		UpdateComponents: &v09.UpdateComponents{
			SurfaceID: "s1",
			Components: []v09.Component{
				a2uibuild.Column("root", a2uibuild.Children("greeting")),
				a2uibuild.Text("greeting", v09.StringLiteral("hello")),
				a2uibuild.Text("extra", v09.StringLiteral("orphan")),
			},
		},
	}
	err := validator.ValidateMessages([]v09.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationOrphanedComponent, "")
}

func TestValidatorRejectsUnknownFunction(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := v09.ServerMessage{
		Version: v09.Version,
		UpdateComponents: &v09.UpdateComponents{
			SurfaceID: "s1",
			Components: []v09.Component{
				a2uibuild.Button("root",
					v09.Action{
						FunctionCall: &v09.FunctionCall{Call: "definitelyUnknown"},
					},
					"label",
				),
				a2uibuild.Text("label", v09.StringLiteral("Run")),
			},
		},
	}
	err := validator.ValidateMessages([]v09.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationUnknownFunction, "")
}

func TestValidatorReportsStructuredInvalidPath(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := v09.ServerMessage{
		Version: v09.Version,
		UpdateDataModel: &v09.UpdateDataModel{
			SurfaceID: "s1",
			Path:      "/bad~path",
			Value:     "value",
		},
	}
	err := validator.ValidateMessages([]v09.ServerMessage{msg})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertValidationError(t, err, ValidationInvalidPath, "")
}

func TestParseAndValidate(t *testing.T) {
	validator := mustBasicValidator(t)
	msg := v09.ServerMessage{
		Version: v09.Version,
		UpdateComponents: &v09.UpdateComponents{
			SurfaceID: "s1",
			Components: []v09.Component{
				a2uibuild.Text("bad", v09.StringLiteral("missing root")),
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
