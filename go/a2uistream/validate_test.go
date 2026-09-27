package a2uistream

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

type recordingValidator struct {
	batches [][]a2ui.AgentMessage
	err     error
}

func (r *recordingValidator) ValidateMessages(msgs []a2ui.AgentMessage) error {
	r.batches = append(r.batches, msgs)
	return r.err
}

const (
	v1Content  = `hi <a2ui-json>{"version":"v1.0","deleteSurface":{"surfaceId":"s1"}}</a2ui-json> bye`
	v09Content = `hi <a2ui-json>{"version":"v0.9","deleteSurface":{"surfaceId":"s1"}}</a2ui-json> bye`
)

func TestParseAndValidate(t *testing.T) {
	var r recordingValidator
	if _, err := ParseAndValidate(v1Content, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.batches) != 1 || r.batches[0][0].DeleteSurface.SurfaceID != "s1" {
		t.Fatalf("validated = %+v", r.batches)
	}

	errInvalid := errors.New("invalid")
	r = recordingValidator{err: errInvalid}
	if _, err := ParseAndValidate(v1Content, &r); !errors.Is(err, errInvalid) {
		t.Fatalf("ParseAndValidate() = %v, want validator error", err)
	}
}

func TestParseAndValidateRejectsOtherVersions(t *testing.T) {
	var r recordingValidator
	for _, content := range []string{
		v09Content,
		`<a2ui-json>{"version":"v0.8","deleteSurface":{"surfaceId":"s1"}}</a2ui-json>`,
		`<a2ui-json>{"deleteSurface":{"surfaceId":"s1"}}</a2ui-json>`,
		// Decodes as a payload but is not a valid 1.x message.
		`<a2ui-json>{"version":"v1.0","deleteSurface":{"surfaceId":"s1"},"updateDataModel":{"surfaceId":"s1"}}</a2ui-json>`,
	} {
		if _, err := ParseAndValidate(content, &r); !errors.Is(err, ErrInvalidPayload) {
			t.Errorf("ParseAndValidate(%s) = %v, want ErrInvalidPayload", content, err)
		}
	}
	if len(r.batches) != 0 {
		t.Fatalf("validator called for rejected content: %+v", r.batches)
	}
}

func TestFixPayloadInvalid(t *testing.T) {
	for _, s := range []string{"", "{", "not json"} {
		if _, err := FixPayload(s); !errors.Is(err, ErrInvalidPayload) {
			t.Errorf("FixPayload(%q) = %v, want ErrInvalidPayload", s, err)
		}
	}
	_, err := FixPayload("{")
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("FixPayload(%q) = %v, want the JSON error wrapped", "{", err)
	}
}

// A message that does not decode has a Payload but no Messages.
// Text after it must still start a new part.
func TestParserFlushAppendsTextAfterUndecodedPayload(t *testing.T) {
	p := NewParser()
	parts, err := p.ProcessChunk(`<a2ui-json>{"version":"v0.9","deleteSurface":{"surfaceId":"s1"}}</a2ui`)
	if err != nil {
		t.Fatal(err)
	}
	flush, err := p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	parts = append(parts, flush...)
	if len(parts) != 2 || len(parts[0].Payload) != 1 || parts[0].Text != "" || parts[1].Text != "</a2ui" {
		t.Fatalf("parts = %+v, want the payload then text %q", parts, "</a2ui")
	}
}
