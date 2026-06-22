---
content_hash: 677e25fe45c569f5f6e1cd05cfb04d911f62e55c38b71146ffd4c0b405bdc14e
created: "2026-06-22"
id: SC-00B
related_changes: []
related_reqs:
    - REQ-008
related_testcases: []
source: manual
status: validated
tags:
    - kubernetes
    - service
title: Service manifest exposes HTTP and gRPC ports
type: happy-path
updated: "2026-06-22"
---

# SC-00B: Service manifest exposes HTTP and gRPC ports

## Preconditions

- `deploy/k8s/service.yaml` exists.

## Steps

1. Inspect `deploy/k8s/service.yaml` for structural correctness.
2. Verify metadata: name is `eve-realm-mcp`, namespace is `eve-realm`, labels include `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
3. Verify spec type is `ClusterIP`.
4. Verify selector matches `app: eve-realm-mcp`.
5. Verify ports: port 8080 (named `http`, targetPort 8080) and port 50051 (named `grpc`, targetPort 50051).

## Expected Result

- The Service manifest is valid YAML and can be parsed by `kubectl apply --dry-run=client`.
- The selector matches the Deployment's pod template labels.
- Both HTTP (8080) and gRPC (50051) ports are exposed and correctly named.
