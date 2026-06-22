---
content_hash: c74ce39d66e9980939f27e15c3498c3ed9af8d3ca88fa4d46b2cdcc609b84408
created: "2026-06-22"
id: SC-012
related_changes: []
related_reqs:
    - REQ-009
related_testcases: []
source: manual
status: implemented
tags:
    - scaffold
    - shutdown
    - signals
title: Graceful shutdown on SIGINT and SIGTERM
type: happy-path
updated: "2026-06-22"
---

# SC-012: Graceful shutdown on SIGINT and SIGTERM

## Preconditions

- The MCP Server binary is running.

## Steps

1. Start the binary: `dist/eve-realm-mcp`.
2. Send `SIGINT` (Ctrl+C) to the process.
3. Observe stdout.
4. Repeat: start the binary again and send `SIGTERM` (`kill <pid>`).
5. Observe stdout.

## Expected Result

- On receiving `SIGINT`, the binary logs `eve-realm-mcp shutting down` and exits with code 0.
- On receiving `SIGTERM`, the binary logs `eve-realm-mcp shutting down` and exits with code 0.
- The HTTP server stops accepting new connections before exiting.
- No panic or unclean exit — the process terminates gracefully in both cases.
