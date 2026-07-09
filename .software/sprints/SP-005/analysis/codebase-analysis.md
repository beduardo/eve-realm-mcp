# Codebase Analysis

**Sprint**: SP-005
**Analyzed**: 2026-07-09
**Entities Mapped**: 4 (REQ-00C, SC-01A, SC-01B, SC-01C)

---

## Entity-to-Code Mapping

| Entity ID | Type | Related Files | Lines | Notes |
|-----------|------|---------------|-------|-------|
| REQ-00C | requirement | `cmd/eve-realm-mcp/main.go` | L105-L113, L185-L191 | Interceptor registered in `startServers`; startup/shutdown `log.*` calls migrated to `slog` |
| REQ-00C | requirement | `gen/proto/mcp/v1/mcp.pb.go` | L171-L224 | `InvokeToolRequest.ToolName` field accessed via `req.GetToolName()` or direct `.ToolName` |
| REQ-00C | requirement | (no existing file) | - | New package `internal/logging` to be created |
| SC-01A | scenario | `internal/mcp/service_test.go` | L25-L57 | `newTestServer` helper pattern; interceptor test will inject `slog.Logger` into a bufconn server |
| SC-01B | scenario | `internal/mcp/service_test.go` | L25-L57 | Same helper; test must inject interceptor + capture log output to verify `tool_name` field |
| SC-01C | scenario | `internal/mcp/service_test.go` | L286-L354 | NotFound error path already tested at service level; interceptor test verifies log at ERROR with status `"NotFound"` |
| SC-01A, SC-01B, SC-01C | scenario | (no existing file) | - | New test file `internal/logging/interceptor_test.go` |

---

## Existing Patterns

### Pattern 1: gRPC UnaryServerInterceptor registration via `grpc.NewServer(opts...)`

- **Reference**: `cmd/eve-realm-mcp/main.go:L112`
- **Description**: Currently `grpc.NewServer()` is called with no options. The interceptor is added by passing `grpc.UnaryInterceptor(...)` or `grpc.ChainUnaryInterceptor(...)` as a `grpc.ServerOption`. The existing gRPC version (v1.81.1 per `go.mod`) fully supports both. No version bump is needed.
- **Entities Using**: REQ-00C

### Pattern 2: In-process bufconn gRPC test server

- **Reference**: `internal/mcp/service_test.go:L25-L57`
- **Description**: `newTestServer(t, reg)` creates a `bufconn.Listen`, starts a `grpc.NewServer()`, registers `MCPService`, and returns a `MCPServiceClient`. The interceptor test will use the same pattern, extended to accept the interceptor as a `grpc.ServerOption` and a `*slog.Logger` backed by a `bytes.Buffer` for log capture.
- **Entities Using**: SC-01A, SC-01B, SC-01C

### Pattern 3: Table-driven test cases with `t.Run`

- **Reference**: `internal/mcp/service_test.go:L94-L140`
- **Description**: `[]struct{ name string; ... }` tables iterated with `for _, tc := range cases { tc := tc; t.Run(tc.name, ...) }`. Interceptor tests follow this structure for the three scenarios.
- **Entities Using**: SC-01A, SC-01B, SC-01C

### Pattern 4: `t.Helper()` on setup functions

- **Reference**: `internal/mcp/service_test.go:L26`, `internal/version/bump_test.go:L17`
- **Description**: All setup/fixture helpers call `t.Helper()` as their first statement. The interceptor test helper that sets up a logged server will follow this convention.
- **Entities Using**: SC-01A, SC-01B, SC-01C

### Pattern 5: `log.Logger` passed as dependency to `startServers` and `ShutdownServer`

- **Reference**: `cmd/eve-realm-mcp/main.go:L86, L105`
- **Description**: Both `ShutdownServer` and `startServers` accept a `*log.Logger`. This parameter is the primary migration target — it will be replaced by `*slog.Logger`. Tests in `main_test.go` construct loggers with `log.New(&logBuf, "", 0)` and verify log output by string-searching `logBuf.String()`. After migration to `slog` with a `JSONHandler`, test assertions must parse JSON or search within JSON fields.
- **Entities Using**: REQ-00C (AC-6)

### Pattern 6: `slog.JSONHandler` with stdout — no existing usage, pure stdlib

- **Reference**: Go stdlib `log/slog` (available since Go 1.21; `go.mod` specifies `go 1.25.0`)
- **Description**: No `slog` is used anywhere in the codebase today. The migration is from `"log"` (stdlib) to `"log/slog"`. `slog.New(slog.NewJSONHandler(os.Stdout, nil))` is the canonical construction.
- **Entities Using**: REQ-00C (AC-6)

### Pattern 7: gRPC method path and request type assertion for `tool_name` extraction

- **Reference**: `gen/proto/mcp/v1/mcp_grpc.pb.go:L22-L23`, `gen/proto/mcp/v1/mcp.pb.go:L172-L224`
- **Description**: The generated constant `MCPService_InvokeTool_FullMethodName = "/mcp.v1.MCPService/InvokeTool"` can be used to detect InvokeTool calls. The `req interface{}` parameter is type-asserted to `*mcpv1.InvokeToolRequest` to extract `.ToolName`. The full import path is `github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1`.
- **Entities Using**: REQ-00C (AC-3), SC-01B, SC-01C

---

## Files to Create

| File | Purpose | Based On | Entities |
|------|---------|----------|----------|
| `internal/logging/interceptor.go` | `UnaryServerInterceptor` function: records start time, invokes handler, computes duration, extracts gRPC status code, conditionally extracts `tool_name` for InvokeTool, logs at INFO/ERROR via `slog` | `internal/mcp/service.go` (package structure), `gen/proto/mcp/v1/mcp_grpc.pb.go` (method constants) | REQ-00C |
| `internal/logging/interceptor_test.go` | Tests for interceptor: log output captured via `bytes.Buffer`-backed `slog.JSONHandler`; bufconn gRPC server with interceptor injected; table-driven cases for SC-01A, SC-01B, SC-01C | `internal/mcp/service_test.go` (bufconn pattern) | SC-01A, SC-01B, SC-01C |

---

## Files to Modify

| File | Modification | Lines | Entities |
|------|--------------|-------|----------|
| `cmd/eve-realm-mcp/main.go` | 1. Replace `"log"` import with `"log/slog"`; 2. Change `startServers` signature from `*log.Logger` to `*slog.Logger`; 3. Pass `grpc.UnaryInterceptor(logging.NewInterceptor(logger))` to `grpc.NewServer()`; 4. Replace `logger.Printf`/`logger.Println` calls with `slog.Info`/`slog.Error`; 5. Change `ShutdownServer` to use `*slog.Logger`; 6. Replace `log.Println(StartupMessage())` and `log.Fatalf` in `main()` with slog calls | L8, L86, L105-L113, L142-L143, L171, L185-L191 | REQ-00C (AC-1, AC-6) |
| `cmd/eve-realm-mcp/main_test.go` | 1. Replace `log.New(&logBuf, "", 0)` with `slog.New(slog.NewJSONHandler(&logBuf, nil))`; 2. Update `TestServer_StartupLog_IncludesGRPCPort` to parse JSON from `logBuf`; 3. Update `TestGracefulShutdown` to check for shutdown message in JSON log output; 4. Update `ShutdownServer` call signature | L8-L18, L413, L467, L525, L566, L599, L703-L735 | REQ-00C (AC-6) |

---

## Integration Points

### Integration 1: gRPC server construction in `startServers`
- **Location**: `cmd/eve-realm-mcp/main.go:L112`
- **Change**: `grpc.NewServer()` becomes `grpc.NewServer(grpc.UnaryInterceptor(logging.NewInterceptor(logger)))`

### Integration 2: `InvokeToolRequest` type assertion inside the interceptor
- **Location**: `internal/logging/interceptor.go` (new file)
- **Mechanism**: `if r, ok := req.(*mcpv1.InvokeToolRequest); ok { toolName = r.ToolName }`

### Integration 3: `slog` logger threaded through `startServers` parameter
- **Location**: `cmd/eve-realm-mcp/main.go:L105` (signature), L190 (call site)
- **Change**: Parameter type `*log.Logger` → `*slog.Logger`

### Integration 4: Test log capture for interceptor tests
- **Location**: `internal/logging/interceptor_test.go` (new file)
- **Mechanism**: `slog.New(slog.NewJSONHandler(&logBuf, nil))` injected into interceptor constructor

---

## Risks and Considerations

1. **`ShutdownServer` signature change breaks existing test assertions**: `TestGracefulShutdown` asserts `strings.Contains(logOutput, "eve-realm-mcp shutting down")` — after migration to JSON, the assertion format changes.

2. **`startServers` and `ShutdownServer` are used in tests**: Every test calling these functions must be updated in the same commit when the logger parameter type changes.

3. **Status code string format**: Scenarios expect `"NotFound"` (CamelCase), not `"NOT_FOUND"`. Must use `codes.Code.String()` which returns CamelCase proto names in grpc-go.

4. **`duration_ms` field type**: Should be `float64`, tests should assert `> 0` not a specific value.

5. **No `go.mod` changes needed**: grpc-go v1.81.1 and Go 1.25.0 have everything required.

6. **Log buffer reset between table-driven test cases**: Shared `logBuf` must be reset between iterations.

7. **`TestServer_StartupLog_IncludesGRPCPort`**: Plain-text `strings.Contains` will still work since the string appears inside JSON `msg` field, but should be updated for consistency.
