---
content_hash: f2ac5fe0e4d9f7dbfda2eb894c1ec34adbda2902d269653600837d685b0ec571
created: "2026-06-22"
id: SC-005
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - versioning
    - makefile
title: Major version bump resets minor and patch segments
type: happy-path
updated: "2026-06-22"
---

# SC-005: Major version bump resets minor and patch segments

## Preconditions

- `VERSION` file contains `0.3.5` (minor and patch have been bumped).

## Steps

1. Run `make bump-major`.
2. Read the `VERSION` file.

## Expected Result

- `VERSION` file now contains `1.0.0`.
- Both minor and patch segments are reset to `0`.
- stdout shows `Version bumped to 1.0.0`.
