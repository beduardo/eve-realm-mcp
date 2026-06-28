// Package registry provides the ToolRegistry interface and its concurrent-safe
// MapRegistry implementation for managing tools in the MCP Server.
package registry

import (
	"fmt"
	"sync"
)

// Tool represents a single registered tool with its metadata and handler.
type Tool struct {
	// Name is the unique identifier for the tool (used as the dispatch key).
	Name string
	// Description is a human-readable explanation of what the tool does.
	Description string
	// InputSchema is a JSON Schema string describing the tool's input parameters.
	InputSchema string
	// Handler is the function called when the tool is invoked. It receives a
	// JSON-encoded input string and returns a JSON-encoded output string or an error.
	Handler func(input string) (string, error)
}

// NotFoundError is returned by Invoke when no tool with the requested name is
// registered. Callers can map this to gRPC status NOT_FOUND.
type NotFoundError struct {
	ToolName string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("tool %q not found", e.ToolName)
}

// ToolRegistry is the interface for managing tool registrations.
type ToolRegistry interface {
	// Register adds a tool to the registry. If a tool with the same name
	// already exists it is replaced.
	Register(tool Tool)
	// List returns a snapshot of all currently registered tools.
	List() []Tool
	// Invoke dispatches the call to the named tool's handler and returns its
	// output. Returns *NotFoundError if the tool name is not registered.
	Invoke(name string, input string) (string, error)
}

// MapRegistry is a concurrent-safe implementation of ToolRegistry backed by a
// plain Go map and a sync.RWMutex. Read operations (List, Invoke lookup) hold
// only a read-lock; Register holds a write-lock.
type MapRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewMapRegistry constructs and returns an empty MapRegistry.
func NewMapRegistry() *MapRegistry {
	return &MapRegistry{
		tools: make(map[string]Tool),
	}
}

// Register stores the tool in the registry under its Name. Concurrent calls are
// safe. If a tool with the same name was previously registered it is replaced.
func (r *MapRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// List returns a point-in-time snapshot of all registered tools. The returned
// slice is independent of the internal map — mutations to the slice do not
// affect the registry.
func (r *MapRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Invoke looks up the tool by name and calls its handler with the provided input.
// Returns *NotFoundError if no tool with the given name is registered.
func (r *MapRegistry) Invoke(name string, input string) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", &NotFoundError{ToolName: name}
	}
	return t.Handler(input)
}
