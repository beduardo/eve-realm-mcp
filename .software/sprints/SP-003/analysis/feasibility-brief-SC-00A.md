# Feasibility Brief

**Sprint**: SP-003
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Target Entity**: SC-00A

## Entity Summary
- **Title**: Deployment manifest defines correct pod spec with health probes
- **Type**: scenario
- **Status**: validated
- **Tags**: kubernetes, deployment, health

Validates: deployment.yaml structure — name, namespace, labels, image with VERSION_PLACEHOLDER, ports (8080, 50051), envFrom, liveness/readiness probes, resource requests/limits.

## Sprint Context
- Current entity count: 6
- Scope score: 6/5
- Other entities in sprint: REQ-008, SC-00B, SC-00C, SC-00D, SC-00E

## Focus Questions
- Does the existing binary (from SP-002) already expose /healthz and /readyz endpoints?
- Are there any structural constraints from the k3d cluster setup that affect the manifest?
- Can kubectl apply --dry-run=client be used in CI/test for validation?
