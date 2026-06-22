# Sprint SP-003: K8s deployment manifests and local deploy pipeline

**Created**: 2026-06-22
**Status**: Specified
**Entities**: 6

---

## Overview

This sprint delivers the Kubernetes deployment manifests and local deploy pipeline that
bring the MCP Server into the k3d cluster. It produces `deploy/k8s/deployment.yaml` and
`deploy/k8s/service.yaml` under the `eve-realm` namespace, along with Makefile targets
(`deploy-local`, `wait-rollout`, `release-patch`, `release-minor`, `release-major`) that
automate version-stamped image deployment and rollout verification. With this sprint
complete, a single `make release-patch` command runs the full cycle — tests, version bump,
Docker build, push to k3d registry, manifest apply, and rollout confirmation — closing the
loop between code change and running cluster pod.

## Entity Inventory

| ID | Type | Title | Partial | Scope Notes |
|----|------|-------|---------|-------------|
| REQ-008 | requirement | K8s deployment manifests and local deploy pipeline | no | - |
| SC-00A | scenario | Deployment manifest defines correct pod spec with health probes | no | - |
| SC-00B | scenario | Service manifest exposes HTTP and gRPC ports | no | - |
| SC-00C | scenario | Deploy-local replaces version placeholder and applies manifests | no | - |
| SC-00D | scenario | Wait-rollout confirms deployment stability | no | - |
| SC-00E | scenario | Release pipeline orchestrates full build-deploy cycle | no | - |

## Technical Context

> Codebase analysis was not performed for this sprint. Implementation should begin
> with a codebase exploration phase to identify relevant patterns and integration
> points.

Prior sprints (SP-001: build pipeline, SP-002: Docker image build) have established the
Makefile structure and VERSION file pattern. The `deploy-local` and `wait-rollout` targets
extend the existing Makefile. The release targets (`release-patch`, `release-minor`,
`release-major`) from SP-001 must be extended to include `docker-build`, `docker-push`,
`deploy-local`, and `wait-rollout` steps. All K8s manifests target namespace `eve-realm`
with image `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER`. The
`VERSION_PLACEHOLDER` token is replaced via `sed` at deploy time using the `VERSION` file
content. Cross-cutting policies REQ-001 (TDD), REQ-002 (release process), REQ-003 (cluster
integration testing), and REQ-004 (topology reference) all apply to this sprint.

## Implementation Sections

### REQ-008: K8s deployment manifests and local deploy pipeline

**Entity**: `.software/entities/requirements/REQ-008.md`
**Type**: requirement
**Priority**: high

**Codebase Mapping**:
To be determined during implementation.

Files to create:
- `deploy/k8s/deployment.yaml` — Deployment manifest for eve-realm-mcp
- `deploy/k8s/service.yaml` — Service manifest for eve-realm-mcp

Files to modify:
- `Makefile` — Add `deploy-local`, `wait-rollout`, and extend `release-patch`, `release-minor`, `release-major` targets

**Acceptance Criteria**:
- **AC-1**: Given `deploy/k8s/deployment.yaml` is authored, when inspected, then it defines a Deployment named `eve-realm-mcp` in namespace `eve-realm` with labels `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
- **AC-2**: Given the Deployment manifest is inspected, when the container image is read, then it is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:VERSION_PLACEHOLDER`.
- **AC-3**: Given the Deployment manifest is inspected, when container ports are read, then port 8080 (HTTP) and port 50051 (gRPC) are declared.
- **AC-4**: Given the Deployment manifest is inspected, when the container environment configuration is read, then it uses `envFrom` referencing the `eve-realm-config` ConfigMap.
- **AC-5**: Given the Deployment manifest is inspected, when the liveness probe is read, then it is HTTP GET `/healthz` on port 8080 with `initialDelaySeconds: 5`, `periodSeconds: 10`, `failureThreshold: 3`.
- **AC-6**: Given the Deployment manifest is inspected, when the readiness probe is read, then it is HTTP GET `/readyz` on port 8080 with `initialDelaySeconds: 3`, `periodSeconds: 5`, `failureThreshold: 3`.
- **AC-7**: Given the Deployment manifest is inspected, when resource requests and limits are read, then requests are 128Mi memory and 100m CPU, and limits are 256Mi memory and 250m CPU.
- **AC-8**: Given `deploy/k8s/service.yaml` is authored, when inspected, then it defines a ClusterIP Service named `eve-realm-mcp` in namespace `eve-realm` exposing port 8080 (HTTP) and port 50051 (gRPC).
- **AC-9**: Given `make deploy-local` is run, when it executes, then it replaces `VERSION_PLACEHOLDER` with the current `VERSION` file content via `sed` and applies both manifests to the cluster.
- **AC-10**: Given `make wait-rollout` is run, when it executes, then it runs `kubectl rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s` and exits 0 on success.
- **AC-11**: Given `make release-patch` is run, when it executes, then it orchestrates: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`; analogous targets exist for `release-minor` and `release-major`.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: key deliverables are `deployment.yaml` and `service.yaml` under `deploy/k8s/`, and Makefile targets `deploy-local`, `wait-rollout`, `release-patch`, `release-minor`, `release-major`. The `VERSION_PLACEHOLDER` sed replacement pattern must be portable across macOS (BSD sed) and Linux (GNU sed) — use the `-e` flag without in-place extension argument, or write to a temp file and apply. The deployment depends on eve-realm-infra (namespace, configmap, NATS, Redis) being applied first. Scope score 6/5 is at the upper boundary; no blockers from SDK or protobuf since this sprint is infrastructure-only (no Go code beyond what was built in prior sprints).

**Verify Expectations** (REQ-003 — cluster integration testing policy):
The following cluster surfaces are affected and must each have a corresponding check function registered in the verification binary under the appropriate category:
- `infrastructure`: `eve-realm-mcp` Deployment pod readiness in namespace `eve-realm`
- `infrastructure`: `eve-realm-mcp` Service availability in namespace `eve-realm`
- `health`: HTTP GET `/healthz` on port 8080 responds 200
- `health`: HTTP GET `/readyz` on port 8080 responds 200
- `configmap`: `eve-realm-config` ConfigMap keys injected via `envFrom` are accessible inside the pod

**Test Expectations**:
- Must test: `make deploy-local` replaces `VERSION_PLACEHOLDER` with the value from the `VERSION` file before applying manifests (validate via dry-run output or manifest content inspection)
- Must test: `make wait-rollout` exits 0 when deployment is ready and exits non-zero when the timeout of 120s elapses without readiness
- Must test: `make release-patch` chains all seven steps in the correct order: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`; verify each step is invoked exactly once and in sequence
- Must test: `make release-minor` and `make release-major` follow the same pipeline with `bump-minor` and `bump-major` substituted for `bump-patch`
- Must NOT rely on: a live k3d cluster in unit tests — use `kubectl --dry-run=client` for manifest validation and mock kubectl calls for pipeline sequencing tests

---

### SC-00A: Deployment manifest defines correct pod spec with health probes

**Entity**: `.software/entities/scenarios/SC-00A.md`
**Type**: scenario
**Priority**: (not specified)

**Codebase Mapping**:
To be determined during implementation.

File under verification:
- `deploy/k8s/deployment.yaml`

**Acceptance Criteria**:
- **AC-1**: Given `deploy/k8s/deployment.yaml` exists, when its metadata is inspected, then name is `eve-realm-mcp`, namespace is `eve-realm`, and labels include `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
- **AC-2**: Given the deployment manifest is inspected, when the container image is read, then it contains `VERSION_PLACEHOLDER` as the image tag.
- **AC-3**: Given the deployment manifest is inspected, when container ports are read, then ports 8080 and 50051 are declared.
- **AC-4**: Given the deployment manifest is inspected, when the liveness probe is read, then it is HTTP GET `/healthz` on port 8080 with the correct `initialDelaySeconds`, `periodSeconds`, and `failureThreshold` values.
- **AC-5**: Given the deployment manifest is inspected, when the readiness probe is read, then it is HTTP GET `/readyz` on port 8080 with the correct probe timing values.
- **AC-6**: Given the deployment manifest is inspected, when resource requests and limits are read, then they match the specified values (128Mi / 100m requests, 256Mi / 250m limits).
- **AC-7**: Given the deployment manifest is applied with `kubectl apply --dry-run=client`, when kubectl parses it, then it exits 0 with no validation errors.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: validates the structural correctness of `deployment.yaml` — name, namespace, labels, image with `VERSION_PLACEHOLDER`, ports (8080, 50051), `envFrom`, liveness/readiness probes, and resource requests/limits. The binary from SP-002 must already expose `/healthz` and `/readyz` for the probes to resolve; if not, probe path definition is correct in the manifest regardless of whether the endpoint exists yet. `kubectl apply --dry-run=client` can be used for structural validation in CI.

**Verify Expectations** (REQ-003):
- `infrastructure` check: Deployment `eve-realm-mcp` exists in namespace `eve-realm` and all pods are ready
- `health` check: HTTP GET to `/healthz` on the pod's port 8080 returns 200

**Test Expectations**:
- Must test: the manifest YAML is parseable and `kubectl apply --dry-run=client` exits 0
- Must test: liveness probe path is `/healthz` and readiness probe path is `/readyz`, both targeting port 8080
- Must test: resource limits and requests match the specified values (128Mi/100m requests, 256Mi/250m limits)
- Must NOT rely on: a live running pod — all checks must use `kubectl --dry-run=client` or static YAML parsing

---

### SC-00B: Service manifest exposes HTTP and gRPC ports

**Entity**: `.software/entities/scenarios/SC-00B.md`
**Type**: scenario
**Priority**: (not specified)

**Codebase Mapping**:
To be determined during implementation.

File under verification:
- `deploy/k8s/service.yaml`

**Acceptance Criteria**:
- **AC-1**: Given `deploy/k8s/service.yaml` exists, when its metadata is inspected, then name is `eve-realm-mcp`, namespace is `eve-realm`, and labels include `app: eve-realm-mcp` and `app.kubernetes.io/part-of: eve-realm`.
- **AC-2**: Given the service manifest is inspected, when the spec type is read, then it is `ClusterIP`.
- **AC-3**: Given the service manifest is inspected, when the selector is read, then it matches `app: eve-realm-mcp` (consistent with the Deployment's pod template labels).
- **AC-4**: Given the service manifest is inspected, when ports are read, then port 8080 is named `http` with targetPort 8080, and port 50051 is named `grpc` with targetPort 50051.
- **AC-5**: Given `kubectl apply --dry-run=client` is run against `service.yaml`, when kubectl parses it, then it exits 0 with no validation errors.
- **AC-6**: Given the Service selector is evaluated against the Deployment's pod template labels, when the selector labels are compared, then they match exactly.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: validates `service.yaml` structure — ClusterIP type, selector matching deployment labels, HTTP (8080) and gRPC (50051) ports correctly named and targeted. The selector must align with any labels defined in the Deployment's pod template; no additional labels beyond `app: eve-realm-mcp` are required unless the eve-realm-infra conventions dictate otherwise.

**Verify Expectations** (REQ-003):
- `infrastructure` check: Service `eve-realm-mcp` exists in namespace `eve-realm` with correct port bindings
- `dns` check: In-cluster DNS name `eve-realm-mcp.eve-realm.svc.cluster.local` resolves and port 8080 is reachable

**Test Expectations**:
- Must test: the service YAML is parseable and `kubectl apply --dry-run=client` exits 0
- Must test: port 8080 is named `http` and port 50051 is named `grpc` in the ports list
- Must test: selector `app: eve-realm-mcp` matches the pod template labels declared in `deployment.yaml`
- Must NOT rely on: a running cluster — all checks use `kubectl --dry-run=client` or static YAML parsing

---

### SC-00C: Deploy-local replaces version placeholder and applies manifests

**Entity**: `.software/entities/scenarios/SC-00C.md`
**Type**: scenario
**Priority**: (not specified)

**Codebase Mapping**:
To be determined during implementation.

Files under verification:
- `Makefile` (`deploy-local` target)
- `deploy/k8s/deployment.yaml`
- `deploy/k8s/service.yaml`

**Acceptance Criteria**:
- **AC-1**: Given `make deploy-local` is run, when it executes, then it reads the `VERSION` file to obtain the current version string.
- **AC-2**: Given `make deploy-local` is run, when it processes the manifests, then it replaces all occurrences of `VERSION_PLACEHOLDER` with the version string from the `VERSION` file using `sed`.
- **AC-3**: Given `make deploy-local` is run, when the processed manifests are applied, then both `deployment.yaml` and `service.yaml` are applied to the cluster via `kubectl apply`.
- **AC-4**: Given the deployment was applied by `make deploy-local`, when the running pod's image tag is inspected, then it matches the version string from the `VERSION` file at deploy time.
- **AC-5**: Given `make deploy-local` is run on both macOS and Linux, when `sed` replacement executes, then it succeeds without requiring GNU-specific flags.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: validates that `make deploy-local` uses `sed` to replace `VERSION_PLACEHOLDER` with `VERSION` file content and applies both manifests. The `sed` portability concern (BSD vs GNU) is a key risk — use `sed -e 's/VERSION_PLACEHOLDER/'"$(cat VERSION)"'/g'` piped to `kubectl apply -f -`, or write to temp files. The k3d registry does not require special image pull configuration if the k3d cluster was created with the registry mapped.

**Verify Expectations** (REQ-003):
- `infrastructure` check: After `make deploy-local`, the Deployment's image tag in the cluster matches the `VERSION` file content (no `VERSION_PLACEHOLDER` remaining)
- `infrastructure` check: Both `deployment.yaml` and `service.yaml` are present and applied in namespace `eve-realm`

**Test Expectations**:
- Must test: `VERSION_PLACEHOLDER` in the manifest is replaced with the actual version string before `kubectl apply` is invoked
- Must test: the `sed` replacement is portable — succeeds on both macOS (BSD sed) and Linux (GNU sed) without GNU-only flags
- Must test: both deployment and service manifests are applied (two `kubectl apply` invocations or one combined apply)
- Must NOT rely on: a live k3d cluster for the unit test — mock or capture the `kubectl apply` command invocation and verify the processed manifest content

---

### SC-00D: Wait-rollout confirms deployment stability

**Entity**: `.software/entities/scenarios/SC-00D.md`
**Type**: scenario
**Priority**: (not specified)

**Codebase Mapping**:
To be determined during implementation.

Files under verification:
- `Makefile` (`wait-rollout` target)

**Acceptance Criteria**:
- **AC-1**: Given `make wait-rollout` is run, when it executes, then it invokes `kubectl rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s`.
- **AC-2**: Given the deployment reaches ready state within 120 seconds, when `make wait-rollout` completes, then the command exits with code 0.
- **AC-3**: Given the deployment does not stabilize within 120 seconds, when the timeout elapses, then `kubectl rollout status` exits non-zero, propagating the failure exit code from the Makefile target.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: validates that `make wait-rollout` runs `kubectl rollout status` with a 120s timeout and propagates the exit code correctly. The 120s timeout must align with the probe configuration — readiness probe with `initialDelaySeconds: 3`, `periodSeconds: 5`, `failureThreshold: 3` gives a worst-case readiness window of ~18 seconds, well within 120s. kubectl must be available in the development environment and CI; this is a prerequisite.

**Verify Expectations** (REQ-003):
- `infrastructure` check: `kubectl rollout status deployment/eve-realm-mcp -n eve-realm` exits 0 after deploy-local completes

**Test Expectations**:
- Must test: the `wait-rollout` target passes the correct arguments to `kubectl rollout status`: deployment name `deployment/eve-realm-mcp`, namespace `-n eve-realm`, timeout `--timeout=120s`
- Must test: a non-zero exit from `kubectl rollout status` propagates as a non-zero exit from `make wait-rollout` (no silent failure swallowing)
- Must NOT rely on: a live cluster for verifying the target definition — inspect the Makefile target recipe directly or use a mock kubectl script

---

### SC-00E: Release pipeline orchestrates full build-deploy cycle

**Entity**: `.software/entities/scenarios/SC-00E.md`
**Type**: scenario
**Priority**: (not specified)

**Codebase Mapping**:
To be determined during implementation.

Files under verification:
- `Makefile` (`release-patch`, `release-minor`, `release-major` targets)
- `VERSION` file

**Acceptance Criteria**:
- **AC-1**: Given `make release-patch` is run with `VERSION` containing `0.1.0`, when it completes, then the pipeline executes in order: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`.
- **AC-2**: Given `make release-patch` completes, when the `VERSION` file is read, then it contains `0.1.1`.
- **AC-3**: Given `make release-patch` completes, when the running pod's image is inspected via `kubectl get pod -l app=eve-realm-mcp -n eve-realm -o jsonpath='{.items[0].spec.containers[0].image}'`, then it is `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:0.1.1`.
- **AC-4**: Given `make release-patch` completes, when the pod logs are read, then they show the startup message `eve-realm-mcp online (v0.1.1, ...)`.
- **AC-5**: Given `make release-minor` and `make release-major` are run, when they execute, then they follow the same seven-step pipeline with `bump-minor` and `bump-major` substituted for `bump-patch` respectively.

**Implementation Notes**:
Feasibility not assessed in full. From the feasibility brief: the existing `release-patch`, `release-minor`, and `release-major` targets from SP-001 need to be extended with `docker-build`, `docker-push`, `deploy-local`, and `wait-rollout` steps. The current pipeline from SP-002 includes `docker-build` and `docker-push`. This sprint appends `deploy-local` and `wait-rollout` to complete the pipeline. Ordering is strict — `docker-push` must precede `deploy-local` so the image exists in the k3d registry before the manifest is applied.

**Verify Expectations** (REQ-003):
- `infrastructure` check: After `make release-patch`, the running pod image tag matches the newly bumped version from the `VERSION` file
- `health` check: Pod logs after `make release-patch` contain the version startup message

**Test Expectations**:
- Must test: `release-patch` executes all seven steps in the exact order: `test` → `bump-patch` → `build-prod` → `docker-build` → `docker-push` → `deploy-local` → `wait-rollout`; no step is skipped or reordered
- Must test: `release-minor` and `release-major` follow the same pipeline with the appropriate bump target substituted
- Must test: if any step in the pipeline exits non-zero, subsequent steps are not executed (Makefile `set -e` or dependency chaining provides this)
- Must NOT rely on: a live cluster or Docker daemon for the step-ordering test — inspect Makefile target dependencies and prerequisites to verify the chain

---

## Documentation Tasks

### RELEASES.md Entry

**Required**: Always

Add an entry to RELEASES.md documenting:
- Sprint ID: SP-003 and title: K8s deployment manifests and local deploy pipeline
- Summary: Delivers `deploy/k8s/deployment.yaml` and `deploy/k8s/service.yaml` for the `eve-realm` namespace, Makefile targets `deploy-local` and `wait-rollout`, and extended release targets (`release-patch`, `release-minor`, `release-major`) that orchestrate the full build-push-deploy-verify pipeline.
- Entity IDs included: REQ-008, SC-00A, SC-00B, SC-00C, SC-00D, SC-00E
- Date of completion: to be filled at sprint completion

This entry should be appended to the existing RELEASES.md file. Do not read or modify existing entries.

### README.md Update

**Required**: User-facing changes detected

Update README.md to reflect:
- New Makefile targets: `deploy-local` (applies K8s manifests to k3d with version replacement) and `wait-rollout` (waits for deployment rollout stability)
- Updated release targets: `make release-patch`, `make release-minor`, `make release-major` now include Docker build, push, cluster deploy, and rollout wait as part of the automated pipeline
- Deployment prerequisites: `eve-realm-infra` (namespace, ConfigMap, NATS, Redis) must be applied before running `make deploy-local`
- K8s manifests location: `deploy/k8s/` containing `deployment.yaml` and `service.yaml`
- Version placeholder pattern: `VERSION_PLACEHOLDER` in image tags is replaced at deploy time from the `VERSION` file

## Pinned Entity Compliance

| Entity | Directive | How Addressed |
|--------|-----------|---------------|
| REQ-005: Cross-cutting requirements catalog for lazy-loaded sprint policy injection | Spec writer must load and apply cross-cutting requirements when their trigger conditions match. For this sprint: REQ-001 (Go code present), REQ-002 (release pipeline being defined), REQ-003 (K8s manifests being created), REQ-004 (K8s topology reference needed). | All four triggered cross-cutting requirements have been loaded. REQ-001: Test Expectations subsections generated for all sprint entities. REQ-002: README update flagged and RELEASES.md entry required. REQ-003: Verify Expectations subsections generated for all entities listing affected cluster surfaces and required check categories. REQ-004: Cluster topology values (namespace `eve-realm`, registry `k3d-eve-realm-registry.localhost:5100`, image pattern with `VERSION_PLACEHOLDER`, deployment order) applied throughout all manifest and pipeline specifications. |

## Out of Scope

- Health probe endpoint implementation (`/healthz`, `/readyz`) — assumed delivered or in progress from SP-002 (minimal MCP server binary); manifests declare the probe paths regardless.
- Cluster integration test verification binary location and implementation — the check function contract is established by REQ-003 but the binary location is TBD; this sprint defines what checks must exist, not the binary itself.
- Ingress or external load balancer configuration — the Service type is ClusterIP only.
- NATS, Redis, and plugin deployments — these are part of `eve-realm-infra` and are prerequisites, not deliverables of this sprint.
- Multi-environment (staging, production) K8s manifests — local k3d only.
- Helm chart packaging or Kustomize overlays.

## Prerequisites

- SP-001 (build pipeline) completed: Makefile targets `test`, `build-prod`, `bump-patch`, `bump-minor`, `bump-major` exist and function.
- SP-002 (Docker image build) completed: Makefile targets `docker-build` and `docker-push` exist and push images to `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`.
- k3d cluster `k3d-eve-realm` is running with the `eve-realm` namespace and `eve-realm-config` ConfigMap available (via `eve-realm-infra`).
- `kubectl` is installed and the kubeconfig context is set to `k3d-eve-realm`.
- The k3d local registry (`k3d-eve-realm-registry.localhost:5100`) is accessible from both the host and within the cluster.
- The binary from SP-002 exposes `/healthz` and `/readyz` on port 8080, or those endpoints are delivered as part of this sprint's scope (REQ-009 may cover this — confirm before starting).
