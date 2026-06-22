---
content_hash: 03ce7c9f8b18dc990ba22132634dc5e49e27bc2ac0c3f1cc5ff44a2982db6eed
created: "2026-06-22"
id: SC-002
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - build
    - makefile
    - ldflags
title: Production build injects version metadata via ldflags
type: happy-path
updated: "2026-06-22"
---

# SC-002: Production build injects version metadata via ldflags

## Preconditions

- `VERSION` file contains `0.1.0`.
- Git repository has at least one commit (so `git rev-parse --short HEAD` succeeds).
- Makefile variables `VERSION`, `GIT_HASH`, and `BUILD_DATE` are computed at the top of the Makefile via shell commands.

## Steps

1. Run `make build-prod` from the repository root.
2. Run `dist/eve-realm-mcp --port 0` and capture the startup log line.
3. Send `GET /version` to the HTTP server.

## Expected Result

- The binary is produced at `dist/eve-realm-mcp`.
- The startup log contains `eve-realm-mcp online (v0.1.0, <7-char-hash>, <YYYY-MM-DD>)`.
- `GET /version` returns `{"version":"0.1.0","git_hash":"<hash>","build_date":"<date>"}` with values matching the VERSION file and current git state.
- The ldflags in the recipe inject `main.Version`, `main.GitHash`, and `main.BuildDate`.
