# Codebase Analysis Brief

**Sprint**: SP-005
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Entity IDs**: REQ-00C, SC-01A, SC-01B, SC-01C

## Entity Details

### REQ-00C: gRPC request logging interceptor
- Type: requirement
- Status: active
- Tags: grpc, logging, observability
- A gRPC unary server interceptor that logs every incoming RPC call with structured JSON output via `slog`. Captures method, duration, status code, and for InvokeTool calls the tool_name. Successful RPCs log at INFO, failed at ERROR. Replaces existing `log.Println` with `slog` JSONHandler.

### SC-01A: ListTools RPC logs method duration and status
- Type: scenario
- Status: validated
- Tags: grpc, logging
- Verifies that a ListTools call produces a JSON log entry with method, duration_ms, status "OK", and level "INFO".

### SC-01B: InvokeTool RPC log includes tool name
- Type: scenario
- Status: validated
- Tags: grpc, logging, invoke-tool
- Verifies that an InvokeTool call for "ping" produces a JSON log entry with method, tool_name "ping", duration_ms, status "OK", and level "INFO".

### SC-01C: Failed RPC logs at error level with status code
- Type: scenario
- Status: validated
- Tags: grpc, logging, error-handling
- Verifies that invoking a nonexistent tool produces a JSON log entry at ERROR level with status "NotFound", tool_name, and error message.

## Focus Areas
- Current gRPC server setup in `startServers` — how the `grpc.Server` is constructed and where interceptors are registered
- Existing logging patterns — current `log.Println` usage that needs migration to `slog`
- InvokeTool request structure — how to extract `tool_name` from the request proto
- Test patterns from SP-004 (gRPC tool registry) — how gRPC tests are structured
- Package layout — whether `internal/logging` or co-located interceptor is the better fit
