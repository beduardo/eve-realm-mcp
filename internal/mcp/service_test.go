package mcp_test

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	"github.com/beduardo/eve-realm-mcp/internal/mcp"
	"github.com/beduardo/eve-realm-mcp/internal/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// newTestServer starts an in-process gRPC server bound to a bufconn listener,
// registers the MCPService with the provided registry, and returns a client
// connected to that server. The test is responsible for cleanup via t.Cleanup.
func newTestServer(t *testing.T, reg registry.ToolRegistry) mcpv1.MCPServiceClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	mcpv1.RegisterMCPServiceServer(srv, mcp.NewMCPService(reg))

	go func() {
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			// Non-fatal: the server is shut down by the test cleanup.
			_ = err
		}
	}()

	t.Cleanup(func() {
		srv.Stop()
		lis.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return mcpv1.NewMCPServiceClient(conn)
}

// ---------------------------------------------------------------------------
// TestMCPService_ListTools_ReturnsDescriptors
// ---------------------------------------------------------------------------

// TestMCPService_ListTools_ReturnsDescriptors verifies that the gRPC handler
// delegates to the registry and returns a well-formed ListToolsResponse.
func TestMCPService_ListTools_ReturnsDescriptors(t *testing.T) {
	reg := registry.NewMapRegistry()
	reg.Register(registry.Tool{
		Name:        "ping",
		Description: "Checks server liveness",
		InputSchema: `{"type":"object","properties":{}}`,
		Handler:     func(input string) (string, error) { return `{"pong":true}`, nil },
	})

	client := newTestServer(t, reg)

	resp, err := client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("ListTools: response is nil")
	}
	if len(resp.Tools) == 0 {
		t.Fatal("ListTools: expected non-empty Tools list, got empty")
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_ListTools_NonEmptyRegistryResponse
// ---------------------------------------------------------------------------

// TestMCPService_ListTools_NonEmptyRegistryResponse verifies that a registry
// containing one known tool produces a non-empty list with that tool's descriptor.
func TestMCPService_ListTools_NonEmptyRegistryResponse(t *testing.T) {
	cases := []struct {
		name        string
		toolName    string
		description string
		inputSchema string
	}{
		{
			name:        "single_ping_tool",
			toolName:    "ping",
			description: "Ping the server",
			inputSchema: `{"type":"object"}`,
		},
		{
			name:        "single_echo_tool",
			toolName:    "echo",
			description: "Echo input back",
			inputSchema: `{"type":"object","properties":{"msg":{"type":"string"}}}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.NewMapRegistry()
			reg.Register(registry.Tool{
				Name:        tc.toolName,
				Description: tc.description,
				InputSchema: tc.inputSchema,
				Handler:     func(input string) (string, error) { return `{}`, nil },
			})

			client := newTestServer(t, reg)

			resp, err := client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}
			if len(resp.Tools) != 1 {
				t.Fatalf("ListTools: got %d tools, want 1", len(resp.Tools))
			}
			if resp.Tools[0].Name != tc.toolName {
				t.Errorf("Tools[0].Name = %q, want %q", resp.Tools[0].Name, tc.toolName)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_ListTools_DescriptorFields_ArePopulated
// ---------------------------------------------------------------------------

// TestMCPService_ListTools_DescriptorFields_ArePopulated verifies that every
// returned descriptor has a non-empty name, non-empty description, and a
// valid-JSON input_schema.
func TestMCPService_ListTools_DescriptorFields_ArePopulated(t *testing.T) {
	reg := registry.NewMapRegistry()
	tools := []registry.Tool{
		{
			Name:        "ping",
			Description: "Ping the server",
			InputSchema: `{"type":"object"}`,
			Handler:     func(input string) (string, error) { return `{}`, nil },
		},
		{
			Name:        "echo",
			Description: "Echo the input",
			InputSchema: `{"type":"object","properties":{"message":{"type":"string"}}}`,
			Handler:     func(input string) (string, error) { return input, nil },
		},
	}
	for _, tool := range tools {
		reg.Register(tool)
	}

	client := newTestServer(t, reg)

	resp, err := client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	for i, d := range resp.Tools {
		if d.Name == "" {
			t.Errorf("Tools[%d].Name is empty", i)
		}
		if d.Description == "" {
			t.Errorf("Tools[%d].Description is empty", i)
		}
		if d.InputSchema == "" {
			t.Errorf("Tools[%d].InputSchema is empty", i)
		}
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(d.InputSchema), &raw); err != nil {
			t.Errorf("Tools[%d].InputSchema is not valid JSON: %v", i, err)
		}
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_InvokeTool_ValidTool_ReturnsHandlerOutput
// ---------------------------------------------------------------------------

// TestMCPService_InvokeTool_ValidTool_ReturnsHandlerOutput verifies that
// calling InvokeTool with a registered tool name returns OK status and the
// handler's JSON output.
func TestMCPService_InvokeTool_ValidTool_ReturnsHandlerOutput(t *testing.T) {
	cases := []struct {
		name           string
		toolName       string
		input          string
		handlerOutput  string
	}{
		{
			name:          "ping_tool",
			toolName:      "ping",
			input:         `{}`,
			handlerOutput: `{"pong":true,"timestamp":"2026-01-01T00:00:00Z"}`,
		},
		{
			name:          "echo_tool",
			toolName:      "echo",
			input:         `{"msg":"hello"}`,
			handlerOutput: `{"msg":"hello"}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.NewMapRegistry()
			out := tc.handlerOutput
			reg.Register(registry.Tool{
				Name:        tc.toolName,
				Description: "test tool",
				InputSchema: `{"type":"object"}`,
				Handler:     func(input string) (string, error) { return out, nil },
			})

			client := newTestServer(t, reg)

			resp, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
				ToolName: tc.toolName,
				Input:    tc.input,
			})
			if err != nil {
				t.Fatalf("InvokeTool(%q): unexpected error: %v", tc.toolName, err)
			}
			if resp.Output != tc.handlerOutput {
				t.Errorf("InvokeTool(%q).Output = %q, want %q", tc.toolName, resp.Output, tc.handlerOutput)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_InvokeTool_OutputIsValidJSON
// ---------------------------------------------------------------------------

// TestMCPService_InvokeTool_OutputIsValidJSON verifies that the output field
// in a successful InvokeTool response is parseable as JSON.
func TestMCPService_InvokeTool_OutputIsValidJSON(t *testing.T) {
	reg := registry.NewMapRegistry()
	reg.Register(registry.Tool{
		Name:        "ping",
		Description: "Ping",
		InputSchema: `{"type":"object"}`,
		Handler:     func(input string) (string, error) { return `{"pong":true}`, nil },
	})

	client := newTestServer(t, reg)

	resp, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
		ToolName: "ping",
		Input:    `{}`,
	})
	if err != nil {
		t.Fatalf("InvokeTool: %v", err)
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(resp.Output), &raw); err != nil {
		t.Errorf("InvokeTool output is not valid JSON: %v (output=%q)", err, resp.Output)
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_InvokeTool_UnknownTool_ReturnsGRPCNotFound
// ---------------------------------------------------------------------------

// TestMCPService_InvokeTool_UnknownTool_ReturnsGRPCNotFound verifies that the
// gRPC handler translates a registry not-found error to gRPC codes.NotFound.
func TestMCPService_InvokeTool_UnknownTool_ReturnsGRPCNotFound(t *testing.T) {
	reg := registry.NewMapRegistry()
	// Empty registry — all tool names are unknown.
	client := newTestServer(t, reg)

	_, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
		ToolName: "nonexistent",
		Input:    `{}`,
	})
	if err == nil {
		t.Fatal("InvokeTool(nonexistent): expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("InvokeTool(nonexistent): error is not a gRPC status: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("InvokeTool(nonexistent): code = %v, want %v", st.Code(), codes.NotFound)
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_InvokeTool_UnknownTool_ReturnsNotFound
// ---------------------------------------------------------------------------

// TestMCPService_InvokeTool_UnknownTool_ReturnsNotFound verifies that gRPC
// status code is codes.NotFound when the tool name is not registered.
func TestMCPService_InvokeTool_UnknownTool_ReturnsNotFound(t *testing.T) {
	cases := []struct {
		name     string
		toolName string
	}{
		{name: "completely_unknown", toolName: "no-such-tool"},
		{name: "similar_name", toolName: "pin"},   // "ping" is registered but not "pin"
		{name: "empty_name", toolName: ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.NewMapRegistry()
			reg.Register(registry.Tool{
				Name:        "ping",
				Description: "ping",
				InputSchema: `{"type":"object"}`,
				Handler:     func(input string) (string, error) { return `{}`, nil },
			})

			client := newTestServer(t, reg)

			_, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
				ToolName: tc.toolName,
				Input:    `{}`,
			})
			if err == nil {
				t.Fatalf("InvokeTool(%q): expected not-found error, got nil", tc.toolName)
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("InvokeTool(%q): error is not a gRPC status: %v", tc.toolName, err)
			}
			if st.Code() != codes.NotFound {
				t.Errorf("InvokeTool(%q): code = %v, want %v", tc.toolName, st.Code(), codes.NotFound)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMCPService_InvokeTool_ErrorMessage_ContainsToolName
// ---------------------------------------------------------------------------

// TestMCPService_InvokeTool_ErrorMessage_ContainsToolName verifies that the
// NOT_FOUND error message contains the requested tool name.
func TestMCPService_InvokeTool_ErrorMessage_ContainsToolName(t *testing.T) {
	cases := []struct {
		name     string
		toolName string
	}{
		{name: "unknown_tool_name_in_message", toolName: "my-missing-tool"},
		{name: "another_tool_name", toolName: "does-not-exist"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.NewMapRegistry()
			client := newTestServer(t, reg)

			_, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
				ToolName: tc.toolName,
				Input:    `{}`,
			})
			if err == nil {
				t.Fatalf("InvokeTool(%q): expected error, got nil", tc.toolName)
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("InvokeTool(%q): error is not a gRPC status: %v", tc.toolName, err)
			}
			if !strings.Contains(st.Message(), tc.toolName) {
				t.Errorf("InvokeTool(%q): error message %q does not contain tool name", tc.toolName, st.Message())
			}
		})
	}
}
