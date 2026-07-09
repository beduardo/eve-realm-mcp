# Spec Writer Brief

**Sprint**: SP-005
**Sprint Title**: gRPC request logging interceptor
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Sprint Folder**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-005
**Date**: 2026-07-09

## Entity List

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-00C | requirement | gRPC request logging interceptor | no | - |
| SC-01A | scenario | ListTools RPC logs method duration and status | no | - |
| SC-01B | scenario | InvokeTool RPC log includes tool name | no | - |
| SC-01C | scenario | Failed RPC logs at error level with status code | no | - |

## Analysis Artifacts

- Codebase Analysis: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-005/analysis/codebase-analysis.md
- Feasibility Reports:
  - REQ-00C: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-005/analysis/feasibility-REQ-00C.md

## Project Context

EVE Realm MCP is the unified MCP Server for the Eve Realm platform. It aggregates tools and skills from all plugins into a single MCP endpoint for AI tools, proxies tool/skill calls to plugin gRPC servers, discovers plugins via NATS, and coordinates background agent execution with progress streaming.

The project currently has 41 entities (12 requirements, 28 scenarios, 1 change) across statuses: 1 active, 5 blocked, 1 draft, 31 implemented, 3 validated. Four sprints have been completed (SP-001 through SP-004), covering build pipeline, Docker image, K8s deployment, and gRPC tool registry with ping diagnostic tool.

The Go module is `github.com/beduardo/eve-realm-mcp` with SDK consumed via Git submodule + replace directive. Build uses `make build`, tests use `make test`, and deployment targets a k3d local cluster.

## Pinned Entities

The entities below are binding project policies. Every agent in every sprint phase (spec, plan, implement) MUST extract and follow all directives from these entities that are relevant to its phase.

### REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection
**Type**: requirement | **Status**: blocked

Cross-cutting requirements registry:

| ID | Title | Trigger condition | Summary |
|----|-------|-------------------|---------|
| REQ-001 | Test-Driven Development Strategy | **Implementing or modifying Go code** in any sprint step | Defines the red→green→refactor TDD cycle, Go test framework rules, test patterns, naming conventions, and pipeline integration. |
| REQ-002 | Sprint completion and release process | **Completing a sprint and preparing a release** | Defines the two-phase release process: version increment, README update, commit → make release → RELEASE.md → deploy-local. |
| REQ-003 | Cluster integration testing policy for infrastructure and inter-pod changes | **Modifying Kubernetes manifests, ConfigMap entries, inter-pod communication** | Defines the policy for keeping cluster integration tests in sync with topology changes. |
| REQ-004 | Local k3d cluster topology reference for operational agents | **Adding, modifying, or verifying Kubernetes deployments, services, ConfigMaps** | Provides the condensed local k3d cluster topology for sprint agents. |

## Flags

- readme_update_needed: false
