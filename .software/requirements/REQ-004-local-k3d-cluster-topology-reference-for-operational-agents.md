---
content_hash: 15b415c130ea53c652e6d8f68353c28a2c0e9f0ed8d446b09e3a924b50d7d812
created: "2026-06-20"
id: REQ-004
priority: medium
related_adrs: []
related_changes: []
related_scenarios: []
related_testcases: []
related_userstories: []
source: manual
status: draft
tags:
    - documentation
    - kubernetes
    - k3d
    - operations
title: Local k3d cluster topology reference for operational agents
updated: "2026-06-20"
---

# REQ-004: Local k3d cluster topology reference for operational agents

## Description

Sprint agents (spec writer, plan generator, step implementer, step verifier) that
produce or validate Kubernetes manifests, Dockerfiles, and deployment commands for
eve-realm-mcp need a single, authoritative reference describing the local k3d
cluster topology. Without this reference, agents hallucinate port numbers, DNS
names, image patterns, and deployment sequences — producing manifests that fail on
apply or services that cannot discover each other.

This requirement establishes that reference as a living document maintained alongside
the K8s manifests in `deploy/k8s/`. It captures every detail an agent needs to
correctly build, push, deploy, and verify a service in the local k3d cluster.

### Cluster quick reference

| Property | Value |
|----------|-------|
| Context | `k3d-eve-realm` (TBD — confirm actual context name) |
| Namespace | `eve-realm` |
| Registry | `k3d-eve-realm-registry.localhost:5100` |

### Service topology (PLACEHOLDER)

The table below will be populated as services are defined and deployed. Current
known services based on project scaffolding:

| Deployment | Ports | DNS (in-cluster) | Health | Notes |
|-----------|-------|-------------------|--------|-------|
| eve-realm-mcp | 50051 (gRPC) | eve-realm-mcp.eve-realm.svc.cluster.local:50051 | TBD | MCP Server — aggregator and proxy |
| eve-realm-nats | 4222/9222 | TBD | TBD | NATS messaging |
| eve-realm-redis | 6379 | TBD | TBD | Redis cache/state |
| (plugins) | TBD | TBD | TBD | Plugin gRPC servers — discovered via NATS |

### Host dependencies (TBD)

External databases or services required on the host (PostgreSQL, Neo4j, etc.) will
be listed here once confirmed.

### Key ConfigMap entries (TBD)

Configuration values injected via the `eve-realm` ConfigMap (NATS URL, Redis URL,
service discovery settings) will be listed here once defined.

### Development workflow commands

| Command | Purpose |
|---------|---------|
| `make build` | Build MCP Server binary with ldflags |
| `make docker-build` | Multi-stage Docker build |
| `make deploy-local` | Apply K8s manifests to k3d |
| `make wait-rollout` | Wait for deployment to stabilize |
| `make test` | Run Go tests (aggregator, proxy, agent runtime) |

### Image and label conventions

- **Image pattern**: `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER`
- **Version placeholder**: `VERSION_PLACEHOLDER` in image tags, replaced at deploy time with the value from the `VERSION` file
- **Labels**: `app: eve-realm-mcp`, `version: <VERSION>`

### Deployment order

1. **eve-realm-infra** — namespace, configmap, NATS, Redis
2. **Plugins** — plugin gRPC servers register tools/skills via NATS
3. **eve-realm-mcp** — the MCP Server discovers plugins at startup via NATS

### Pipeline integration

Sprint agents consume this reference at each workflow stage:

- **Spec writer**: references the service topology table when specifying services,
  ports, or inter-service communication
- **Plan generator**: includes the correct `build → push → deploy → wait → verify`
  command sequence using the development workflow commands above
- **Step implementer**: uses correct image patterns (`VERSION_PLACEHOLDER`), labels
  (`app: eve-realm-mcp`), resource limits, DNS names, and namespace (`eve-realm`)
- **Step verifier**: probes correct health endpoints for each service using the DNS
  names and ports from the topology table

## Acceptance Criteria

- Given a sprint agent needs to add a K8s service, when it reads this topology
  reference, then it can specify correct ports, DNS names, and health endpoints
  without hallucinating values

- Given a sprint step deploys to the cluster, when the plan is generated, then it
  includes the correct `make deploy-local` and `make wait-rollout` sequence along
  with the appropriate image registry and version placeholder

- Given K8s manifests are written, when the step implementer creates them, then
  they use the image pattern `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`,
  the `VERSION_PLACEHOLDER` tag, labels `app: eve-realm-mcp`, and namespace
  `eve-realm`

- Given a new service is added to the cluster, when the topology changes, then
  this reference is updated to reflect the new service's deployment, ports, DNS
  name, and health endpoint

- Given the step verifier checks a deployment, when it probes health endpoints,
  then it uses the correct DNS names and ports from this reference rather than
  guessing or defaulting

## Notes

- Cross-cutting policy via REQ-005 lazy-load catalog (not pinned directly)
- **PLACEHOLDER STATUS**: The service topology table, host dependencies, and
  ConfigMap entries are placeholders. They will be populated when the actual
  cluster topology is defined and services are deployed.
- Full operational reference will live at `DOCS/LOCAL_K8S_TOPOLOGY.md` once created
- K8s manifests are located at `deploy/k8s/`
- Deployment order is infrastructure-first: eve-realm-infra → plugins → MCP Server
  (the MCP Server discovers plugins at startup via NATS)
- The MCP Server exposes gRPC on port 50051 by default for tool/skill proxying
