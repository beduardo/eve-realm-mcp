# Sprint SP-004: gRPC tool registry service with ping diagnostic tool

**Created**: 2026-06-28
**Status**: Specified
**Entities**: 9

---

## Overview

This sprint introduces the core gRPC service layer of the MCP Server: a concurrent-safe
tool registry exposed via `MCPService` on port 50051, with `ListTools` and `InvokeTool`
RPCs backed by an internal `ToolRegistry` interface. The registry is validated end-to-end
by the `ping` built-in diagnostic tool, which proves the full dispatch pipeline (gRPC
request → registry → handler → JSON response) works correctly from day one. Together,
these two requirements lay the foundation for all future tool interactions — whether
from built-in handlers or eventually plugin-sourced tools discovered via NATS.

## Entity Inventory

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-00A | requirement | gRPC tool registry service | no | - |
| REQ-00B | requirement | Ping built-in diagnostic tool | no | - |
| SC-013 | scenario | gRPC server starts alongside HTTP server | no | - |
| SC-014 | scenario | ListTools returns registered built-in tools | no | - |
| SC-015 | scenario | InvokeTool dispatches to handler and returns result | no | - |
| SC-016 | scenario | InvokeTool with unknown tool returns NOT_FOUND | no | - |
| SC-017 | scenario | Tool registry supports concurrent access | no | - |
| SC-018 | scenario | Ping tool registered at startup | no | - |
| SC-019 | scenario | Ping invocation returns pong with RFC 3339 timestamp | no | - |

## Technical Context

The codebase analysis identifies the following integration points and implementation
areas for this sprint:

**New directories and files to create:**
- `proto/mcp/v1/mcp.proto` — local protobuf definition for `MCPService`
- `gen/proto/mcp/v1/` — generated Go code output directory (via `make proto`)
- `internal/registry/` — `ToolRegistry` interface and concurrent-safe implementation
- `internal/tools/ping.go` — ping tool handler

**Files to modify:**
- `cmd/` entry point — add gRPC server startup alongside existing HTTP server on 8080; add `--grpc-port` flag (default 50051); extend graceful shutdown to cover gRPC server
- `Makefile` — add `make proto` target invoking `protoc` with Go/gRPC plugins, output to `gen/proto/mcp/v1/`
- `go.mod` — add `google.golang.org/grpc` and `google.golang.org/protobuf` dependencies
- `deploy/k8s/service.yaml` — already updated to `type: NodePort` with gRPC port 50051 mapped to NodePort 30051 (no further changes needed)

**K8s service manifest status:** `deploy/k8s/service.yaml` already contains `type: NodePort` with the gRPC entry (`port: 50051`, `targetPort: 50051`, `nodePort: 30051`). AC-6 of REQ-00A is satisfied by the current manifest state — no manifest changes are required.

**Implementation patterns to follow:**
- gRPC server runs as a separate goroutine alongside HTTP; both shut down via a shared context or `errgroup`
- `ToolRegistry` uses `sync.RWMutex` for concurrent-safe `Register`/`List`/`Invoke` operations
- `MCPService` gRPC handler delegates entirely to the `ToolRegistry` interface — no logic in the handler
- Cluster integration: a gRPC connectivity check function must be added to `deploy/k8s/verify/checks.go` verifying reachability at `localhost:30051` (REQ-003 cluster verification policy triggered by the new NodePort gRPC service)

## Implementation Sections

### REQ-00A: gRPC tool registry service

**Entity**: `.software/entities/requirements/REQ-00A.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
- **Create** `proto/mcp/v1/mcp.proto` — `MCPService` service with `ListTools` (empty request → `ListToolsResponse`) and `InvokeTool` (`InvokeToolRequest` → `InvokeToolResponse`) RPCs
- **Create** `gen/proto/mcp/v1/` — protoc output directory for generated Go stubs
- **Create** `internal/registry/registry.go` — `ToolRegistry` interface (`Register`, `List`, `Invoke`) and `sync.RWMutex`-based implementation
- **Create** `internal/registry/registry_test.go` — unit tests for registry operations and race detection
- **Modify** `cmd/` entry point — add `--grpc-port` flag, start gRPC server via `grpc.NewServer()`, bind `MCPService` handler, integrate into graceful shutdown
- **Modify** `Makefile` — add `proto` target: `protoc --go_out=gen/proto/mcp/v1 --go-grpc_out=gen/proto/mcp/v1 proto/mcp/v1/mcp.proto`
- **Modify** `go.mod` / `go.sum` — add `google.golang.org/grpc` and `google.golang.org/protobuf`
- **Modify** `deploy/k8s/verify/checks.go` — add gRPC NodePort check function verifying `localhost:30051` is reachable (per REQ-003 cluster integration testing policy)

**Acceptance Criteria**:
- **AC-1**: Given the implementation is built, when `proto/mcp/v1/mcp.proto` is inspected, then it defines `MCPService` with `ListTools` and `InvokeTool` RPCs.
- **AC-2**: Given a gRPC client calls `ListTools`, when tools are registered in the registry, then the response contains repeated tool descriptors each with `name`, `description`, and `input_schema` fields.
- **AC-3**: Given a gRPC client calls `InvokeTool`, when it provides a valid `tool_name` and JSON-encoded `input`, then the response `output` field contains a valid JSON-encoded string produced by the handler.
- **AC-4**: Given a gRPC client calls `InvokeTool` with an unregistered `tool_name`, when the registry lookup fails, then the gRPC response returns status `NOT_FOUND` with a descriptive message containing the tool name.
- **AC-5**: Given the MCP Server starts without the `--grpc-port` flag, when both servers initialize, then the gRPC server listens on port 50051 and the HTTP server listens on port 8080; given `--grpc-port 9090` is passed, then the gRPC server listens on 9090 while HTTP remains on 8080.
- **AC-6**: Given `deploy/k8s/service.yaml` is applied to the cluster, when the service is inspected, then `type: NodePort` is set and the gRPC port (50051) is mapped to NodePort 30051, making the service reachable at `localhost:30051` from the host.
- **AC-7**: Given the internal `ToolRegistry` interface, when implemented, then it exposes `Register(tool)`, `List() []Tool`, and `Invoke(name, input) (output, error)` operations.
- **AC-8**: Given concurrent goroutines call `Register`, `List`, and `Invoke` simultaneously, when the test runs with `go test -race`, then no data races are detected.
- **AC-9**: Given `make proto` is executed, when protoc runs, then Go code is generated into `gen/proto/mcp/v1/`.

**Implementation Notes**:
The feasibility analysis identifies the following key considerations for REQ-00A:

- **gRPC server startup**: The existing `cmd/` entry point must be extended to launch a second server goroutine. An `errgroup.Group` (from `golang.org/x/sync/errgroup`) is the idiomatic pattern for running multiple servers under a shared cancellation context — verify whether this dependency is already present before adding it.
- **Flag parsing**: Confirm whether the existing binary uses `flag` stdlib or a third-party library (e.g., `cobra`). The `--grpc-port` flag must be added consistently with the existing pattern.
- **gRPC dependencies**: `go.mod` likely does not yet include `google.golang.org/grpc` or `google.golang.org/protobuf` — these must be added before code generation can proceed.
- **Makefile proto target**: Requires `protoc` and the `protoc-gen-go` / `protoc-gen-go-grpc` plugins installed in the build environment; document these as prerequisites in the Makefile comment or a dev-setup doc.
- **K8s service**: `deploy/k8s/service.yaml` already satisfies AC-6 — no manifest changes needed, but a gRPC connectivity check function must be added to `deploy/k8s/verify/checks.go`.
- **Complexity**: High — involves protobuf schema design, code generation wiring, concurrent registry implementation, dual-server startup, and cluster verification.

**Test Expectations**:
- Must test: `TestRegistry_Register_AddsToolToList` — after `Register`, `List` returns the registered tool with correct name, description, and input_schema
- Must test: `TestRegistry_Invoke_DispatchesToHandler` — given a registered tool with a known handler, `Invoke` returns the handler's output without error
- Must test: `TestRegistry_Invoke_UnknownTool_ReturnsNotFound` — given no tool named `"nonexistent"` is registered, `Invoke` returns an error mappable to gRPC `NOT_FOUND`
- Must test: `TestRegistry_ConcurrentAccess_NoRace` — concurrent `Register`, `List`, and `Invoke` calls pass `go test -race` with no data races detected (table-driven with `sync.WaitGroup` goroutines)
- Must test: `TestMCPService_ListTools_ReturnsDescriptors` — gRPC handler delegates to registry and returns well-formed `ListToolsResponse`
- Must test: `TestMCPService_InvokeTool_UnknownTool_ReturnsGRPCNotFound` — gRPC handler translates registry not-found error to gRPC `codes.NotFound` status
- Must test: `TestServer_GRPCPortFlag_Override` — process substitution test verifying gRPC server binds to the port specified by `--grpc-port`
- Must NOT rely on: global registry state shared between test cases; use a fresh `ToolRegistry` instance per test case

---

### REQ-00B: Ping built-in diagnostic tool

**Entity**: `.software/entities/requirements/REQ-00B.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
- **Create** `internal/tools/ping.go` — ping tool handler implementing the handler signature defined by `ToolRegistry`; returns `{"message":"pong","timestamp":"<RFC3339>"}` using `time.Now().Format(time.RFC3339)`
- **Create** `internal/tools/ping_test.go` — unit tests for ping handler output correctness and timestamp validity
- **Modify** `cmd/` entry point — register the ping tool at server startup by calling `registry.Register(ping.NewTool())`

**Acceptance Criteria**:
- **AC-1**: Given the MCP Server starts, when the registry is initialized, then a tool named `ping` is registered before the gRPC server accepts connections.
- **AC-2**: Given the tool is registered, when its descriptor is inspected, then the `description` field equals `"Diagnostic tool that returns pong with server timestamp"`.
- **AC-3**: Given the tool is registered, when its input schema is inspected, then it declares no required parameters (empty JSON object schema: `{}`).
- **AC-4**: Given a gRPC client calls `InvokeTool` with `tool_name: "ping"`, when the handler executes, then the JSON response is `{"message": "pong", "timestamp": "<RFC 3339 timestamp>"}`.
- **AC-5**: Given `InvokeTool("ping")` is called, when the response is parsed, then the `timestamp` field contains the server's current time at moment of invocation, formatted as RFC 3339 (e.g., `2026-06-27T14:30:00Z`).
- **AC-6**: Given the handler is implemented, when the file is inspected, then the handler code resides in `internal/tools/ping.go`.
- **AC-7**: Given a gRPC client calls `ListTools`, when the ping tool is registered, then the response includes a tool descriptor with `name: "ping"`, the correct description, and the empty input schema.

**Implementation Notes**:
The feasibility analysis for REQ-00B identifies the following considerations:

- **Dependency on REQ-00A**: The `ToolRegistry` interface and handler function signature must exist before the ping handler can be implemented. REQ-00A must be completed or at minimum its interface defined before REQ-00B implementation begins.
- **Directory creation**: `internal/tools/` does not yet exist and must be created.
- **Timestamp formatting**: Use `time.Now().Format(time.RFC3339)` — no utility wrapper needed.
- **Startup registration**: The registration call in `cmd/` must occur before `grpc.Serve()` is called to ensure the ping tool is available from the first client connection.
- **Complexity**: Low — the handler itself is trivial; the primary risk is integration ordering with REQ-00A.

**Test Expectations**:
- Must test: `TestPingHandler_Invoke_ReturnsPongMessage` — handler returns JSON with `message` equal to `"pong"`
- Must test: `TestPingHandler_Invoke_TimestampIsRFC3339` — the `timestamp` field in the response parses successfully as RFC 3339 and falls within a reasonable window of the invocation time
- Must test: `TestPingHandler_Invoke_OutputIsValidJSON` — handler output is valid JSON (no marshal error, no extra fields beyond `message` and `timestamp`)
- Must test: `TestPingTool_Descriptor_Name` — tool name equals `"ping"`
- Must test: `TestPingTool_Descriptor_Description` — description equals `"Diagnostic tool that returns pong with server timestamp"`
- Must test: `TestPingTool_Descriptor_InputSchema_NoRequiredParams` — input schema is the empty JSON object schema with no required fields
- Must NOT rely on: fixed/hardcoded timestamps in assertions; use time window comparison (`T1 <= parsed <= T2`) instead

---

### SC-013: gRPC server starts alongside HTTP server

**Entity**: `.software/entities/scenarios/SC-013.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/` entry point — dual-server startup logic and `--grpc-port` flag handling
- Integration / process substitution test file under `cmd/`

**Acceptance Criteria**:
- **AC-1**: Given the MCP Server binary starts without flags, when initialization completes, then the HTTP server is listening on port 8080 and the gRPC server is listening on port 50051, and startup log output includes the gRPC listen port.
- **AC-2**: Given the MCP Server binary starts with `--grpc-port 9090`, when initialization completes, then the gRPC server listens on port 9090 and the HTTP server remains on port 8080.

**Implementation Notes**:
No separate feasibility report for SC-013. This scenario is covered by REQ-00A's implementation of dual-server startup (AC-5). The process substitution test pattern (per REQ-001 TDD policy) applies for binary-level port verification.

**Test Expectations**:
- Must test: `TestServer_DefaultPorts_BothServersListen` — process substitution test verifying the server binary binds both 8080 (HTTP) and 50051 (gRPC) when started without flags
- Must test: `TestServer_GRPCPortFlag_OverridesDefault` — process substitution test verifying `--grpc-port 9090` causes gRPC to bind on 9090 while HTTP stays on 8080
- Must test: `TestServer_StartupLog_IncludesGRPCPort` — startup log output contains the gRPC listen port
- Must NOT rely on: sleep-based port polling; use retry-with-timeout or readiness signaling

---

### SC-014: ListTools returns registered built-in tools

**Entity**: `.software/entities/scenarios/SC-014.md`
**Type**: scenario

**Codebase Mapping**:
- `internal/registry/` — `ToolRegistry` implementation
- gRPC handler in `internal/` or `cmd/` wiring `MCPService` to registry

**Acceptance Criteria**:
- **AC-1**: Given a gRPC client connects to port 50051 and calls `ListTools`, when at least one built-in tool is registered, then the response contains a non-empty list of tool descriptors each with a non-empty `name`, non-empty `description`, and a valid JSON Schema string in `input_schema`.

**Implementation Notes**:
No separate feasibility report for SC-014. Covered by the `ListTools` RPC implementation in REQ-00A. Test doubles for the gRPC client must be implemented in the same test file using narrow interfaces (per REQ-001).

**Test Expectations**:
- Must test: `TestMCPService_ListTools_NonEmptyRegistryResponse` — with a registry containing one known tool, `ListTools` returns a response list with that tool's descriptor intact
- Must test: `TestMCPService_ListTools_DescriptorFields_ArePopulated` — every returned descriptor has non-empty `name`, non-empty `description`, and `input_schema` that is valid JSON
- Must NOT rely on: a live gRPC network connection in unit tests; use a test server via `grpc.NewServer()` bound to a `bufconn` listener

---

### SC-015: InvokeTool dispatches to handler and returns result

**Entity**: `.software/entities/scenarios/SC-015.md`
**Type**: scenario

**Codebase Mapping**:
- `internal/registry/` — `Invoke` method dispatching to registered handler
- gRPC `MCPService.InvokeTool` handler

**Acceptance Criteria**:
- **AC-1**: Given a gRPC client calls `InvokeTool` with a valid `tool_name` and JSON-encoded `input`, when the handler for that tool is dispatched, then the response status is OK and `output` contains the valid JSON-encoded result produced by the handler.

**Implementation Notes**:
No separate feasibility report for SC-015. Covered by REQ-00A AC-3 and the `Invoke` path in the registry. The ping tool (REQ-00B) serves as the primary live handler for this scenario.

**Test Expectations**:
- Must test: `TestMCPService_InvokeTool_ValidTool_ReturnsHandlerOutput` — calling `InvokeTool` with a registered tool name returns OK status and the handler's JSON output
- Must test: `TestMCPService_InvokeTool_OutputIsValidJSON` — the `output` field in the response is parseable JSON
- Must NOT rely on: production handler side effects or real network calls in unit tests

---

### SC-016: InvokeTool with unknown tool returns NOT_FOUND

**Entity**: `.software/entities/scenarios/SC-016.md`
**Type**: scenario

**Codebase Mapping**:
- `internal/registry/` — `Invoke` returning not-found error for unregistered name
- gRPC `MCPService.InvokeTool` handler — error translation to `codes.NotFound`

**Acceptance Criteria**:
- **AC-1**: Given no tool named `"nonexistent"` is registered in the ToolRegistry, when a gRPC client calls `InvokeTool` with `tool_name: "nonexistent"`, then the response returns gRPC status code `NOT_FOUND`, the error message contains the string `"nonexistent"`, and no server crash or panic occurs.

**Implementation Notes**:
No separate feasibility report for SC-016. Covered by REQ-00A AC-4. Critical: the gRPC handler must translate the registry's not-found error into `status.Error(codes.NotFound, ...)` — returning a raw Go error would produce `codes.Unknown`.

**Test Expectations**:
- Must test: `TestMCPService_InvokeTool_UnknownTool_ReturnsNotFound` — gRPC status code is `codes.NotFound` when tool name is not registered
- Must test: `TestMCPService_InvokeTool_ErrorMessage_ContainsToolName` — error message from the NOT_FOUND status contains the requested tool name
- Must test: `TestRegistry_Invoke_UnknownTool_ReturnsError` — registry `Invoke` returns a sentinel or typed not-found error for unregistered names (unit test of registry in isolation)
- Must NOT rely on: string matching gRPC status descriptions as the sole correctness check; assert `codes.NotFound` using `status.Code(err)`

---

### SC-017: Tool registry supports concurrent access

**Entity**: `.software/entities/scenarios/SC-017.md`
**Type**: scenario

**Codebase Mapping**:
- `internal/registry/registry.go` — `sync.RWMutex`-protected `Register`, `List`, `Invoke`
- `internal/registry/registry_test.go` — race-detected concurrent test

**Acceptance Criteria**:
- **AC-1**: Given multiple goroutines concurrently call `Register`, `List`, and `Invoke` on the same `ToolRegistry` instance using `sync.WaitGroup`, when the test runs with `go test -race`, then no data races are detected, all `Register` calls complete without error, `List` calls return consistent snapshots, `Invoke` calls for registered tools return correct results, and `Invoke` calls for unregistered tools return appropriate errors.

**Implementation Notes**:
No separate feasibility report for SC-017. Covered by REQ-00A AC-8. The `sync.RWMutex` pattern provides read-concurrent / write-exclusive semantics: `RLock` for `List` and `Invoke` lookup; `Lock` for `Register`.

**Test Expectations**:
- Must test: `TestRegistry_ConcurrentRegisterListInvoke_NoRace` — launch goroutines for all three operations simultaneously under `sync.WaitGroup`, run with `-race` flag
- Must test: `TestRegistry_ConcurrentRegister_AllToolsVisible` — after concurrent `Register` calls complete, `List` returns all registered tools (no silent drops)
- Must NOT rely on: sequential execution guarantees; test must exercise genuine concurrency with multiple goroutines per operation type

---

### SC-018: Ping tool registered at startup

**Entity**: `.software/entities/scenarios/SC-018.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/` entry point — startup registration of `ping` tool before `grpc.Serve()`
- `internal/tools/ping.go` — tool descriptor values

**Acceptance Criteria**:
- **AC-1**: Given the MCP Server starts normally, when a gRPC client calls `ListTools`, then the response includes a tool descriptor with `name: "ping"`, `description: "Diagnostic tool that returns pong with server timestamp"`, and an `input_schema` defining an empty object with no required parameters.

**Implementation Notes**:
No separate feasibility report for SC-018. Covered by REQ-00B AC-1, AC-2, AC-3, and AC-7. This scenario validates the startup wiring in `cmd/` — the ping tool must be registered before the server begins accepting requests.

**Test Expectations**:
- Must test: `TestStartup_PingToolRegistered_InListTools` — process-level or in-process test verifying `ListTools` response contains the `ping` descriptor after server initialization
- Must test: `TestPingTool_Descriptor_MatchesSpec` — unit test verifying the tool's `name`, `description`, and `input_schema` match the specified values exactly
- Must NOT rely on: polling or sleep-based startup detection; use server readiness signaling

---

### SC-019: Ping invocation returns pong with RFC 3339 timestamp

**Entity**: `.software/entities/scenarios/SC-019.md`
**Type**: scenario

**Codebase Mapping**:
- `internal/tools/ping.go` — handler implementation
- `internal/tools/ping_test.go` — timestamp window assertion

**Acceptance Criteria**:
- **AC-1**: Given the MCP Server is running with the `ping` tool registered, when a gRPC client calls `InvokeTool` with `tool_name: "ping"` and empty JSON input, then the response status is OK, the output is valid JSON with exactly two fields (`message` and `timestamp`), `message` equals `"pong"`, `timestamp` is a valid RFC 3339 formatted datetime string, and the parsed timestamp falls within the window [T1, T2] recorded around the invocation (with 1 second tolerance).

**Implementation Notes**:
No separate feasibility report for SC-019. Covered by REQ-00B AC-4 and AC-5. The timestamp window assertion [T1, T2] is critical — record `time.Now()` before and after the `InvokeTool` call and verify the parsed timestamp falls within that range.

**Test Expectations**:
- Must test: `TestPingHandler_Invoke_MessageIsPong` — output JSON `message` field equals `"pong"`
- Must test: `TestPingHandler_Invoke_TimestampParsesAsRFC3339` — `time.Parse(time.RFC3339, timestamp)` succeeds without error
- Must test: `TestPingHandler_Invoke_TimestampWithinWindow` — parsed timestamp is between T1 and T2 recorded around the invocation, with 1 second tolerance
- Must test: `TestPingHandler_Invoke_OutputHasExactlyTwoFields` — the output JSON object contains exactly `message` and `timestamp`, no additional fields
- Must NOT rely on: fixed expected timestamp strings; always use a time window comparison

---

## Documentation Tasks

### RELEASES.md Entry

**Required**: Always

Add an entry to RELEASES.md documenting:
- Sprint ID and title: SP-004 — gRPC tool registry service with ping diagnostic tool
- Summary of changes delivered: gRPC `MCPService` on port 50051 with `ListTools` and `InvokeTool` RPCs; concurrent-safe internal `ToolRegistry`; `proto/mcp/v1/mcp.proto` schema with `make proto` code generation; `ping` built-in diagnostic tool registered at startup; K8s Service NodePort 30051 for host-accessible gRPC
- Entity IDs included: REQ-00A, REQ-00B, SC-013, SC-014, SC-015, SC-016, SC-017, SC-018, SC-019
- Date of completion: to be filled at completion time

This entry should be appended to the existing RELEASES.md file. Do not read or modify existing entries.

### README.md Update

**Required**: User-facing changes detected

Update README.md to reflect:
- New `--grpc-port` flag on the MCP Server binary (default: 50051)
- The MCP Server now exposes a gRPC endpoint on port 50051 (NodePort 30051 in the local k3d cluster) with `MCPService` operations: `ListTools` and `InvokeTool`
- The `ping` built-in diagnostic tool is available via `InvokeTool(tool_name: "ping")` — returns `{"message":"pong","timestamp":"<RFC3339>"}` and can be used to verify server reachability
- `make proto` target: regenerates Go code from `proto/mcp/v1/mcp.proto` into `gen/proto/mcp/v1/`
- Local development: the gRPC service is accessible at `localhost:30051` when deployed to k3d

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection | No spec-phase directive in REQ-005 body itself; the catalog triggers REQ-001 (TDD), REQ-002 (release), REQ-003 (cluster integration testing), and REQ-004 (k3d topology). Spec writer is instructed (via REQ-001) to generate "Test Expectations" subsections per requirement; REQ-003 instructs spec writer to generate "Verify Expectations" for cluster-facing changes. | Test Expectations subsections generated for all REQ and SC entities. REQ-003 cluster integration check noted in REQ-00A Codebase Mapping (gRPC NodePort check in `deploy/k8s/verify/checks.go`). REQ-002 release process reflected in Documentation Tasks. REQ-004 k3d topology context applied to K8s Service and NodePort 30051 notes. |

## Out of Scope

- Plugin-sourced tool registration via NATS discovery (REQ-00A registry is designed to accommodate this but does not implement it in this sprint)
- Migration of `proto/mcp/v1/mcp.proto` to the SDK submodule (deferred; proto lives locally for now)
- MCP transport (stdio/HTTP+SSE) integration with the tool registry — the gRPC layer is the direct interface for this sprint
- Additional built-in tools beyond `ping`
- Authentication or authorization on the gRPC endpoint
- gRPC reflection service

## Prerequisites

- Go module dependencies `google.golang.org/grpc` and `google.golang.org/protobuf` must be added to `go.mod` before protobuf code generation
- `protoc` compiler and `protoc-gen-go` / `protoc-gen-go-grpc` plugins must be installed in the development environment before `make proto` can be executed
- REQ-00A `ToolRegistry` interface must be defined before REQ-00B ping handler implementation begins
- The existing HTTP server startup and graceful shutdown pattern in `cmd/` must be understood before the dual-server extension is implemented
