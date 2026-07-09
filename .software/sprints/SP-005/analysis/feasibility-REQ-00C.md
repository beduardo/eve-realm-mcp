# Feasibility Report: REQ-00C

**Entity**: gRPC request logging interceptor
**Type**: requirement
**Analyzed**: 2026-07-09

---

## Recommendation

**PROCEED-WITH-CAVEATS**

REQ-00C is technically feasible with zero dependency changes required. However, two implementation decisions carry non-trivial risk: migrating the `*log.Logger`-based `startServers` signature to `*slog.Logger` will break existing test assertions that rely on `log.New(&logBuf, "", 0)`, and the status string format mandated by SC-01C ("NotFound") differs subtly from some gRPC conventions and must be verified against grpc-go v1.81.1.

---

## Prerequisite Status

| Prerequisite | Type | Required Status | Current Status | Ready? |
|---|---|---|---|---|
| REQ-00A (gRPC tool registry service) | requirement | active/delivered | active | Yes |
| REQ-00B (ping built-in diagnostic tool) | requirement | active/delivered | active | Yes |
| SC-01A (ListTools RPC logs method, duration, status) | scenario | validated | validated | Yes |
| SC-01B (InvokeTool RPC log includes tool name) | scenario | validated | validated | Yes |
| SC-01C (Failed RPC logs at error level with status code) | scenario | validated | validated | Yes |

All prerequisite entities are in place.

---

## Dependency Graph Analysis

**Direct dependencies**: 3 (REQ-00A, REQ-00B, SC-01A/01B/01C)
**Transitive dependencies**: 0
**Blocking dependencies**: 0

- REQ-00C → REQ-00A: Interceptor registered on the `grpc.Server` built in `startServers` (confirmed at `cmd/eve-realm-mcp/main.go:L112`)
- REQ-00C → REQ-00B: SC-01B and SC-01C scenarios use the ping tool. Confirmed registered at line 109.
- REQ-00C → SC-01A, SC-01B, SC-01C: Define acceptance shape of log output. All validated.

---

## Complexity Estimate

**Size**: M

| Factor | Assessment | Notes |
|---|---|---|
| Code changes | M | New `internal/logging` package (~80-120 LOC), changes to `main.go` (~30 LOC net) |
| Test coverage | M | 3 new unit tests + migration of existing test logger callsites |
| Integration risk | Low | Pure server-side middleware; no proto changes; no new external dependencies |
| Architectural impact | Low | Additive: new package, no structural reorganization |

---

## Risk Factors

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing tests break on `*log.Logger` → `*slog.Logger` migration | High | Medium | Update all 7 test callsites in `main_test.go` in lockstep. Tests using `log.New(&logBuf, ...)` become `slog.New(slog.NewJSONHandler(&logBuf, nil))`. |
| SC-01C expects `"NotFound"` but `codes.NotFound.String()` may differ | High | Medium | Verify `codes.Code.String()` returns CamelCase proto names in grpc-go v1.81.1. |
| `tool_name` extraction requires type assertion on interceptor `req interface{}` | Low | Low | Direct type assertion `req.(*mcpv1.InvokeToolRequest)` is safe; `FullMethod` string gates the check. |
| `grpc.UnaryInterceptor` vs `grpc.ChainUnaryInterceptor` | Low | Low | Single interceptor: `grpc.UnaryInterceptor` is appropriate. Can migrate to `ChainUnaryInterceptor` later. |
| Stdout capture in unit tests conflicts with global logger | Low | Low | Constructor injection of `*slog.Logger` avoids global state. |

---

## Blockers

None.

---

## Testability of Scenarios

**SC-01A** (ListTools logs method, duration, status): Fully testable. Bufconn-based server with interceptor, `bytes.Buffer`-backed `slog.JSONHandler`, JSON assertion on `method`, `duration_ms`, `status`, `level`.

**SC-01B** (InvokeTool log includes tool_name): Fully testable. Same pattern, register ping tool, verify `tool_name: "ping"`.

**SC-01C** (Failed RPC logs at error level): Fully testable. Call with `"nonexistent"` tool, verify `level: "ERROR"`, `status: "NotFound"`, `error` field present.

---

## Notes

- **Go 1.25.0**: `log/slog` is stdlib since Go 1.21. No import changes needed.
- **grpc-go v1.81.1**: `grpc.UnaryInterceptor` and `grpc.ChainUnaryInterceptor` both available.
- **No existing slog usage**: Migration scope is `main.go` only — 4 callsites.
- **Test breakage scope is bounded**: 7 callsites in `main_test.go`, predictable mechanical changes.
- **`grpc.NewServer()` at main.go:L112**: One-line change to add interceptor option.
- **Generated constants available**: `MCPService_InvokeTool_FullMethodName` exported for use in interceptor.
