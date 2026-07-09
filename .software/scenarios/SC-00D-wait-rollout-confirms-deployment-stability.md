---
content_hash: 8916ed497d287b4fb1918b7f3687742cfbaa0fa42a2c20e0a596bace82a2c1cc
created: "2026-06-22"
id: SC-00D
related_changes: []
related_reqs:
    - REQ-008
related_testcases: []
source: manual
status: implemented
tags:
    - kubernetes
    - deploy
    - makefile
title: Wait-rollout confirms deployment stability
type: happy-path
updated: "2026-06-29"
---

# SC-00D: Wait-rollout confirms deployment stability

## Preconditions

- `make deploy-local` has been run and the deployment exists in the cluster.
- The pod's readiness probe is responding (i.e., the binary is functional).

## Steps

1. Run `make wait-rollout`.

## Expected Result

- The target executes `kubectl rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s`.
- The command exits 0 when the deployment reaches ready state.
- If the deployment fails to stabilize within 120 seconds, the command exits non-zero.
