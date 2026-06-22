# Spec Writer Brief

**Sprint**: SP-002
**Sprint Title**: Docker image build and minimal MCP Server binary
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Sprint Folder**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-002
**Date**: 2026-06-22

## Entity List

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

## Analysis Artifacts

- Codebase Analysis: .software/sprints/SP-002/analysis/codebase-analysis.md
- Feasibility Reports:
  - REQ-007: .software/sprints/SP-002/analysis/feasibility-REQ-007.md
  - REQ-009: .software/sprints/SP-002/analysis/feasibility-REQ-009.md

## Feasibility Caveats That MUST Be Addressed in SPEC.md

The feasibility analysis identified caveats that the spec must resolve. These are NOT optional — they must be explicitly addressed in the specification:

### Caveat 1: go.sum is missing
`go.sum` does not exist on disk. The Dockerfile's `COPY go.sum .` and `go mod download` will fail without it. The spec must include an early step to generate `go.sum` via `go mod tidy` before any Docker build work.

### Caveat 2: eve-realm-sdk submodule is absent
`go.mod` has `replace github.com/beduardo/eve-realm-sdk => ./eve-realm-sdk` but no `.gitmodules` or `eve-realm-sdk/` directory exists. The Docker builder's `COPY . .` will fail. The spec must resolve this — either initialize the submodule or clean up go.mod (remove unused require/replace since main.go has zero SDK imports).

### Caveat 3: SC-007 depends on REQ-009's /healthz endpoint
SC-007 step 4 calls `GET /healthz`, which doesn't exist yet. Implementation ordering must ensure REQ-009 health probes land BEFORE SC-007 Docker scenario validation.

### Caveat 4: SP-001 already delivered ~60% of REQ-009
`cmd/eve-realm-mcp/main.go` already has: --port flag, startup log, /version handler, ldflags vars (ACs 1, 2, 3, 6, 7). The spec MUST extend the existing file, NOT recreate it. Only ACs 4, 5, 8 need new code.

### Caveat 5: SC-00F and SC-011 are already satisfied by SP-001
These scenarios are fully covered by existing code and tests. The spec should mark them as "verify only" — no new implementation, just confirm existing tests pass.

## SP-001 Overlap Summary

SP-001 delivered these artifacts that SP-002 builds on:
- `cmd/eve-realm-mcp/main.go` — binary with --port, startup log, /version handler, ldflags vars
- `cmd/eve-realm-mcp/main_test.go` — 252-line test suite with httptest patterns
- `Makefile` — build/test/bump targets, ldflags injection via build-prod
- `go.mod` — module definition with SDK replace directive
- `VERSION` — 0.1.0
- `internal/version/` — bump logic and integration tests

## Existing Code Patterns (from codebase analysis)

### Handler Pattern
Named constructor functions returning `http.Handler` (e.g., `VersionHandler() http.Handler`). Sets Content-Type, writes StatusOK, encodes typed response struct. New handlers: `HealthzHandler()`, `ReadyzHandler()`.

### Test Pattern
- Unit tests with `httptest.NewServer` — create server, HTTP GET, assert status + Content-Type + JSON body
- Table-driven tests with `t.Run` subtests
- Integration tests with `startBinary`, `freePort`, `runMake` helpers

### Makefile Conventions
- Variables at top, `$(shell ...)` for dynamic values
- Target naming: `lower-hyphen-case`
- `.PHONY` declaration for all targets

### Ldflags Pattern
`-X main.Version=$(VERSION) -X main.GitHash=$(GIT_HASH) -X main.BuildDate=$(BUILD_DATE)`

## Project Context

EVE Realm MCP — Go project. 28 total entities: 9 requirements, 18 scenarios, 1 change. Sprint SP-001 (build pipeline) completed. SP-002 adds Docker packaging and completes the MCP Server scaffold with health probes.

See CLAUDE.md at project root for full conventions (Go backend, testing patterns, K8s deployment, sprint workflow critic policy).

## Pinned Entities

### REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection

This is the single pinned entity. It contains the cross-cutting requirements registry:

| ID | Title | Trigger condition |
|----|-------|-------------------|
| REQ-001 | Test-Driven Development Strategy | **Implementing or modifying Go code** |
| REQ-002 | Sprint completion and release process | **Completing a sprint and preparing a release** |
| REQ-003 | Cluster integration testing policy | **Modifying K8s manifests, ConfigMap entries, inter-pod communication, health endpoints** |
| REQ-004 | Local k3d cluster topology reference | **Adding, modifying, or verifying K8s deployments, services, ConfigMaps** |

All four triggers are relevant to SP-002:
- REQ-001: SP-002 implements Go code (health probes, shutdown)
- REQ-002: SP-002 will complete with a release
- REQ-003: SP-002 adds health endpoints that K8s probes will use
- REQ-004: SP-002 adds Docker image to the k3d cluster topology

The spec writer MUST load each of these via `eve_software_show` and incorporate their directives.

## Flags

- readme_update_needed: true (new Makefile targets docker-build/docker-push, new HTTP endpoints /healthz /readyz)
