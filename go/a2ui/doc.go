// Package a2ui provides Go types for the A2UI (Agent-to-User Interface)
// protocol, a declarative JSON format for AI agents to generate
// rich, interactive user interfaces.
//
// This package implements protocol version 1.x. An [AgentMessage] travels
// from the agent to the renderer; a [RendererMessage] travels from the
// renderer to the agent.
//
// The pre-1.0 protocols are in [github.com/a2ui-project/a2ui/go/a2ui/v09]
// and [github.com/a2ui-project/a2ui/go/a2ui/v091].
package a2ui
