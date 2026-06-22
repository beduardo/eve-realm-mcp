# Feasibility Report: REQ-009

## Recommendation

PROCEED-WITH-CAVEATS

SP-001 has already delivered roughly 60% of REQ-009's acceptance criteria. The remaining work is well-scoped: two health probe handlers, graceful signal handling with `http.Server.Shutdown`, and corresponding test coverage. There are no blockers, but the spec must acknowledge existing implementation to avoid redundant work and test drift.

## Summary

`cmd/eve-realm-mcp/main.go` exists and is fully functional for ACs 1, 2, 3, 6, and 7 (binary, --port flag, startup log, ldflags vars, /version endpoint). The Makefile already wires ldflags correctly. Only ACs 4, 5, and 8 require new code: `/healthz`, `/readyz`, and SIGINT/SIGTERM graceful shutdown. The existing test suite covers SP-001 scope thoroughly but has no tests for the three missing ACs; those tests must be added. No architectural conflicts exist — the additions slot cleanly into the existing mux and main function pattern.

## Prerequisites

- SP-001 merged to main — confirmed by presence of `cmd/eve-realm-mcp/main.go`
- Makefile `build-prod` target already injects all three ldflags vars
- `go.mod` declares `go 1.25` with only the SDK replace directive — no third-party dependencies needed

## Blockers

None.

## Caveats

- The spec must treat `main.go` as an existing file to be extended, not created. Creating from scratch would overwrite SP-001's work.
- SC-00F and SC-011 are already fully satisfiable by the current codebase. The sprint spec should acknowledge this explicitly — verify, not re-implement.
- `http.ListenAndServe` must be replaced with `http.Server` + `Shutdown(ctx)` for graceful shutdown (AC-8). This is the most structurally significant change.
- SC-012 graceful shutdown test is harder to write than handler tests — standard approach uses `context.WithTimeout` + `httptest`.
- Existing test patterns extend naturally to new handlers — no refactoring needed, only additions.

## Findings

### AC Coverage by SP-001 Delivery

| AC | Description | Covered by SP-001? |
|----|-------------|-------------------|
| 1 | `cmd/eve-realm-mcp/main.go` exists and compiles | Yes |
| 2 | `--port` flag with default 8080 | Yes |
| 3 | Startup log with version triplet | Yes |
| 4 | `GET /healthz` returns 200 + JSON | No — handler missing |
| 5 | `GET /readyz` returns 200 + JSON | No — handler missing |
| 6 | `Version`, `GitHash`, `BuildDate` package vars via ldflags | Yes |
| 7 | `GET /version` returns 200 + JSON | Yes |
| 8 | SIGINT/SIGTERM graceful shutdown | No — `ListenAndServe` blocks; no signal handling |

### New Code Required (ACs 4, 5, 8)

1. `HealthzHandler() http.Handler` — ~10 LOC
2. `ReadyzHandler() http.Handler` — ~10 LOC
3. Signal handling in `main()` — replace `http.ListenAndServe` with `http.Server` + `signal.NotifyContext` + `server.Shutdown(ctx)` — ~20 LOC

Total new production code: ~40-50 LOC in one file.

### Test Suite Gap

Tests needed: `TestHealthzHandler` (~25 LOC), `TestReadyzHandler` (~25 LOC), graceful shutdown test (~30 LOC). ~80 LOC total. No existing tests need modification.

## SP-001 Overlap Analysis

SP-001 delivered:
- `cmd/eve-realm-mcp/main.go` — complete with `--port`, startup log, `/version` handler, ldflags vars (ACs 1, 2, 3, 6, 7)
- `cmd/eve-realm-mcp/main_test.go` — 252-line test file covering all SP-001 ACs
- `Makefile` — `build-prod` target with ldflags injection

What SP-002 must add: `/healthz` handler, `/readyz` handler, signal handling loop, and corresponding tests. Complexity: **S** (<100 LOC new production code).
