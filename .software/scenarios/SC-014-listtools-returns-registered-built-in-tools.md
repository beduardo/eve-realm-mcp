---
content_hash: e1b614a862fa9ffce22d83ef80b7238a559e93ab2673cd7a8b2951f2b4f4656b
created: "2026-06-27"
id: SC-014
related_changes: []
related_reqs:
    - REQ-00A
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - list-tools
title: ListTools returns registered built-in tools
type: happy-path
updated: "2026-06-28"
---

# SC-014: ListTools returns registered built-in tools

## Preconditions

- MCP Server running with at least one built-in tool registered in the ToolRegistry

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call the `ListTools` RPC with an empty request

## Expected Result

- Response contains a list of tool descriptors
- Each tool descriptor includes `name`, `description`, and `input_schema` fields
- Every tool has a non-empty `name` and non-empty `description`
- The `input_schema` field contains a valid JSON Schema string
