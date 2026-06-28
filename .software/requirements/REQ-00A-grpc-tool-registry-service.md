---
content_hash: 629c5f9fa5636bd85f426328b9620a56572175a285661fbb2202656eb76b62ba
created: "2026-06-27"
id: REQ-00A
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-013
    - SC-014
    - SC-015
    - SC-016
    - SC-017
related_testcases: []
related_userstories: []
source: manual
status: active
tags:
    - grpc
    - tool-registry
    - mcp-server
title: gRPC tool registry service
updated: "2026-06-28"
---

# REQ-00A: gRPC tool registry service

## Description

The MCP Server exposes a gRPC service on port 50051 that allows clients (the CLI, future
AI tool integrations) to discover and invoke registered tools. This is the core service
interface of the MCP Server — all tool interaction flows through it.

The service provides two operations:

- **ListTools** — Returns all tools currently registered in the server, each with its name,
  description, and JSON Schema for input parameters. The response is the live state of the
  registry (tools can be added or removed at runtime as plugins connect/disconnect).

- **InvokeTool** — Accepts a tool name and a JSON-encoded input payload, dispatches to the
  matching tool handler, and returns the tool's result as a JSON-encoded response. Returns
  a gRPC NOT_FOUND error if the tool name is not registered.

The tool registry is an internal component that manages tool registrations. For this first
iteration, only built-in tools (registered at server startup) are supported. The registry
interface is designed to accommodate future plugin-sourced tools (via NATS discovery) without
breaking changes — tools are identified by name regardless of origin.

The proto definition lives locally in this project (`proto/mcp/v1/mcp.proto`) for now. It
will migrate to the SDK submodule once the shared proto workflow is established, so both this
server and the CLI can consume the same generated Go code.

## Acceptance Criteria

1. A protobuf service definition exists at `proto/mcp/v1/mcp.proto` defining `MCPService` with `ListTools` and `InvokeTool` RPCs.
2. `ListTools` returns a repeated list of tool descriptors, each containing `name` (string), `description` (string), and `input_schema` (string, JSON Schema).
3. `InvokeTool` accepts `tool_name` (string) and `input` (string, JSON-encoded), returns `output` (string, JSON-encoded).
4. `InvokeTool` with an unregistered tool name returns gRPC status `NOT_FOUND` with a descriptive message.
5. The gRPC server starts on port 50051 (configurable via `--grpc-port` flag, default 50051) alongside the existing HTTP server on port 8080.
6. The K8s Service (`deploy/k8s/service.yaml`) uses `type: NodePort` with the gRPC port mapped to NodePort 30051, making the service accessible from the host at `localhost:30051`.
7. An internal `ToolRegistry` interface supports `Register(tool)`, `List() []Tool`, and `Invoke(name, input) (output, error)` operations.
8. The registry is safe for concurrent access (future plugin registration will happen asynchronously).
9. `make proto` generates Go code from the proto definition into `gen/proto/mcp/v1/`.
