# Sprint SP-005: gRPC request logging interceptor

**Created**: 2026-07-09
**Status**: Specified
**Entities**: 4

---

## Overview

This sprint introduces structured request logging to the MCP Server's gRPC layer. A new `UnaryServerInterceptor` in the `internal/logging` package wraps every RPC handler to emit a JSON log entry on completion — capturing method path, duration, status code, and (for `InvokeTool`) the target tool name. As part of the same change, the server's existing `log.Logger` usage for startup and shutdown is migrated to `log/slog` with a `JSONHandler`, ensuring all pod log output is uniformly structured and consumable by Kubernetes log aggregation. The sprint directly improves observability: operators can now confirm that requests arrive, measure latency, and diagnose failures from `kubectl logs` alone.

## Entity Inventory

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-00C | requirement | gRPC request logging interceptor | no | - |
| SC-01A | scenario | ListTools RPC logs method duration and status | no | - |
| SC-01B | scenario | InvokeTool RPC log includes tool name | no | - |
| SC-01C | scenario | Failed RPC logs at error level with status code | no | - |

## Technical Context

The codebase analysis reveals that `grpc.NewServer()` in `cmd/eve-realm-mcp/main.go:L112` is currently called with no options, making interceptor injection a one-line addition. The existing test infrastructure in `internal/mcp/service_test.go` follows a `newTestServer(t, reg)` bufconn pattern that the new interceptor test file will extend.

**Entity-to-code highlights:**

- `cmd/eve-realm-mcp/main.go` (L105-L113, L185-L191): The `startServers` and `ShutdownServer` functions accept `*log.Logger` today. Both signatures migrate to `*slog.Logger`, and four `logger.Printf`/`logger.Println` callsites are updated to `slog.Info`/`slog.Error`.
- `gen/proto/mcp/v1/mcp.pb.go` (L171-L224): `InvokeToolRequest.ToolName` is extracted via type assertion inside the interceptor using the generated constant `MCPService_InvokeTool_FullMethodName`.
- `internal/mcp/service_test.go` (L25-L57, L286-L354): Establishes the bufconn + `t.Helper()` + table-driven patterns that interceptor tests replicate.

**Files to create:**

| File | Purpose |
|------|---------|
| `internal/logging/interceptor.go` | `NewInterceptor` function returning a `grpc.UnaryServerInterceptor`: records start time, invokes handler, computes `duration_ms`, extracts gRPC status, conditionally reads `tool_name` for `InvokeTool` calls, logs at INFO/ERROR via `slog`. |
| `internal/logging/interceptor_test.go` | Table-driven test cases for SC-01A, SC-01B, SC-01C using a bufconn server with the interceptor injected and a `bytes.Buffer`-backed `slog.JSONHandler` for log capture. |

**Files to modify:**

| File | Lines | Change |
|------|-------|--------|
| `cmd/eve-realm-mcp/main.go` | L8, L86, L105-L113, L142-L143, L171, L185-L191 | Replace `"log"` with `"log/slog"`; change `startServers`/`ShutdownServer` signatures to `*slog.Logger`; wire `grpc.UnaryInterceptor(logging.NewInterceptor(logger))` into `grpc.NewServer()`; replace all `logger.Printf`/`logger.Println` callsites. |
| `cmd/eve-realm-mcp/main_test.go` | L8-L18, L413, L467, L525, L566, L599, L703-L735 | Replace `log.New(&logBuf, "", 0)` with `slog.New(slog.NewJSONHandler(&logBuf, nil))`; update test assertions to handle JSON log output format. |

**Critical integration points:**

1. `grpc.NewServer()` at `main.go:L112` receives `grpc.UnaryInterceptor(logging.NewInterceptor(logger))` as the sole server option.
2. Inside `interceptor.go`, `req` is type-asserted to `*mcpv1.InvokeToolRequest` only when `info.FullMethod == MCPService_InvokeTool_FullMethodName`.
3. `codes.Code.String()` must be used for the `status` field — grpc-go v1.81.1 returns CamelCase proto names (e.g., `"NotFound"`, not `"NOT_FOUND"`), matching SC-01C's expectation.
4. The shared `logBuf` in table-driven interceptor tests must be reset between iterations to avoid log bleed between cases.

## Implementation Sections

### REQ-00C: gRPC request logging interceptor

**Entity**: `.software/entities/requirements/REQ-00C.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:

Files to create:
- `internal/logging/interceptor.go` — new package; `NewInterceptor(*slog.Logger) grpc.UnaryServerInterceptor`

Files to modify:
- `cmd/eve-realm-mcp/main.go` — L8 (imports), L86 (ShutdownServer signature), L105-L113 (startServers signature + grpc.NewServer call), L142-L143 (log call), L171 (log call), L185-L191 (log calls in shutdown)
- `cmd/eve-realm-mcp/main_test.go` — L8-L18 (imports + logger construction), L413, L467, L525, L566, L599 (ShutdownServer call sites), L703-L735 (log assertion update)

Related generated files (read-only, referenced only):
- `gen/proto/mcp/v1/mcp.pb.go` — `InvokeToolRequest` struct and `ToolName` field
- `gen/proto/mcp/v1/mcp_grpc.pb.go` — `MCPService_InvokeTool_FullMethodName` constant

**Acceptance Criteria**:

- **AC-1**: Given the gRPC server starts, when `grpc.NewServer` is called in `startServers`, then a `grpc.UnaryServerInterceptor` produced by `logging.NewInterceptor` is registered as a server option.
- **AC-2**: Given any unary RPC completes, when the interceptor runs post-handler, then a structured JSON log entry is emitted containing `method` (full gRPC path), `duration_ms` (numeric elapsed milliseconds), and `status` (gRPC code name such as `"OK"` or `"NotFound"`).
- **AC-3**: Given an `InvokeTool` RPC completes, when the interceptor logs the entry, then the entry additionally contains a `tool_name` field extracted from `InvokeToolRequest.ToolName`.
- **AC-4**: Given a unary RPC returns status OK, when the interceptor logs, then the log level is `slog.LevelInfo`.
- **AC-5**: Given a unary RPC returns any non-OK status, when the interceptor logs, then the log level is `slog.LevelError` and the log entry includes an `error` field with the error message.
- **AC-6**: Given the server starts and shuts down, when logging occurs during those lifecycle events, then the server uses `slog` with a `JSONHandler` writing to stdout, and `*log.Logger` usage is fully replaced in `main.go`.
- **AC-7**: Given the pod is deployed to Kubernetes, when an RPC is processed, then log output is visible and parseable via `kubectl logs`.

**Implementation Notes**:

Feasibility: PROCEED-WITH-CAVEATS. Complexity is M (medium). No dependency changes are required — grpc-go v1.81.1 and Go 1.25.0 provide everything needed.

Key risks and mitigations:
- **`*log.Logger` → `*slog.Logger` migration breaks existing test assertions** (High likelihood, Medium impact): All 7 callsites in `main_test.go` that use `log.New(&logBuf, "", 0)` must be updated in lockstep to `slog.New(slog.NewJSONHandler(&logBuf, nil))`. Test assertions checking log output must be updated from plain-string contains to JSON-aware checks.
- **`status` string format for SC-01C** (High likelihood, Medium impact): Use `codes.Code.String()` which returns CamelCase proto names in grpc-go v1.81.1 — produces `"NotFound"`, not `"NOT_FOUND"`.
- **`tool_name` type assertion** (Low likelihood, Low impact): The `info.FullMethod` check gates the assertion; safe to apply directly.
- **Global logger state in tests** (Low likelihood, Low impact): Constructor-injected `*slog.Logger` avoids any global state conflict.

Prerequisites: REQ-00A (gRPC tool registry service) must be delivered; REQ-00B (ping diagnostic tool) must be delivered. Both are confirmed active/delivered.

---

### SC-01A: ListTools RPC logs method duration and status

**Entity**: `.software/entities/scenarios/SC-01A.md`
**Type**: scenario
**Priority**: (derived from REQ-00C — high)

**Codebase Mapping**:

Files to create:
- `internal/logging/interceptor_test.go` — test case for this scenario using bufconn server with interceptor, `bytes.Buffer`-backed `slog.JSONHandler`

Related reference files:
- `internal/mcp/service_test.go` (L25-L57) — `newTestServer` helper pattern to replicate
- `internal/mcp/service_test.go` (L94-L140) — table-driven test structure to follow

**Acceptance Criteria**:

- **AC-1**: Given the MCP Server runs with the gRPC logging interceptor enabled and the ping tool is registered, when a gRPC client calls `ListTools` with an empty request, then the `ListTools` response succeeds with the list of registered tools.
- **AC-2**: Given the `ListTools` call completes successfully, when the interceptor emits its log entry to stdout, then the JSON entry contains `"method": "/mcp.v1.MCPService/ListTools"`.
- **AC-3**: Given the `ListTools` call completes, when the interceptor logs, then the entry contains a `"duration_ms"` field with a numeric value greater than zero.
- **AC-4**: Given the `ListTools` call returns status OK, when the interceptor logs, then the entry contains `"status": "OK"` and `"level": "INFO"`.

**Implementation Notes**:

Fully testable via in-process bufconn gRPC server. The test injects the interceptor as a `grpc.ServerOption` and captures log output through a `bytes.Buffer`-backed `slog.JSONHandler`. JSON assertions verify each required field. Follows the `newTestServer` + `t.Helper()` + table-driven pattern established in `internal/mcp/service_test.go`.

---

### SC-01B: InvokeTool RPC log includes tool name

**Entity**: `.software/entities/scenarios/SC-01B.md`
**Type**: scenario
**Priority**: (derived from REQ-00C — high)

**Codebase Mapping**:

Files to create:
- `internal/logging/interceptor_test.go` — test case for this scenario (same file as SC-01A; separate table entry)

Related reference files:
- `internal/mcp/service_test.go` (L25-L57) — `newTestServer` helper pattern
- `gen/proto/mcp/v1/mcp_grpc.pb.go` — `MCPService_InvokeTool_FullMethodName` constant used in interceptor
- `gen/proto/mcp/v1/mcp.pb.go` (L172-L224) — `InvokeToolRequest` struct with `ToolName` field

**Acceptance Criteria**:

- **AC-1**: Given the MCP Server runs with the interceptor enabled and the ping tool is registered, when a gRPC client calls `InvokeTool` with `tool_name = "ping"` and empty input, then the `InvokeTool` response succeeds with the ping result (pong + RFC 3339 timestamp).
- **AC-2**: Given the `InvokeTool` call completes, when the interceptor logs the entry, then the JSON entry contains `"tool_name": "ping"` extracted from the request.
- **AC-3**: Given the `InvokeTool` call returns status OK, when the interceptor logs, then the entry contains `"method": "/mcp.v1.MCPService/InvokeTool"`, `"status": "OK"`, `"level": "INFO"`, and a numeric `"duration_ms"` field.

**Implementation Notes**:

Fully testable. Same bufconn setup as SC-01A, with the ping tool registered via the tool registry. The test must verify that `tool_name` is present only on `InvokeTool` entries — i.e., the type assertion path in the interceptor activates correctly when `info.FullMethod == MCPService_InvokeTool_FullMethodName`.

---

### SC-01C: Failed RPC logs at error level with status code

**Entity**: `.software/entities/scenarios/SC-01C.md`
**Type**: scenario
**Priority**: (derived from REQ-00C — high)

**Codebase Mapping**:

Files to create:
- `internal/logging/interceptor_test.go` — test case for this scenario (same file; separate table entry)

Related reference files:
- `internal/mcp/service_test.go` (L286-L354) — existing NotFound error path test for reference
- `gen/proto/mcp/v1/mcp_grpc.pb.go` — `MCPService_InvokeTool_FullMethodName` constant

**Acceptance Criteria**:

- **AC-1**: Given the MCP Server runs with the interceptor enabled and no tool named `"nonexistent"` is registered, when a gRPC client calls `InvokeTool` with `tool_name = "nonexistent"`, then the gRPC response returns status `NOT_FOUND` (as per REQ-00A).
- **AC-2**: Given the `InvokeTool` call returns `NOT_FOUND`, when the interceptor logs the entry, then the JSON entry contains `"level": "ERROR"`.
- **AC-3**: Given the failed RPC is logged at error level, when the JSON entry is inspected, then it contains `"status": "NotFound"` (CamelCase, using `codes.Code.String()`), `"tool_name": "nonexistent"`, `"method": "/mcp.v1.MCPService/InvokeTool"`, a numeric `"duration_ms"` field, and an `"error"` field with the descriptive error message.

**Implementation Notes**:

Fully testable. The `NOT_FOUND` error path is already exercised at the service level in `internal/mcp/service_test.go:L286-L354`; this test verifies the interceptor layer specifically. The critical implementation note is that `codes.Code.String()` must be used — not `status.Code(err).String()` with different casing conventions — to match the `"NotFound"` format expected by this scenario. The `error` field must contain the handler's error message, not a generic string.

---

## Documentation Tasks

### RELEASES.md Entry

**Required**: Always

Add an entry to RELEASES.md documenting:
- Sprint ID: SP-005 and title: gRPC request logging interceptor
- Summary: introduces `internal/logging.NewInterceptor`, wires it into `grpc.NewServer` in `startServers`, migrates `main.go` startup/shutdown logging from `log.Logger` to `slog` with JSON output, and updates `main_test.go` log assertion patterns
- Entity IDs included: REQ-00C, SC-01A, SC-01B, SC-01C
- Date of completion

This entry should be appended to the existing RELEASES.md file. Do not read or modify existing entries.

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection | Acknowledged — no spec-phase action required | The catalog is acknowledged; implementation and plan phases must load REQ-001 (TDD Strategy) as this sprint implements and modifies Go code, REQ-002 (sprint completion process) at release time, and REQ-004 (k3d topology) if K8s manifests are touched. No spec-phase directive is embedded in this catalog entity. |

## Out of Scope

- Runtime-configurable log levels (reserved for a future requirement as noted in REQ-00C)
- `StreamServerInterceptor` for streaming RPCs (noted as a future extension in REQ-00C; agent progress streaming is not part of this sprint)
- Log aggregation infrastructure or external log shipping configuration
- Changes to K8s manifests (AC-7 is satisfied by the existing pod stdout capture; no manifest changes are required)
- Any modification to the SDK submodule or protobuf definitions

## Prerequisites

- REQ-00A (gRPC tool registry service): The `grpc.Server` constructed in `startServers` must exist and be fully wired. Confirmed active/delivered.
- REQ-00B (ping built-in diagnostic tool): The ping tool must be registered at startup for SC-01B and SC-01C test cases to produce meaningful results. Confirmed active/delivered.
- Go 1.25.0 and grpc-go v1.81.1 as specified in `go.mod`: Both confirmed present; no version changes needed.
