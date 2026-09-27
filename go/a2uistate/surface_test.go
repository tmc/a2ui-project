package a2uistate

import (
	"strings"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func text(id, s string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringLiteral(s)}}
}

func create(surfaceID string, components ...a2ui.Component) a2ui.AgentMessage {
	return a2ui.AgentMessage{CreateSurface: &a2ui.CreateSurface{
		SurfaceID:  surfaceID,
		CatalogID:  "cat",
		Components: components,
		DataModel:  map[string]any{"n": 1},
	}}
}

func update(surfaceID string, components ...a2ui.Component) a2ui.AgentMessage {
	return a2ui.AgentMessage{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: surfaceID, Components: components}}
}

func setData(surfaceID, path string, value any) a2ui.AgentMessage {
	return a2ui.AgentMessage{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: surfaceID, Path: path, Value: value}}
}

func del(surfaceID string) a2ui.AgentMessage {
	return a2ui.AgentMessage{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: surfaceID}}
}

var callFunction = a2ui.AgentMessage{CallRendererFunction: &a2ui.CallRendererFunction{}}

// state summarizes a surface as "catalog|components|data|deleted".
func state(s *Surface) string {
	var ids []string
	for _, id := range []string{"root", "a", "b"} {
		if c, ok := s.Component(id); ok {
			ids = append(ids, id+"="+*c.Text.Text.Literal)
		}
	}
	data, _ := s.Data().Get("")
	d := ""
	if s.Deleted() {
		d = "deleted"
	}
	return strings.Join([]string{s.CatalogID(), strings.Join(ids, ","), toString(data), d}, "|")
}

func TestSurfaceApply(t *testing.T) {
	tests := []struct {
		name string
		msgs []a2ui.AgentMessage
		want string
		err  string // substring of the error, if any
	}{
		{
			name: "create",
			msgs: []a2ui.AgentMessage{create("s", text("root", "r"))},
			want: `cat|root=r|{"n":1}|`,
		},
		{
			name: "update before create",
			msgs: []a2ui.AgentMessage{update("s", text("a", "x")), setData("s", "/m", "y")},
			want: `|a=x|{"m":"y"}|`,
		},
		{
			name: "upsert",
			msgs: []a2ui.AgentMessage{create("s", text("root", "r"), text("a", "x")), update("s", text("a", "y"), text("b", "z"))},
			want: `cat|root=r,a=y,b=z|{"n":1}|`,
		},
		{
			name: "update data",
			msgs: []a2ui.AgentMessage{create("s"), setData("s", "/list/0", "x"), setData("s", "/n", nil)},
			want: `cat||{"list":["x"]}|`,
		},
		{
			name: "replace data",
			msgs: []a2ui.AgentMessage{create("s"), setData("s", "", map[string]any{"m": 2})},
			want: `cat||{"m":2}|`,
		},
		{
			name: "create resets",
			msgs: []a2ui.AgentMessage{update("s", text("a", "x")), create("s", text("root", "r"))},
			want: `cat|root=r|{"n":1}|`,
		},
		{
			name: "delete",
			msgs: []a2ui.AgentMessage{create("s", text("root", "r")), del("s")},
			want: `||{}|deleted`,
		},
		{
			name: "recreate",
			msgs: []a2ui.AgentMessage{create("s", text("root", "r")), del("s"), create("s", text("a", "x"))},
			want: `cat|a=x|{"n":1}|`,
		},
		{
			name: "create twice",
			msgs: []a2ui.AgentMessage{create("s", text("root", "r")), create("s", text("a", "x"))},
			want: `cat|root=r|{"n":1}|`,
			err:  `surface "s": createSurface for existing surface`,
		},
		{
			name: "update after delete",
			msgs: []a2ui.AgentMessage{create("s"), del("s"), update("s", text("a", "x"))},
			want: `||{}|deleted`,
			err:  `surface "s": updateComponents for deleted surface`,
		},
		{
			name: "wrong surface",
			msgs: []a2ui.AgentMessage{create("s"), setData("t", "/x", 1), update("s", text("a", "x"))},
			want: `cat||{"n":1}|`,
			err:  `surface "s": updateDataModel for surface "t"`,
		},
		{
			name: "not a surface message",
			msgs: []a2ui.AgentMessage{callFunction},
			want: `||{}|`,
			err:  "message addresses no surface",
		},
		{
			name: "bad data update",
			msgs: []a2ui.AgentMessage{create("s"), setData("s", "/n/x", 1)},
			want: `cat||{"n":1}|`,
			err:  `surface "s": updateDataModel: a2uistate: set /n/x`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSurface("s")
			err := s.Apply(tt.msgs...)
			if tt.err == "" && err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if tt.err != "" && (err == nil || !strings.Contains(err.Error(), tt.err)) {
				t.Fatalf("Apply: error %v, want %q", err, tt.err)
			}
			if got := state(s); got != tt.want {
				t.Errorf("state = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSurfaceAccessors(t *testing.T) {
	s := NewSurface("s")
	if _, ok := s.Root(); ok {
		t.Errorf("Root reports a component before any update")
	}
	meta := &a2ui.Metadata{}
	err := s.Apply(a2ui.AgentMessage{CreateSurface: &a2ui.CreateSurface{
		SurfaceID:     "s",
		CatalogID:     a2ui.BasicCatalogID,
		SendDataModel: true,
		Metadata:      meta,
		Components:    []a2ui.Component{text("root", "r")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID() != "s" || s.CatalogID() != a2ui.BasicCatalogID || !s.SendDataModel() || s.Metadata() != meta {
		t.Errorf("accessors = %q, %q, %v, %p, want s, %q, true, %p", s.ID(), s.CatalogID(), s.SendDataModel(), s.Metadata(), a2ui.BasicCatalogID, meta)
	}
	if root, ok := s.Root(); !ok || root.ID != "root" {
		t.Errorf("Root = %v, %v, want the root component", root.ID, ok)
	}
}

func TestSurfacesApply(t *testing.T) {
	tests := []struct {
		name string
		msgs []a2ui.AgentMessage
		want map[string]string // surface ID to state
		err  string
	}{
		{
			name: "route",
			msgs: []a2ui.AgentMessage{create("s"), create("t"), update("s", text("a", "x")), setData("t", "/m", 2)},
			want: map[string]string{"s": `cat|a=x|{"n":1}|`, "t": `cat||{"m":2,"n":1}|`},
		},
		{
			name: "delete",
			msgs: []a2ui.AgentMessage{create("s"), create("t"), del("s")},
			want: map[string]string{"t": `cat||{"n":1}|`},
		},
		{
			name: "recreate",
			msgs: []a2ui.AgentMessage{create("s"), del("s"), create("s", text("a", "x"))},
			want: map[string]string{"s": `cat|a=x|{"n":1}|`},
		},
		{
			name: "create twice",
			msgs: []a2ui.AgentMessage{create("s"), create("s")},
			want: map[string]string{"s": `cat||{"n":1}|`},
			err:  `surface "s": createSurface for existing surface`,
		},
		{
			name: "unknown surface",
			msgs: []a2ui.AgentMessage{create("s"), update("t", text("a", "x"))},
			want: map[string]string{"s": `cat||{"n":1}|`},
			err:  `updateComponents for unknown surface "t"`,
		},
		{
			name: "delete unknown surface",
			msgs: []a2ui.AgentMessage{del("t")},
			want: map[string]string{},
			err:  `deleteSurface for unknown surface "t"`,
		},
		{
			name: "not a surface message",
			msgs: []a2ui.AgentMessage{callFunction},
			want: map[string]string{},
			err:  "message addresses no surface",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss Surfaces
			err := ss.Apply(tt.msgs...)
			if tt.err == "" && err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if tt.err != "" && (err == nil || !strings.Contains(err.Error(), tt.err)) {
				t.Fatalf("Apply: error %v, want %q", err, tt.err)
			}
			for _, id := range []string{"s", "t"} {
				s, ok := ss.Surface(id)
				want, wantOK := tt.want[id]
				if ok != wantOK {
					t.Errorf("Surface(%q) reports %v, want %v", id, ok, wantOK)
					continue
				}
				if ok {
					if got := state(s); got != want {
						t.Errorf("surface %q: state = %s, want %s", id, got, want)
					}
				}
			}
		})
	}
}

func TestSurfacesAdd(t *testing.T) {
	var ss Surfaces
	ss.Add(NewSurface("s"))
	if err := ss.Apply(update("s", text("a", "x"))); err != nil {
		t.Fatal(err)
	}
	if err := ss.Apply(create("s")); err == nil {
		t.Errorf("createSurface for an added surface succeeded")
	}
}

func TestSurfaceCreateFailureLeavesState(t *testing.T) {
	s := NewSurface("s")
	if err := s.Apply(update("s", text("a", "x")), setData("s", "/m", "y")); err != nil {
		t.Fatal(err)
	}
	bad := create("s", text("root", "r"))
	bad.CreateSurface.DataModel = map[string]any{"f": func() {}}
	if err := s.Apply(bad); err == nil {
		t.Fatal("createSurface with an unencodable data model succeeded")
	}
	if got, want := state(s), `|a=x|{"m":"y"}|`; got != want {
		t.Errorf("state after failed createSurface = %s, want %s", got, want)
	}
	if err := s.Apply(create("s")); err != nil {
		t.Errorf("createSurface after failed createSurface: %v", err)
	}
}
