---
content_hash: 3160ce82dc373bfd4f8f78479cd1b71c4d272247d01bdbb75bc62ddf7edcd6cc
created: "2026-06-22"
id: SC-006
related_changes: []
related_reqs:
    - REQ-006
related_testcases: []
source: manual
status: implemented
tags:
    - test
    - makefile
title: Test failure exits non-zero
type: happy-path
updated: "2026-06-22"
---

# SC-006: Test failure exits non-zero

## Preconditions

- Repository contains at least one Go test file.
- One test is intentionally failing (or a test can be made to fail for verification).

## Steps

1. Introduce a failing test (e.g., `func TestFail(t *testing.T) { t.Fatal("forced") }`).
2. Run `make test`.
3. Capture the exit code.

## Expected Result

- `make test` runs `go test -count=1 ./...`.
- The failing test is reported in stdout.
- The exit code is non-zero (1).
- A subsequent `make release-patch` (which depends on `test`) would abort at the test step.
