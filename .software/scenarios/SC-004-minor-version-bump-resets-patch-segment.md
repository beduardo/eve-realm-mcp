---
content_hash: b1cc126da825c47d5740246f32ceca7e3a542102c000e9b568dafb7ed5ac089f
created: "2026-06-22"
id: SC-004
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - versioning
    - makefile
title: Minor version bump resets patch segment
type: happy-path
updated: "2026-06-22"
---

# SC-004: Minor version bump resets patch segment

## Preconditions

- `VERSION` file contains `0.1.3` (patch has been bumped several times).

## Steps

1. Run `make bump-minor`.
2. Read the `VERSION` file.

## Expected Result

- `VERSION` file now contains `0.2.0`.
- The patch segment is reset to `0`.
- stdout shows `Version bumped to 0.2.0`.
