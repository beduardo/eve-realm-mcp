# Codebase Analysis Brief

**Sprint**: SP-002
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Entity IDs**: REQ-007, REQ-009, SC-007, SC-008, SC-009, SC-00F, SC-010, SC-011, SC-012

## Entity Details

### REQ-007: Docker image build and local registry push
- Type: requirement
- Status: active
- Multi-stage Dockerfile, Makefile docker-build/docker-push targets, k3d registry push

### REQ-009: Minimal MCP Server binary with health probes
- Type: requirement
- Status: active
- /healthz, /readyz endpoints, graceful SIGINT/SIGTERM shutdown, extends existing cmd/eve-realm-mcp/main.go from SP-001

### SC-007: Dockerfile multi-stage build produces runnable distroless image
- Type: scenario
- Status: validated
- Verifies Dockerfile builds and container runs with /healthz returning 200

### SC-008: Docker build tags image with semantic version
- Type: scenario
- Status: validated
- Verifies image tagged with VERSION file content

### SC-009: Docker push delivers image to k3d registry
- Type: scenario
- Status: validated
- Verifies image pushed to k3d-eve-realm-registry.localhost:5100

### SC-00F: Binary compiles and starts with default port
- Type: scenario
- Status: validated
- Verifies binary starts on port 8080 (largely covered by SP-001)

### SC-010: Health probe endpoints return 200
- Type: scenario
- Status: validated
- Verifies GET /healthz and GET /readyz return 200 + JSON

### SC-011: Version endpoint reports ldflags-injected metadata
- Type: scenario
- Status: validated
- Verifies GET /version returns version triplet (largely covered by SP-001)

### SC-012: Graceful shutdown on SIGINT and SIGTERM
- Type: scenario
- Status: validated
- Verifies clean shutdown on signal receipt

## Focus Areas
- Map entities to existing source files (cmd/eve-realm-mcp/main.go, main_test.go, Makefile, go.mod)
- Identify SP-001 artifacts that SP-002 builds on vs. new files to create (Dockerfile)
- Check submodule and go.sum state for Docker build readiness
- Identify implementation patterns from existing code that SP-002 should follow
