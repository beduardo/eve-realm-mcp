---
content_hash: 6ecdc4a06e836d851a1ac7548ebf43647c7e3b5a09d635ae6d0cf774fe9e49ff
created: "2026-06-27"
id: REQ-00B
priority: high
related_adrs: []
related_changes: []
related_scenarios:
    - SC-018
    - SC-019
related_testcases: []
related_userstories: []
source: manual
status: active
tags:
    - tool
    - built-in
    - diagnostic
    - ping
title: Ping built-in diagnostic tool
updated: "2026-06-28"
---

# REQ-00B: Ping built-in diagnostic tool

## Description

The first built-in tool registered in the MCP Server's tool registry (REQ-00A). It serves
two purposes: proving the end-to-end tool pipeline works (gRPC → registry → handler →
response), and providing a permanent diagnostic tool for connectivity checks.

The ping tool has no required inputs. When invoked, it returns a JSON response containing
the word "pong" and the server's current date and time in RFC 3339 format. This confirms
that the MCP Server is reachable, the tool registry is operational, and tool invocation
works correctly.

The tool is registered at server startup as a built-in tool — it does not come from any
plugin and is always available regardless of NATS/plugin state.

## Acceptance Criteria

1. A tool named `ping` is registered in the tool registry at server startup.
2. The tool's description is `Diagnostic tool that returns pong with server timestamp`.
3. The tool's input schema declares no required parameters (empty JSON object schema).
4. When invoked via `InvokeTool(tool_name: "ping")`, the tool returns a JSON response: `{"message": "pong", "timestamp": "<RFC 3339 timestamp>"}`.
5. The `timestamp` field reflects the server's current time at the moment of invocation, formatted as RFC 3339 (e.g., `2026-06-27T14:30:00Z`).
6. The tool handler lives in `internal/tools/ping.go`.
7. The ping tool appears in the `ListTools` response with its name, description, and input schema.
