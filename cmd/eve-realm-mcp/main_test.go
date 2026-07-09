package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestDefaultVariableValues verifies that the package-level variables carry the
// expected default values when the binary is built without ldflags injection.
func TestDefaultVariableValues(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "Version defaults to dev", got: Version, expected: "dev"},
		{name: "GitHash defaults to unknown", got: GitHash, expected: "unknown"},
		{name: "BuildDate defaults to unknown", got: BuildDate, expected: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

// TestStartupMessage verifies the startup log format produced by StartupMessage.
func TestStartupMessage(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		gitHash         string
		buildDate       string
		expectedMessage string
	}{
		{
			name:            "default values produce correct format",
			version:         "dev",
			gitHash:         "unknown",
			buildDate:       "unknown",
			expectedMessage: "eve-realm-mcp online (vdev, unknown, unknown)",
		},
		{
			name:            "injected version values produce correct format",
			version:         "1.2.3",
			gitHash:         "abc1234",
			buildDate:       "2026-06-22",
			expectedMessage: "eve-realm-mcp online (v1.2.3, abc1234, 2026-06-22)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Temporarily set package-level vars and restore after.
			origVersion, origGitHash, origBuildDate := Version, GitHash, BuildDate
			Version, GitHash, BuildDate = tc.version, tc.gitHash, tc.buildDate
			defer func() {
				Version, GitHash, BuildDate = origVersion, origGitHash, origBuildDate
			}()

			got := StartupMessage()
			if got != tc.expectedMessage {
				t.Errorf("StartupMessage() = %q, want %q", got, tc.expectedMessage)
			}
		})
	}
}

// TestStartupMessageContainsBinaryName verifies the startup message always begins
// with the binary name prefix.
func TestStartupMessageContainsBinaryName(t *testing.T) {
	msg := StartupMessage()
	if !strings.HasPrefix(msg, "eve-realm-mcp online") {
		t.Errorf("startup message %q does not start with %q", msg, "eve-realm-mcp online")
	}
}

// TestStartupMessageFormat verifies the parenthesised version triplet format.
func TestStartupMessageFormat(t *testing.T) {
	origVersion, origGitHash, origBuildDate := Version, GitHash, BuildDate
	Version, GitHash, BuildDate = "0.1.0", "deadbeef", "2026-01-01"
	defer func() {
		Version, GitHash, BuildDate = origVersion, origGitHash, origBuildDate
	}()

	msg := StartupMessage()
	expected := fmt.Sprintf("eve-realm-mcp online (v%s, %s, %s)", Version, GitHash, BuildDate)
	if msg != expected {
		t.Errorf("StartupMessage() = %q, want %q", msg, expected)
	}
}

// TestVersionHandlerJSONSchema verifies that GET /version returns HTTP 200 with
// the correct JSON schema: {"version":"...","git_hash":"...","build_date":"..."}.
func TestVersionHandlerJSONSchema(t *testing.T) {
	server := httptest.NewServer(VersionHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /version failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	requiredKeys := []string{"version", "git_hash", "build_date"}
	for _, key := range requiredKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("JSON response missing required key %q", key)
		}
	}
}

// TestVersionHandlerDefaultValues verifies that the /version handler reflects
// the package-level variable defaults when built without ldflags.
func TestVersionHandlerDefaultValues(t *testing.T) {
	// Ensure we are testing against the known default state.
	origVersion, origGitHash, origBuildDate := Version, GitHash, BuildDate
	Version, GitHash, BuildDate = "dev", "unknown", "unknown"
	defer func() {
		Version, GitHash, BuildDate = origVersion, origGitHash, origBuildDate
	}()

	server := httptest.NewServer(VersionHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /version failed: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Version   string `json:"version"`
		GitHash   string `json:"git_hash"`
		BuildDate string `json:"build_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "version field is dev", got: body.Version, expected: "dev"},
		{name: "git_hash field is unknown", got: body.GitHash, expected: "unknown"},
		{name: "build_date field is unknown", got: body.BuildDate, expected: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

// TestVersionHandlerInjectedValues verifies that the /version handler reflects
// variable values as they would appear when injected via ldflags.
func TestVersionHandlerInjectedValues(t *testing.T) {
	origVersion, origGitHash, origBuildDate := Version, GitHash, BuildDate
	Version, GitHash, BuildDate = "1.2.3", "abc1234", "2026-06-22"
	defer func() {
		Version, GitHash, BuildDate = origVersion, origGitHash, origBuildDate
	}()

	server := httptest.NewServer(VersionHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /version failed: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Version   string `json:"version"`
		GitHash   string `json:"git_hash"`
		BuildDate string `json:"build_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body.Version != "1.2.3" {
		t.Errorf("version field: expected %q, got %q", "1.2.3", body.Version)
	}
	if body.GitHash != "abc1234" {
		t.Errorf("git_hash field: expected %q, got %q", "abc1234", body.GitHash)
	}
	if body.BuildDate != "2026-06-22" {
		t.Errorf("build_date field: expected %q, got %q", "2026-06-22", body.BuildDate)
	}
}

// TestVersionHandlerNoExtraKeys verifies that the /version JSON response contains
// exactly the three required keys and no additional ones.
func TestVersionHandlerNoExtraKeys(t *testing.T) {
	server := httptest.NewServer(VersionHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /version failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	allowedKeys := map[string]bool{
		"version":    true,
		"git_hash":   true,
		"build_date": true,
	}

	for key := range body {
		if !allowedKeys[key] {
			t.Errorf("unexpected key %q in /version JSON response", key)
		}
	}

	if len(body) != 3 {
		t.Errorf("expected exactly 3 keys in /version response, got %d", len(body))
	}
}

// healthzResponse is the expected JSON schema for the /healthz endpoint.
type healthzResponse struct {
	Status string `json:"status"`
}

// TestHealthzHandler verifies that GET /healthz returns HTTP 200 with
// Content-Type application/json and a body of {"status":"ok"}.
func TestHealthzHandler(t *testing.T) {
	server := httptest.NewServer(HealthzHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var body healthzResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("expected status field %q, got %q", "ok", body.Status)
	}
}

// TestHealthzHandlerNoExtraKeys verifies that the /healthz JSON response contains
// exactly the one required key and no additional ones.
func TestHealthzHandlerNoExtraKeys(t *testing.T) {
	server := httptest.NewServer(HealthzHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if len(body) != 1 {
		t.Errorf("expected exactly 1 key in /healthz response, got %d", len(body))
	}

	if _, ok := body["status"]; !ok {
		t.Errorf("JSON response missing required key %q", "status")
	}
}

// readyzResponse is the expected JSON schema for the /readyz endpoint.
type readyzResponse struct {
	Status string `json:"status"`
}

// TestReadyzHandler verifies that GET /readyz returns HTTP 200 with
// Content-Type application/json and a body of {"status":"ok"}.
func TestReadyzHandler(t *testing.T) {
	server := httptest.NewServer(ReadyzHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /readyz failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var body readyzResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("expected status field %q, got %q", "ok", body.Status)
	}
}

// TestReadyzHandlerNoExtraKeys verifies that the /readyz JSON response contains
// exactly the one required key and no additional ones.
func TestReadyzHandlerNoExtraKeys(t *testing.T) {
	server := httptest.NewServer(ReadyzHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /readyz failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if len(body) != 1 {
		t.Errorf("expected exactly 1 key in /readyz response, got %d", len(body))
	}

	if _, ok := body["status"]; !ok {
		t.Errorf("JSON response missing required key %q", "status")
	}
}

// ---------------------------------------------------------------------------
// Dual-server startup tests
// ---------------------------------------------------------------------------

// serverConfig holds the parameters for startServers, mirroring the flags.
type testServerConfig struct {
	httpPort int
	grpcPort int
}

// ---------------------------------------------------------------------------
// TestServer_DefaultPorts_BothServersListen
// ---------------------------------------------------------------------------

// TestServer_DefaultPorts_BothServersListen verifies that when startServers is
// called with default-equivalent ports, both an HTTP server and a gRPC server
// are bound and accepting connections. Ephemeral ports are used to avoid
// conflicts with any running services.
func TestServer_DefaultPorts_BothServersListen(t *testing.T) {
	// Obtain two free ephemeral ports.
	httpPort := freePort(t)
	grpcPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	ready := make(chan struct{})
	go func() {
		if err := startServers(ctx, httpPort, grpcPort, logger, ready); err != nil {
			// Errors after context cancellation are expected.
			_ = err
		}
	}()

	// Wait for the servers to signal readiness (with timeout).
	waitReady(t, ready, 5*time.Second)

	// Verify HTTP server is accepting connections.
	httpAddr := fmt.Sprintf("http://127.0.0.1:%d/healthz", httpPort)
	resp, err := http.Get(httpAddr)
	if err != nil {
		t.Fatalf("HTTP server not accepting connections on port %d: %v", httpPort, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("HTTP /healthz: expected 200, got %d", resp.StatusCode)
	}

	// Verify gRPC server is accepting connections.
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("gRPC dial failed on port %d: %v", grpcPort, err)
	}
	defer conn.Close()

	client := mcpv1.NewMCPServiceClient(conn)
	_, err = client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools RPC failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestServer_GRPCPortFlag_Override / TestServer_GRPCPortFlag_OverridesDefault
// ---------------------------------------------------------------------------

// TestServer_GRPCPortFlag_OverridesDefault verifies that passing a non-default
// gRPC port causes the gRPC server to bind on that port while the HTTP server
// remains on its own port.
func TestServer_GRPCPortFlag_OverridesDefault(t *testing.T) {
	httpPort := freePort(t)
	grpcPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	ready := make(chan struct{})
	go func() {
		if err := startServers(ctx, httpPort, grpcPort, logger, ready); err != nil {
			_ = err
		}
	}()

	waitReady(t, ready, 5*time.Second)

	// Confirm gRPC is reachable on the overridden port.
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("gRPC dial failed on overridden port %d: %v", grpcPort, err)
	}
	defer conn.Close()

	client := mcpv1.NewMCPServiceClient(conn)
	_, err = client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools on overridden port %d: %v", grpcPort, err)
	}

	// Confirm HTTP is reachable on its port (unchanged).
	httpAddr := fmt.Sprintf("http://127.0.0.1:%d/healthz", httpPort)
	resp, err := http.Get(httpAddr)
	if err != nil {
		t.Fatalf("HTTP server not reachable on port %d: %v", httpPort, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("HTTP /healthz: expected 200, got %d", resp.StatusCode)
	}
}

// TestServer_GRPCPortFlag_Override is an alias table-driven form of the above
// that iterates over a set of port combinations, confirming each gRPC port
// binding is independent of the HTTP port.
func TestServer_GRPCPortFlag_Override(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "custom_grpc_port_1"},
		{name: "custom_grpc_port_2"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			httpPort := freePort(t)
			grpcPort := freePort(t)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

			ready := make(chan struct{})
			go func() {
				if err := startServers(ctx, httpPort, grpcPort, logger, ready); err != nil {
					_ = err
				}
			}()

			waitReady(t, ready, 5*time.Second)

			grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
			conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("[%s] gRPC dial failed on port %d: %v", tc.name, grpcPort, err)
			}
			defer conn.Close()

			client := mcpv1.NewMCPServiceClient(conn)
			_, err = client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
			if err != nil {
				t.Fatalf("[%s] ListTools on port %d: %v", tc.name, grpcPort, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestServer_StartupLog_IncludesGRPCPort
// ---------------------------------------------------------------------------

// TestServer_StartupLog_IncludesGRPCPort verifies that the startup log output
// contains the gRPC listen address (e.g. "grpc listening on :50051").
func TestServer_StartupLog_IncludesGRPCPort(t *testing.T) {
	httpPort := freePort(t)
	grpcPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	ready := make(chan struct{})
	go func() {
		if err := startServers(ctx, httpPort, grpcPort, logger, ready); err != nil {
			_ = err
		}
	}()

	waitReady(t, ready, 5*time.Second)

	logOutput := logBuf.String()
	wantAddr := fmt.Sprintf(":%d", grpcPort)
	found := false
	for _, line := range strings.Split(strings.TrimSpace(logOutput), "\n") {
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		if msg, _ := entry["msg"].(string); msg == "grpc listening" {
			if addr, _ := entry["addr"].(string); addr == wantAddr {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("startup log does not contain grpc listening entry with addr %q; got:\n%s", wantAddr, logOutput)
	}
}

// ---------------------------------------------------------------------------
// TestStartup_PingToolRegistered_InListTools
// ---------------------------------------------------------------------------

// TestStartup_PingToolRegistered_InListTools verifies that after startServers
// initialises the ToolRegistry, a ListTools RPC response contains a descriptor
// for the "ping" tool.
func TestStartup_PingToolRegistered_InListTools(t *testing.T) {
	httpPort := freePort(t)
	grpcPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	ready := make(chan struct{})
	go func() {
		if err := startServers(ctx, httpPort, grpcPort, logger, ready); err != nil {
			_ = err
		}
	}()

	waitReady(t, ready, 5*time.Second)

	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("gRPC dial failed: %v", err)
	}
	defer conn.Close()

	client := mcpv1.NewMCPServiceClient(conn)
	resp, err := client.ListTools(context.Background(), &mcpv1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	var found bool
	for _, d := range resp.Tools {
		if d.Name == "ping" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, 0, len(resp.Tools))
		for _, d := range resp.Tools {
			names = append(names, d.Name)
		}
		t.Errorf("ListTools response does not contain ping tool; got: %v", names)
	}
}

// ---------------------------------------------------------------------------
// TestPingTool_Descriptor_MatchesSpec
// ---------------------------------------------------------------------------

// TestPingTool_Descriptor_MatchesSpec verifies the ping tool's name, description,
// and input_schema match the values specified in SC-018 and REQ-00B.
func TestPingTool_Descriptor_MatchesSpec(t *testing.T) {
	tool := pingTool()

	if tool.Name != "ping" {
		t.Errorf("Name = %q, want %q", tool.Name, "ping")
	}

	wantDesc := "Diagnostic tool that returns pong with server timestamp"
	if tool.Description != wantDesc {
		t.Errorf("Description = %q, want %q", tool.Description, wantDesc)
	}

	if tool.InputSchema != "{}" {
		t.Errorf("InputSchema = %q, want %q", tool.InputSchema, "{}")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// freePort returns an available TCP port by opening a listener on :0 and
// immediately closing it. There is a brief TOCTOU window but it is acceptable
// in tests where ports are allocated just before use.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: net.Listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// waitReady blocks until the ready channel is closed or the deadline is
// exceeded, in which case the test is fatally failed.
func waitReady(t *testing.T, ready <-chan struct{}, timeout time.Duration) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for servers to become ready after %s", timeout)
	}
}

// ---------------------------------------------------------------------------
// Original graceful shutdown test
// ---------------------------------------------------------------------------

// TestGracefulShutdown verifies that calling Shutdown on a running http.Server:
//  1. causes http.Server.ListenAndServe to return http.ErrServerClosed
//  2. emits the canonical shutdown log line "eve-realm-mcp shutting down"
//  3. completes without panic or goroutine leak
//
// The test does NOT use syscall.SIGINT or syscall.SIGTERM — it drives shutdown
// directly via context cancellation passed to ShutdownServer.
func TestGracefulShutdown(t *testing.T) {
	// Capture log output to verify the shutdown message.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	mux := http.NewServeMux()
	mux.Handle("/healthz", HealthzHandler())

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: mux,
	}

	// listenErr receives the error returned by ListenAndServe.
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- srv.ListenAndServe()
	}()

	// Use a context that is cancelled immediately to trigger shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // signal shutdown now

	// ShutdownServer logs the shutdown message and calls srv.Shutdown(ctx).
	ShutdownServer(ctx, srv, logger)

	err := <-listenErr
	if err != http.ErrServerClosed {
		t.Errorf("expected http.ErrServerClosed, got %v", err)
	}

	logOutput := logBuf.String()
	found := false
	for _, line := range strings.Split(strings.TrimSpace(logOutput), "\n") {
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		if msg, _ := entry["msg"].(string); msg == "eve-realm-mcp shutting down" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected JSON log entry with msg %q, got:\n%s", "eve-realm-mcp shutting down", logOutput)
	}
}
