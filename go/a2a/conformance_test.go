package a2a

import (
	"reflect"
	"testing"
)

type conformanceCase struct {
	Name   string
	Action string
	Args   map[string]any
	Expect any
}

func TestA2AConformance(t *testing.T) {
	for _, tc := range a2aConformanceCases() {
		t.Run(tc.Name, func(t *testing.T) {
			runA2AConformanceCase(t, tc)
		})
	}
}

func runA2AConformanceCase(t *testing.T, tc conformanceCase) {
	switch tc.Action {
	case "create_a2ui_part":
		part, err := CreateDataPart(tc.Args["data"])
		if err != nil {
			t.Fatal(err)
		}
		if !IsA2UIPart(part) {
			t.Fatal("part is not A2UI")
		}
		expect := expectMap(t, tc.Expect)
		if got, want := part.Metadata[MIMETypeKey], expect["mime_type"]; got != want {
			t.Fatalf("mime type = %q, want %q", got, want)
		}
	case "is_a2ui_part":
		part := DataPart{Data: map[string]any{}, Metadata: map[string]any{MIMETypeKey: tc.Args["mime_type"]}}
		if got, want := IsA2UIPart(part), tc.Expect; got != want {
			t.Fatalf("IsA2UIPart = %v, want %v", got, want)
		}
	case "get_extension":
		opts := AgentExtensionOptions{
			Version: tc.Args["version"].(string),
		}
		if v, ok := tc.Args["accepts_inline_catalogs"].(bool); ok {
			opts.AcceptsInlineCatalogs = v
		}
		if v, ok := tc.Args["supported_catalog_ids"].([]any); ok {
			opts.SupportedCatalogIDs = stringSlice(v)
		}
		ext := NewAgentExtension(opts)
		expect := expectMap(t, tc.Expect)
		if got, want := ext.URI, expect["uri"]; got != want {
			t.Fatalf("uri = %q, want %q", got, want)
		}
		if !reflect.DeepEqual(normalizeNilMap(ext.Params), normalizeMap(expect["params"])) {
			t.Fatalf("params = %#v, want %#v", ext.Params, expect["params"])
		}
	case "try_activate":
		activated, version, ok := TryActivateExtension(stringSlice(tc.Args["requested"].([]any)), stringSlice(tc.Args["advertised"].([]any)))
		expect := expectMap(t, tc.Expect)
		if expect["activated"] == nil {
			if ok {
				t.Fatalf("activated = %q, version = %q, want no activation", activated, version)
			}
			return
		}
		if !ok {
			t.Fatal("activation failed")
		}
		if got, want := activated, expect["activated"]; got != want {
			t.Fatalf("activated = %q, want %q", got, want)
		}
		if got, want := version, expect["version"]; got != want {
			t.Fatalf("version = %q, want %q", got, want)
		}
	case "select_newest":
		got, _ := SelectNewestRequestedExtension(stringSlice(tc.Args["requested"].([]any)), stringSlice(tc.Args["advertised"].([]any)))
		expect := expectMap(t, tc.Expect)
		if want := expect["newest"]; got != want {
			t.Fatalf("newest = %q, want %q", got, want)
		}
	default:
		t.Fatalf("unsupported action %q", tc.Action)
	}
}

func expectMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expect has type %T, want map", v)
	}
	return m
}

func stringSlice(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	return out
}

func normalizeMap(v any) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			out[key] = normalizeMap(value)
		}
		return out
	case []any:
		return stringSlice(v)
	default:
		return v
	}
}

func normalizeNilMap(m map[string]any) any {
	if m == nil {
		return nil
	}
	return m
}

func a2aConformanceCases() []conformanceCase {
	// These cases mirror conformance/extensions/a2a/a2a_integration.yaml.
	return []conformanceCase{
		{
			Name:   "test_create_a2ui_part",
			Action: "create_a2ui_part",
			Args: map[string]any{
				"data": map[string]any{"foo": "bar"},
			},
			Expect: map[string]any{"mime_type": "application/a2ui+json"},
		},
		{
			Name:   "test_is_a2ui_part",
			Action: "is_a2ui_part",
			Args:   map[string]any{"mime_type": "application/a2ui+json"},
			Expect: true,
		},
		{
			Name:   "test_get_extension_minimal",
			Action: "get_extension",
			Args:   map[string]any{"version": "1.0"},
			Expect: map[string]any{
				"uri":    "https://a2ui.org/a2a-extension/a2ui/v1.0",
				"params": nil,
			},
		},
		{
			Name:   "test_get_extension_with_inline",
			Action: "get_extension",
			Args: map[string]any{
				"version":                 "1.0",
				"accepts_inline_catalogs": true,
			},
			Expect: map[string]any{
				"uri":    "https://a2ui.org/a2a-extension/a2ui/v1.0",
				"params": map[string]any{"acceptsInlineCatalogs": true},
			},
		},
		{
			Name:   "test_get_extension_with_catalogs",
			Action: "get_extension",
			Args: map[string]any{
				"version":               "1.0",
				"supported_catalog_ids": []any{"a", "b", "c"},
			},
			Expect: map[string]any{
				"uri":    "https://a2ui.org/a2a-extension/a2ui/v1.0",
				"params": map[string]any{"supportedCatalogIds": []any{"a", "b", "c"}},
			},
		},
		{
			Name:   "test_try_activate_success",
			Action: "try_activate",
			Args: map[string]any{
				"requested":  []any{"https://a2ui.org/a2a-extension/a2ui/v1.0"},
				"advertised": []any{"https://a2ui.org/a2a-extension/a2ui/v1.0"},
			},
			Expect: map[string]any{
				"activated": "https://a2ui.org/a2a-extension/a2ui/v1.0",
				"version":   "1.0",
			},
		},
		{
			Name:   "test_try_activate_not_requested",
			Action: "try_activate",
			Args: map[string]any{
				"requested":  []any{},
				"advertised": []any{"https://a2ui.org/a2a-extension/a2ui/v1.0"},
			},
			Expect: map[string]any{"activated": nil},
		},
		{
			Name:   "test_select_newest",
			Action: "select_newest",
			Args: map[string]any{
				"requested": []any{
					"https://a2ui.org/a2a-extension/a2ui/v1.0.0",
					"https://a2ui.org/a2a-extension/a2ui/v1.2.0",
					"https://a2ui.org/a2a-extension/a2ui/v1.1.0",
					"https://a2ui.org/a2a-extension/a2ui/v1.10.0",
				},
				"advertised": []any{
					"https://a2ui.org/a2a-extension/a2ui/v1.0.0",
					"https://a2ui.org/a2a-extension/a2ui/v1.2.0",
					"https://a2ui.org/a2a-extension/a2ui/v1.10.0",
					"https://a2ui.org/a2a-extension/a2ui/v2.0.0",
				},
			},
			Expect: map[string]any{"newest": "https://a2ui.org/a2a-extension/a2ui/v1.10.0"},
		},
	}
}
