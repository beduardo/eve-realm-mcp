# Sprint SP-002: Docker image build and minimal MCP Server binary

**Created**: 2026-06-22
**Status**: Specified
**Entities**: 9

---

## Overview

SP-002 completes the MCP Server scaffold by adding health probe endpoints and graceful
signal handling to the existing binary, then packages it as a distroless Docker image
pushed to the local k3d registry. Building on the build pipeline delivered in SP-001,
this sprint proves the end-to-end path from source to a running container: `go build` →
`docker build` → `docker push` → container start → `/healthz` 200. The result is a
deployable, version-tagged image ready to accept K8s liveness and readiness probes.

---

## Entity Inventory

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-007 | requirement | Docker image build and local registry push | No | - |
| REQ-009 | requirement | Minimal MCP Server binary with health probes | No | - |
| SC-007 | scenario | Dockerfile multi-stage build produces runnable distroless image | No | - |
| SC-008 | scenario | Docker build tags image with semantic version | No | - |
| SC-009 | scenario | Docker push delivers image to k3d registry | No | - |
| SC-00F | scenario | Binary compiles and starts with default port | No | - |
| SC-010 | scenario | Health probe endpoints return 200 | No | - |
| SC-011 | scenario | Version endpoint reports ldflags-injected metadata | No | - |
| SC-012 | scenario | Graceful shutdown on SIGINT and SIGTERM | No | - |

---

## Technical Context

### Entity-to-Code Mapping Summary

| File | Action | Entities |
|------|--------|----------|
| `cmd/eve-realm-mcp/main.go` | **Modify** (extend — never recreate) | REQ-009, SC-010, SC-012 |
| `cmd/eve-realm-mcp/main_test.go` | **Modify** (extend) | REQ-009, SC-010, SC-012 |
| `Dockerfile` | **Create** (at repository root) | REQ-007, SC-007, SC-008 |
| `Makefile` | **Modify** (add `docker-build`, `docker-push`, `DOCKER_IMAGE`) | REQ-007, SC-008, SC-009 |
| `go.sum` | **Create** (via `go mod tidy`) | SC-007 precondition |
| `go.mod` | **Possibly modify** (remove SDK require/replace if submodule not initialized) | SC-007 precondition |

### SP-001 Overlap — Already Delivered

Two scenarios are fully covered by SP-001 and require no new code in SP-002:

- **SC-00F** (binary compiles and starts): `--port` flag, default 8080, startup log, `make build` — all present in `cmd/eve-realm-mcp/main.go`.
- **SC-011** (version endpoint ldflags metadata): `VersionHandler()`, `build-prod` ldflags injection, comprehensive test coverage — all present.

These scenarios are **verify-only** in SP-002. Confirm existing `go test ./...` passes; no new production code is needed.

### Implementation Patterns to Follow

**Handler pattern** (`cmd/eve-realm-mcp/main.go`): Named constructor functions returning `http.Handler` — e.g., `HealthzHandler() http.Handler`, `ReadyzHandler() http.Handler`. Each handler sets `Content-Type: application/json`, writes `http.StatusOK`, and encodes a typed response struct. Registered in `main()` via `mux.Handle("/path", HandlerFunc())`.

**Test pattern** (`cmd/eve-realm-mcp/main_test.go`): Unit tests with `httptest.NewServer`, HTTP GET, assert status + `Content-Type` + decoded JSON fields. Table-driven tests use `[]struct{ name string; ... }` with `t.Run`. Integration helpers (`startBinary`, `freePort`, `runMake`) live in `internal/version/integration_test.go`.

**Makefile conventions**: Variables at top via `$(shell ...)`, `lower-hyphen-case` target names, `.PHONY` declaration for all non-file targets. Ldflags pattern: `-X main.Version=$(VERSION) -X main.GitHash=$(GIT_HASH) -X main.BuildDate=$(BUILD_DATE)`.

**Dockerfile pattern** (from eve-cli): `golang:1.25-alpine` builder with `CGO_ENABLED=0`; `gcr.io/distroless/static-debian12:nonroot` runtime. Copy `go.mod`/`go.sum` first, run `go mod download`, then copy source and build.

### Critical Integration Points

- `http.ListenAndServe` must be replaced with `http.Server` + `signal.NotifyContext` + `server.Shutdown(ctx)` for graceful shutdown (AC-8 of REQ-009). This is the only structural change to `main()`.
- SC-007 step 4 calls `GET /healthz` — the health probe endpoint must be present before SC-007 can be validated. REQ-009 implementation (health handlers) must land before SC-007 Docker scenario validation.
- The Dockerfile `COPY go.sum .` and `go mod download` will fail without `go.sum`. Generating `go.sum` via `go mod tidy` is a prerequisite for any Docker build work.

---

## Feasibility Caveats — Mandatory Resolutions

The following five caveats identified in feasibility analysis must be explicitly resolved before implementation proceeds. They are not optional — skipping any one will block the Docker build.

### Caveat 1: go.sum is missing

`go.sum` does not exist on disk. The Dockerfile's `COPY go.sum .` and subsequent `go mod download` will both fail. Resolution: run `go mod tidy` as the first implementation action and commit `go.sum` before writing the Dockerfile or any Docker Makefile targets.

### Caveat 2: eve-realm-sdk submodule is absent

`go.mod` contains `require github.com/beduardo/eve-realm-sdk v0.1.0` and `replace github.com/beduardo/eve-realm-sdk => ./eve-realm-sdk`. No `.gitmodules` file and no `eve-realm-sdk/` directory exist. The Docker builder's `COPY . .` will copy an empty path, causing the module graph resolution to fail. Since `cmd/eve-realm-mcp/main.go` currently has zero SDK imports, the resolution is: **remove the `require` and `replace` directives from `go.mod`** (and re-add them when SDK integration begins). If the team decides to initialize the submodule instead, a minimal `eve-realm-sdk/go.mod` stub (`module github.com/beduardo/eve-realm-sdk`) suffices. Whichever path is taken, `go mod tidy` must succeed cleanly afterwards.

### Caveat 3: REQ-009 before SC-007 ordering

SC-007 step 4 executes `GET /healthz` against the running container. The `/healthz` endpoint does not yet exist (it is AC-4 of REQ-009). Implementation must sequence as follows: (1) implement and test health handlers in `main.go`, (2) only then build and validate the Dockerfile scenario. Plan steps must enforce this ordering.

### Caveat 4: Extend existing main.go — never recreate

SP-001 already delivered ACs 1, 2, 3, 6, and 7 of REQ-009 (`--port` flag, startup log, ldflags vars, `/version` handler). The file `cmd/eve-realm-mcp/main.go` must be **modified in place**. Creating a replacement file would overwrite SP-001's work and break the existing test suite. Only ACs 4, 5, and 8 require new code.

### Caveat 5: SC-00F and SC-011 are verify-only

SC-00F (binary compiles and starts with default port) and SC-011 (version endpoint reports ldflags-injected metadata) are fully satisfied by SP-001's delivery. No new implementation is required. The plan must include a verification step confirming that `go test ./...` continues to pass for these scenarios — but no new production code or test code is written for them.

---

## Implementation Sections

### REQ-007: Docker image build and local registry push

**Entity**: `.software/entities/requirements/REQ-007.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
- `Dockerfile` — Create at repository root. Follows eve-cli two-stage pattern.
- `Makefile` — Modify: add `DOCKER_IMAGE` variable and `docker-build`, `docker-push` targets; extend `.PHONY`.
- `go.sum` — Must exist before Dockerfile can be executed (see Caveat 1).
- `go.mod` — May require modification to remove unused SDK require/replace (see Caveat 2).

**Acceptance Criteria**:

- **AC-1**: Given the repository root, when `Dockerfile` is read, then it contains two named stages: (a) `builder` using `golang:1.25-alpine` that builds with `CGO_ENABLED=0` and ldflags for version injection, and (b) a runtime stage using `gcr.io/distroless/static-debian12:nonroot`.
- **AC-2**: Given the builder stage executes, when the build context is processed, then it copies `go.mod` and `go.sum` first, runs `go mod download`, then copies the full source and builds `./cmd/eve-realm-mcp` to `/out/eve-realm-mcp`.
- **AC-3**: Given the runtime stage executes, when the image is assembled, then the binary is copied to `/usr/local/bin/eve-realm-mcp`, port 8080 is exposed, and the entrypoint is set to `/usr/local/bin/eve-realm-mcp`.
- **AC-4**: Given `make docker-build` is run, when the Makefile target executes, then it produces an image tagged `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:$(VERSION)`. A `:latest` tag may exist locally but must never be referenced in K8s manifests.
- **AC-5**: Given `make docker-push` is run, when the target executes, then only the versioned tag (`k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:$(VERSION)`) is pushed to the k3d registry.
- **AC-6**: Given the Makefile invokes `docker build`, when the command is assembled, then it passes `--build-arg VERSION=$(VERSION)` so the embedded binary version matches the `VERSION` file.

**Implementation Notes**:
Complexity estimate: Size S — approximately 30 LOC for the Dockerfile and 6 LOC for Makefile additions. No Go source changes required for REQ-007 itself.

Key risks:
- The `golang:1.25-alpine` tag may not be available at implementation time; use the nearest available stable Go 1.25.x alpine tag.
- Docker build will fail if `go.sum` is absent (Caveat 1) or if the SDK submodule path cannot be resolved (Caveat 2). Both must be resolved as first-order steps before `docker build` is attempted.

Prerequisites confirmed present from SP-001: `go.mod`, `VERSION` (0.1.0), `Makefile` (build targets), `cmd/eve-realm-mcp/main.go` (binary entrypoint).

**Test Expectations**:
- Must test: `make docker-build` produces an image tagged with the exact value of the `VERSION` file (`k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0` given `VERSION=0.1.0`)
- Must test: The `VERSION` build arg is passed through to the binary — `GET /version` on the running container returns `{"version":"0.1.0",...}`
- Must test: The runtime image contains only the binary at `/usr/local/bin/eve-realm-mcp` and no Go toolchain artifacts
- Must test: `make docker-push` pushes only the versioned tag (not `:latest`) — registry tags list includes `0.1.0`
- Must NOT rely on: Live k3d registry availability for unit-level image build assertions; use `docker inspect` or `docker run` locally before push tests

**Verify Expectations** (REQ-003 cluster surface):
Health endpoint `/healthz` is now exposed inside the container image. Once the image is deployed to the k3d cluster, a `health` category check must verify that `GET http://eve-realm-mcp.eve-realm.svc.cluster.local:8080/healthz` returns HTTP 200. This check is not part of this sprint's implementation (the K8s deployment manifests are in REQ-008, a future sprint), but the spec records the expectation so the plan for the deployment sprint can include it.

---

### REQ-009: Minimal MCP Server binary with health probes

**Entity**: `.software/entities/requirements/REQ-009.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
- `cmd/eve-realm-mcp/main.go` — **Extend only** (never recreate). Add `HealthzHandler()`, `ReadyzHandler()`, signal handling loop. ACs 1, 2, 3, 6, 7 already implemented by SP-001.
- `cmd/eve-realm-mcp/main_test.go` — Extend: add `TestHealthzHandler`, `TestReadyzHandler`, and graceful shutdown test.

**Acceptance Criteria**:

- **AC-1** (SP-001 — verify only): Given `cmd/eve-realm-mcp/main.go` exists, when `go build ./cmd/eve-realm-mcp` is run, then it compiles to a standalone binary without errors.
- **AC-2** (SP-001 — verify only): Given the binary is run without flags, when it starts, then it listens on port 8080 by default; given it is run with `--port 9090`, when it starts, then it listens on port 9090.
- **AC-3** (SP-001 — verify only): Given the binary was built without ldflags, when it starts, then the startup log reads `eve-realm-mcp online (vdev, unknown, unknown)`; given it was built with ldflags, when it starts, then the log includes the injected version, hash, and date.
- **AC-4** (new): Given the binary is running, when `GET /healthz` is sent, then it returns HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- **AC-5** (new): Given the binary is running, when `GET /readyz` is sent, then it returns HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- **AC-6** (SP-001 — verify only): Given the binary is inspected, when package-level variable types are checked, then `Version`, `GitHash`, and `BuildDate` are `string` type and are populated by `-ldflags "-X main.Version=..."` at build time.
- **AC-7** (SP-001 — verify only): Given the binary is running, when `GET /version` is sent, then it returns HTTP 200 with body `{"version":"X.Y.Z","git_hash":"...","build_date":"..."}` and header `Content-Type: application/json`.
- **AC-8** (new): Given the binary is running, when it receives `SIGINT` or `SIGTERM`, then it logs `eve-realm-mcp shutting down`, stops accepting new connections via `http.Server.Shutdown(ctx)`, and exits with code 0 without panicking.

**Implementation Notes**:
SP-001 delivered approximately 60% of this requirement. Only ACs 4, 5, and 8 require new code — approximately 40-50 LOC in production and 80 LOC in tests.

The most structurally significant change is AC-8: `http.ListenAndServe` must be replaced with an explicit `http.Server` struct combined with `signal.NotifyContext` (or `signal.Notify`) and `server.Shutdown(ctx)`. This is an in-place refactor of the `main()` function bottom half.

`HealthzHandler()` and `ReadyzHandler()` follow the identical constructor pattern as the existing `VersionHandler()` — approximately 10 LOC each.

The shutdown test is the hardest test in this sprint. The recommended approach uses `context.WithTimeout` and `httptest.NewServer` to start the server, send a shutdown signal via channel, and confirm the server stops accepting connections. No real OS signal is required in the unit test.

**Test Expectations**:
- Must test: `GET /healthz` returns status 200, `Content-Type: application/json`, and body `{"status":"ok"}` (unit test with `httptest.NewServer`)
- Must test: `GET /readyz` returns status 200, `Content-Type: application/json`, and body `{"status":"ok"}` (unit test with `httptest.NewServer`)
- Must test: on receiving shutdown signal, the server logs `eve-realm-mcp shutting down` and the HTTP server stops accepting new connections
- Must test: graceful shutdown completes without panic and exits cleanly
- Must NOT rely on: real OS signals (`syscall.SIGINT`) in unit tests — use channel or context cancellation to trigger shutdown path in the test

**Verify Expectations** (REQ-003 cluster surface):
`/healthz` and `/readyz` are the endpoints that K8s liveness and readiness probes will target once the deployment manifest (REQ-008, future sprint) is applied. A `health` category check function must be written when K8s manifests are added. This sprint records the expectation; the check itself is deferred to the deployment sprint.

---

### SC-007: Dockerfile multi-stage build produces runnable distroless image

**Entity**: `.software/entities/scenarios/SC-007.md`
**Type**: scenario

**Codebase Mapping**:
- `Dockerfile` — New file; this scenario validates it directly.
- `go.sum` — Precondition: must exist before `docker build` is attempted.

**Acceptance Criteria**:

- **AC-1**: Given a `Dockerfile` at the repository root with `go.mod` and `go.sum` present, when `docker build --build-arg VERSION=0.1.0 -t eve-realm-mcp:test .` is run, then the build completes with two named stages: `builder` (golang:1.25-alpine) and runtime (gcr.io/distroless/static-debian12:nonroot).
- **AC-2**: Given the builder stage, when it executes, then it copies `go.mod`/`go.sum` first, runs `go mod download`, then copies source and builds with `CGO_ENABLED=0`.
- **AC-3**: Given the runtime image, when it is inspected, then it contains only `/usr/local/bin/eve-realm-mcp` — no Go toolchain and no source code.
- **AC-4**: Given the runtime image, when inspected, then port 8080 is declared as `EXPOSE` and the entrypoint is `/usr/local/bin/eve-realm-mcp`.
- **AC-5**: Given the container is run with `-p 8080:8080`, when `GET http://localhost:8080/healthz` is sent, then the container returns HTTP 200.

> **Dependency note**: AC-5 requires `/healthz` to be present. This scenario cannot be fully validated until REQ-009 ACs 4 and 5 are implemented. REQ-009 must be implemented first.

**Implementation Notes**:
This scenario is entirely new — no existing Dockerfile. Two preconditions are blockers (see Caveats 1 and 2). Once `go.sum` is generated and the `go.mod` SDK directives are resolved, the Dockerfile itself is straightforward (~30 LOC). Validate AC-5 only after REQ-009 health handlers are in place.

**Test Expectations**:
- Must test: `docker build` succeeds from repository root with `--build-arg VERSION=0.1.0`
- Must test: `docker history` confirms two build stages
- Must test: `docker run --rm -p 8080:8080 eve-realm-mcp:test` starts and `GET /healthz` returns 200
- Must NOT rely on: k3d registry connectivity for this scenario — use a local image tag (`eve-realm-mcp:test`)

---

### SC-008: Docker build tags image with semantic version

**Entity**: `.software/entities/scenarios/SC-008.md`
**Type**: scenario

**Codebase Mapping**:
- `Makefile` — Modify: `docker-build` target must pass `--build-arg VERSION=$(VERSION)` and tag as `$(DOCKER_IMAGE):$(VERSION)`.
- `DOCKER_IMAGE` variable — Add to Makefile with default `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`.

**Acceptance Criteria**:

- **AC-1**: Given `VERSION` file contains `0.1.0`, when `make docker-build` runs, then a local image tagged `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0` exists.
- **AC-2**: Given the Makefile `docker-build` target, when inspected, then it passes `--build-arg VERSION=$(VERSION)` to `docker build`.
- **AC-3**: Given the image built by `make docker-build`, when the running container's `GET /version` is called, then the response body contains `{"version":"0.1.0",...}`, confirming the build arg propagated through ldflags into the binary.
- **AC-4**: Given a `:latest` tag may exist locally, when K8s deployment manifests are examined, then they reference only the versioned tag — never `:latest`.

**Implementation Notes**:
This scenario validates the `docker-build` Makefile target. It is a direct consequence of REQ-007 AC-4. No Go source changes needed. The `DOCKER_IMAGE` Makefile variable should follow the naming convention established by existing variables at the top of the Makefile.

**Test Expectations**:
- Must test: `make docker-build` exits 0 and produces an image with the tag matching `$(DOCKER_IMAGE):$(VERSION)`
- Must test: the `VERSION` build arg is propagated — `GET /version` on the container returns `"version":"0.1.0"`
- Must NOT rely on: K8s manifest state in this scenario — the `:latest` prohibition is validated by manifest inspection, not by this scenario's Docker commands

---

### SC-009: Docker push delivers image to k3d registry

**Entity**: `.software/entities/scenarios/SC-009.md`
**Type**: scenario

**Codebase Mapping**:
- `Makefile` — Modify: add `docker-push` target that pushes `$(DOCKER_IMAGE):$(VERSION)`.

**Acceptance Criteria**:

- **AC-1**: Given a k3d cluster running with a registry at `k3d-eve-realm-registry.localhost:5100` and `make docker-build` has been run, when `make docker-push` runs, then the versioned tag `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0` is pushed successfully.
- **AC-2**: Given the push completes, when `curl -s http://k3d-eve-realm-registry.localhost:5100/v2/_catalog` is queried, then `eve-realm-mcp` appears in the catalog.
- **AC-3**: Given the push completes, when `curl -s http://k3d-eve-realm-registry.localhost:5100/v2/eve-realm-mcp/tags/list` is queried, then `0.1.0` appears in the tags list.
- **AC-4**: Given `make docker-push`, when inspected, then it pushes only the versioned tag — not `:latest`.

**Implementation Notes**:
This scenario requires a live k3d cluster with the registry running. It cannot be validated in a pure unit test context. The `docker-push` target is approximately 2 LOC in the Makefile. Prerequisite: `make docker-build` must succeed first (which itself depends on Caveats 1 and 2 being resolved).

**Test Expectations**:
- Must test: `make docker-push` succeeds when the k3d registry is reachable and the versioned image exists locally
- Must test: registry API confirms the pushed tag via `v2/eve-realm-mcp/tags/list`
- Must NOT rely on: public Docker Hub or any external registry — all push operations target `k3d-eve-realm-registry.localhost:5100` exclusively

---

### SC-00F: Binary compiles and starts with default port

**Entity**: `.software/entities/scenarios/SC-00F.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/eve-realm-mcp/main.go` — Already present from SP-001. No modifications needed for this scenario.
- `cmd/eve-realm-mcp/main_test.go` — Already covers this scenario. No new tests needed.

> **VERIFY ONLY**: This scenario is fully satisfied by SP-001. No new production code or test code is required. Confirm `go test ./...` passes and the binary starts on port 8080 without flags.

**Acceptance Criteria**:

- **AC-1**: Given `cmd/eve-realm-mcp/main.go` exists, when `go build -o dist/eve-realm-mcp ./cmd/eve-realm-mcp` is run without ldflags, then the binary compiles without errors.
- **AC-2**: Given the compiled binary is run without flags, when it starts, then it listens on port 8080 and logs `eve-realm-mcp online (vdev, unknown, unknown)`.
- **AC-3**: Given the compiled binary is run with `--port 9090`, when it starts, then it listens on port 9090 instead of 8080.

**Implementation Notes**:
SP-001 fully implemented this scenario. Feasibility confirmed no blockers. The only action required in SP-002 is to run the existing test suite and confirm these cases still pass after the SP-002 changes to `main.go` (health handlers + shutdown refactor).

---

### SC-010: Health probe endpoints return 200

**Entity**: `.software/entities/scenarios/SC-010.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/eve-realm-mcp/main.go` — Modify: add `HealthzHandler()` and `ReadyzHandler()` constructor functions; register `/healthz` and `/readyz` in `main()`.
- `cmd/eve-realm-mcp/main_test.go` — Extend: add `TestHealthzHandler` and `TestReadyzHandler`.

**Acceptance Criteria**:

- **AC-1**: Given the MCP Server binary is running, when `GET http://localhost:8080/healthz` is sent, then the response is HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- **AC-2**: Given the MCP Server binary is running, when `GET http://localhost:8080/readyz` is sent, then the response is HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- **AC-3**: Given both endpoints, when called, then they respond within milliseconds with no external dependencies.

**Implementation Notes**:
New handlers follow the existing `VersionHandler()` constructor pattern. Each is approximately 10 LOC. Tests follow the `httptest.NewServer` pattern already established in `main_test.go`. This scenario must be implemented and passing before SC-007 is validated (see Caveat 3).

**Test Expectations**:
- Must test: `GET /healthz` on `httptest.NewServer` returns status 200, `Content-Type: application/json`, decoded body `{"status":"ok"}`
- Must test: `GET /readyz` on `httptest.NewServer` returns status 200, `Content-Type: application/json`, decoded body `{"status":"ok"}`
- Must test: table-driven test covering both `/healthz` and `/readyz` paths in a single `TestHealthProbes` function (or separate `TestHealthzHandler` / `TestReadyzHandler` functions per naming convention)
- Must NOT rely on: external service availability or global state — both handlers are pure scaffold responses with no dependencies

---

### SC-011: Version endpoint reports ldflags-injected metadata

**Entity**: `.software/entities/scenarios/SC-011.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/eve-realm-mcp/main.go` — Already present from SP-001. No modifications needed.
- `cmd/eve-realm-mcp/main_test.go` — Already covers this scenario. No new tests needed.

> **VERIFY ONLY**: This scenario is fully satisfied by SP-001. `VersionHandler()` is implemented, `build-prod` injects all three ldflags, and comprehensive test coverage exists. No new production code or test code is required.

**Acceptance Criteria**:

- **AC-1**: Given the binary was built with `make build-prod`, when `GET /version` is sent to the running binary, then the response is HTTP 200 with `Content-Type: application/json`.
- **AC-2**: Given the VERSION file contains `0.1.0`, when `GET /version` is called on a binary built with `make build-prod`, then the body is `{"version":"0.1.0","git_hash":"<7-char-hash>","build_date":"<YYYY-MM-DD>"}`.
- **AC-3**: Given the binary was built without ldflags, when `GET /version` is called, then `version` returns `dev`, `git_hash` returns `unknown`, and `build_date` returns `unknown`.

**Implementation Notes**:
SP-001 fully implemented this scenario. Existing tests cover all three acceptance criteria. After SP-002 changes to `main.go` (health handlers + shutdown), confirm the version handler tests still pass without modification.

---

### SC-012: Graceful shutdown on SIGINT and SIGTERM

**Entity**: `.software/entities/scenarios/SC-012.md`
**Type**: scenario

**Codebase Mapping**:
- `cmd/eve-realm-mcp/main.go` — Modify: replace `http.ListenAndServe` with `http.Server` struct + `signal.NotifyContext` (or `signal.Notify`) + `server.Shutdown(ctx)`.
- `cmd/eve-realm-mcp/main_test.go` — Extend: add graceful shutdown test (~30 LOC).

**Acceptance Criteria**:

- **AC-1**: Given the MCP Server binary is running, when it receives `SIGINT`, then it logs `eve-realm-mcp shutting down` to stdout and exits with code 0.
- **AC-2**: Given the MCP Server binary is running, when it receives `SIGTERM`, then it logs `eve-realm-mcp shutting down` to stdout and exits with code 0.
- **AC-3**: Given a shutdown signal is received, when shutdown executes, then the HTTP server stops accepting new connections before the process exits (no panic, no unclean exit).

**Implementation Notes**:
This is the most structurally significant change in SP-002. The `main()` function bottom half must be refactored from `http.ListenAndServe` (blocking call) to an `http.Server` struct with a context that is cancelled on signal receipt, followed by `server.Shutdown(ctx)`. The standard Go pattern uses `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.

The shutdown test does not need to send real OS signals. The recommended test approach: start the server in a goroutine, call `server.Shutdown(context.Background())` directly after a brief delay, and assert that the server goroutine unblocks and the log message is emitted.

**Test Expectations**:
- Must test: when `server.Shutdown` is called, the HTTP server stops accepting new connections and the shutdown log message `eve-realm-mcp shutting down` is emitted
- Must test: after shutdown, `http.Server.ListenAndServe` returns `http.ErrServerClosed` (the expected non-error sentinel on clean shutdown)
- Must test: exit is clean — no panic, goroutine leak, or error propagation
- Must NOT rely on: real `syscall.SIGINT` or `syscall.SIGTERM` in unit tests — trigger shutdown via context cancellation or direct `server.Shutdown` call to avoid OS-level signal delivery complexity in test environments

---

## Documentation Tasks

### RELEASES.md Entry

**Required**: Always

Append an entry to `RELEASES.md` documenting:
- Sprint ID: SP-002
- Sprint title: Docker image build and minimal MCP Server binary
- Summary: Added health probe endpoints (`/healthz`, `/readyz`) and graceful SIGINT/SIGTERM shutdown to the MCP Server binary. Added a two-stage Dockerfile (golang:1.25-alpine builder + distroless runtime) and Makefile targets `docker-build` and `docker-push` for building and pushing versioned images to the k3d registry.
- Entity IDs: REQ-007, REQ-009, SC-007, SC-008, SC-009, SC-00F, SC-010, SC-011, SC-012
- Date of completion: to be filled at release time

Do not read or modify existing entries. Append only.

### README.md Update

**Required**: User-facing changes detected

Update `README.md` to reflect the following new capabilities introduced in SP-002:

- **New Makefile targets**: `docker-build` (builds versioned Docker image) and `docker-push` (pushes to k3d registry at `k3d-eve-realm-registry.localhost:5100`).
- **New HTTP endpoints**: `GET /healthz` and `GET /readyz` — both return `{"status":"ok"}` with HTTP 200. These are the K8s liveness and readiness probe targets.
- **Graceful shutdown**: The binary now handles `SIGINT` and `SIGTERM`, logging `eve-realm-mcp shutting down` before a clean exit.
- **Docker image**: `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:<VERSION>` — versioned tag, distroless runtime, entrypoint at `/usr/local/bin/eve-realm-mcp`.

---

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog | Spec writer must load all four triggered cross-cutting requirements (REQ-001, REQ-002, REQ-003, REQ-004) via `eve_software_show` when their trigger conditions match the sprint scope | All four were loaded. REQ-001 (TDD): Test Expectations subsections generated for all REQ and SC entities with new code. REQ-002 (release process): RELEASES.md and README.md tasks included; version increment type (patch) and README update flag recorded. REQ-003 (cluster testing): Verify Expectations subsections added to REQ-007 and REQ-009 noting health endpoint cluster checks deferred to REQ-008 (deployment sprint). REQ-004 (k3d topology): Image pattern `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION`, registry, namespace `eve-realm`, and deployment order noted throughout. |
| REQ-001: Test-Driven Development Strategy | Spec writer generates a "Test Expectations" subsection for each requirement specifying what tests must exist before the requirement is considered met | Test Expectations subsections generated for REQ-007, REQ-009, SC-007, SC-008, SC-009, SC-010, SC-012. SC-00F and SC-011 are verify-only and correctly excluded from new test expectations. ADR-type entities not present. |
| REQ-002: Sprint completion and release process | Spec-time decisions: record version increment type (major/minor/patch) and whether README.md needs updating | Version increment: **patch** (0.1.0 → 0.1.1). README update: **required** (new endpoints `/healthz`, `/readyz`; new Makefile targets `docker-build`, `docker-push`; graceful shutdown behavior). Both recorded. |
| REQ-003: Cluster integration testing policy | Spec writer generates a "Verify Expectations" subsection listing the cluster surfaces affected by the sprint | Verify Expectations included in REQ-007 (health endpoint surface) and REQ-009 (health + readyz probe surface). Cluster check implementation deferred to REQ-008 (K8s deployment manifests sprint) as the cluster surface is not yet deployed. |
| REQ-004: Local k3d cluster topology reference | Spec writer references the service topology table when specifying services, ports, or inter-service communication | Registry URL `k3d-eve-realm-registry.localhost:5100`, namespace `eve-realm`, image pattern `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION`, port 8080, and deployment order (infra → plugins → MCP Server) referenced consistently throughout the spec. |

---

## Out of Scope

- K8s deployment manifests (`deploy/k8s/`) — covered by REQ-008 in a future sprint.
- NATS plugin discovery, gRPC proxy, agent runtime — no business logic in this sprint; the binary remains a pure scaffold.
- Cluster integration test check functions for `/healthz` and `/readyz` — deferred to the deployment sprint when the cluster topology is finalized.
- MCP protocol handling (stdio/HTTP+SSE transports) — future sprint.
- `:latest` tag push to k3d registry — versioned tags only per REQ-007 AC-4.
- SDK submodule initialization beyond what is required to unblock `go mod tidy` — full submodule wiring deferred to when SDK imports are introduced.
- `make verify-cluster` target — the verification binary location is TBD per REQ-003 notes; check registration is deferred.

---

## Prerequisites

- `go.sum` must be generated via `go mod tidy` before any Docker build work begins. This is the first implementation action.
- The `eve-realm-sdk` `require`/`replace` directives in `go.mod` must be resolved (either removed since there are zero SDK imports currently, or the submodule initialized with a minimal stub) before `go mod tidy` can complete and before Docker build can run.
- REQ-009 health handlers (`/healthz`, `/readyz`) must be implemented and tested before SC-007 Docker scenario validation is attempted (SC-007 step 4 calls `GET /healthz`).
- SP-001 artifacts must be on the `main` branch: `cmd/eve-realm-mcp/main.go`, `cmd/eve-realm-mcp/main_test.go`, `Makefile`, `go.mod`, `VERSION`. All confirmed present per feasibility reports.
- Docker daemon must be running locally for SC-007, SC-008, SC-009 validation.
- k3d cluster with registry at `k3d-eve-realm-registry.localhost:5100` must be running for SC-009 validation.
