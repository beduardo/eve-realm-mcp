---
content_hash: a7691cd60a5b97d31be9f9ad164559787db874138f49aceb1585f915b5535a7b
created: "2026-06-22"
id: SC-00C
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
title: Deploy-local replaces version placeholder and applies manifests
type: happy-path
updated: "2026-06-29"
---

# SC-00C: Deploy-local replaces version placeholder and applies manifests

## Preconditions

- k3d cluster is running with the `eve-realm` namespace and `eve-realm-config` ConfigMap already applied (from eve-realm-infra).
- The versioned Docker image has been pushed to the k3d registry.
- `VERSION` file contains `0.1.0`.

## Steps

1. Run `make deploy-local`.
2. Inspect the running deployment: `kubectl get deployment eve-realm-mcp -n eve-realm -o yaml`.
3. Check the pod's image tag.

## Expected Result

- `make deploy-local` applies both `deployment.yaml` and `service.yaml` using `sed` to replace `VERSION_PLACEHOLDER` with `0.1.0`.
- The running deployment's container image is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0` (not `VERSION_PLACEHOLDER`, not `:latest`).
- The service is reachable within the cluster at `eve-realm-mcp.eve-realm.svc.cluster.local:8080`.
