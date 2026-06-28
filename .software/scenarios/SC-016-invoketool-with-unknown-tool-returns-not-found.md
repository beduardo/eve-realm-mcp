---
content_hash: 81dadcc7d08cfad2f51975177d0d870de1bea16cc0ca1daec920f7615af43ba4
created: "2026-06-27"
id: SC-016
related_changes: []
related_reqs:
    - REQ-00A
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - invoke-tool
    - error
title: InvokeTool with unknown tool returns NOT_FOUND
type: error-path
updated: "2026-06-28"
---

# SC-016: InvokeTool with unknown tool returns NOT_FOUND

## Preconditions

- MCP Server running
- No tool named `nonexistent` is registered in the ToolRegistry

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call the `InvokeTool` RPC with `tool_name` set to `"nonexistent"` and empty `input`

## Expected Result

- gRPC response returns status code `NOT_FOUND`
- Error message contains the unknown tool name `"nonexistent"`
- No server crash or panic occurs
