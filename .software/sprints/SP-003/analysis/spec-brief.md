# Spec Writer Brief

**Sprint**: SP-003
**Sprint Title**: K8s deployment manifests and local deploy pipeline
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Sprint Folder**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003
**Date**: 2026-06-22

## Entity List

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-008 | requirement | K8s deployment manifests and local deploy pipeline | no | - |
| SC-00A | scenario | Deployment manifest defines correct pod spec with health probes | no | - |
| SC-00B | scenario | Service manifest exposes HTTP and gRPC ports | no | - |
| SC-00C | scenario | Deploy-local replaces version placeholder and applies manifests | no | - |
| SC-00D | scenario | Wait-rollout confirms deployment stability | no | - |
| SC-00E | scenario | Release pipeline orchestrates full build-deploy cycle | no | - |

## Analysis Artifacts

- Codebase Analysis: not available
- Feasibility Reports:
  - REQ-008: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-REQ-008.md
  - SC-00A: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-SC-00A.md
  - SC-00B: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-SC-00B.md
  - SC-00C: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-SC-00C.md
  - SC-00D: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-SC-00D.md
  - SC-00E: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-003/analysis/feasibility-brief-SC-00E.md

## Project Context

EVE Realm MCP is the unified MCP Server for the Eve Realm platform. It aggregates tools and skills from all plugins into a single MCP endpoint for AI tools, proxies tool/skill calls to plugin gRPC servers, discovers plugins via NATS, and coordinates background agent execution with progress streaming.

Project stats: 28 total entities (9 requirements, 18 scenarios, 1 change). Status distribution: 16 implemented, 5 validated, 5 blocked, 1 active, 1 draft. Two prior sprints (SP-001: build pipeline, SP-002: Docker image build) are completed.

The project uses Go with module path `github.com/beduardo/eve-realm-mcp`. The SDK is wired via Git submodule with a `replace` directive. The Dockerfile uses multi-stage builds. K8s deployment targets namespace `eve-realm` with image from `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`.

## Pinned Entities

### REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection

Cross-cutting requirements registry:

| ID | Title | Trigger condition | Summary |
|----|-------|-------------------|---------|
| REQ-001 | Test-Driven Development Strategy | **Implementing or modifying Go code** in any sprint step | Defines the red→green→refactor TDD cycle, Go test framework rules (`testing` stdlib only), test patterns (table-driven, temp dirs, interface mocking), naming conventions, and pipeline integration. |
| REQ-002 | Sprint completion and release process | **Completing a sprint and preparing a release** | Defines the two-phase release process: Phase 1 (spec-time decisions: version increment, README update) and Phase 2 (post-implementation release sequence). Also defines Docker image tagging and K8s deployment rules. |
| REQ-003 | Cluster integration testing policy for infrastructure and inter-pod changes | **Modifying Kubernetes manifests, ConfigMap entries, inter-pod communication, health endpoints, or adding new services to the cluster** | Defines the policy for keeping cluster integration tests in sync with topology changes. Every cluster-facing change must be accompanied by a corresponding check function. |
| REQ-004 | Local k3d cluster topology reference for operational agents | **Adding, modifying, or verifying Kubernetes deployments, services, ConfigMaps, or inter-pod networking in the local k3d cluster** | Provides the condensed local k3d cluster topology for sprint agents: cluster identity, service topology table, host dependencies, key ConfigMap entries, development workflow commands. |

## Flags

- readme_update_needed: true
