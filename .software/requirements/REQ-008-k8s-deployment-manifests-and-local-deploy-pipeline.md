---
content_hash: 87dc4f18a3b95cc2c543108dead0d448d0ddb2ea09537f157f4ffc9ae6bcdc8a
created: "2026-06-22"
id: REQ-008
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-00A
    - SC-00B
    - SC-00C
    - SC-00D
    - SC-00E
related_testcases: []
related_userstories: []
source: manual
status: active
tags:
    - kubernetes
    - deploy
    - k3d
title: K8s deployment manifests and local deploy pipeline
updated: "2026-06-22"
---

# REQ-008: K8s deployment manifests and local deploy pipeline

## Description

The MCP Server requires K8s manifests (Deployment + Service) under `deploy/k8s/` and
Makefile targets to deploy, monitor, and release to the local k3d cluster. Manifests use
the `VERSION_PLACEHOLDER` pattern — `sed` replaces it with the actual version at deploy
time, ensuring the running image matches the VERSION file.

The deployment depends on eve-realm-infra (namespace, configmap, NATS, Redis) being
already applied. The MCP Server is the last component deployed because it discovers
plugins at startup.

## Reference

Pattern derived from `eve-cli/deploy/k8s/software-deployment.yaml`,
`eve-cli/deploy/k8s/software-service.yaml`, and `eve-cli/Makefile` targets:
`deploy-local`, `wait-rollouts`, `release-patch`, `release-minor`, `release-major`.

## Acceptance Criteria

1. `deploy/k8s/deployment.yaml` defines a Deployment named `eve-realm-mcp` in namespace `eve-realm` with label `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
2. The Deployment's container image is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER`.
3. The container exposes port 8080 (HTTP, for MCP HTTP+SSE endpoint and health probes) and port 50051 (gRPC, for future inter-service communication).
4. The container uses `envFrom` referencing `eve-realm-config` ConfigMap.
5. Liveness probe: HTTP GET `/healthz` on port 8080 (initialDelaySeconds: 5, periodSeconds: 10, failureThreshold: 3).
6. Readiness probe: HTTP GET `/readyz` on port 8080 (initialDelaySeconds: 3, periodSeconds: 5, failureThreshold: 3).
7. Resource requests: 128Mi memory, 100m CPU. Limits: 256Mi memory, 250m CPU.
8. `deploy/k8s/service.yaml` defines a ClusterIP Service named `eve-realm-mcp` in namespace `eve-realm`, exposing port 8080 (HTTP) and port 50051 (gRPC).
9. `make deploy-local` applies the deployment and service manifests, replacing `VERSION_PLACEHOLDER` with the current VERSION via `sed`.
10. `make wait-rollout` waits for `deployment/eve-realm-mcp` rollout to complete with a 120s timeout.
11. `make release-patch` orchestrates: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`. Analogous targets exist for `release-minor` and `release-major`.
