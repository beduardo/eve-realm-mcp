# Spec Writer Brief

**Sprint**: SP-001
**Sprint Title**: Build pipeline with semantic versioning
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Sprint Folder**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main/.software/sprints/SP-001
**Date**: 2026-06-22

## Entity List

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-006 | requirement | Build pipeline with semantic versioning | No | - |
| SC-001 | scenario | Development build compiles without version injection | No | - |
| SC-002 | scenario | Production build injects version metadata via ldflags | No | - |
| SC-003 | scenario | Patch version bump increments correctly | No | - |
| SC-004 | scenario | Minor version bump resets patch segment | No | - |
| SC-005 | scenario | Major version bump resets minor and patch segments | No | - |
| SC-006 | scenario | Test failure exits non-zero | No | - |

## Analysis Artifacts

- Codebase Analysis: not available
- Feasibility Reports:
  - REQ-006: not assessed
  - SC-001: not assessed
  - SC-002: not assessed
  - SC-003: not assessed
  - SC-004: not assessed
  - SC-005: not assessed
  - SC-006: not assessed

## Project Context

**Project**: EVE Realm MCP — Go-based unified MCP Server for the Eve Realm platform.
**Total Entities**: 28 (9 requirements, 18 scenarios, 1 change).
**Module**: `github.com/beduardo/eve-realm-mcp`
**Binary**: `cmd/eve-realm-mcp` compiled to `dist/eve-realm-mcp`
**Build conventions**: Makefile-based, ldflags injection for version metadata, `VERSION` file as single source of truth.
**SDK**: Git submodule at `eve-realm-sdk/` with `replace` directive in `go.mod`.
**Docker**: Multi-stage Dockerfile, distroless runtime image.
**K8s**: Namespace `eve-realm`, image `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`, `VERSION_PLACEHOLDER` substitution at deploy time.

## Pinned Entities

### REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection

Registry of cross-cutting requirements with trigger conditions:

| ID | Title | Trigger condition | Summary |
|----|-------|-------------------|---------|
| REQ-001 | Test-Driven Development Strategy | **Implementing or modifying Go code** in any sprint step | Defines the red->green->refactor TDD cycle, Go test framework rules (`testing` stdlib only), test patterns (table-driven, temp dirs, interface mocking), naming conventions, and pipeline integration. |
| REQ-002 | Sprint completion and release process | **Completing a sprint and preparing a release** | Defines the two-phase release process: Phase 1 (spec-time decisions: version increment, README update) and Phase 2 (post-implementation release sequence). |
| REQ-003 | Cluster integration testing policy for infrastructure and inter-pod changes | **Modifying Kubernetes manifests, ConfigMap entries, inter-pod communication** | Defines the policy for keeping cluster integration tests in sync with topology changes. |
| REQ-004 | Local k3d cluster topology reference for operational agents | **Adding, modifying, or verifying Kubernetes deployments, services, ConfigMaps** | Provides the condensed local k3d cluster topology for sprint agents. |

**Mandatory loading rule**: If a trigger condition matches the sprint scope, the spec writer MUST call `eve_software_show <ID>` to load the full requirement before proceeding.

**Trigger evaluation for this sprint**:
- REQ-001: MATCHES — this sprint implements Go code (build pipeline, version injection in `cmd/eve-realm-mcp/main.go`).
- REQ-002: MATCHES — this sprint establishes the build/release pipeline that REQ-002's release process depends on.
- REQ-003: Does NOT match — no K8s manifests or inter-pod communication changes in this sprint.
- REQ-004: Does NOT match — no K8s deployment changes in this sprint.

## Flags

- readme_update_needed: false
