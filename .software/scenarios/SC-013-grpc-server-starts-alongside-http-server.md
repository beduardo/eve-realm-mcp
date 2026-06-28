---
content_hash: 703d2cb094548ac9fb624f1626bae330b5cf5b21a23e80039e476979cd6e2812
created: "2026-06-27"
id: SC-013
related_changes: []
related_reqs:
    - REQ-00A
related_testcases: []
source: manual
status: validated
tags:
    - grpc
    - server
title: gRPC server starts alongside HTTP server
type: happy-path
updated: "2026-06-28"
---

# SC-013: gRPC server starts alongside HTTP server

## Preconditions

- MCP Server binary compiled with default flags
- No other process occupying ports 8080, 50051, or 9090

## Steps

1. Start the MCP Server binary without any flags
2. Verify HTTP server is listening on port 8080
3. Verify gRPC server is listening on port 50051
4. Stop the server
5. Start the MCP Server binary with `--grpc-port 9090`
6. Verify gRPC server is listening on port 9090
7. Verify HTTP server remains on port 8080

## Expected Result

- Both HTTP and gRPC servers start successfully on their respective default ports (8080 and 50051)
- Startup log output includes the gRPC listen port
- The `--grpc-port` flag overrides the default gRPC port; gRPC server listens on the specified custom port
- HTTP server port is unaffected by the gRPC port override
