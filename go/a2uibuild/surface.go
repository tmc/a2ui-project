package a2uibuild

import (
	"maps"

	"github.com/a2ui-project/a2ui/go/a2ui/v09"
)

// Surface builds a complete A2UI surface as a sequence of server messages.
type Surface struct {
	surfaceID  string
	catalogID  string
	theme      *v09.Theme
	components []v09.Component
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

// WithTheme returns a surface with theme parameters set.
func (s Surface) WithTheme(t v09.Theme) Surface {
	s.theme = &t
	return s
}

// WithSendDataModel returns a surface that requests client data in actions.
func (s Surface) WithSendDataModel() Surface {
	s.sendData = true
	return s
}

// Add returns a surface with c appended.
func (s Surface) Add(c v09.Component) Surface {
	s.components = append(slicesClone(s.components), c)
	return s
}

// WithData returns a surface with the initial data model set.
func (s Surface) WithData(data map[string]any) Surface {
	s.data = maps.Clone(data)
	return s
}

// Messages returns the server messages needed to render this surface.
func (s Surface) Messages() []v09.ServerMessage {
	var msgs []v09.ServerMessage

	msgs = append(msgs, v09.ServerMessage{
		Version: v09.Version,
		CreateSurface: &v09.CreateSurface{
			SurfaceID:     s.surfaceID,
			CatalogID:     s.catalogID,
			Theme:         s.theme,
			SendDataModel: s.sendData,
		},
	})

	if len(s.components) > 0 {
		msgs = append(msgs, v09.ServerMessage{
			Version: v09.Version,
			UpdateComponents: &v09.UpdateComponents{
				SurfaceID:  s.surfaceID,
				Components: slicesClone(s.components),
			},
		})
	}

	if len(s.data) > 0 {
		msgs = append(msgs, v09.ServerMessage{
			Version: v09.Version,
			UpdateDataModel: &v09.UpdateDataModel{
				SurfaceID: s.surfaceID,
				Value:     maps.Clone(s.data),
			},
		})
	}

	return msgs
}

func slicesClone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S(nil), s...)
}
