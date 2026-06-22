---
content_hash: 66e74ad6026dbd012a4e85c11be94b54c26363804663366bde5b02f7a16ac731
created: "2026-06-22"
id: SC-009
related_changes: []
related_reqs:
    - REQ-007
related_testcases: []
source: manual
status: implemented
tags:
    - docker
    - registry
    - k3d
title: Docker push delivers image to k3d registry
type: happy-path
updated: "2026-06-22"
---

# SC-009: Docker push delivers image to k3d registry

## Preconditions

- k3d cluster is running with a registry at `k3d-eve-realm-registry.localhost:5100`.
- `make docker-build` has been run, producing the versioned image locally.
- `VERSION` file contains `0.1.0`.

## Steps

1. Run `make docker-push`.
2. Query the registry catalog: `curl -s http://k3d-eve-realm-registry.localhost:5100/v2/_catalog`.
3. Query the image tags: `curl -s http://k3d-eve-realm-registry.localhost:5100/v2/eve-realm-mcp/tags/list`.

## Expected Result

- `docker push` succeeds for the versioned tag `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0`.
- The registry catalog includes `eve-realm-mcp`.
- The tags list includes `0.1.0`.
- The image is now available for K8s deployments within the k3d cluster to pull.
