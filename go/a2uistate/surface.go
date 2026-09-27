package a2uistate

import (
	"fmt"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// RootID is the ID of the root component of a surface.
const RootID = "root"

// A Surface is the renderer-side state of one A2UI surface.
//
// A Surface exists from [NewSurface] on and accepts updates at once, as
// for a surface that an agent knows to exist; a createSurface message
// sets its catalog ID and initial state. After a deleteSurface message,
// it accepts only a createSurface message, which creates it anew.
type Surface struct {
	id            string
	created       bool // a createSurface message has been applied
	deleted       bool
	catalogID     string
	sendDataModel bool
	metadata      *a2ui.Metadata
	components    map[string]a2ui.Component
	data          DataModel
}

// NewSurface returns an empty surface with the given ID.
func NewSurface(id string) *Surface {
	return &Surface{id: id, components: make(map[string]a2ui.Component)}
}

// ID returns the ID of s.
func (s *Surface) ID() string { return s.id }

// CatalogID returns the catalog ID set by createSurface, or "".
func (s *Surface) CatalogID() string { return s.catalogID }

// SendDataModel reports whether createSurface asked the renderer to
// send the data model to the agent with each message.
func (s *Surface) SendDataModel() bool { return s.sendDataModel }

// Metadata returns the metadata set by createSurface, or nil.
func (s *Surface) Metadata() *a2ui.Metadata { return s.metadata }

// Deleted reports whether a deleteSurface message deleted s and no
// createSurface message has created it again.
func (s *Surface) Deleted() bool { return s.deleted }

// Component returns the component with the given ID.
func (s *Surface) Component(id string) (a2ui.Component, bool) {
	c, ok := s.components[id]
	return c, ok
}

// Root returns the root component, the component with ID [RootID].
// Until it arrives, a surface has nothing to render.
func (s *Surface) Root() (a2ui.Component, bool) {
	return s.Component(RootID)
}

// Data returns the data model of s.
func (s *Surface) Data() *DataModel { return &s.data }

// Apply applies msgs to s in order:
//
//   - createSurface sets the catalog ID, metadata and sendDataModel,
//     and then adds its components and replaces the data model with its
//     dataModel. It is an error if a createSurface message was already
//     applied, unless a deleteSurface message was applied after it.
//   - updateComponents adds components, replacing those with the same ID.
//   - updateDataModel sets the value at its path; see [DataModel.Set].
//   - deleteSurface clears s and marks it deleted.
//
// It is an error for a message to address another surface, to address
// s after it was deleted, other than to create it, or not to address a
// surface at all, as callRendererFunction and agentFunctionResponse do.
// Apply stops at the first error; the messages before it stay applied.
func (s *Surface) Apply(msgs ...a2ui.AgentMessage) error {
	for _, msg := range msgs {
		if err := s.apply(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Surface) apply(msg a2ui.AgentMessage) error {
	id, kind := surfaceID(msg)
	if id == "" && kind == "" {
		return fmt.Errorf("a2uistate: message addresses no surface")
	}
	if id != s.id {
		return fmt.Errorf("a2uistate: surface %q: %s for surface %q", s.id, kind, id)
	}
	if c := msg.CreateSurface; c != nil {
		if s.created && !s.deleted {
			return fmt.Errorf("a2uistate: surface %q: createSurface for existing surface", s.id)
		}
		*s = Surface{
			id:            s.id,
			created:       true,
			catalogID:     c.CatalogID,
			sendDataModel: c.SendDataModel,
			metadata:      c.Metadata,
			components:    make(map[string]a2ui.Component, len(c.Components)),
		}
		s.addComponents(c.Components)
		if err := s.data.Set("", c.DataModel); err != nil {
			return fmt.Errorf("a2uistate: surface %q: createSurface: %w", s.id, err)
		}
		return nil
	}
	if s.deleted {
		return fmt.Errorf("a2uistate: surface %q: %s for deleted surface", s.id, kind)
	}
	switch {
	case msg.UpdateComponents != nil:
		s.addComponents(msg.UpdateComponents.Components)
	case msg.UpdateDataModel != nil:
		u := msg.UpdateDataModel
		if err := s.data.Set(u.Path, u.Value); err != nil {
			return fmt.Errorf("a2uistate: surface %q: updateDataModel: %w", s.id, err)
		}
	case msg.DeleteSurface != nil:
		*s = Surface{id: s.id, created: s.created, deleted: true, components: make(map[string]a2ui.Component)}
	}
	return nil
}

func (s *Surface) addComponents(components []a2ui.Component) {
	for _, c := range components {
		s.components[c.ID] = c
	}
}

// surfaceID returns the surface ID and the name of the kind of msg,
// or two empty strings if msg does not address a surface.
func surfaceID(msg a2ui.AgentMessage) (id, kind string) {
	switch {
	case msg.CreateSurface != nil:
		return msg.CreateSurface.SurfaceID, "createSurface"
	case msg.UpdateComponents != nil:
		return msg.UpdateComponents.SurfaceID, "updateComponents"
	case msg.UpdateDataModel != nil:
		return msg.UpdateDataModel.SurfaceID, "updateDataModel"
	case msg.DeleteSurface != nil:
		return msg.DeleteSurface.SurfaceID, "deleteSurface"
	}
	return "", ""
}

// Surfaces holds the surfaces of a renderer by ID.
// The zero value is an empty set, ready to use.
type Surfaces struct {
	m map[string]*Surface
}

// Surface returns the surface with the given ID.
func (ss *Surfaces) Surface(id string) (*Surface, bool) {
	s, ok := ss.m[id]
	return s, ok
}

// Add adds s, as for a surface that exists before any createSurface
// message, replacing any surface with the same ID.
func (ss *Surfaces) Add(s *Surface) {
	if ss.m == nil {
		ss.m = make(map[string]*Surface)
	}
	ss.m[s.ID()] = s
}

// Apply applies each of msgs to the surface it addresses, as
// [Surface.Apply] does. A createSurface message adds a surface, and it
// is an error if the surface exists. A deleteSurface message removes
// the surface. It is an error for any other message to address a
// surface that does not exist, or not to address a surface at all.
// Apply stops at the first error; the messages before it stay applied.
func (ss *Surfaces) Apply(msgs ...a2ui.AgentMessage) error {
	for _, msg := range msgs {
		if err := ss.apply(msg); err != nil {
			return err
		}
	}
	return nil
}

func (ss *Surfaces) apply(msg a2ui.AgentMessage) error {
	id, kind := surfaceID(msg)
	if id == "" && kind == "" {
		return fmt.Errorf("a2uistate: message addresses no surface")
	}
	s, ok := ss.m[id]
	switch {
	case msg.CreateSurface != nil:
		if ok {
			return fmt.Errorf("a2uistate: surface %q: createSurface for existing surface", id)
		}
		s = NewSurface(id)
		if err := s.Apply(msg); err != nil {
			return err
		}
		ss.Add(s)
		return nil
	case !ok:
		return fmt.Errorf("a2uistate: %s for unknown surface %q", kind, id)
	case msg.DeleteSurface != nil:
		delete(ss.m, id)
		return nil
	}
	return s.Apply(msg)
}
