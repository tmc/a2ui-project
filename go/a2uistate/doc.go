// Package a2uistate holds the renderer-side state of A2UI 1.x surfaces
// ([github.com/a2ui-project/a2ui/go/a2ui]).
//
// A [Surface] applies the agent messages addressed to one surface: its
// catalog ID, its components by ID, and its [DataModel]. [Surfaces]
// routes messages to surfaces by ID. A DataModel stores a JSON value
// addressed by JSON Pointers and resolves data bindings, including the
// relative paths used inside list templates.
//
// Function calls are not evaluated: resolving a dynamic value that is a
// function call reports that no value is available.
//
// The types in this package are not safe for concurrent use.
package a2uistate
