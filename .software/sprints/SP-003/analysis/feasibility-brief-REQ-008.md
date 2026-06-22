# Feasibility Brief

**Sprint**: SP-003
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Target Entity**: REQ-008

## Entity Summary
- **Title**: K8s deployment manifests and local deploy pipeline
- **Type**: requirement
- **Status**: active
- **Tags**: kubernetes, deploy, k3d

Key deliverables: deployment.yaml, service.yaml under deploy/k8s/, Makefile targets (deploy-local, wait-rollout, release-patch/minor/major). VERSION_PLACEHOLDER sed replacement pattern. Depends on eve-realm-infra (namespace, configmap, NATS, Redis).

## Sprint Context
- Current entity count: 6
- Scope score: 6/5
- Other entities in sprint: SC-00A, SC-00B, SC-00C, SC-00D, SC-00E (all scenarios for this REQ)

## Focus Questions
- Are all prerequisites (eve-realm-infra, k3d cluster, Docker image build) already in place from prior sprints?
- Are there any blockers from the SDK submodule or protobuf definitions?
- Is the Makefile pattern from SP-001/SP-002 sufficient to extend for deploy targets?
- Are there any missing dependencies (kubectl, sed, k3d) that need to be addressed?
