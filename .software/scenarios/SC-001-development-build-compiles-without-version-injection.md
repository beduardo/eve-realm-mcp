---
content_hash: 23fcb69db2a584450f8b8c7c3c267c9b3dcc70d16cd1d32fc87f21630cfa83bf
created: "2026-06-22"
id: SC-001
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - build
    - makefile
title: Development build compiles without version injection
type: happy-path
updated: "2026-06-22"
---

# SC-001: Development build compiles without version injection

## Preconditions

- Repository contains `cmd/eve-realm-mcp/main.go` with a compilable Go binary.
- Makefile exists at the repository root with a `build` target.

## Steps

1. Run `make build` from the repository root.

## Expected Result

- The binary is produced at `dist/eve-realm-mcp`.
- The binary is executable.
- Running `dist/eve-realm-mcp --port 0` starts the server and logs a startup message containing the default version values (`dev`, `unknown`, `unknown`) since no ldflags were injected.
- No `-ldflags` appear in the `make build` recipe.
