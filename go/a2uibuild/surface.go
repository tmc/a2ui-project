package a2uibuild

import (
	"maps"
	"slices"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// Surface builds a complete A2UI surface as a sequence of agent messages.
type Surface struct {
	surfaceID  string
	catalogID  string
	components []a2ui.Component
	data       map[string]any
	sendData   bool
}

// NewSurface returns a surface builder with the given surface and catalog IDs.
func NewSurface(surfaceID, catalogID string) Surface {
	return Surface{
		surfaceID: surfaceID,
		catalogID: catalogID,
	}
}

// WithSendDataModel returns a surface that requests client data in actions.
func (s Surface) WithSendDataModel() Surface {
	s.sendData = true
	return s
}

// Add returns a surface with c appended.
func (s Surface) Add(c a2ui.Component) Surface {
	s.components = append(slices.Clone(s.components), c)
	return s
}

// WithData returns a surface with the initial data model set.
func (s Surface) WithData(data map[string]any) Surface {
	s.data = maps.Clone(data)
	return s
}

// Messages returns the agent messages needed to render this surface.
func (s Surface) Messages() []a2ui.AgentMessage {
	var msgs []a2ui.AgentMessage

	msgs = append(msgs, a2ui.AgentMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID:     s.surfaceID,
			CatalogID:     s.catalogID,
			SendDataModel: s.sendData,
		},
	})

	if len(s.components) > 0 {
		msgs = append(msgs, a2ui.AgentMessage{
			Version: a2ui.Version,
			UpdateComponents: &a2ui.UpdateComponents{
				SurfaceID:  s.surfaceID,
				Components: slices.Clone(s.components),
			},
		})
	}

	if len(s.data) > 0 {
		msgs = append(msgs, a2ui.AgentMessage{
			Version: a2ui.Version,
			UpdateDataModel: &a2ui.UpdateDataModel{
				SurfaceID: s.surfaceID,
				Value:     maps.Clone(s.data),
			},
		})
	}

	return msgs
}
