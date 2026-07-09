---
content_hash: f7a2fc7b6b8dc78eed57e10751d35341bb1401b3f50f016570461dfef507fdfc
created: "2026-06-27"
id: SC-015
related_changes: []
related_reqs:
    - REQ-00A
related_testcases: []
source: manual
status: implemented
tags:
    - grpc
    - invoke-tool
title: InvokeTool dispatches to handler and returns result
type: happy-path
updated: "2026-06-29"
---

# SC-015: InvokeTool dispatches to handler and returns result

## Preconditions

- MCP Server running with at least one tool registered (e.g., the `ping` built-in tool)

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call the `InvokeTool` RPC with a valid `tool_name` and JSON-encoded `input`

## Expected Result

- Response status is OK (no gRPC error)
- Response `output` field contains a valid JSON-encoded string produced by the tool's handler
- The output corresponds to the expected behavior of the invoked tool
