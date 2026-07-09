---
content_hash: 88e54b8887ff83f2874f840251250ef65feb22b39ca87514acdb02f4197ed545
created: "2026-07-01"
id: SC-01C
related_changes: []
related_reqs:
    - REQ-00C
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - logging
    - error-handling
title: Failed RPC logs at error level with status code
type: error-path
updated: "2026-07-01"
---

# SC-01C: Failed RPC logs at error level with status code

## Preconditions

- MCP Server is running with the gRPC logging interceptor enabled
- No tool named "nonexistent" is registered in the registry
- Server log output is captured (stdout)

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call `InvokeTool` with `tool_name = "nonexistent"` and empty input
3. Observe that the gRPC response returns status `NOT_FOUND`
4. Inspect the server's stdout log output

## Expected Result

- The `InvokeTool` call returns gRPC status `NOT_FOUND` (as per REQ-00A)
- A single structured JSON log entry is emitted to stdout containing:
  - `"method"` field with value `"/mcp.v1.MCPService/InvokeTool"`
  - `"tool_name"` field with value `"nonexistent"`
  - `"duration_ms"` field with a numeric value representing elapsed milliseconds
  - `"status"` field with value `"NotFound"`
  - `"level"` field with value `"ERROR"`
  - `"error"` field containing the descriptive error message
