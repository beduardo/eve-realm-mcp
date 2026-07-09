# Implementation Plan: gRPC request logging interceptor

**Sprint**: SP-005
**Created**: 2026-07-09
**Spec**: SPEC.md
**Status**: Ready for Implementation

## Summary

This plan delivers a `UnaryServerInterceptor` in a new `internal/logging` package that wraps every gRPC RPC handler to emit a structured JSON log entry on completion — capturing method path, duration in milliseconds, gRPC status code, and (for `InvokeTool` calls) the tool name extracted from the request. In lockstep, the plan migrates the server's existing `log.Logger` usage in `cmd/eve-realm-mcp/main.go` to `log/slog` with a `JSONHandler`, and updates the corresponding test assertions in `main_test.go`. The result is uniform, Kubernetes-aggregatable JSON log output for every pod lifecycle event and every gRPC request.

## Entity Coverage

| Entity  | Type        | Partial | Scope                  |
|---------|-------------|---------|------------------------|
| REQ-00C | requirement | no      | Full implementation    |
| SC-01A  | scenario    | no      | Full implementation    |
| SC-01B  | scenario    | no      | Full implementation    |
| SC-01C  | scenario    | no      | Full implementation    |

## Implementation Steps

### Step 1: Interceptor tests — ListTools, InvokeTool OK, InvokeTool NotFound

**Description**: Create `internal/logging/interceptor_test.go` with three table-driven test cases covering SC-01A (ListTools logs method, duration_ms, status OK at INFO level), SC-01B (InvokeTool logs tool_name for a registered ping tool), and SC-01C (InvokeTool with an unregistered tool logs at ERROR level with status "NotFound" and an error field). The test helper follows the `newTestServer` + `t.Helper()` + bufconn pattern established in `internal/mcp/service_test.go`, injecting the (not-yet-written) interceptor via `grpc.UnaryInterceptor` and capturing output via a `bytes.Buffer`-backed `slog.JSONHandler`. The `logBuf` is reset between table iterations to prevent log bleed. These tests must be written first and must fail (red phase) before Step 2 introduces the production interceptor.
**Entities**: REQ-00C, SC-01A, SC-01B, SC-01C
**Files to modify**:
- `internal/logging/interceptor_test.go` (create)
**Acceptance criteria**:
- [ ] File compiles (the `NewInterceptor` symbol may be a stub or forward declaration to allow compilation)
- [ ] `TestInterceptor_ListTools_LogsMethodDurationStatus` case exists and fails without the production implementation
- [ ] `TestInterceptor_InvokeTool_LogsToolName` case exists and verifies `tool_name` field is present in the JSON log entry
- [ ] `TestInterceptor_InvokeTool_NotFound_LogsAtErrorLevel` case exists and verifies `level: ERROR`, `status: "NotFound"`, and `error` field
- [ ] Each test case resets `logBuf` between iterations
- [ ] No external test frameworks are used — only `testing` stdlib
- [ ] Test follows table-driven `[]struct{ name string; ... }` with `t.Run` pattern
**Estimated complexity**: M
**Depends on**: None

**Test Expectations (from SPEC)**:
- SC-01A AC-2: JSON entry contains `"method": "/mcp.v1.MCPService/ListTools"`
- SC-01A AC-3: Entry contains `"duration_ms"` with a numeric value greater than zero
- SC-01A AC-4: Entry contains `"status": "OK"` and `"level": "INFO"`
- SC-01B AC-2: JSON entry contains `"tool_name": "ping"`
- SC-01B AC-3: Entry contains `"method": "/mcp.v1.MCPService/InvokeTool"`, `"status": "OK"`, `"level": "INFO"`, and numeric `"duration_ms"`
- SC-01C AC-2: JSON entry contains `"level": "ERROR"`
- SC-01C AC-3: Entry contains `"status": "NotFound"`, `"tool_name": "nonexistent"`, `"method": "/mcp.v1.MCPService/InvokeTool"`, numeric `"duration_ms"`, and `"error"` field

**Testing Approach**: TDD

---

### Step 2: Interceptor production implementation

**Description**: Create `internal/logging/interceptor.go` containing `NewInterceptor(*slog.Logger) grpc.UnaryServerInterceptor`. The interceptor records the start time, invokes the handler, computes `duration_ms` as elapsed milliseconds (float64), extracts the gRPC status code via `status.FromError`, uses `codes.Code.String()` for the status string field (producing CamelCase names such as `"OK"` and `"NotFound"`), and conditionally extracts `tool_name` by type-asserting `req` to `*mcpv1.InvokeToolRequest` only when `info.FullMethod == mcpv1.MCPService_InvokeTool_FullMethodName`. Successful (OK) calls log at `slog.LevelInfo`; any non-OK call logs at `slog.LevelError` with an additional `error` attribute. Running the Step 1 tests after this step must produce a green result.
**Entities**: REQ-00C, SC-01A, SC-01B, SC-01C
**Files to modify**:
- `internal/logging/interceptor.go` (create)
**Acceptance criteria**:
- [ ] `NewInterceptor` is exported from package `logging` with signature `func NewInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor`
- [ ] Log entry always includes `method` (string), `duration_ms` (float64), and `status` (string from `codes.Code.String()`)
- [ ] `tool_name` field is present in log entries for `InvokeTool` RPCs and absent for all other RPCs
- [ ] Non-OK status produces `slog.LevelError` log with an `error` attribute containing the handler error message
- [ ] OK status produces `slog.LevelInfo` log
- [ ] `go test ./internal/logging/...` passes (all three scenario tests green)
- [ ] No external libraries beyond `google.golang.org/grpc` and `log/slog` stdlib are introduced
**Estimated complexity**: M
**Depends on**: Step 1

**Test Expectations (from SPEC)**:
- REQ-00C AC-1: `grpc.UnaryServerInterceptor` produced by `logging.NewInterceptor` can be registered as a server option
- REQ-00C AC-2: Structured JSON log entry contains `method`, `duration_ms`, and `status` on every unary RPC completion
- REQ-00C AC-3: `InvokeTool` log entries additionally contain `tool_name`
- REQ-00C AC-4: OK calls log at `slog.LevelInfo`
- REQ-00C AC-5: Non-OK calls log at `slog.LevelError` with `error` field

**Testing Approach**: TDD

---

### Step 3: Migrate main.go from log.Logger to slog and wire interceptor

**Description**: Modify `cmd/eve-realm-mcp/main.go` to replace the `"log"` import with `"log/slog"`, change the `startServers` and `ShutdownServer` function signatures from `*log.Logger` to `*slog.Logger`, wire `grpc.UnaryInterceptor(logging.NewInterceptor(logger))` into the `grpc.NewServer()` call at L112, and update the four `logger.Printf`/`logger.Println` callsites (L142-L143, L171, L185-L191 and `main()`) to the equivalent `slog` method calls. The `main()` function constructs its logger as `slog.New(slog.NewJSONHandler(os.Stdout, nil))` and passes it through. This step satisfies REQ-00C AC-6 (full `log.Logger` removal from main.go) and AC-1 (interceptor wired into server).
**Entities**: REQ-00C
**Files to modify**:
- `cmd/eve-realm-mcp/main.go` (modify)
**Acceptance criteria**:
- [ ] `"log"` import is removed; `"log/slog"` is imported instead
- [ ] `startServers` signature accepts `*slog.Logger` as the logger parameter
- [ ] `ShutdownServer` signature accepts `*slog.Logger` as the logger parameter
- [ ] `grpc.NewServer()` is called with `grpc.UnaryInterceptor(logging.NewInterceptor(logger))` as a server option
- [ ] All four `logger.Printf`/`logger.Println` callsites are replaced with `slog` equivalents (`slog.Info`, `slog.Error`, or `logger.Info`, `logger.Error`)
- [ ] `main()` creates logger as `slog.New(slog.NewJSONHandler(os.Stdout, nil))`
- [ ] `go build ./cmd/eve-realm-mcp/` succeeds with no errors
**Estimated complexity**: M
**Depends on**: Step 2

**Test Expectations (from SPEC)**:
- REQ-00C AC-6: `*log.Logger` usage fully replaced in `main.go`; server uses `slog` with `JSONHandler` writing to stdout

**Testing Approach**: TDD

---

### Step 4: Update main_test.go log assertions for slog/JSON output

**Description**: Modify `cmd/eve-realm-mcp/main_test.go` to replace all seven `log.New(&logBuf, "", 0)` logger constructions with `slog.New(slog.NewJSONHandler(&logBuf, nil))`, update the `"log"` import to `"log/slog"`, and revise any log output assertions that use plain-string `strings.Contains` checks to instead parse the captured buffer as newline-delimited JSON and assert on the relevant JSON fields. The key assertion in `TestServer_StartupLog_IncludesGRPCPort` (L578-L581) that checks for `"grpc listening on :PORT"` must be adapted to match the JSON format produced by `slog`. `TestGracefulShutdown` (L703-L735) that checks for `"eve-realm-mcp shutting down"` likewise requires JSON-aware assertion.
**Entities**: REQ-00C
**Files to modify**:
- `cmd/eve-realm-mcp/main_test.go` (modify)
**Acceptance criteria**:
- [ ] All seven `log.New(&logBuf, "", 0)` occurrences replaced with `slog.New(slog.NewJSONHandler(&logBuf, nil))`
- [ ] `"log"` import replaced with `"log/slog"`
- [ ] `TestServer_StartupLog_IncludesGRPCPort` passes with JSON-format log output
- [ ] `TestGracefulShutdown` passes: JSON log entry contains the shutdown message text
- [ ] All other tests in `main_test.go` that call `startServers` or `ShutdownServer` compile and pass
- [ ] `go test ./cmd/eve-realm-mcp/` passes with no failures
**Estimated complexity**: M
**Depends on**: Step 3

**Test Expectations (from SPEC)**:
- REQ-00C AC-6: Test assertions reflect JSON log output format from `slog.JSONHandler`
- main_test.go L413, L467, L525, L566, L599: All `log.New` constructions updated

**Testing Approach**: TDD

---

### Step 5: Full test suite verification

**Description**: Run `go test ./...` from the repository root to confirm all packages pass: `internal/logging/...` (interceptor tests for SC-01A, SC-01B, SC-01C), `internal/mcp/...` (existing service tests, unaffected), and `cmd/eve-realm-mcp/` (updated main_test.go). This step is a verification gate — no new production code is written. If any test fails, the relevant step above is revisited before proceeding to documentation.
**Entities**: REQ-00C, SC-01A, SC-01B, SC-01C
**Files to modify**:
- N/A (verification only)
**Acceptance criteria**:
- [ ] `go test ./...` exits with code 0
- [ ] `internal/logging` package tests all pass (SC-01A, SC-01B, SC-01C covered)
- [ ] `internal/mcp` package tests all pass (no regression)
- [ ] `cmd/eve-realm-mcp` package tests all pass (log migration complete)
- [ ] No new test failures introduced relative to the SP-004 baseline
**Estimated complexity**: S
**Depends on**: Step 4

**Testing Approach**: TDD

---

### Step 6: RELEASES.md Append

**Description**: Append a release entry to `RELEASES.md` documenting the SP-005 delivery. The entry records the sprint ID, title, date, a 2-3 sentence summary covering the new `internal/logging.NewInterceptor`, its wiring into `grpc.NewServer` in `startServers`, the migration of `main.go` startup/shutdown logging from `log.Logger` to `slog` with JSON output, and the update to `main_test.go` log assertion patterns. Entity IDs REQ-00C, SC-01A, SC-01B, and SC-01C are listed. The existing entries in RELEASES.md are not modified.
**Entities**: REQ-00C, SC-01A, SC-01B, SC-01C
**Files to modify**:
- `RELEASES.md` (modify)
**Acceptance criteria**:
- [ ] RELEASES.md has a new entry with sprint ID SP-005 and date 2026-07-09
- [ ] Entry lists entity IDs REQ-00C, SC-01A, SC-01B, SC-01C
- [ ] Entry summarizes the interceptor introduction, slog migration, and test update in 2-3 sentences
- [ ] Existing RELEASES.md entries are unchanged
**Estimated complexity**: S
**Depends on**: Step 5

---

## Step Dependency Graph

```
Step 1 (interceptor tests — red)
  └── Step 2 (interceptor implementation — green)
        └── Step 3 (main.go migration + interceptor wiring)
              └── Step 4 (main_test.go log assertion update)
                    └── Step 5 (full test suite verification)
                          └── Step 6 (RELEASES.md)
```

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection | Catalog trigger for REQ-001 fires: this sprint implements and modifies Go code. REQ-001 (TDD Strategy) must be loaded and followed. REQ-002 applies at release time (Step 6 + post-sprint release sequence). REQ-003 and REQ-004 do not apply — no K8s manifests, ConfigMap entries, or inter-pod communication is touched. | REQ-001 (TDD Strategy) is fully enforced: test file is created in Step 1 before production code in Step 2 (red→green ordering); table-driven tests with `testing` stdlib only; bufconn pattern at the gRPC boundary; `logBuf` reset between iterations. REQ-002 release process is addressed by Step 6 (RELEASES.md entry). REQ-003 and REQ-004 are confirmed out of scope for this sprint. |
