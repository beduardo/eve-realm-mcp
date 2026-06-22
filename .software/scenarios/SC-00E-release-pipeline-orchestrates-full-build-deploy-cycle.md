---
content_hash: 5ca6f284a41427c390998a7a0bb66c10fb449152a1e9a64dc9ed6fc82767a01e
created: "2026-06-22"
id: SC-00E
related_changes: []
related_reqs:
    - REQ-008
related_testcases: []
source: manual
status: validated
tags:
    - release
    - makefile
    - deploy
title: Release pipeline orchestrates full build-deploy cycle
type: happy-path
updated: "2026-06-22"
---

# SC-00E: Release pipeline orchestrates full build-deploy cycle

## Preconditions

- k3d cluster is running with eve-realm-infra deployed.
- All Go tests pass.
- `VERSION` file contains `0.1.0`.

## Steps

1. Run `make release-patch`.
2. After completion, read the `VERSION` file.
3. Check the running pod's image tag: `kubectl get pod -l app=eve-realm-mcp -n eve-realm -o jsonpath='{.items[0].spec.containers[0].image}'`.
4. Check the pod logs for the startup message.

## Expected Result

- The pipeline executes in order: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`.
- `VERSION` file now contains `0.1.1`.
- The running pod's image tag is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.1`.
- Pod logs show `eve-realm-mcp online (v0.1.1, ...)`.
- `make release-minor` and `make release-major` follow the same orchestration with their respective bump targets.
