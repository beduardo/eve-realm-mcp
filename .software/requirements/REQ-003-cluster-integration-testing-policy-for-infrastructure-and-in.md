---
content_hash: ae1a45123750570c642163fa7601f4f5202e270cdebaf969148a4d107352efea
created: "2026-06-20"
id: REQ-003
priority: high
related_adrs: []
related_changes: []
related_scenarios: []
related_testcases: []
related_userstories: []
source: manual
status: draft
tags:
    - testing
    - kubernetes
    - cross-cutting
    - infrastructure
title: Cluster integration testing policy for infrastructure and inter-pod changes
updated: "2026-06-20"
---

# REQ-003: Cluster integration testing policy for infrastructure and inter-pod changes

## Description

Every change that touches the cluster surface of eve-realm-mcp must be accompanied by a corresponding verification check in the cluster integration test suite. The cluster surface includes Kubernetes manifests (Deployments, Services, ConfigMaps), NATS subjects (plugin discovery, tool announcements), Redis key patterns (caching and state), gRPC endpoints (plugin proxy connections), health endpoints, and MCP transport endpoints (HTTP+SSE).

The MCP Server is deployed to a k3d cluster in the `eve-realm` namespace. It discovers plugins via NATS pub/sub and proxies tool calls to plugin gRPC servers. Cluster integration tests validate that the deployed topology functions correctly end-to-end: services resolve, NATS subscriptions deliver, gRPC connections establish, Redis keys are accessible, health endpoints respond, and MCP transport endpoints accept connections.

A verification binary (location TBD — will be defined when the cluster topology is finalized) runs all registered checks via `make verify-cluster`. Each check is a pure function conforming to the standard check contract:

```go
type CheckFunc func(ctx context.Context) CheckResult
```

Check functions must:

- Complete within 10 seconds
- Return descriptive errors (not just "connection refused" — include the service name, endpoint, and what was expected)
- Be idempotent with no side effects on cluster state
- Clean up any test data created during execution

### Check categories

| Category | Covers |
|----------|--------|
| `infrastructure` | K8s Deployments, Services, pod readiness |
| `configmap` | ConfigMap keys, environment variable injection |
| `dns` | In-cluster DNS resolution for services |
| `messaging` | NATS subjects — plugin discovery, tool announcements, heartbeats |
| `cache` | Redis connectivity, key pattern validation |
| `grpc` | gRPC endpoint connectivity to plugins |
| `health` | Health and readiness probe endpoints |
| `mcp-transport` | MCP HTTP+SSE transport endpoint availability |

### Per-change flow

1. **Identify** the cluster surface impacted by the change (manifests, NATS subjects, Redis keys, gRPC endpoints, health endpoints, MCP transport)
2. **Write** a check function in the verification binary (location TBD)
3. **Register** the check in the checks slice with the appropriate category
4. **Run** `make verify-cluster` to confirm all checks pass

### Pipeline integration (via REQ-005 lazy-load)

| Pipeline stage | Responsibility |
|----------------|----------------|
| **Spec writer** | Generates a "Verify Expectations" subsection listing the cluster surfaces affected by the sprint |
| **Plan generator** | Adds a dedicated verification step after every step that modifies infrastructure |
| **Step implementer** | Writes the check function, registers it in the checks slice, runs `make verify-cluster` |
| **Step verifier** | Confirms every cluster-facing change has a corresponding check; a missing check is a **step failure** |

## Acceptance Criteria

- Given a sprint step modifies Kubernetes manifests (Deployments, Services, ConfigMaps), when the step is implemented, then a corresponding verification check is written and registered in the appropriate category
- Given a sprint step adds or modifies a NATS subject (plugin discovery, tool announcements), when the step is implemented, then a messaging check verifies that subscription and publish work correctly on that subject
- Given a sprint step modifies gRPC endpoints (plugin proxy connections), when the step is implemented, then a grpc check verifies connectivity and basic request/response to the endpoint
- Given a sprint step adds or modifies Redis key patterns, when the step is implemented, then a cache check verifies key accessibility and expected behavior
- Given a sprint step adds or modifies health or MCP transport endpoints, when the step is implemented, then a health or mcp-transport check verifies endpoint availability and response
- Given the step verifier validates a cluster-facing step, when a corresponding verification check is missing for any changed cluster surface, then the step fails verification
- Given `make verify-cluster` runs, when all registered checks execute, then each check completes within 10 seconds and reports descriptive results including service name, endpoint, and expected vs actual outcome
- Given the spec writer processes a sprint that includes cluster-facing changes, when generating the spec, then it includes a "Verify Expectations" subsection listing all affected cluster surfaces

## Notes

- This is a cross-cutting policy loaded via REQ-005 lazy-load catalog (not pinned directly to sprints). Trigger conditions: changes to K8s manifests, NATS subjects, Redis patterns, gRPC endpoints, health endpoints, ConfigMap entries, or MCP transport endpoints.
- Complements REQ-001 (unit tests via TDD) — REQ-001 covers function-level testing within the Go codebase; this policy covers cluster-level integration testing of deployed infrastructure and inter-pod communication.
- The actual verification binary location, pod name, and deployment mechanism are **TBD** — they will be defined when the cluster topology is finalized.
- The actual service topology (which pods exist, how they interconnect) is **TBD** — placeholder until the cluster design is populated with real topology data.
- The check function contract (`CheckFunc`) and per-change flow follow the same pattern established in eve-cli's cluster verification, adapted for eve-realm-mcp's Go codebase and k3d deployment target.
