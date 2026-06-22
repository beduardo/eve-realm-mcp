# Codebase Analysis: SP-002

## Entity-to-Code Mapping

### REQ-007: Docker image build and local registry push
- Existing files: `Makefile` (has `build-prod` with ldflags pattern, `VERSION` var, `GIT_HASH` var, `BUILD_DATE` var)
- New files needed: `Dockerfile` (at repository root), `go.sum` (required by Dockerfile's `go mod download`)
- Modifications needed: `Makefile` — add `docker-build` and `docker-push` targets; add `DOCKER_IMAGE` variable; add both targets to `.PHONY`

### REQ-009: Minimal MCP Server binary with health probes
- Existing files: `cmd/eve-realm-mcp/main.go` (SP-001: `--port`, startup log, `/version`, ldflags), `cmd/eve-realm-mcp/main_test.go`
- New files needed: none
- Modifications needed: `main.go` — add `/healthz`, `/readyz` handlers + graceful shutdown; `main_test.go` — add tests for new handlers and shutdown

### SC-007: Dockerfile multi-stage build produces runnable distroless image
- New files needed: `Dockerfile`, `go.sum`
- Modifications needed: none — purely a new artifact

### SC-008: Docker build tags image with semantic version
- Modifications needed: `Makefile` — `docker-build` target must tag as `$(DOCKER_IMAGE):$(VERSION)`

### SC-009: Docker push delivers image to k3d registry
- Modifications needed: `Makefile` — add `docker-push` target

### SC-00F: Binary compiles and starts with default port
- **Already fully satisfied by SP-001** — verify only

### SC-010: Health probe endpoints return 200
- Modifications needed: `main.go` — register `/healthz` and `/readyz`; `main_test.go` — add tests

### SC-011: Version endpoint reports ldflags-injected metadata
- **Already fully satisfied by SP-001** — verify only

### SC-012: Graceful shutdown on SIGINT and SIGTERM
- Modifications needed: `main.go` — replace `http.ListenAndServe` with `http.Server` + signal handling; `main_test.go` — add shutdown test

## Existing Patterns

### Handler Pattern
`cmd/eve-realm-mcp/main.go` defines handlers as named constructor functions returning `http.Handler` (e.g., `VersionHandler() http.Handler`). Handler sets `Content-Type: application/json`, writes `http.StatusOK`, encodes a typed response struct. New handlers must follow: `HealthzHandler() http.Handler` and `ReadyzHandler() http.Handler`.

Registration in `main()` via `mux.Handle("/path", HandlerFunc())`.

### Test Pattern
Two patterns in `main_test.go`:
1. **Unit tests with `httptest.NewServer`**: Create test server, make HTTP GET, assert status code, Content-Type, and decoded JSON fields.
2. **Table-driven tests**: `[]struct{ name, got, expected string }` with `t.Run` subtests.

Integration tests in `internal/version/integration_test.go`: `projectRoot(t)`, `freePort(t)`, `startBinary(t, path, port)`, `runMake(t, root, target)` helpers.

### Makefile Conventions
- Variables at top: `VERSION`, `GIT_HASH`, `BUILD_DATE`, `BINARY`, `MAIN_PKG`, `VERSION_FILE`
- Shell expansion via `$(shell ...)`
- Target naming: `lower-hyphen-case`
- `.PHONY` declaration for all non-file targets

### Ldflags Pattern
`go build -ldflags "-X main.Version=$(VERSION) -X main.GitHash=$(GIT_HASH) -X main.BuildDate=$(BUILD_DATE)"`. Dockerfile builder stage must replicate this.

## Implementation Gaps

| Gap | Severity | Resolution |
|-----|----------|-----------|
| `go.sum` missing | Blocker for Docker | Run `go mod tidy` and commit |
| `eve-realm-sdk/` submodule absent | Blocker for Docker | Initialize submodule or remove unused require/replace from go.mod |
| `Dockerfile` absent | Blocker for SC-007/008/009 | Create at repository root |
| `docker-build` target absent | Blocker for SC-008 | Add to Makefile |
| `docker-push` target absent | Blocker for SC-009 | Add to Makefile |
| `/healthz` and `/readyz` absent | Fail for SC-010 | Add to main.go |
| Graceful shutdown absent | Fail for SC-012 | Refactor main() to http.Server + signal |
| Tests for new handlers absent | TDD policy | Add before implementation per REQ-001 |

## Files Summary

| File | Action | Entity Coverage |
|------|--------|----------------|
| `cmd/eve-realm-mcp/main.go` | Modify | REQ-009, SC-010, SC-00F (verify), SC-012 |
| `cmd/eve-realm-mcp/main_test.go` | Modify | REQ-009, SC-010, SC-012 |
| `Dockerfile` | Create | REQ-007, SC-007, SC-008 |
| `Makefile` | Modify | REQ-007, SC-008, SC-009 |
| `go.sum` | Create | SC-007 (precondition) |
| `go.mod` | Possibly modify | SC-007 (unblock Docker if submodule not initialized) |
| `internal/version/integration_test.go` | Possibly extend | REQ-009, SC-00F, SC-010 |

## SP-001 Overlap

Two scenarios fully satisfied by SP-001 — verify only, no new code:
- **SC-00F**: Binary compiles and starts with default port — `--port` flag (default 8080), startup log, `make build` all delivered
- **SC-011**: Version endpoint reports ldflags-injected metadata — `VersionHandler()` fully implemented, `build-prod` injects ldflags, comprehensive test coverage exists
