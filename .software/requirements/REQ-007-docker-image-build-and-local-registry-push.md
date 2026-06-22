---
content_hash: 6de7e00ef7e2fd6de092595be0dc6359a9ca7c63667684ad20530bfe81d0b7f5
created: "2026-06-22"
id: REQ-007
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-007
    - SC-008
    - SC-009
related_testcases: []
related_userstories: []
source: manual
status: implemented
tags:
    - docker
    - registry
    - build
title: Docker image build and local registry push
updated: "2026-06-22"
---

# REQ-007: Docker image build and local registry push

## Description

The MCP Server is packaged as a Docker image using a multi-stage Dockerfile. The image
is pushed to the local k3d registry for cluster deployment. The image tag uses the
semantic version from the VERSION file — K8s manifests reference the versioned tag,
never `:latest`.

The Dockerfile follows the eve-cli pattern: a Go builder stage producing a statically
linked binary, and a distroless runtime stage.

## Reference

Pattern derived from `eve-cli/Dockerfile` (multi-stage, CGO_ENABLED=0, distroless
runtime) and `eve-cli/Makefile` targets: `docker-build`, `docker-push`.

## Acceptance Criteria

1. A `Dockerfile` exists at the repository root with two stages: (a) `builder` using `golang:1.25-alpine`, building with `CGO_ENABLED=0` and ldflags for version injection; (b) runtime using `gcr.io/distroless/static-debian12:nonroot`.
2. The builder stage copies `go.mod`, `go.sum`, runs `go mod download`, then copies source and builds `cmd/eve-realm-mcp` to `/out/eve-realm-mcp`.
3. The runtime stage copies the binary to `/usr/local/bin/eve-realm-mcp`, exposes port 8080, and sets the entrypoint to the binary.
4. `make docker-build` builds the image tagged as `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:$(VERSION)`. A `:latest` tag may be built locally but is never referenced in K8s manifests.
5. `make docker-push` pushes the versioned tag to the k3d registry.
6. The `VERSION` build arg is passed from the Makefile to the Dockerfile to ensure the embedded version matches the VERSION file.
