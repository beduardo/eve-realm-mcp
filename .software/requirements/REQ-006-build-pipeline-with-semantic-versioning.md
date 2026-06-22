---
content_hash: 8c0fc2b3ed1b820fd86a1ef843eb897b9cba25012b8a6d78edd329b943247b5f
created: "2026-06-22"
id: REQ-006
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-001
    - SC-002
    - SC-003
    - SC-004
    - SC-005
    - SC-006
related_testcases: []
related_userstories: []
source: manual
status: implemented
tags:
    - build
    - makefile
    - versioning
title: Build pipeline with semantic versioning
updated: "2026-06-22"
---

# REQ-006: Build pipeline with semantic versioning

## Description

The eve-realm-mcp repository requires a Makefile-based build pipeline following the same
conventions as the eve-cli monorepo, adapted for a single Go binary (no frontend). A
`VERSION` file at the repository root is the single source of truth for the current version.

The pipeline injects version metadata (Version, GitHash, BuildDate) into the binary via
Go ldflags at build time. This applies to both local builds (`build-prod`) and Docker
builds.

## Reference

Pattern derived from `eve-cli/Makefile` targets: `build-prod`, `bump-patch`, `bump-minor`,
`bump-major`, `test`.

## Acceptance Criteria

1. A `VERSION` file exists at the repository root, initialized to `0.1.0`.
2. `make build` compiles `cmd/eve-realm-mcp` to `dist/eve-realm-mcp` without ldflags injection (development build).
3. `make build-prod` compiles `cmd/eve-realm-mcp` to `dist/eve-realm-mcp` with `-ldflags` injecting `main.Version` (from VERSION file), `main.GitHash` (from `git rev-parse --short HEAD`), and `main.BuildDate` (UTC date).
4. `make test` runs `go test -count=1 ./...` and exits non-zero on failure.
5. `make bump-patch` increments the patch segment of VERSION (e.g., 0.1.0 → 0.1.1).
6. `make bump-minor` increments the minor segment and resets patch to 0 (e.g., 0.1.1 → 0.2.0).
7. `make bump-major` increments the major segment and resets minor and patch to 0 (e.g., 0.2.0 → 1.0.0).
8. Makefile variables `GIT_HASH`, `VERSION`, and `BUILD_DATE` are computed at the top of the Makefile using shell commands, consistent with the eve-cli pattern.
