# SP-004 Recovery Prompt

## What happened

Sprint SP-004 ("gRPC tool registry service with ping diagnostic tool") was fully
implemented and verified across 10 steps with TDD. All tests passed, `make build`
succeeded, the sprint was marked `completed` in eve-software. Then an incompetent
agent came along and destroyed the work by reverting all tracked file modifications
back to HEAD — likely via `git checkout -- .` or `git restore .`. This agent blindly
wiped out every modification to tracked files without checking what it was destroying.
It saw uncommitted changes and decided to "clean up" without understanding that those
changes were the entire sprint delivery. Absolute clown behavior.

The silver lining: because the agent only reverted tracked files, all NEW files
(untracked) survived on disk. The core implementation is intact. Only the
modifications to pre-existing tracked files need to be rebuilt.

## Current repository state

**HEAD commit**: `7678f11` — "Deliver SP-001 through SP-003 with k3d cluster bootstrap"
(unchanged, no commits were lost)

**Sprint status**: `completed` in eve-software (the manifest says completed, but the
code is partially destroyed on disk)

## What survived (DO NOT touch these files — they are correct and complete)

All new files from Steps 2-6 survived because they were untracked (new directories):

| File | Lines | Step | Notes |
|------|-------|------|-------|
| `internal/registry/registry.go` | 91 | 3 | ToolRegistry interface + MapRegistry with sync.RWMutex |
| `internal/registry/registry_test.go` | 315 | 3 | 6 tests including 3 concurrency tests |
| `internal/mcp/service.go` | 56 | 4 | MCPService gRPC handler with NotFoundError mapping |
| `internal/mcp/service_test.go` | 394 | 4 | 8 tests using bufconn listener |
| `internal/tools/ping.go` | 42 | 5 | Ping handler returning pong + RFC3339 timestamp |
| `internal/tools/ping_test.go` | 171 | 5 | 10 tests with time window comparison |
| `cmd/eve-realm-mcp/main.go` | 193 | 6 | Dual-server startup with errgroup, --grpc-port flag |
| `cmd/eve-realm-mcp/main_test.go` | 737 | 6 | 20 tests (14 original + 6 new gRPC tests) |
| `proto/mcp/v1/mcp.proto` | 51 | 2 | MCPService proto3 schema |
| `gen/proto/mcp/v1/mcp.pb.go` | 351 | 2 | Generated protobuf Go code |
| `gen/proto/mcp/v1/mcp_grpc.pb.go` | 167 | 2 | Generated gRPC Go code |
| `tools.go` | 14 | 1 | Build-tag-guarded tool dependency imports |
| `go.sum` | — | 1 | Module checksums |
| `.software/sprints/SP-004/*` | — | — | SPEC.md, PLAN.md, IMPLEMENTATION.md, manifest.json |
| `.software/requirements/REQ-00A-*.md` | — | — | gRPC tool registry service requirement |
| `.software/requirements/REQ-00B-*.md` | — | — | Ping built-in diagnostic tool requirement |
| `.software/scenarios/SC-013 through SC-019` | — | — | All sprint scenarios |

## What was destroyed (THESE are what need to be rebuilt)

These tracked files were reverted to HEAD (the SP-003 state). Each needs specific
modifications re-applied:

### 1. `go.mod` — Step 1

**Current state**: Only 3 lines (module declaration + go version, zero dependencies)

**Required changes**: Add these direct dependencies:
```
require (
    golang.org/x/sync v0.15.0
    google.golang.org/grpc v1.73.0
    google.golang.org/protobuf v1.36.6
)
```
Plus all indirect dependencies that `go mod tidy` resolves. The `go.sum` file already
exists on disk with the correct checksums, so `go mod tidy` should work.

**Verification**: `go build ./...` must succeed.

### 2. `Makefile` — Step 1

**Current state**: Has all SP-001 through SP-003 targets but is MISSING the `proto` target.

**Required changes**:
- Add `PROTO_SRC`, `PROTO_OUT`, `PROTO_FILES` variables
- Add `proto` target that invokes `protoc` with `--go_out` and `--go-grpc_out`
  pointing to `gen/proto/mcp/v1/`
- Add `proto` to the `.PHONY` line
- Include a comment documenting `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`
  prerequisites

**Verification**: `make proto` must regenerate the Go stubs (the proto file and gen
directory already exist).

### 3. `deploy/k8s/verify/checks.go` — Step 7

**Current state**: Has 5 checks (deployment-ready, service-exists, healthz, readyz,
configmap-injected). Missing the gRPC NodePort check.

**Required changes**:
- Add `GRPCClient` interface with a `Dial` method (following the same narrow-interface
  pattern as `KubeClient` and `HTTPClient`)
- Add `grpcNodePort` constant (30051)
- Add `CheckGRPCNodePort(client GRPCClient) CheckFunc` function that returns a
  CheckResult with `Category: "grpc"` and `Name: "grpc-nodeport"`, verifying TCP/gRPC
  reachability at `localhost:30051`
- Add `defaultGRPCClient` var (nil placeholder, same pattern as other default clients)
- Add 6th entry in `Checks` slice for gRPC NodePort check

**Verification**: `go test ./deploy/k8s/verify/...` must pass with 6 registered checks.

### 4. `deploy/k8s/verify/checks_test.go` — Step 7

**Current state**: Has tests for the 5 existing checks (579 lines). Missing gRPC tests.

**Required changes**:
- Add `mockGRPCClient` struct implementing the `GRPCClient` interface
- Add 4 test functions:
  - `TestCheckGRPCNodePort_Success`
  - `TestCheckGRPCNodePort_ConnectionRefused`
  - `TestCheckGRPCNodePort_Timeout`
  - `TestCheckGRPCNodePort_DescriptiveError`
- Update `TestChecks_AllRegistered` to expect 6 checks instead of 5
- Update `TestChecks_CategoryCounts` to include `"grpc": 1`

**Verification**: `go test -race ./deploy/k8s/verify/...` must pass.

### 5. `README.md` — Step 9

**Current state**: SP-003 version. Missing all gRPC documentation.

**Required changes**:
- Add "gRPC Service" section documenting MCPService with ListTools and InvokeTool RPCs
- Add "Built-in Tools" section documenting the ping tool (invocation, response format)
- Update "Local development build" to include `--grpc-port` flag (default 50051)
- Update Docker run command to include gRPC port mapping (-p 50051:50051)
- Update "Graceful Shutdown" to mention both HTTP and gRPC servers
- Add `proto` target to Makefile Targets table with prerequisites subsection
- Document NodePort 30051 for k3d access

**Verification**: README is valid markdown, port numbers/flag names/JSON shapes match code.

### 6. `RELEASES.md` — Step 10

**Current state**: Ends at v0.3.0 (SP-003). Missing SP-004 entry.

**Required changes**: Append v0.4.0 entry:

```markdown
## v0.4.0 -- SP-004: gRPC tool registry service with ping diagnostic tool

**Date**: 2026-06-28
**Sprint**: SP-004
**Entities**: REQ-00A, REQ-00B, SC-013, SC-014, SC-015, SC-016, SC-017, SC-018, SC-019

Added the gRPC `MCPService` on port 50051 with `ListTools` and `InvokeTool` RPCs backed
by a concurrent-safe `ToolRegistry` (sync.RWMutex). Defined the protobuf schema at
`proto/mcp/v1/mcp.proto` with `make proto` code generation. Delivered the `ping` built-in
diagnostic tool returning `{"message":"pong","timestamp":"<RFC3339>"}`. Extended
`cmd/eve-realm-mcp` with `--grpc-port` flag, errgroup dual-server startup, and graceful
shutdown covering both HTTP and gRPC. Added K8s Service NodePort 30051 for local k3d gRPC
access with a `CheckGRPCNodePort` cluster verification check.
```

## CRITICAL BUG TO FIX: `.gitignore`

The `.gitignore` contains:
```
dist/
eve-realm-mcp
```

The pattern `eve-realm-mcp` is intended to ignore the compiled binary, but it ALSO
matches the directory `cmd/eve-realm-mcp/`, causing ALL source code under that directory
to be gitignored. This is why `cmd/eve-realm-mcp/main.go` was never committed despite
being the main entry point created in SP-001.

**Fix**: Change `eve-realm-mcp` to `/eve-realm-mcp` (anchored to repo root) so it only
matches the binary at the root level, not the source directory inside `cmd/`.

Or alternatively, use a negation pattern:
```
dist/
eve-realm-mcp
!cmd/eve-realm-mcp/
```

The anchored pattern `/eve-realm-mcp` is cleaner.

## Instructions for the implementing agent

This is a RECOVERY run. The sprint's core implementation already exists and is correct.
You must NOT rewrite, modify, or "improve" the surviving files listed above. Your job is
strictly to:

1. **Fix `.gitignore`**: Change `eve-realm-mcp` to `/eve-realm-mcp`
2. **Rebuild `go.mod`**: Add the three direct dependencies and run `go mod tidy`
3. **Rebuild Makefile `proto` target**: Add the proto target with PROTO_SRC/PROTO_OUT vars
4. **Rebuild `deploy/k8s/verify/checks.go`**: Add GRPCClient interface + CheckGRPCNodePort
5. **Rebuild `deploy/k8s/verify/checks_test.go`**: Add gRPC mock + 4 tests + update counts
6. **Rebuild `README.md`**: Add gRPC documentation sections
7. **Rebuild `RELEASES.md`**: Append v0.4.0 entry

After each change, verify:
- `go build ./...` succeeds
- `go test -race -count=1 ./...` passes with zero failures and zero data races
- `make build` succeeds

Read the surviving source files to understand the exact interfaces, types, and patterns
before writing any code. The existing `checks.go` file has 5 checks using `KubeClient`
and `HTTPClient` narrow interfaces — follow the exact same pattern for `GRPCClient`.

Do NOT re-run the full 10-step plan. Do NOT create new test files for the surviving code.
Do NOT modify any file in `internal/`, `cmd/`, `proto/`, or `gen/`. Those are done and
verified.

## Sprint entities for reference

- **REQ-00A**: gRPC tool registry service
- **REQ-00B**: Ping built-in diagnostic tool
- **SC-013**: gRPC server starts alongside HTTP server
- **SC-014**: ListTools returns registered built-in tools
- **SC-015**: InvokeTool dispatches to handler and returns result
- **SC-016**: InvokeTool with unknown tool returns NOT_FOUND
- **SC-017**: Tool registry supports concurrent access
- **SC-018**: Ping tool registered at startup
- **SC-019**: Ping invocation returns pong with RFC 3339 timestamp
