package registry_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/beduardo/eve-realm-mcp/internal/registry"
)

// ---------------------------------------------------------------------------
// TestRegistry_Register_AddsToolToList
// ---------------------------------------------------------------------------

func TestRegistry_Register_AddsToolToList(t *testing.T) {
	cases := []struct {
		name        string
		toolName    string
		description string
		inputSchema string
	}{
		{
			name:        "single_tool_returned_by_list",
			toolName:    "my-tool",
			description: "A test tool",
			inputSchema: `{"type":"object"}`,
		},
		{
			name:        "tool_with_empty_schema",
			toolName:    "no-schema-tool",
			description: "Tool without schema",
			inputSchema: "",
		},
		{
			name:        "tool_with_complex_schema",
			toolName:    "complex-tool",
			description: "Tool with complex schema",
			inputSchema: `{"type":"object","properties":{"x":{"type":"integer"}}}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := registry.NewMapRegistry()

			tool := registry.Tool{
				Name:        tc.toolName,
				Description: tc.description,
				InputSchema: tc.inputSchema,
				Handler: func(input string) (string, error) {
					return "", nil
				},
			}
			r.Register(tool)

			tools := r.List()
			if len(tools) != 1 {
				t.Fatalf("List() returned %d tools, want 1", len(tools))
			}
			got := tools[0]
			if got.Name != tc.toolName {
				t.Errorf("List()[0].Name = %q, want %q", got.Name, tc.toolName)
			}
			if got.Description != tc.description {
				t.Errorf("List()[0].Description = %q, want %q", got.Description, tc.description)
			}
			if got.InputSchema != tc.inputSchema {
				t.Errorf("List()[0].InputSchema = %q, want %q", got.InputSchema, tc.inputSchema)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRegistry_Invoke_DispatchesToHandler
// ---------------------------------------------------------------------------

func TestRegistry_Invoke_DispatchesToHandler(t *testing.T) {
	cases := []struct {
		name       string
		toolName   string
		input      string
		handlerOut string
	}{
		{
			name:       "handler_echoes_input",
			toolName:   "echo-tool",
			input:      `{"msg":"hello"}`,
			handlerOut: `{"msg":"hello"}`,
		},
		{
			name:       "handler_returns_fixed_output",
			toolName:   "fixed-tool",
			input:      `{}`,
			handlerOut: `{"result":"ok"}`,
		},
		{
			name:       "handler_with_empty_input",
			toolName:   "empty-input-tool",
			input:      "",
			handlerOut: `{"status":"done"}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := registry.NewMapRegistry()

			captured := tc.handlerOut // capture for closure
			r.Register(registry.Tool{
				Name:        tc.toolName,
				Description: "test",
				InputSchema: `{}`,
				Handler: func(input string) (string, error) {
					return captured, nil
				},
			})

			got, err := r.Invoke(tc.toolName, tc.input)
			if err != nil {
				t.Fatalf("Invoke(%q, %q) returned unexpected error: %v", tc.toolName, tc.input, err)
			}
			if got != tc.handlerOut {
				t.Errorf("Invoke(%q, %q) = %q, want %q", tc.toolName, tc.input, got, tc.handlerOut)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRegistry_Invoke_UnknownTool_ReturnsNotFound
// ---------------------------------------------------------------------------

func TestRegistry_Invoke_UnknownTool_ReturnsNotFound(t *testing.T) {
	cases := []struct {
		name     string
		register []string // tool names to register before invoke
		invoke   string   // tool name to invoke (not registered)
	}{
		{
			name:     "empty_registry",
			register: nil,
			invoke:   "nonexistent",
		},
		{
			name:     "other_tools_registered",
			register: []string{"tool-a", "tool-b"},
			invoke:   "nonexistent",
		},
		{
			name:     "similar_name_does_not_match",
			register: []string{"ping"},
			invoke:   "pin",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := registry.NewMapRegistry()
			for _, name := range tc.register {
				n := name
				r.Register(registry.Tool{
					Name:    n,
					Handler: func(input string) (string, error) { return "", nil },
				})
			}

			_, err := r.Invoke(tc.invoke, "{}")
			if err == nil {
				t.Fatalf("Invoke(%q) expected not-found error, got nil", tc.invoke)
			}
			var nfe *registry.NotFoundError
			if !errors.As(err, &nfe) {
				t.Errorf("Invoke(%q) error type = %T, want *registry.NotFoundError", tc.invoke, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRegistry_ConcurrentAccess_NoRace
// ---------------------------------------------------------------------------

func TestRegistry_ConcurrentAccess_NoRace(t *testing.T) {
	r := registry.NewMapRegistry()

	// Pre-register one tool so Invoke calls have something to find.
	r.Register(registry.Tool{
		Name:        "seed-tool",
		Description: "pre-seeded",
		InputSchema: `{}`,
		Handler: func(input string) (string, error) {
			return `{"ok":true}`, nil
		},
	})

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent Register goroutines.
	for i := range goroutines {
		i := i
		go func() {
			defer wg.Done()
			r.Register(registry.Tool{
				Name:    fmt.Sprintf("race-tool-%d", i),
				Handler: func(input string) (string, error) { return "", nil },
			})
		}()
	}

	// Concurrent List goroutines.
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = r.List()
		}()
	}

	// Concurrent Invoke goroutines (seed-tool is always registered).
	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = r.Invoke("seed-tool", "{}")
		}()
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// TestRegistry_ConcurrentRegisterListInvoke_NoRace
// ---------------------------------------------------------------------------

func TestRegistry_ConcurrentRegisterListInvoke_NoRace(t *testing.T) {
	r := registry.NewMapRegistry()

	// Pre-register a tool that all Invoke goroutines will call.
	r.Register(registry.Tool{
		Name:    "stable-tool",
		Handler: func(input string) (string, error) { return `"pong"`, nil },
	})

	const n = 30
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		i := i
		go func() {
			defer wg.Done()
			switch i % 3 {
			case 0:
				r.Register(registry.Tool{
					Name:    fmt.Sprintf("dyn-tool-%d", i),
					Handler: func(input string) (string, error) { return "", nil },
				})
			case 1:
				_ = r.List()
			case 2:
				_, _ = r.Invoke("stable-tool", "{}")
			}
		}()
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// TestRegistry_ConcurrentRegister_AllToolsVisible
// ---------------------------------------------------------------------------

func TestRegistry_ConcurrentRegister_AllToolsVisible(t *testing.T) {
	r := registry.NewMapRegistry()

	const count = 50
	var wg sync.WaitGroup
	wg.Add(count)

	for i := range count {
		i := i
		go func() {
			defer wg.Done()
			r.Register(registry.Tool{
				Name:    fmt.Sprintf("bulk-tool-%d", i),
				Handler: func(input string) (string, error) { return "", nil },
			})
		}()
	}

	wg.Wait()

	tools := r.List()
	if len(tools) != count {
		t.Errorf("List() returned %d tools after %d concurrent Register calls, want %d",
			len(tools), count, count)
	}

	// Verify all expected tool names are present (no silent drops).
	seen := make(map[string]bool, count)
	for _, tool := range tools {
		seen[tool.Name] = true
	}
	for i := range count {
		name := fmt.Sprintf("bulk-tool-%d", i)
		if !seen[name] {
			t.Errorf("tool %q not found in List() after concurrent Register", name)
		}
	}
}
