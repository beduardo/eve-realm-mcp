# Spec Writer Brief

**Sprint**: SP-004
**Sprint Title**: gRPC tool registry service with ping diagnostic tool
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Sprint Folder**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-004
**Date**: 2026-06-28

## Entity List

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-00A | requirement | gRPC tool registry service | no | - |
| REQ-00B | requirement | Ping built-in diagnostic tool | no | - |
| SC-013 | scenario | gRPC server starts alongside HTTP server | no | - |
| SC-014 | scenario | ListTools returns registered built-in tools | no | - |
| SC-015 | scenario | InvokeTool dispatches to handler and returns result | no | - |
| SC-016 | scenario | InvokeTool with unknown tool returns NOT_FOUND | no | - |
| SC-017 | scenario | Tool registry supports concurrent access | no | - |
| SC-018 | scenario | Ping tool registered at startup | no | - |
| SC-019 | scenario | Ping invocation returns pong with RFC 3339 timestamp | no | - |

## Analysis Artifacts

- Codebase Analysis: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-004/analysis/codebase-analysis-brief.md
- Feasibility Reports:
  - REQ-00A: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-004/analysis/feasibility-brief-REQ-00A.md
  - REQ-00B: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-004/analysis/feasibility-brief-REQ-00B.md

## Project Context

Project: EVE Realm MCP — unified MCP Server for the Eve Realm platform. Go backend with gRPC, NATS discovery, and K8s deployment. Module path: github.com/beduardo/eve-realm-mcp. SDK consumed via Git submodule with replace directive.

Entity counts: 37 total (11 requirements, 25 scenarios, 1 change). Status distribution: 22 implemented, 7 validated, 5 blocked, 2 active, 1 draft. Three prior sprints completed (SP-001: build pipeline, SP-002: Docker image, SP-003: K8s deployment).

Key build commands: `make build`, `make test`, `make proto`, `make docker-build`, `make deploy-local`, `make wait-rollout`.

Project structure: cmd/ (entry point), internal/ (aggregator, proxy, agent subsystems), deploy/k8s/ (manifests), eve-realm-sdk/ (submodule).

## Pinned Entities

### REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection
**Type**: requirement | **Status**: blocked

Cross-cutting requirements registry:

| ID | Title | Trigger condition | Summary |
|----|-------|-------------------|---------|
| REQ-001 | Test-Driven Development Strategy | **Implementing or modifying Go code** in any sprint step | Defines the red→green→refactor TDD cycle, Go test framework rules (`testing` stdlib only), test patterns (table-driven, temp dirs, interface mocking), naming conventions, and pipeline integration (spec writer generates test expectations, plan propagates them, implementer writes tests first, verifier validates coverage). |
| REQ-002 | Sprint completion and release process | **Completing a sprint and preparing a release** (typically the final steps of an implementation) | Defines the two-phase release process: Phase 1 (spec-time decisions: version increment, README update) and Phase 2 (post-implementation release sequence: commit → `make release-*` → collect metadata → append RELEASE.md → conditional README update → commit release artifacts → `make deploy-local`). Also defines Docker image tagging and K8s deployment rules. |
| REQ-003 | Cluster integration testing policy for infrastructure and inter-pod changes | **Modifying Kubernetes manifests, ConfigMap entries, inter-pod communication (NATS subjects, gRPC endpoints, Redis key patterns), health endpoints, or adding new services to the cluster** | Defines the policy for keeping cluster integration tests in sync with topology changes. Every cluster-facing change must be accompanied by a corresponding check function. Includes the check function contract (signature, timeout, idempotency), check categories (infrastructure, configmap, dns, messaging, cache, grpc, health, mcp-transport), and pipeline integration (spec writer generates "Verify Expectations", plan adds verification steps, implementer writes checks, verifier confirms coverage). |
| REQ-004 | Local k3d cluster topology reference for operational agents | **Adding, modifying, or verifying Kubernetes deployments, services, ConfigMaps, or inter-pod networking in the local k3d cluster** — including adding new services, changing ports or health endpoints, modifying `deploy/k8s/` manifests, or verifying deployment correctness | Provides the condensed local k3d cluster topology for sprint agents: cluster identity (namespace `eve-realm`), service topology table (with ports, DNS names, health endpoints), host dependencies, key ConfigMap entries, development workflow commands, and pipeline integration instructions for spec/plan/implement/verify agents. |

## Flags

- readme_update_needed: true
