---
content_hash: e3b115ac635d108c2c7e80743ee844486f200ff2b332efd75b12c9c8dd5df13e
created: "2026-07-01"
id: REQ-00C
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-01A
    - SC-01B
    - SC-01C
related_testcases: []
related_userstories: []
source: manual
status: active
tags:
    - grpc
    - logging
    - observability
title: gRPC request logging interceptor
updated: "2026-07-01"
---

# REQ-00C: gRPC request logging interceptor

## Description

The MCP Server's gRPC service (REQ-00A) executes ListTools and InvokeTool RPCs without
producing any log output. When a client calls the ping tool or lists available tools,
nothing appears in the pod logs — making it impossible to confirm that requests arrived,
diagnose failures, or audit tool usage.

A gRPC unary server interceptor logs every incoming RPC call. The interceptor is middleware
registered on the `grpc.Server` at construction time, wrapping all service handlers without
modifying individual method code. It captures the full request lifecycle: method arrival,
execution outcome, and timing.

For `InvokeTool` calls, the interceptor extracts the tool name from the request to provide
actionable context (e.g., "InvokeTool tool=ping duration=1.2ms status=OK"). For all other
RPCs, the gRPC method path is sufficient.

Logging uses Go's `log/slog` structured logger (stdlib since Go 1.21), producing JSON
output suitable for K8s log aggregation. The existing `log.Println` calls in startup and
shutdown are migrated to `slog` for consistency.

## Acceptance Criteria

1. A gRPC `UnaryServerInterceptor` is registered on the `grpc.Server` in `startServers`.
2. Every unary RPC logs a structured entry on completion containing: gRPC method (full path), duration (milliseconds), gRPC status code name (e.g., "OK", "NOT_FOUND", "Internal").
3. `InvokeTool` log entries additionally include the `tool_name` field extracted from the request.
4. Successful RPCs (status OK) are logged at `slog.LevelInfo`.
5. Failed RPCs (any non-OK status) are logged at `slog.LevelError`, with the error message included.
6. The server uses `slog` with a `JSONHandler` writing to stdout, replacing the existing `log.Logger` usage for startup and shutdown messages.
7. Log output is visible via `kubectl logs` on the deployed pod.

## Notes

- The interceptor lives in a new package `internal/logging` or alongside the MCP service — to be decided during planning.
- Future extension: when streaming RPCs are added (agent progress), a `StreamServerInterceptor` will follow the same pattern.
- This requirement does not cover log levels configurable at runtime — that is a future concern.
