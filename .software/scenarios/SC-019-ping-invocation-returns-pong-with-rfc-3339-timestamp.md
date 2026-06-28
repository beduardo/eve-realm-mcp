---
content_hash: e2aa93a12adf4c8be01b92517649862a2392d76b545dec024cff815ceaadcd94
created: "2026-06-27"
id: SC-019
related_changes: []
related_reqs:
    - REQ-00B
related_testcases: []
source: manual
status: validated
tags:
    - ping
    - invoke
title: Ping invocation returns pong with RFC 3339 timestamp
type: happy-path
updated: "2026-06-28"
---

# SC-019: Ping invocation returns pong with RFC 3339 timestamp

## Preconditions

- MCP Server running with the `ping` tool registered

## Steps

1. Record the current time (T1)
2. Connect a gRPC client to the MCP Server on port 50051
3. Call the `InvokeTool` RPC with `tool_name` set to `"ping"` and empty JSON `input`
4. Record the current time (T2)
5. Parse the JSON output from the response

## Expected Result

- Response status is OK
- Output is valid JSON with exactly two fields: `message` and `timestamp`
- The `message` field equals `"pong"`
- The `timestamp` field is a valid RFC 3339 formatted datetime string
- The parsed timestamp falls within the window [T1, T2] (within 1 second tolerance)
