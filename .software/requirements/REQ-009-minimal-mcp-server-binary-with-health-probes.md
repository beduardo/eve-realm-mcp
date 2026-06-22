---
content_hash: a7feb78fcf1165da3c59eef88d147189e7104cbe15fb58adb582abdb742e6308
created: "2026-06-22"
id: REQ-009
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-00F
    - SC-010
    - SC-011
    - SC-012
related_testcases: []
related_userstories: []
source: manual
status: implemented
tags:
    - scaffold
    - health
    - server
title: Minimal MCP Server binary with health probes
updated: "2026-06-22"
---

# REQ-009: Minimal MCP Server binary with health probes

## Description

A minimal Go binary at `cmd/eve-realm-mcp/main.go` that serves as the scaffold for the
MCP Server. It provides just enough functionality to prove the build/Docker/deploy pipeline
works end-to-end: an HTTP server with health probes and a startup log.

No business logic (no NATS, no gRPC, no MCP protocol handling) — those are future
requirements. This scaffold is the foundation that the other pipeline requirements
(REQ-006, REQ-007, REQ-008) exercise.

## Reference

Health probe pattern derived from `eve-cli/deploy/k8s/software-deployment.yaml` (liveness
at `/healthz`, readiness at `/readyz`). Ldflags injection pattern from `eve-cli/Makefile`
(`build-prod` target).

## Acceptance Criteria

1. `cmd/eve-realm-mcp/main.go` exists and compiles to a standalone binary.
2. The binary accepts a `--port` flag (default: `8080`) controlling the HTTP listen port.
3. On startup, the binary logs `eve-realm-mcp online (vX.Y.Z, <git-hash>, <build-date>)` to stdout, where version/hash/date come from ldflags-injected variables (defaulting to `dev`, `unknown`, `unknown` when not injected).
4. `GET /healthz` returns HTTP 200 with body `{"status":"ok"}` and `Content-Type: application/json`.
5. `GET /readyz` returns HTTP 200 with body `{"status":"ok"}` and `Content-Type: application/json`.
6. The binary has package-level variables `Version`, `GitHash`, `BuildDate` (type `string`) that are populated via `-ldflags "-X main.Version=..."` at build time.
7. `GET /version` returns HTTP 200 with body `{"version":"X.Y.Z","git_hash":"...","build_date":"..."}` and `Content-Type: application/json`.
8. The binary handles `SIGINT` and `SIGTERM` gracefully, logging `eve-realm-mcp shutting down` before exiting.
