---
content_hash: 539b7294d76396cc5876087dd73579616e6a4c0f02ca260c4a7097b647d902a5
created: "2026-06-22"
id: SC-008
related_changes: []
related_reqs:
    - REQ-007
related_testcases: []
source: manual
status: implemented
tags:
    - docker
    - versioning
title: Docker build tags image with semantic version
type: happy-path
updated: "2026-06-22"
---

# SC-008: Docker build tags image with semantic version

## Preconditions

- `VERSION` file contains `0.1.0`.
- Makefile has a `docker-build` target.
- `DOCKER_IMAGE` variable defaults to `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`.

## Steps

1. Run `make docker-build`.
2. List local images: `docker images | grep eve-realm-mcp`.
3. Inspect the image: `docker inspect k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0`.
4. Run the image and send `GET /version`.

## Expected Result

- An image tagged `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0` exists.
- The `VERSION` build arg is passed from the Makefile to the Dockerfile (`--build-arg VERSION=$(VERSION)`).
- `GET /version` returns `{"version":"0.1.0",...}`, confirming the build arg propagated through ldflags into the binary.
- A `:latest` tag may also exist locally but is never referenced in K8s deployment manifests.
