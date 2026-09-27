// Package a2uistate holds the renderer-side state of A2UI 1.x surfaces
// ([github.com/a2ui-project/a2ui/go/a2ui]).
//
// A [Surface] applies the agent messages addressed to one surface: its
// catalog ID, its components by ID, and its [DataModel]. [Surfaces]
// routes messages to surfaces by ID. A DataModel stores a JSON value
// addressed by JSON Pointers and resolves data bindings, including the
// relative paths used inside list templates.
//
// An [Evaluator] resolves dynamic values against a data model and
// evaluates their function calls, including the ${...} expressions of
// formatString templates. By default it has the functions of the basic
// catalog, which [BasicFunctions] lists along with how they differ from
// the web renderers. A renderer with other functions adds them to the
// map that BasicFunctions returns and sets it as [Evaluator.Funcs],
// which replaces the basic functions when it is not nil.
// [Evaluator.Check] evaluates the checks of a component and
// returns the failures. The Resolve methods of DataModel evaluate only
// the basic functions and drop errors, reporting only whether a value
// is available.
//
// Functions that perform actions, such as openUrl, are not performed:
// evaluating one for its value fails with [ErrAction]. A renderer
// performs actions itself, using [Evaluator.ResolveArgs] to resolve
// their arguments.
//
// The errors of an Evaluator wrap [ErrNoValue], [ErrWrongType],
// [ErrUnknownFunction], [ErrInvalidArgs] or [ErrAction].
//
// The types in this package are not safe for concurrent use.
package a2uistate
