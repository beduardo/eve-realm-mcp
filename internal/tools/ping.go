// Package tools provides built-in tool implementations for the MCP Server.
package tools

import (
	"encoding/json"
	"time"

	"github.com/beduardo/eve-realm-mcp/internal/registry"
)

// pingResponse is the JSON payload returned by the ping handler.
type pingResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// NewTool returns a registry.Tool descriptor for the ping diagnostic tool.
// The handler produces a JSON object with exactly two fields: "message" (always
// "pong") and "timestamp" (the current server time in RFC 3339 format).
func NewTool() registry.Tool {
	return registry.Tool{
		Name:        "ping",
		Description: "Diagnostic tool that returns pong with server timestamp",
		InputSchema: "{}",
		Handler:     pingHandler,
	}
}

// pingHandler implements the ping tool. It ignores its input and returns a JSON
// object containing a static "pong" message and the current server timestamp
// formatted as RFC 3339.
func pingHandler(_ string) (string, error) {
	resp := pingResponse{
		Message:   "pong",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	out, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
