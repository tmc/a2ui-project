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
			msgs: `[{"version":"v1.0","createSurface":{"surfaceId":"s1"}},{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Column","children":["greeting"]},{"id":"greeting","component":"Text","text":"hello"},{"id":"extra","component":"Text","text":"orphan"}]}}]`,
			want: ErrInvalidTree,
			path: "/1/updateComponents/components/2",
		},
		{
			name: "missing root",
			msgs: `[{"version":"v1.0","createSurface":{"surfaceId":"s1"}},{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"a","component":"Text","text":"a"}]}}]`,
			want: ErrInvalidTree,
			path: "/0/createSurface",
		},
		{
			name: "cycle without root",
			msgs: `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"a","component":"Card","child":"b"},{"id":"b","component":"Card","child":"a"}]}}`,
			want: ErrInvalidTree,
			path: "/updateComponents/components/0",
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

func TestValidatorUnknownCustomComponent(t *testing.T) {
	const msgs = `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Gauge","value":3}]}}`
	err := mustBasicValidator(t).ValidateJSON([]byte(msgs))
	if !errors.Is(err, ErrUnknownComponent) {
		t.Fatalf("ValidateJSON() = %v, want ErrUnknownComponent", err)
	}
}

func TestValidatorCustomCatalog(t *testing.T) {
	manager, err := NewSchemaManager([]CatalogConfig{
		CatalogConfigFromPath("custom", "testdata/custom_catalog.json", ""),
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.SelectedCatalog(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	validator := catalog.Validator()

	const create = `{"version":"v1.0","createSurface":{"surfaceId":"s1","catalogId":"https://example.com/custom_catalog.json"}}`
	update := func(components string) string {
		return `{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[` + components + `]}}`
	}
	tests := []struct {
		name string
		msgs []string
		err  error
		path string
	}{
		{"all refs", []string{create, update(`{"id":"root","component":"Panel","header":"h","items":["a","b"],"sections":[{"body":"c"}]},{"id":"h","component":"Gauge"},{"id":"a","component":"Gauge"},{"id":"b","component":"Gauge"},{"id":"c","component":"Gauge"}`)}, nil, ""},
		{"template", []string{create, update(`{"id":"root","component":"Panel","items":{"componentId":"t","path":"/xs"}},{"id":"t","component":"Gauge"}`)}, nil, ""},
		{"orphan", []string{create, update(`{"id":"root","component":"Panel","header":"h"},{"id":"h","component":"Gauge"},{"id":"x","component":"Gauge"}`)}, ErrInvalidTree, "/1/updateComponents/components/2"},
		{"unknown ref", []string{create, update(`{"id":"root","component":"Panel","sections":[{"body":"missing"}]}`)}, ErrInvalidTree, "/1/updateComponents/components/0/sections/0/body"},
		{"cycle", []string{create, update(`{"id":"root","component":"Panel","header":"p"},{"id":"p","component":"Panel","items":["root"]}`)}, ErrInvalidTree, "/1/updateComponents/components/0"},
		{"unknown type", []string{create, update(`{"id":"root","component":"Dial"}`)}, ErrUnknownComponent, "/1/updateComponents/components/0/component"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte("[" + strings.Join(tt.msgs, ",") + "]"))
			if tt.err == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tt.err) {
				t.Fatalf("ValidateJSON() = %v, want %v", err, tt.err)
			}
			var ve *ValidationError
			if errors.As(err, &ve) && ve.Path != tt.path {
				t.Errorf("Path = %q, want %q", ve.Path, tt.path)
			}
		})
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
		// Updates to a surface created before the batch may refer to
		// components outside it, and need not include root.
		{"existing_surface_ref", []string{root}, false},
		{"existing_surface_no_root", []string{body}, false},
		{"existing_surface_orphan", []string{`{"version":"v1.0","updateComponents":{"surfaceId":"s1","components":[{"id":"root","component":"Card","child":"body"},{"id":"body","component":"Text","text":"hi"},{"id":"extra","component":"Text","text":"hi"}]}}`}, false},
		{"recreated_surface", []string{root, deleteS, create, body}, true},
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

	// See testdata/composition_catalog.json for the constraints.
	create := func(surfaceID string) string {
		return `{"version":"v1.0","createSurface":{"surfaceId":"` + surfaceID + `","catalogId":"https://example.com/composition_catalog.json"}}`
	}
	update := func(surfaceID, components string) string {
		return `{"version":"v1.0","updateComponents":{"surfaceId":"` + surfaceID + `","components":[` + components + `]}}`
	}
	const (
		label  = `{"id":"label","component":"Text","text":"Go"}`
		button = `{"id":"button","component":"Button","child":"label","action":{"event":{"name":"go"}}},` + label
		card   = `{"id":"card","component":"Card","child":"label"},` + label
	)
	tests := []struct {
		name string
		msgs []string
		err  error // nil if valid
		path string
	}{
		{"allowed", []string{create("s1"), update("s1", `{"id":"root","component":"Column","children":["card","text"]},{"id":"card","component":"Card","child":"button"},{"id":"text","component":"Text","text":"hi"},`+button)}, nil, ""},
		{"disallowed parent", []string{create("s1"), update("s1", `{"id":"root","component":"Card","child":"inner"},{"id":"inner","component":"Card","child":"label"},`+label)}, ErrUnallowedParent, "/1/updateComponents/components/0/child"},
		{"disallowed child", []string{create("s1"), update("s1", `{"id":"root","component":"Column","children":["text","inner"]},{"id":"text","component":"Text","text":"hi"},{"id":"inner","component":"Column","children":["label"]},`+label)}, ErrUnallowedChild, "/1/updateComponents/components/0/children/1"},
		{"disallowed root", []string{create("s1"), update("s1", `{"id":"root","component":"Button","child":"label","action":{"event":{"name":"go"}}},`+label)}, ErrUnallowedParent, "/1/updateComponents/components/0"},
		{"allowed across messages", []string{create("s1"), update("s1", `{"id":"root","component":"Column","children":["card"]}`), update("s1", card)}, nil, ""},
		{"disallowed across messages", []string{create("s1"), update("s1", `{"id":"root","component":"Column","children":["button"]}`), update("s1", button)}, ErrUnallowedParent, "/1/updateComponents/components/0/children/0"},

		{"modal allowed", []string{create("s1"), update("s1", `{"id":"root","component":"Modal","trigger":"button","content":"card"},`+button+`,{"id":"card","component":"Card","child":"label"}`)}, nil, ""},
		{"modal trigger", []string{create("s1"), update("s1", `{"id":"root","component":"Modal","trigger":"label","content":"card"},`+card)}, ErrUnallowedChild, "/1/updateComponents/components/0/trigger"},
		{"modal content", []string{create("s1"), update("s1", `{"id":"root","component":"Modal","trigger":"button","content":"tabs"},{"id":"tabs","component":"Tabs","tabs":[{"title":"A","child":"card"}]},{"id":"card","component":"Card","child":"label"},`+button)}, ErrUnallowedChild, "/1/updateComponents/components/0/content"},
		{"tabs allowed", []string{create("s1"), update("s1", `{"id":"root","component":"Tabs","tabs":[{"title":"A","child":"card"}]},`+card)}, nil, ""},
		{"tabs child", []string{create("s1"), update("s1", `{"id":"root","component":"Tabs","tabs":[{"title":"A","child":"card"},{"title":"B","child":"label"}]},{"id":"card","component":"Card","child":"label"},`+label)}, ErrUnallowedChild, "/1/updateComponents/components/0/tabs/1/child"},
		{"template allowed", []string{create("s1"), update("s1", `{"id":"root","component":"List","children":{"componentId":"card","path":"/items"}},`+card)}, nil, ""},
		{"template child", []string{create("s1"), update("s1", `{"id":"root","component":"List","children":{"componentId":"label","path":"/items"}},`+label)}, ErrUnallowedChild, "/1/updateComponents/components/0/children/componentId"},

		// An empty list allows no component; an omitted list allows any.
		{"empty allowedChildren", []string{create("s1"), update("s1", `{"id":"root","component":"Row","children":["label"]},`+label)}, ErrUnallowedChild, "/1/updateComponents/components/0/children/0"},
		{"empty allowedParents", []string{create("s1"), update("s1", `{"id":"root","component":"Divider"}`)}, ErrUnallowedParent, "/1/updateComponents/components/0"},
		{"omitted lists", []string{create("s1"), update("s1", `{"id":"root","component":"Card","child":"col"},{"id":"col","component":"Column","children":["label"]},`+label)}, nil, ""},

		{"multiple surfaces", []string{create("s1"), create("s2"), update("s1", `{"id":"root","component":"Card","child":"label"},`+label), update("s2", `{"id":"root","component":"Column","children":["button"]},`+button)}, ErrUnallowedParent, "/3/updateComponents/components/0/children/0"},
		{"no edges across surfaces", []string{update("s1", `{"id":"root","component":"Column","children":["button"]}`), update("s2", button)}, nil, ""},
		{"replaced child", []string{create("s1"), update("s1", `{"id":"root","component":"Column","children":["x"]},{"id":"x","component":"Card","child":"label"},`+label), update("s1", `{"id":"x","component":"Button","child":"label","action":{"event":{"name":"go"}}}`)}, ErrUnallowedParent, "/1/updateComponents/components/0/children/0"},
		{"replaced root", []string{create("s1"), update("s1", `{"id":"root","component":"Card","child":"label"},`+label), update("s1", `{"id":"root","component":"Divider"}`)}, ErrUnallowedParent, "/2/updateComponents/components/0"},
		{"recreated surface", []string{create("s1"), update("s1", `{"id":"root","component":"Card","child":"col"},{"id":"col","component":"Column","children":["x"]},{"id":"x","component":"Card","child":"label"},`+label), create("s1"), update("s1", `{"id":"root","component":"Card","child":"x"},{"id":"x","component":"Button","child":"label","action":{"event":{"name":"go"}}},`+label)}, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte("[" + strings.Join(tt.msgs, ",") + "]"))
			if tt.err == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tt.err) {
				t.Fatalf("ValidateJSON() = %v, want %v", err, tt.err)
			}
			var ve *ValidationError
			if errors.As(err, &ve) && ve.Path != tt.path {
				t.Errorf("Path = %q, want %q", ve.Path, tt.path)
			}
		})
	}
}
