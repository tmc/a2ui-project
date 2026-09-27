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

func TestValidatorErrors(t *testing.T) {
	tests := []struct {
		name string
		msgs string
		want error
		path string
	}{
		{
			name: "version",
			msgs: `{"version":"v0.9","deleteSurface":{"surfaceId":"s1"}}`,
			want: ErrVersionMismatch,
			path: "/version",
		},
		{
			name: "missing field",
			msgs: `[{"version":"v1.0","deleteSurface":{}}]`,
			want: ErrInvalidMessage,
			path: "/0/deleteSurface/surfaceId",
		},
		{
			name: "syntax",
			msgs: `{"version":`,
			want: ErrInvalidMessage,
		},
		{
			name: "duplicate id",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["dup"]},{"id":"dup","component":"Text","text":"one"},{"id":"dup","component":"Text","text":"two"}]}}`,
			want: ErrInvalidTree,
			path: "/updateComponents/components/2/id",
		},
		{
			name: "unknown function",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Button","child":"label","action":{"functionCall":{"call":"definitelyUnknown"}}},{"id":"label","component":"Text","text":"Run"}]}}`,
			want: ErrUnknownFunction,
			path: "/updateComponents/components/0/action/functionCall/call",
		},
		{
			name: "invalid path",
			msgs: `{"version":"v1.0","updateDataModel":{"surfaceId":"s1","path":"/bad~path","value":"value"}}`,
			want: ErrInvalidMessage,
			path: "/updateDataModel/path",
		},
		{
			name: "invalid binding",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Text","text":{"path":"~bad"}}]}}`,
			want: ErrInvalidMessage,
			path: "/updateComponents/components/0/text/path",
		},
		{
			name: "orphan",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["greeting"]},{"id":"greeting","component":"Text","text":"hello"},{"id":"extra","component":"Text","text":"orphan"}]}}`,
			want: ErrInvalidTree,
			path: "/updateComponents/components/2",
		},
		{
			name: "unknown reference",
			msgs: `[{"version":"v1.0","createSurface":{"surfaceId":"s1"}},{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["a","b"]},{"id":"a","component":"Text","text":"a"}]}}]`,
			want: ErrInvalidTree,
			path: "/1/updateComponents/components/0/children/1",
		},
	}
	validator := mustBasicValidator(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte(tt.msgs))
			if !errors.Is(err, tt.want) {
				t.Fatalf("ValidateJSON() = %v, want %v", err, tt.want)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(%v, *ValidationError) = false", err)
			}
			if ve.Path != tt.path {
				t.Errorf("Path = %q, want %q", ve.Path, tt.path)
			}
			if !strings.HasPrefix(err.Error(), "a2uischema: ") {
				t.Errorf("Error() = %q, want a2uischema: prefix", err)
			}
		})
	}
}

func TestValidatorUnknownComponent(t *testing.T) {
	catalog, err := mustBasicManager(t).SelectedCatalog(nil, []string{"Text"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	const msgs = `[{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["t"]},{"id":"t","component":"Text","text":"hi"}]}}]`
	err = catalog.Validator().ValidateJSON([]byte(msgs))
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Err != ErrUnknownComponent {
		t.Fatalf("ValidateJSON() = %v, want ErrUnknownComponent", err)
	}
	if want := "/0/updateComponents/components/0/component"; ve.Path != want {
		t.Errorf("Path = %q, want %q", ve.Path, want)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	const msgs = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Card"}]}}`
	err := mustBasicValidator(t).ValidateJSON([]byte(msgs))
	const want = "a2uischema: message[0]: updateComponents: component[0] (root): card.child is required"
	if err == nil || err.Error() != want {
		t.Fatalf("ValidateJSON() = %v, want %s", err, want)
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

func TestValidatorComposition(t *testing.T) {
	manager, err := NewSchemaManager([]CatalogConfig{
		CatalogConfigFromPath("composition", "testdata/composition_catalog.json", ""),
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	validator := catalog.Validator()

	const (
		create = `{"version":"v1.0","createSurface":{"surfaceId":"s1","catalogId":"https://example.com/composition_catalog.json"}}`
		// Column allows Card and Text children; Card allows Surface and
		// Column parents; Button allows only a Card parent.
		allowed     = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["card","text"]},{"id":"card","component":"Card","child":"button"},{"id":"button","component":"Button","child":"label","action":{"event":{"name":"go"}}},{"id":"label","component":"Text","text":"Go"},{"id":"text","component":"Text","text":"hi"}]}}`
		badParent   = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Card","child":"inner"},{"id":"inner","component":"Card","child":"label"},{"id":"label","component":"Text","text":"Go"}]}}`
		badChild    = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["text","inner"]},{"id":"text","component":"Text","text":"hi"},{"id":"inner","component":"Column","children":["text2"]},{"id":"text2","component":"Text","text":"hi"}]}}`
		badRoot     = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Button","child":"label","action":{"event":{"name":"go"}}},{"id":"label","component":"Text","text":"Go"}]}}`
		parentFirst = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["later"]}]}}`
		laterButton = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"later","component":"Button","child":"label","action":{"event":{"name":"go"}}},{"id":"label","component":"Text","text":"Go"}]}}`
		laterCard   = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"later","component":"Card","child":"label"},{"id":"label","component":"Text","text":"Go"}]}}`
	)
	tests := []struct {
		name string
		msgs []string
		path string // "" if valid
	}{
		{"allowed", []string{create, allowed}, ""},
		{"disallowed parent", []string{create, badParent}, "/1/updateComponents/components/0/child"},
		{"disallowed child", []string{create, badChild}, "/1/updateComponents/components/0/children/1"},
		{"disallowed root", []string{create, badRoot}, "/1/updateComponents/components/0"},
		{"allowed across messages", []string{create, parentFirst, laterCard}, ""},
		{"disallowed across messages", []string{create, parentFirst, laterButton}, "/1/updateComponents/components/0/children/0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte("[" + strings.Join(tt.msgs, ",") + "]"))
			if tt.path == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, ErrNotAllowed) {
				t.Fatalf("ValidateJSON() = %v, want ErrNotAllowed", err)
			}
			var ve *ValidationError
			if errors.As(err, &ve) && ve.Path != tt.path {
				t.Errorf("Path = %q, want %q", ve.Path, tt.path)
			}
		})
	}
}
