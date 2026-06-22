---
content_hash: ea72bdf19b7736497a8884254af7e34fbdc2228e62902e770e9295e6130db4b7
created: "2026-06-22"
id: SC-011
related_changes: []
related_reqs:
    - REQ-009
related_testcases: []
source: manual
status: implemented
tags:
    - scaffold
    - version
    - ldflags
title: Version endpoint reports ldflags-injected metadata
type: happy-path
updated: "2026-06-22"
---

# SC-011: Version endpoint reports ldflags-injected metadata

## Preconditions

- The binary was built with `make build-prod` (ldflags injected).
- `VERSION` file contains `0.1.0`.
- The binary is running.

## Steps

1. Send `GET http://localhost:8080/version`.
2. Parse the JSON response.

## Expected Result

- HTTP 200 with `Content-Type: application/json`.
- Body is `{"version":"0.1.0","git_hash":"<7-char-hash>","build_date":"<YYYY-MM-DD>"}`.
- The `version` field matches the VERSION file contents.
- The `git_hash` field is a valid short git hash (7 hex characters).
- The `build_date` field is a valid UTC date.
- The package-level variables `Version`, `GitHash`, `BuildDate` are of type `string` and populated via `-ldflags "-X main.Version=..."`.
