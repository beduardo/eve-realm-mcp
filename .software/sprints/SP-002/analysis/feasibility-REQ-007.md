# Feasibility Report: REQ-007

## Recommendation

**PROCEED-WITH-CAVEATS**

REQ-007 is well-scoped and the SP-001 delivery provides everything the Dockerfile needs from the project side (binary entrypoint, ldflags wiring, VERSION file, go.mod). Two caveats must be resolved before `docker build` can succeed: the `eve-realm-sdk` submodule does not exist on disk, and `go.sum` is absent — both required by the Go module system during the builder stage.

## Prerequisites

- REQ-006 (Build pipeline with semantic versioning) — completed in SP-001
- `cmd/eve-realm-mcp/main.go` compiles — present
- `go.mod` with module and replace directive — present
- `VERSION` file — present (0.1.0)
- `dist/eve-realm-mcp` binary — present

## Blockers

| Blocker | Severity | Resolution Path |
|---------|----------|-----------------|
| `go.sum` is absent | Critical | Run `go mod tidy` and commit `go.sum`. Required by Dockerfile COPY before `go mod download`. |
| `eve-realm-sdk` submodule missing from disk | Critical | Either initialize the submodule or create a minimal `eve-realm-sdk/go.mod` stub to satisfy the replace directive. |
| `Makefile` has no `docker-build` / `docker-push` targets | Major | Deliverable for this sprint. |

## Caveats

- SC-007's expected result includes `GET /healthz` returning HTTP 200. That endpoint is not in `main.go` as delivered by SP-001. REQ-009 (also in SP-002) adds health probes. SC-007 cannot be fully validated until REQ-009 is also implemented.
- The AC specifies `golang:1.25-alpine` as the builder base image. If the tag is not available in Docker Hub at implementation time, use the nearest available stable tag.
- AC-4 allows a `:latest` tag locally but forbids it in K8s manifests. The `docker-push` target should push only the versioned tag.

## Findings

### SP-001 Artifacts That REQ-007 Builds On

SP-001 delivered: `go.mod` (module + replace directive), `VERSION` (0.1.0), `Makefile` (build/test/bump targets), `cmd/eve-realm-mcp/main.go` (binary with ldflags injection), `dist/eve-realm-mcp` (compiled binary).

### `go.sum` Is Absent — Root Cause

The SP-001 implementation created `go.mod` with a `require` and `replace` directive for the SDK. Since the SDK submodule does not exist on disk, `go mod tidy` could not have been run. The current source has zero SDK imports, so `go build` works on the host, but Docker's `go mod download` will fail without `go.sum`.

### eve-realm-sdk Submodule State

No `.gitmodules` file exists. The `eve-realm-sdk/` directory is not present. The `replace` directive in `go.mod` points to `./eve-realm-sdk`. Minimum requirement: `eve-realm-sdk/go.mod` with `module github.com/beduardo/eve-realm-sdk`.

### Dockerfile Is Entirely New

No `Dockerfile` exists. The eve-cli Dockerfile provides a matching pattern: `golang:1.25-alpine` builder, `CGO_ENABLED=0`, `gcr.io/distroless/static-debian12:nonroot` runtime.

### Complexity Estimate

**Size S**: ~30 LOC Dockerfile, ~6 LOC Makefile additions. No Go source changes needed.

## Overlap with SP-001

SP-001 delivered the complete build foundation. REQ-007 is purely additive: Dockerfile + two Makefile targets on top of SP-001 artifacts. The only gap is the missing `go.sum` and submodule directory.
