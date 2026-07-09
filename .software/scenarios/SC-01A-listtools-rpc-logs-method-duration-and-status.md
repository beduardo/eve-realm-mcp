---
content_hash: c402c3765da3be7263a49ffead74a2946fba96e6197c3cf33a3113bff476c589
created: "2026-07-01"
id: SC-01A
related_changes: []
related_reqs:
    - REQ-00C
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - logging
title: ListTools RPC logs method duration and status
type: happy-path
updated: "2026-07-01"
---

# SC-01A: ListTools RPC logs method duration and status

## Preconditions

- MCP Server is running with the gRPC logging interceptor enabled
- At least one tool (ping) is registered in the registry
- Server log output is captured (stdout)

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call the `ListTools` RPC with an empty request
3. Inspect the server's stdout log output

## Expected Result

- The `ListTools` response succeeds with the list of registered tools
- A single structured JSON log entry is emitted to stdout containing:
  - `"method"` field with value `"/mcp.v1.MCPService/ListTools"`
  - `"duration_ms"` field with a numeric value representing elapsed milliseconds
  - `"status"` field with value `"OK"`
  - `"level"` field with value `"INFO"`
