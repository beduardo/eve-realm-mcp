---
content_hash: a5dcd11387b2bf7a97fed0bdc880aaaeddbe9137532e8c2d7ace6ffc02f4be4d
created: "2026-06-22"
id: SC-00F
related_changes: []
related_reqs:
    - REQ-009
related_testcases: []
source: manual
status: implemented
tags:
    - scaffold
    - server
    - startup
title: Binary compiles and starts with default port
type: happy-path
updated: "2026-06-22"
---

# SC-00F: Binary compiles and starts with default port

## Preconditions

- `cmd/eve-realm-mcp/main.go` exists with the minimal scaffold implementation.
- Go toolchain is available.

## Steps

1. Run `go build -o dist/eve-realm-mcp ./cmd/eve-realm-mcp` (no ldflags).
2. Run `dist/eve-realm-mcp` without any flags.
3. Observe the startup log output.
4. Run `dist/eve-realm-mcp --port 9090` and verify it listens on the alternate port.

## Expected Result

- The binary compiles without errors.
- Without flags, the server listens on port 8080 (default).
- The startup log line reads `eve-realm-mcp online (vdev, unknown, unknown)` since no ldflags were injected.
- With `--port 9090`, the server listens on port 9090 instead.
