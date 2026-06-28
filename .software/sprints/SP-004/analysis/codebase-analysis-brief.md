# Codebase Analysis Brief

**Sprint**: SP-004
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Entity IDs**: REQ-00A, REQ-00B, SC-013, SC-014, SC-015, SC-016, SC-017, SC-018, SC-019

## Entity Details

### REQ-00A: gRPC tool registry service
- Type: requirement
- Status: active
- Tags: grpc, tool-registry, mcp-server
- Exposes a gRPC service on port 50051 with ListTools and InvokeTool RPCs. Internal ToolRegistry interface with Register, List, Invoke. Proto at `proto/mcp/v1/mcp.proto`, generated code at `gen/proto/mcp/v1/`. Concurrent-safe registry. `make proto` for codegen.
- **AC-6 (NEW)**: The K8s Service (`deploy/k8s/service.yaml`) uses `type: NodePort` with the gRPC port mapped to NodePort 30051, making the service accessible from the host at `localhost:30051`.

### Acceptance Criteria (full list, 9 ACs):
1. Proto definition at `proto/mcp/v1/mcp.proto` with MCPService, ListTools, InvokeTool
2. ListTools returns repeated tool descriptors (name, description, input_schema)
3. InvokeTool accepts tool_name + JSON input, returns JSON output
4. InvokeTool with unknown tool returns gRPC NOT_FOUND
5. gRPC server on port 50051, configurable via `--grpc-port`, alongside HTTP on 8080
6. K8s Service uses `type: NodePort` with gRPC mapped to NodePort 30051 (`localhost:30051`)
7. Internal ToolRegistry interface: Register, List, Invoke
8. Registry safe for concurrent access
9. `make proto` generates Go code to `gen/proto/mcp/v1/`

### REQ-00B: Ping built-in diagnostic tool
- Type: requirement
- Status: active
- Tags: tool, built-in, diagnostic, ping
- First built-in tool in the registry. Returns `{"message":"pong","timestamp":"<RFC3339>"}`. Handler at `internal/tools/ping.go`. Registered at startup.

### SC-013: gRPC server starts alongside HTTP server
- Type: scenario
- Status: validated
- Tags: grpc, server
- Verifies both HTTP (8080) and gRPC (50051) servers start, and `--grpc-port` flag overrides gRPC port.

### SC-014: ListTools returns registered built-in tools
- Type: scenario
- Status: validated
- Tags: grpc, list-tools
- Verifies ListTools RPC returns tool descriptors with name, description, input_schema.

### SC-015: InvokeTool dispatches to handler and returns result
- Type: scenario
- Status: validated
- Tags: grpc, invoke-tool
- Verifies InvokeTool RPC dispatches to correct handler and returns JSON output.

### SC-016: InvokeTool with unknown tool returns NOT_FOUND
- Type: scenario
- Status: validated
- Tags: grpc, invoke-tool, error
- Verifies NOT_FOUND gRPC status for unregistered tool names.

### SC-017: Tool registry supports concurrent access
- Type: scenario
- Status: validated
- Tags: registry, concurrency
- Verifies no data races with concurrent Register/List/Invoke using `go test -race`.

### SC-018: Ping tool registered at startup
- Type: scenario
- Status: validated
- Tags: ping, startup
- Verifies ping tool appears in ListTools with correct name, description, schema.

### SC-019: Ping invocation returns pong with RFC 3339 timestamp
- Type: scenario
- Status: validated
- Tags: ping, invoke
- Verifies InvokeTool("ping") returns pong message with valid RFC 3339 timestamp.

## Focus Areas
- Existing HTTP server startup code (cmd/ entry point) — how to add gRPC server alongside it
- Existing Makefile targets — where to add `make proto`
- Existing project structure — where `internal/tools/`, `proto/`, `gen/` fit
- Go module dependencies needed (grpc, protobuf)
- Existing health check / graceful shutdown patterns to extend for gRPC
- **K8s service manifest** (`deploy/k8s/service.yaml`) — current service type and port configuration; what changes are needed for NodePort 30051
- **K8s verify checks** (`deploy/k8s/verify/checks.go`) — whether a gRPC check function exists or needs creation for NodePort 30051
