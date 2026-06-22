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
