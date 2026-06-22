---
content_hash: 716320894399f3e173f3634bcf44b3935e6443720eac5adcb9dd0e4a7f3bde5c
created: "2026-06-22"
id: SC-00A
related_changes: []
related_reqs:
    - REQ-008
related_testcases: []
source: manual
status: validated
tags:
    - kubernetes
    - deployment
    - health
title: Deployment manifest defines correct pod spec with health probes
type: happy-path
updated: "2026-06-22"
---

# SC-00A: Deployment manifest defines correct pod spec with health probes

## Preconditions

- `deploy/k8s/deployment.yaml` exists.

## Steps

1. Inspect `deploy/k8s/deployment.yaml` for structural correctness.
2. Verify metadata: name is `eve-realm-mcp`, namespace is `eve-realm`, labels include `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
3. Verify the container image is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER`.
4. Verify container ports: 8080 (named `http`) and 50051 (named `grpc`).
5. Verify `envFrom` references `eve-realm-config` ConfigMap.
6. Verify liveness probe: HTTP GET `/healthz` on port 8080, initialDelaySeconds 5, periodSeconds 10, failureThreshold 3.
7. Verify readiness probe: HTTP GET `/readyz` on port 8080, initialDelaySeconds 3, periodSeconds 5, failureThreshold 3.
8. Verify resource requests (128Mi memory, 100m CPU) and limits (256Mi memory, 250m CPU).

## Expected Result

- All structural checks pass.
- The manifest is valid YAML and can be parsed by `kubectl apply --dry-run=client`.
- The VERSION_PLACEHOLDER token is present in the image tag (not a hardcoded version or `:latest`).
