---
content_hash: fb40f33832918e1636e2bf2bc5c38efc93a2ef5eb7224bcc9c9cae2c2e70f656
created: "2026-06-27"
id: SC-018
related_changes: []
related_reqs:
    - REQ-00B
related_testcases: []
source: manual
status: implemented
tags:
    - ping
    - startup
title: Ping tool registered at startup
type: happy-path
updated: "2026-06-29"
---

# SC-018: Ping tool registered at startup

## Preconditions

- MCP Server starts normally with default configuration

## Steps

1. Connect a gRPC client to the MCP Server on port 50051
2. Call the `ListTools` RPC with an empty request
3. Search the returned tool list for a tool named `ping`

## Expected Result

- The `ListTools` response includes a tool with `name` equal to `"ping"`
- The tool's `description` is `"Diagnostic tool that returns pong with server timestamp"`
- The tool's `input_schema` defines an empty object (no required parameters)
