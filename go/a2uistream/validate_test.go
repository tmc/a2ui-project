package a2uistream

import (
	"errors"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
	"github.com/a2ui-project/a2ui/go/a2ui/v09"
)

type recordingValidator struct {
	v1  [][]a2ui.AgentMessage
	v09 [][]v09.ServerMessage
	err error
}

func (r *recordingValidator) ValidateMessages(msgs []a2ui.AgentMessage) error {
	r.v1 = append(r.v1, msgs)
	return r.err
}

func (r *recordingValidator) ValidateMessagesV09(msgs []v09.ServerMessage) error {
	r.v09 = append(r.v09, msgs)
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
	if len(r.v1) != 1 || r.v1[0][0].DeleteSurface.SurfaceID != "s1" || len(r.v09) != 0 {
		t.Fatalf("validated v1 = %+v, v09 = %+v", r.v1, r.v09)
	}

	r = recordingValidator{err: errors.New("invalid")}
	if _, err := ParseAndValidate(v1Content, &r); err == nil {
		t.Fatal("validator error not returned")
	}
}

func TestParseAndValidateV09(t *testing.T) {
	var r recordingValidator
	if _, err := ParseAndValidateV09(v09Content, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.v09) != 1 || r.v09[0][0].DeleteSurface.SurfaceID != "s1" || len(r.v1) != 0 {
		t.Fatalf("validated v1 = %+v, v09 = %+v", r.v1, r.v09)
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
		if _, err := ParseAndValidate(content, &r); err == nil {
			t.Errorf("ParseAndValidate(%s): no error", content)
		}
	}
	if _, err := ParseAndValidateV09(v1Content, &r); err == nil {
		t.Error("ParseAndValidateV09(v1.0 content): no error")
	}
	if len(r.v1)+len(r.v09) != 0 {
		t.Fatalf("validator called for rejected content: %+v %+v", r.v1, r.v09)
	}
}

func TestMessagesV09SkipsOtherVersions(t *testing.T) {
	part := ResponsePart{Payload: []map[string]any{
		{"version": "v1.0", "deleteSurface": map[string]any{"surfaceId": "a"}},
		{"version": "v0.9", "deleteSurface": map[string]any{"surfaceId": "b"}},
		{"version": "v0.9.1", "deleteSurface": map[string]any{"surfaceId": "c"}},
	}}
	msgs, err := part.MessagesV09()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].DeleteSurface.SurfaceID != "b" || msgs[1].DeleteSurface.SurfaceID != "c" {
		t.Fatalf("MessagesV09 = %+v, want b and c", msgs)
	}
}
