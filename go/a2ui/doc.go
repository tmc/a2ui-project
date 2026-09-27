// Package a2ui provides Go types for the A2UI (Agent-to-User Interface)
// protocol, a declarative JSON format for AI agents to generate
// rich, interactive user interfaces.
//
// This package implements protocol version 1.x. An [AgentMessage] travels
// from the agent to the renderer; a [RendererMessage] travels from the
// renderer to the agent.
package a2ui
