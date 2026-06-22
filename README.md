# eve-realm-mcp

Eve Realm MCP Server — the unified MCP (Model Context Protocol) endpoint that aggregates
tools and skills from eve-realm plugins and proxies tool calls to plugin gRPC servers.

## Build and Run

### Requirements

- Go 1.25+
- Docker (for image build and push)
- k3d local registry running at `k3d-eve-realm-registry.localhost:5100` (for push)

### Local development build

```sh
make build        # fast build, no ldflags injection
./dist/eve-realm-mcp
```

The binary starts an HTTP server on port 8080 by default. Use the `-port` flag to override:

```sh
./dist/eve-realm-mcp -port 9090
```

### Production build

```sh
make build-prod   # injects VERSION, git hash, and build date via ldflags
```

## Docker

### Build the image

```sh
make docker-build
```

Builds a versioned image tagged `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:<VERSION>`,
where `<VERSION>` is read from the `VERSION` file (e.g., `0.1.0`).

The image uses a two-stage Dockerfile:

1. **Builder** — `golang:1.25-alpine`: compiles the binary with ldflags.
2. **Runtime** — `gcr.io/distroless/static-debian12:nonroot`: minimal distroless base with no shell.

Binary location in the container: `/usr/local/bin/eve-realm-mcp`

### Push to the local k3d registry

```sh
make docker-push
```

Pushes the versioned image to `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:<VERSION>`.

### Run the container

```sh
docker run --rm -p 8080:8080 k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.0
```

## Kubernetes Deployment

### Manifests

K8s manifests live in `deploy/k8s/`:

| File | Kind | Purpose |
|------|------|---------|
| `deploy/k8s/deployment.yaml` | Deployment | Runs `eve-realm-mcp` in the `eve-realm` namespace |
| `deploy/k8s/service.yaml` | Service (ClusterIP) | Exposes HTTP (`8080`) and gRPC (`50051`) inside the cluster |

### VERSION_PLACEHOLDER pattern

The image tag in both manifests is the literal string `VERSION_PLACEHOLDER`:

```yaml
image: k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER
```

At deploy time `make deploy-local` replaces this token with the content of the `VERSION`
file using `sed` before piping to `kubectl apply`. The manifests are never edited
in-place — the substitution happens only in the shell pipeline.

### Prerequisites

The following infrastructure must exist in the cluster before applying the MCP Server
manifests. It is provided by the `eve-realm-infra` stack:

| Resource | Type | Required by |
|----------|------|-------------|
| `eve-realm` | Namespace | All manifests |
| `eve-realm-config` | ConfigMap | `envFrom` in Deployment |
| NATS | Deployment/Service | Plugin discovery at runtime |
| Redis | Deployment/Service | Agent state at runtime |

Apply `eve-realm-infra` first, then run `make deploy-local`.

### Deploy to k3d

```sh
make deploy-local
```

Replaces `VERSION_PLACEHOLDER` with the current `VERSION` file content in both
`deploy/k8s/deployment.yaml` and `deploy/k8s/service.yaml`, then applies both via
`kubectl apply`.

### Wait for rollout

```sh
make wait-rollout
```

Runs `kubectl rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s`.
Exits with status `0` when the rollout stabilises within 120 seconds, or non-zero
if it times out.

### Resource configuration

The Deployment requests `128Mi` memory and `100m` CPU, with limits of `256Mi` and
`250m`. The liveness probe polls `/healthz` and the readiness probe polls `/readyz`,
both on port `8080`.

## HTTP Endpoints

All endpoints are served on the configured port (default `8080`).

| Endpoint    | Method | Response body        | Status | Purpose |
|-------------|--------|----------------------|--------|---------|
| `/version`  | GET    | `{"version":"...","git_hash":"...","build_date":"..."}` | 200 | Build metadata |
| `/healthz`  | GET    | `{"status":"ok"}`    | 200    | K8s liveness probe |
| `/readyz`   | GET    | `{"status":"ok"}`    | 200    | K8s readiness probe |

`/healthz` and `/readyz` are always `{"status":"ok"}` while the process is running and
ready to accept traffic. Configure them as K8s liveness and readiness probes respectively.

## Graceful Shutdown

The binary handles `SIGINT` and `SIGTERM`. On receipt of either signal it:

1. Logs `eve-realm-mcp shutting down`
2. Calls `http.Server.Shutdown` to drain active connections
3. Exits cleanly

## Makefile Targets

| Target          | Description |
|-----------------|-------------|
| `build`         | Build binary to `dist/eve-realm-mcp` (no ldflags) |
| `build-prod`    | Build binary with VERSION, git hash, and build date injected |
| `test`          | Run all Go tests (`go test -count=1 ./...`) |
| `docker-build`  | Build Docker image tagged with current VERSION |
| `docker-push`   | Push Docker image to the k3d local registry |
| `deploy-local`  | Replace `VERSION_PLACEHOLDER` and apply K8s manifests to the k3d cluster |
| `wait-rollout`  | Wait up to 120s for Deployment `eve-realm-mcp` rollout to stabilise |
| `bump-patch`    | Increment patch version in `VERSION` file (e.g., `0.1.0` → `0.1.1`) |
| `bump-minor`    | Increment minor version in `VERSION` file (e.g., `0.1.0` → `0.2.0`) |
| `bump-major`    | Increment major version in `VERSION` file (e.g., `0.1.0` → `1.0.0`) |
| `release-patch` | Seven-step pipeline: `test → bump-patch → build-prod → docker-build → docker-push → deploy-local → wait-rollout` |
| `release-minor` | Seven-step pipeline: `test → bump-minor → build-prod → docker-build → docker-push → deploy-local → wait-rollout` |
| `release-major` | Seven-step pipeline: `test → bump-major → build-prod → docker-build → docker-push → deploy-local → wait-rollout` |

## Docker Image Naming

```
k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:<VERSION>
```

`<VERSION>` matches the content of the `VERSION` file at build time.
