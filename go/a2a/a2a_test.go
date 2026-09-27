package a2a

import "testing"

func TestCreateDataPart(t *testing.T) {
	part, err := CreateDataPart(map[string]any{"version": "v1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !IsA2UIPart(part) {
		t.Fatal("expected A2UI part")
	}
	if _, ok := A2UIData(part); !ok {
		t.Fatal("expected A2UI data")
	}
	if got := part.Metadata[MIMETypeKey]; got != A2UIMIMEType {
		t.Fatalf("mime type = %q, want %q", got, A2UIMIMEType)
	}
}

func TestIsA2UIPartRejectsOtherMIMETypes(t *testing.T) {
	for _, mimeType := range []string{"application/json+a2ui", "application/json", ""} {
		part := DataPart{Metadata: map[string]any{MIMETypeKey: mimeType}}
		if IsA2UIPart(part) {
			t.Errorf("IsA2UIPart(%q) = true", mimeType)
		}
	}
}

func TestMarshalA2UIDataClonesMapPayload(t *testing.T) {
	payload := map[string]any{"version": "v1.0"}
	data, err := MarshalA2UIData(payload)
	if err != nil {
		t.Fatal(err)
	}
	data["version"] = "changed"
	if got := payload["version"]; got != "v1.0" {
		t.Fatalf("payload version = %q, want unchanged", got)
	}
}

func TestCreateDataPartRejectsNonObject(t *testing.T) {
	if _, err := CreateDataPart([]string{"not", "an", "object"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewAgentExtension(t *testing.T) {
	ext := NewAgentExtension(AgentExtensionOptions{
		Version:               "1.1",
		AcceptsInlineCatalogs: true,
		SupportedCatalogIDs:   []string{"catalog"},
	})
	if ext.URI != "https://a2ui.org/a2a-extension/a2ui/v1.1" {
		t.Fatalf("uri = %q", ext.URI)
	}
	if ext.Params[AcceptsInlineCatalogsKey] != true {
		t.Fatal("expected acceptsInlineCatalogs param")
	}
}

func TestSelectNewestRequestedExtension(t *testing.T) {
	got, ok := SelectNewestRequestedExtension(
		[]string{
			"https://a2ui.org/a2a-extension/a2ui/v1.0",
			"https://a2ui.org/a2a-extension/a2ui/v1.1",
		},
		[]string{
			"https://a2ui.org/a2a-extension/a2ui/v1.0",
			"https://a2ui.org/a2a-extension/a2ui/v1.1",
		},
	)
	if !ok {
		t.Fatal("expected a match")
	}
	if want := "https://a2ui.org/a2a-extension/a2ui/v1.1"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultExtensionVersion(t *testing.T) {
	ext := NewAgentExtension(AgentExtensionOptions{})
	if want := A2UIExtensionBaseURI + "/v1.0"; ext.URI != want {
		t.Fatalf("default extension URI = %q, want %q", ext.URI, want)
	}
}
