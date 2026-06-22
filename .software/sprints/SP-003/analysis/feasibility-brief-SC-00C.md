# Feasibility Brief

**Sprint**: SP-003
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Target Entity**: SC-00C

## Entity Summary
- **Title**: Deploy-local replaces version placeholder and applies manifests
- **Type**: scenario
- **Status**: validated
- **Tags**: kubernetes, deploy, makefile

Validates: make deploy-local uses sed to replace VERSION_PLACEHOLDER with VERSION file content, applies both manifests, resulting image tag matches version, service is reachable.

## Sprint Context
- Current entity count: 6
- Scope score: 6/5
- Other entities in sprint: REQ-008, SC-00A, SC-00B, SC-00D, SC-00E

## Focus Questions
- Does the existing Makefile from SP-001/SP-002 already have a deploy-local skeleton or is it net-new?
- Is the sed replacement pattern portable across macOS and Linux (BSD vs GNU sed)?
- Does the k3d registry need any special configuration for image pulls?
