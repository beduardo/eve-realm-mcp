# Releases

## v0.1.1 -- SP-001: Build pipeline with semantic versioning

**Date**: 2026-06-22
**Sprint**: SP-001
**Entities**: REQ-006, SC-001, SC-002, SC-003, SC-004, SC-005, SC-006

Established the foundational Makefile-based build pipeline with semantic versioning. Delivered the VERSION file (initialized to 0.1.0), Makefile targets (`build`, `build-prod`, `test`, `bump-patch`, `bump-minor`, `bump-major`, `release-patch`), ldflags version injection (`main.Version`, `main.GitHash`, `main.BuildDate`), and the minimal `cmd/eve-realm-mcp` binary entry point with startup logging and a `/version` HTTP endpoint.

## v0.2.0 -- SP-002: Docker image build and minimal MCP Server binary

**Date**: 2026-06-22
**Sprint**: SP-002
**Entities**: REQ-007, REQ-009, SC-007, SC-008, SC-009, SC-00F, SC-010, SC-011, SC-012

Added health probe endpoints (`/healthz`, `/readyz`) returning `{"status":"ok"}` for K8s liveness and readiness probes. Implemented graceful SIGINT/SIGTERM shutdown with canonical log message. Created a two-stage Dockerfile (`golang:1.25-alpine` builder + `gcr.io/distroless/static-debian12:nonroot` runtime) producing a versioned image at `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:<VERSION>`. Added Makefile targets `docker-build` and `docker-push` for image build and local k3d registry push.

## v0.3.0 -- SP-003: K8s deployment manifests and local deploy pipeline

**Date**: 2026-06-22
**Sprint**: SP-003
**Entities**: REQ-008, SC-00A, SC-00B, SC-00C, SC-00D, SC-00E

Created K8s deployment manifests (`deploy/k8s/deployment.yaml` and `deploy/k8s/service.yaml`) for the `eve-realm` namespace with `VERSION_PLACEHOLDER` image tag replacement at deploy time. Added Makefile targets `deploy-local` (portable `sed -e` substitution piped to `kubectl apply`) and `wait-rollout` (`kubectl rollout status` with 120s timeout). Extended release targets (`release-patch`, `release-minor`, `release-major`) to orchestrate the full seven-step pipeline: `test → bump-* → build-prod → docker-build → docker-push → deploy-local → wait-rollout`. Implemented five REQ-003 cluster verification checks (deployment-ready, service-exists, healthz, readyz, configmap-injected) with table-driven unit tests using interface-based mocking.

## SP-004: gRPC tool registry service with ping diagnostic tool

**Date**: 2026-06-28
**Sprint**: SP-004
**Entities**: REQ-00A, REQ-00B, SC-013, SC-014, SC-015, SC-016, SC-017, SC-018, SC-019

Implemented the gRPC tool registry service (`internal/registry`, `internal/mcp`) with `MCPServiceServer` exposing `ListTools` and `InvokeTool` RPCs, backed by a concurrent-safe `MapRegistry`. Added the built-in `ping` diagnostic tool (`internal/tools`) returning `{"pong":true,"timestamp":"<RFC3339>"}`. Wired the gRPC server into `startServers` alongside the existing HTTP server with shared lifecycle and graceful shutdown.

## SP-005: gRPC request logging interceptor

**Date**: 2026-07-09
**Sprint**: SP-005
**Entities**: REQ-00C, SC-01A, SC-01B, SC-01C

Introduced `internal/logging.NewInterceptor`, a `grpc.UnaryServerInterceptor` that emits structured JSON log entries on every RPC completion with method path, duration in milliseconds, gRPC status code, and (for `InvokeTool` calls) the tool name. Migrated `cmd/eve-realm-mcp/main.go` from `log.Logger` to `log/slog` with `JSONHandler` writing to stdout, producing uniform Kubernetes-aggregatable JSON output for all pod lifecycle events and gRPC requests. Updated `main_test.go` assertions to parse JSON log entries.
