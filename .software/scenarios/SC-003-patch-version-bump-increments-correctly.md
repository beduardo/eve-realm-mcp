---
content_hash: eaa64da88a8cea2e7223b5406983ef305f2613aba31aaf703b88a13654aec735
created: "2026-06-22"
id: SC-003
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - versioning
    - makefile
title: Patch version bump increments correctly
type: happy-path
updated: "2026-06-22"
---

# SC-003: Patch version bump increments correctly

## Preconditions

- `VERSION` file contains `0.1.0`.

## Steps

1. Run `make bump-patch`.
2. Read the `VERSION` file.

## Expected Result

- `VERSION` file now contains `0.1.1`.
- stdout shows `Version bumped to 0.1.1`.
- Running `make bump-patch` again produces `0.1.2`.
