---
content_hash: 75d0c8262eae28173b3107f5491d4682f5a355097bf89a071b9d39684ba43783
created: "2026-06-22"
id: SC-007
related_changes: []
related_reqs:
    - REQ-007
related_testcases: []
source: manual
status: implemented
tags:
    - docker
    - dockerfile
title: Dockerfile multi-stage build produces runnable distroless image
type: happy-path
updated: "2026-06-22"
---

# SC-007: Dockerfile multi-stage build produces runnable distroless image

## Preconditions

- `Dockerfile` exists at the repository root.
- `cmd/eve-realm-mcp/main.go` compiles successfully.
- `go.mod` and `go.sum` are present and valid.

## Steps

1. Run `docker build --build-arg VERSION=0.1.0 -t eve-realm-mcp:test .` from the repository root.
2. Inspect the image stages with `docker history eve-realm-mcp:test`.
3. Run the image: `docker run --rm -p 8080:8080 eve-realm-mcp:test`.
4. Send `GET http://localhost:8080/healthz`.

## Expected Result

- The build completes with two stages: `builder` (golang:1.25-alpine) and runtime (gcr.io/distroless/static-debian12:nonroot).
- The builder stage copies `go.mod`/`go.sum` first, runs `go mod download`, then copies source and builds with `CGO_ENABLED=0`.
- The runtime image contains only `/usr/local/bin/eve-realm-mcp` — no Go toolchain, no source code.
- Port 8080 is exposed in the Dockerfile.
- The entrypoint is `/usr/local/bin/eve-realm-mcp`.
- The container starts and `/healthz` returns HTTP 200.
