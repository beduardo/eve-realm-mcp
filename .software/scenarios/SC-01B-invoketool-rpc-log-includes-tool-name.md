---
content_hash: d513d9cf3fe571819b82eb83793a27af45dbc02ce984fb64cda20ae14432b327
created: "2026-07-01"
id: SC-01B
related_changes: []
related_reqs:
    - REQ-00C
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - logging
    - invoke-tool
title: InvokeTool RPC log includes tool name
type: happy-path
updated: "2026-07-01"
---

# SC-01B: InvokeTool RPC log includes tool name

## Preconditions

- MCP Server is running with the gRPC logging interceptor enabled
- The ping tool is registered in the registry
- Server log output is captured (stdout)

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call `InvokeTool` with `tool_name = "ping"` and empty input
3. Inspect the server's stdout log output

## Expected Result

- The `InvokeTool` response succeeds with the ping result (pong + timestamp)
- A single structured JSON log entry is emitted to stdout containing:
  - `"method"` field with value `"/mcp.v1.MCPService/InvokeTool"`
  - `"tool_name"` field with value `"ping"`
  - `"duration_ms"` field with a numeric value representing elapsed milliseconds
  - `"status"` field with value `"OK"`
  - `"level"` field with value `"INFO"`
