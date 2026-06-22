---
content_hash: 8509e07c48a501ca832012592ba59c7d1535487757b63d373e848a29a11dec33
created: "2026-06-22"
id: SC-010
related_changes: []
related_reqs:
    - REQ-009
related_testcases: []
source: manual
status: implemented
tags:
    - scaffold
    - health
    - http
title: Health probe endpoints return 200
type: happy-path
updated: "2026-06-22"
---

# SC-010: Health probe endpoints return 200

## Preconditions

- The MCP Server binary is running on port 8080.

## Steps

1. Send `GET http://localhost:8080/healthz`.
2. Send `GET http://localhost:8080/readyz`.

## Expected Result

- `/healthz` returns HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- `/readyz` returns HTTP 200 with body `{"status":"ok"}` and header `Content-Type: application/json`.
- Both endpoints respond within milliseconds (no external dependencies in the scaffold).
