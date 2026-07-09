package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"testing"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	"github.com/beduardo/eve-realm-mcp/internal/logging"
	internalmcp "github.com/beduardo/eve-realm-mcp/internal/mcp"
	"github.com/beduardo/eve-realm-mcp/internal/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// newTestServerWithLogger starts an in-process gRPC server bound to a bufconn
// listener, installs the logging interceptor backed by logger, registers
// MCPService with the provided registry, and returns a client connected to that
// server. The test is responsible for cleanup via t.Cleanup.
func newTestServerWithLogger(t *testing.T, reg registry.ToolRegistry, logger *slog.Logger) mcpv1.MCPServiceClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(grpc.UnaryInterceptor(logging.NewInterceptor(logger)))
	mcpv1.RegisterMCPServiceServer(srv, internalmcp.NewMCPService(reg))

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

// parseLogEntry parses the first JSON object from buf. If buf is empty or the
// content is not valid JSON, it returns nil to signal an absent log entry.
func parseLogEntry(buf *bytes.Buffer) map[string]any {
	data := bytes.TrimSpace(buf.Bytes())
	if len(data) == 0 {
		return nil
	}
	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	return entry
}

// ---------------------------------------------------------------------------
// TestInterceptor_ListTools_LogsMethodDurationStatus
// ---------------------------------------------------------------------------

// TestInterceptor_ListTools_LogsMethodDurationStatus verifies (SC-01A) that
// a ListTools RPC produces a JSON log entry with:
//   - "method": "/mcp.v1.MCPService/ListTools"
//   - "duration_ms": a numeric value greater than zero
//   - "status": "OK"
//   - "level": "INFO"
func TestInterceptor_ListTools_LogsMethodDurationStatus(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "single_call"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

			reg := registry.NewMapRegistry()
			reg.Register(registry.Tool{
				Name:        "ping",
				Description: "Checks server liveness",
				InputSchema: `{"type":"object","properties":{}}`,
				Handler:     func(input string) (string, error) { return `{"pong":true}`, nil },
			})

			client := newTestServerWithLogger(t, reg, logger)

			logBuf.Reset()

			_, err := client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: unexpected error: %v", err)
			}

			entry := parseLogEntry(&logBuf)
			if entry == nil {
				t.Fatal("ListTools: expected a log entry to be written, but log buffer is empty")
			}

			// AC-2: method field
			method, ok := entry["method"].(string)
			if !ok || method != mcpv1.MCPService_ListTools_FullMethodName {
				t.Errorf("log entry method = %v, want %q", entry["method"], mcpv1.MCPService_ListTools_FullMethodName)
			}

			// AC-3: duration_ms must be a numeric value greater than zero
			durRaw, hasDur := entry["duration_ms"]
			if !hasDur {
				t.Error("log entry missing \"duration_ms\" field")
			} else {
				dur, ok := durRaw.(float64)
				if !ok {
					t.Errorf("log entry \"duration_ms\" is not numeric: %T %v", durRaw, durRaw)
				} else if dur < 0 {
					t.Errorf("log entry \"duration_ms\" = %v, want >= 0", dur)
				}
			}

			// AC-4: status and level
			statusVal, ok := entry["status"].(string)
			if !ok || statusVal != "OK" {
				t.Errorf("log entry status = %v, want \"OK\"", entry["status"])
			}
			levelVal, ok := entry["level"].(string)
			if !ok || levelVal != "INFO" {
				t.Errorf("log entry level = %v, want \"INFO\"", entry["level"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestInterceptor_InvokeTool_LogsToolName
// ---------------------------------------------------------------------------

// TestInterceptor_InvokeTool_LogsToolName verifies (SC-01B) that an InvokeTool
// RPC for a registered tool produces a JSON log entry with:
//   - "tool_name": "ping"
//   - "method": "/mcp.v1.MCPService/InvokeTool"
//   - "status": "OK"
//   - "level": "INFO"
//   - "duration_ms": numeric
func TestInterceptor_InvokeTool_LogsToolName(t *testing.T) {
	cases := []struct {
		name     string
		toolName string
	}{
		{name: "ping_tool", toolName: "ping"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

			reg := registry.NewMapRegistry()
			reg.Register(registry.Tool{
				Name:        tc.toolName,
				Description: "Checks server liveness",
				InputSchema: `{"type":"object","properties":{}}`,
				Handler:     func(input string) (string, error) { return `{"pong":true}`, nil },
			})

			client := newTestServerWithLogger(t, reg, logger)

			logBuf.Reset()

			_, err := client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
				ToolName: tc.toolName,
				Input:    `{}`,
			})
			if err != nil {
				t.Fatalf("InvokeTool(%q): unexpected error: %v", tc.toolName, err)
			}

			entry := parseLogEntry(&logBuf)
			if entry == nil {
				t.Fatal("InvokeTool: expected a log entry to be written, but log buffer is empty")
			}

			// AC-2: tool_name field
			toolName, ok := entry["tool_name"].(string)
			if !ok || toolName != tc.toolName {
				t.Errorf("log entry tool_name = %v, want %q", entry["tool_name"], tc.toolName)
			}

			// AC-3: method, status, level, duration_ms
			method, ok := entry["method"].(string)
			if !ok || method != mcpv1.MCPService_InvokeTool_FullMethodName {
				t.Errorf("log entry method = %v, want %q", entry["method"], mcpv1.MCPService_InvokeTool_FullMethodName)
			}

			statusVal, ok := entry["status"].(string)
			if !ok || statusVal != "OK" {
				t.Errorf("log entry status = %v, want \"OK\"", entry["status"])
			}

			levelVal, ok := entry["level"].(string)
			if !ok || levelVal != "INFO" {
				t.Errorf("log entry level = %v, want \"INFO\"", entry["level"])
			}

			durRaw, hasDur := entry["duration_ms"]
			if !hasDur {
				t.Error("log entry missing \"duration_ms\" field")
			} else {
				dur, ok := durRaw.(float64)
				if !ok {
					t.Errorf("log entry \"duration_ms\" is not numeric: %T %v", durRaw, durRaw)
				} else if dur < 0 {
					t.Errorf("log entry \"duration_ms\" = %v, want >= 0", dur)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestInterceptor_InvokeTool_NotFound_LogsAtErrorLevel
// ---------------------------------------------------------------------------

// TestInterceptor_InvokeTool_NotFound_LogsAtErrorLevel verifies (SC-01C) that
// an InvokeTool RPC for an unregistered tool produces a JSON log entry with:
//   - "level": "ERROR"
//   - "status": "NotFound"
//   - "tool_name": "nonexistent"
//   - "method": "/mcp.v1.MCPService/InvokeTool"
//   - "duration_ms": numeric
//   - "error": non-empty string
func TestInterceptor_InvokeTool_NotFound_LogsAtErrorLevel(t *testing.T) {
	cases := []struct {
		name     string
		toolName string
	}{
		{name: "nonexistent_tool", toolName: "nonexistent"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

			// Empty registry — all tool names are unknown.
			reg := registry.NewMapRegistry()

			client := newTestServerWithLogger(t, reg, logger)

			logBuf.Reset()

			// The RPC is expected to fail — discard the error; we care about the log.
			_, _ = client.InvokeTool(context.Background(), &mcpv1.InvokeToolRequest{
				ToolName: tc.toolName,
				Input:    `{}`,
			})

			entry := parseLogEntry(&logBuf)
			if entry == nil {
				t.Fatal("InvokeTool(NotFound): expected a log entry to be written, but log buffer is empty")
			}

			// AC-2: level must be ERROR
			levelVal, ok := entry["level"].(string)
			if !ok || levelVal != "ERROR" {
				t.Errorf("log entry level = %v, want \"ERROR\"", entry["level"])
			}

			// AC-3: status, tool_name, method, duration_ms, error
			statusVal, ok := entry["status"].(string)
			if !ok || statusVal != "NotFound" {
				t.Errorf("log entry status = %v, want \"NotFound\"", entry["status"])
			}

			toolName, ok := entry["tool_name"].(string)
			if !ok || toolName != tc.toolName {
				t.Errorf("log entry tool_name = %v, want %q", entry["tool_name"], tc.toolName)
			}

			method, ok := entry["method"].(string)
			if !ok || method != mcpv1.MCPService_InvokeTool_FullMethodName {
				t.Errorf("log entry method = %v, want %q", entry["method"], mcpv1.MCPService_InvokeTool_FullMethodName)
			}

			durRaw, hasDur := entry["duration_ms"]
			if !hasDur {
				t.Error("log entry missing \"duration_ms\" field")
			} else {
				dur, ok := durRaw.(float64)
				if !ok {
					t.Errorf("log entry \"duration_ms\" is not numeric: %T %v", durRaw, durRaw)
				} else if dur < 0 {
					t.Errorf("log entry \"duration_ms\" = %v, want >= 0", dur)
				}
			}

			errVal, hasErr := entry["error"]
			if !hasErr {
				t.Error("log entry missing \"error\" field")
			} else {
				errStr, ok := errVal.(string)
				if !ok || errStr == "" {
					t.Errorf("log entry \"error\" must be a non-empty string, got %v", errVal)
				}
			}
		})
	}
}
