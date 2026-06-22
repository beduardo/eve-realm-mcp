# Sprint SP-001: Build pipeline with semantic versioning

**Created**: 2026-06-22
**Status**: Specified
**Entities**: 7

---

## Overview

This sprint establishes the foundational Makefile-based build pipeline for the eve-realm-mcp
repository, introducing semantic versioning infrastructure that all future sprints depend on.
The pipeline covers development builds (no ldflags), production builds (ldflags inject Version,
GitHash, and BuildDate), version bump targets (patch, minor, major), and a test target that
gates the release sequence with a non-zero exit on failure. Delivering this pipeline is a
prerequisite for the release process defined in REQ-002 — no sprint can be released until the
`make release-*` targets exist and the `VERSION` file is the single source of truth.

## Entity Inventory

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-006 | requirement | Build pipeline with semantic versioning | No | - |
| SC-001 | scenario | Development build compiles without version injection | No | - |
| SC-002 | scenario | Production build injects version metadata via ldflags | No | - |
| SC-003 | scenario | Patch version bump increments correctly | No | - |
| SC-004 | scenario | Minor version bump resets patch segment | No | - |
| SC-005 | scenario | Major version bump resets minor and patch segments | No | - |
| SC-006 | scenario | Test failure exits non-zero | No | - |

## Technical Context

> Codebase analysis was not performed for this sprint. Implementation should begin
> with a codebase exploration phase to identify relevant patterns and integration
> points.

## Implementation Sections

### REQ-006: Build pipeline with semantic versioning

**Entity**: `.software/entities/requirements/REQ-006.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given the repository root, when inspected, then a `VERSION` file exists initialized to `0.1.0`.
- **AC-2**: Given a `VERSION` file and compilable `cmd/eve-realm-mcp/main.go`, when `make build` runs, then the binary is produced at `dist/eve-realm-mcp` without any `-ldflags` in the recipe.
- **AC-3**: Given a `VERSION` file and a git repository with at least one commit, when `make build-prod` runs, then the binary at `dist/eve-realm-mcp` is built with `-ldflags` injecting `main.Version` (from VERSION), `main.GitHash` (from `git rev-parse --short HEAD`), and `main.BuildDate` (UTC date).
- **AC-4**: Given at least one Go test file in the repository, when `make test` runs, then `go test -count=1 ./...` executes and exits non-zero on any test failure.
- **AC-5**: Given `VERSION` contains `0.1.0`, when `make bump-patch` runs, then VERSION is updated to `0.1.1`.
- **AC-6**: Given `VERSION` contains a version with a non-zero patch, when `make bump-minor` runs, then the minor segment increments and the patch segment resets to `0`.
- **AC-7**: Given `VERSION` contains a version with non-zero minor and/or patch, when `make bump-major` runs, then the major segment increments and both minor and patch segments reset to `0`.
- **AC-8**: Given the Makefile, when it is read, then `GIT_HASH`, `VERSION`, and `BUILD_DATE` variables are computed at the top using shell commands, consistent with the eve-cli pattern.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

Files to create:
- `VERSION` — initialized to `0.1.0`
- `Makefile` — with targets: `build`, `build-prod`, `test`, `bump-patch`, `bump-minor`, `bump-major`; Makefile variables `VERSION`, `GIT_HASH`, `BUILD_DATE` computed via shell at the top
- `cmd/eve-realm-mcp/main.go` — minimal compilable binary entry point with `var Version`, `var GitHash`, `var BuildDate` package-level variables (defaulting to `dev`, `unknown`, `unknown`) used in startup logging and an HTTP `/version` endpoint

**Test Expectations**:
- Must test: `make build` produces a binary at `dist/eve-realm-mcp` without ldflags (verify the binary is executable and the default version values `dev`/`unknown`/`unknown` are present)
- Must test: `make build-prod` injects `main.Version` matching the content of `VERSION`, `main.GitHash` matching the short git HEAD, and `main.BuildDate` matching the UTC date
- Must test: `bump-patch` increments the patch segment exactly by 1 (e.g., `0.1.0` → `0.1.1`; `0.1.1` → `0.1.2`)
- Must test: `bump-minor` increments the minor segment by 1 and resets patch to `0` (e.g., `0.1.3` → `0.2.0`)
- Must test: `bump-major` increments the major segment by 1 and resets minor and patch to `0` (e.g., `0.3.5` → `1.0.0`)
- Must test: `make test` exits non-zero when at least one test fails
- Must test: VERSION file boundary — bump targets handle leading zeros correctly (no octal interpretation)
- Must NOT rely on: global state, external service availability, or the git state of the test runner's working directory when testing version string parsing

---

### SC-001: Development build compiles without version injection

**Entity**: `.software/entities/scenarios/SC-001.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given `cmd/eve-realm-mcp/main.go` is compilable and a `build` target exists in the Makefile, when `make build` runs from the repository root, then the binary is produced at `dist/eve-realm-mcp`.
- **AC-2**: Given the binary was produced by `make build`, when it is inspected, then it is executable.
- **AC-3**: Given the binary was produced by `make build` (no ldflags), when `dist/eve-realm-mcp --port 0` starts and logs a startup message, then that message contains the default version values `dev`, `unknown`, `unknown`.
- **AC-4**: Given the `build` target in the Makefile, when the recipe is read, then no `-ldflags` flag appears in the `go build` command.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: `make build` produces `dist/eve-realm-mcp` that is executable (mode check)
- Must test: binary built without ldflags reports `Version=dev`, `GitHash=unknown`, `BuildDate=unknown` through its startup log or `/version` endpoint
- Must test: the `build` Makefile target recipe does not contain `-ldflags`
- Must NOT rely on: a running process or network socket — use process substitution or exec to capture output without binding a real port

---

### SC-002: Production build injects version metadata via ldflags

**Entity**: `.software/entities/scenarios/SC-002.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given `VERSION` contains `0.1.0` and the git repository has at least one commit, when `make build-prod` runs, then the binary is produced at `dist/eve-realm-mcp`.
- **AC-2**: Given the binary was produced by `make build-prod`, when it starts and logs a startup message, then the message contains `eve-realm-mcp online (v0.1.0, <7-char-hash>, <YYYY-MM-DD>)` with values matching the VERSION file and current git state.
- **AC-3**: Given the binary is running, when `GET /version` is called, then the response is `{"version":"0.1.0","git_hash":"<hash>","build_date":"<date>"}` with values matching the VERSION file and git HEAD.
- **AC-4**: Given the `build-prod` Makefile target, when the recipe is read, then the ldflags explicitly reference `main.Version`, `main.GitHash`, and `main.BuildDate`.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: binary built with `make build-prod` reports the exact version string from the `VERSION` file (not `dev`) via the `/version` endpoint or startup log
- Must test: `main.GitHash` injected matches `git rev-parse --short HEAD` at build time
- Must test: `main.BuildDate` injected is a valid UTC date in `YYYY-MM-DD` format
- Must test: the JSON response from `GET /version` matches the schema `{"version":"...","git_hash":"...","build_date":"..."}`
- Must NOT rely on: a real bound network port in tests — use `httptest.NewServer` for HTTP handler verification

---

### SC-003: Patch version bump increments correctly

**Entity**: `.software/entities/scenarios/SC-003.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given `VERSION` contains `0.1.0`, when `make bump-patch` runs, then the `VERSION` file contains `0.1.1`.
- **AC-2**: Given `make bump-patch` ran successfully, when stdout is captured, then it contains `Version bumped to 0.1.1`.
- **AC-3**: Given `VERSION` now contains `0.1.1`, when `make bump-patch` runs again, then the `VERSION` file contains `0.1.2`.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: `bump-patch` starting from `0.1.0` produces `0.1.1`
- Must test: `bump-patch` applied twice starting from `0.1.0` produces `0.1.2`
- Must test: stdout includes the "Version bumped to X.Y.Z" confirmation message
- Must NOT rely on: the actual state of the repository's `VERSION` file — tests must use `t.TempDir()` with a controlled VERSION file to avoid mutating repository state

---

### SC-004: Minor version bump resets patch segment

**Entity**: `.software/entities/scenarios/SC-004.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given `VERSION` contains `0.1.3`, when `make bump-minor` runs, then the `VERSION` file contains `0.2.0`.
- **AC-2**: Given the bump completed, when `VERSION` is read, then the patch segment is `0`.
- **AC-3**: Given `make bump-minor` ran successfully, when stdout is captured, then it contains `Version bumped to 0.2.0`.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: `bump-minor` starting from `0.1.3` produces `0.2.0` (minor incremented, patch reset)
- Must test: `bump-minor` starting from `0.0.9` produces `0.1.0` (patch reset regardless of its value)
- Must test: stdout includes the "Version bumped to X.Y.0" confirmation message
- Must NOT rely on: the actual state of the repository's `VERSION` file — tests must use `t.TempDir()` with a controlled VERSION file

---

### SC-005: Major version bump resets minor and patch segments

**Entity**: `.software/entities/scenarios/SC-005.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given `VERSION` contains `0.3.5`, when `make bump-major` runs, then the `VERSION` file contains `1.0.0`.
- **AC-2**: Given the bump completed, when `VERSION` is read, then both minor and patch segments are `0`.
- **AC-3**: Given `make bump-major` ran successfully, when stdout is captured, then it contains `Version bumped to 1.0.0`.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: `bump-major` starting from `0.3.5` produces `1.0.0` (major incremented, minor and patch both reset)
- Must test: `bump-major` starting from `2.7.9` produces `3.0.0`
- Must test: stdout includes the "Version bumped to X.0.0" confirmation message
- Must NOT rely on: the actual state of the repository's `VERSION` file — tests must use `t.TempDir()` with a controlled VERSION file

---

### SC-006: Test failure exits non-zero

**Entity**: `.software/entities/scenarios/SC-006.md`
**Type**: scenario
**Priority**: -

**Codebase Mapping**:
To be determined during implementation.

**Acceptance Criteria**:
- **AC-1**: Given at least one Go test file exists, when a test is introduced that calls `t.Fatal("forced")`, then `make test` runs `go test -count=1 ./...`.
- **AC-2**: Given `make test` ran with a failing test, when the exit code is captured, then it is non-zero (1).
- **AC-3**: Given `make test` ran with a failing test, when stdout is examined, then the failing test is reported.
- **AC-4**: Given a `release-patch` target that depends on `test`, when `make release-patch` runs with a failing test, then execution aborts at the test step and the release does not proceed.

**Implementation Notes**:
Feasibility not assessed. Review dependencies before starting.

**Test Expectations**:
- Must test: the `test` Makefile recipe runs `go test -count=1 ./...` (not a different invocation)
- Must test: when a test file contains a `t.Fatal` call, `make test` exits with code 1
- Must test: the `release-patch` target declares `test` as a prerequisite, so a test failure prevents the release sequence from continuing
- Must NOT rely on: permanently mutating the test suite — use a temporary test file injected during the test run or verify Makefile dependency structure through static inspection

---

## Documentation Tasks

### RELEASES.md Entry

**Required**: Always

Add an entry to RELEASES.md documenting:
- Sprint ID and title: SP-001 — Build pipeline with semantic versioning
- Summary of changes delivered: Makefile-based build pipeline, VERSION file, `build`, `build-prod`, `test`, `bump-patch`, `bump-minor`, `bump-major` targets, ldflags version injection, and the minimal `cmd/eve-realm-mcp` binary entry point
- Entity IDs included: REQ-006, SC-001, SC-002, SC-003, SC-004, SC-005, SC-006
- Date of completion: to be filled at release time

This entry should be appended to the existing RELEASES.md file. Do not read or modify existing entries.

### Version Increment Decision (REQ-002 Phase 1)

**Required**: Yes — REQ-002 trigger matched for this sprint

**Version increment type**: `patch` (0.1.0 → 0.1.1)

This sprint delivers the initial build and versioning infrastructure. Because it is the first code-producing sprint and adds new infrastructure without breaking changes, a patch increment is appropriate. The release pipeline target to run after implementation is `make release-patch`.

**README update required**: No

---

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection | Spec writer must evaluate trigger conditions in the registry and call `eve_software_show <ID>` for each matching trigger before generating the spec. Spec writer generates "Test Expectations" per REQ-001 pipeline integration instructions when REQ-001 trigger matches. | REQ-001 and REQ-002 triggers both evaluated as matching. REQ-001 loaded: Test Expectations subsections generated for REQ-006, SC-001, SC-002, SC-003, SC-004, SC-005, SC-006. REQ-002 loaded: Version Increment Decision recorded in Documentation Tasks (patch, 0.1.0 → 0.1.1; README update not required). REQ-003 and REQ-004 triggers evaluated as not matching — no K8s or inter-pod changes in scope. |

## Out of Scope

- Docker image build (`make docker-build`) — covered by a separate sprint (REQ-007)
- Kubernetes deployment manifests and `make deploy-local` — covered by a separate sprint (REQ-008)
- MCP Server runtime logic (aggregator, proxy, agent) — covered by downstream sprints
- Any release pipeline steps beyond `make test`, `make build-prod`, and the bump targets (full `make release-*` composition depends on Docker build being in place first)
- README.md updates — not required for this sprint

## Prerequisites

- A Git repository with at least one commit must exist at the project root (required for `git rev-parse --short HEAD` in `make build-prod` and the ldflags `GIT_HASH` variable).
- `cmd/eve-realm-mcp/main.go` must be a compilable Go file before `make build` can succeed — the minimal binary entry point is part of this sprint's delivery.
- The `go.mod` file must declare the module `github.com/beduardo/eve-realm-mcp` with the SDK submodule wired via `replace` before `go build` and `go test` can run cleanly.
