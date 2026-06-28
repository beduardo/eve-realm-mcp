# Feasibility Brief

**Sprint**: SP-004
**Project Root**: /Users/bruno/repo-pessoal/eve-realm/eve-realm-mcp/main
**Target Entity**: REQ-00A

## Entity Summary
gRPC tool registry service — exposes MCPService on port 50051 with ListTools and InvokeTool RPCs. Requires protobuf definition, code generation, ToolRegistry interface (concurrent-safe), gRPC server startup alongside existing HTTP server, `--grpc-port` flag, K8s Service NodePort configuration, and `make proto` target.

## Acceptance Criteria (9 ACs)
1. Proto definition at `proto/mcp/v1/mcp.proto` with MCPService, ListTools, InvokeTool
2. ListTools returns repeated tool descriptors (name, description, input_schema)
3. InvokeTool accepts tool_name + JSON input, returns JSON output
4. InvokeTool with unknown tool returns gRPC NOT_FOUND
5. gRPC server on port 50051, configurable via `--grpc-port`, alongside HTTP on 8080
6. **K8s Service (`deploy/k8s/service.yaml`) uses `type: NodePort` with gRPC port mapped to NodePort 30051, accessible from host at `localhost:30051`**
7. Internal ToolRegistry interface: Register, List, Invoke
8. Registry safe for concurrent access
9. `make proto` generates Go code to `gen/proto/mcp/v1/`

## Sprint Context
- Current entity count: 9
- Scope score: 9/5
- Other entities in sprint: REQ-00B, SC-013, SC-014, SC-015, SC-016, SC-017, SC-018, SC-019

## Focus Questions
- Does the existing cmd/ entry point support adding a second server (gRPC) cleanly?
- Is there an existing graceful shutdown pattern that can be extended?
- Are protobuf/gRPC dependencies already in go.mod or need adding?
- Does the Makefile structure support adding a `proto` target?
- Is there an existing flag parsing mechanism to add `--grpc-port`?
- **What is the current `type` of the K8s Service in `deploy/k8s/service.yaml`? Does it already have a gRPC port entry? What changes are needed for `type: NodePort` with NodePort 30051?**
- **Does `deploy/k8s/verify/checks.go` have a gRPC connectivity check, or does one need to be added for NodePort 30051?**
- **Are there existing cluster verification patterns for NodePort services that should be followed?**
