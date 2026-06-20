---
content_hash: 1e2eb8b6094d0e8306ba7487e02dcbaa54d26444bf49c12b0e9e054376ab7307
created: "2026-06-20"
id: REQ-002
priority: high
related_adrs: []
related_changes: []
related_scenarios: []
related_testcases: []
related_userstories: []
source: manual
status: draft
tags:
    - process
    - release
    - cross-cutting
title: Sprint completion and release process
updated: "2026-06-20"
---

# REQ-002: Sprint completion and release process

## Description

Defines the end-to-end process that every eve-realm-mcp sprint follows once implementation is complete, covering version bumping, Docker image building, release documentation, and Kubernetes deployment.

The process is split into two phases:

**Phase 1 — Spec-time decisions.** During sprint spec generation the user decides the version increment type (major, minor, or patch) and whether `README.md` needs updating. These decisions are recorded in the sprint `SPEC.md` so they are available when the release pipeline runs.

**Phase 2 — Post-implementation release sequence.** After all implementation work is done and tests pass, the following steps execute in strict order:

1. **Single implementation commit** — All source code changes, sprint artifacts, and modified `.software/` entities are committed together in one commit.
2. **Run release pipeline** — `make release-{patch,minor,major}` runs tests, bumps the `VERSION` file, builds the Go binary inside a multi-stage Docker build, and pushes the Docker image to the k3d registry (`k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:vX.Y.Z`).
3. **Collect metadata** — The new version string, the git hash of the implementation commit (step 1), and the current date are captured.
4. **Append to RELEASE.md** — A new release entry is appended (never reading or modifying existing entries). Format:
   ```
   ## vX.Y.Z — YYYY-MM-DD
   **Commit:** <short-hash>
   **Summary:** <sprint summary>
   ### Changes
   - <change 1>
   - <change 2>
   ```
5. **Update README.md** — Only if flagged at spec time.
6. **Commit release artifacts** — `VERSION`, `RELEASE.md`, and optionally `README.md` are committed as the release commit.
7. **Deploy to k3d** — `make deploy-local` updates the running k3d cluster with the new image.

**Build artifact rules:**

- Docker image is tagged with the version: `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp:vX.Y.Z`.
- The Go binary is built inside the Docker multi-stage build and is not placed locally.
- The `VERSION` file at the project root tracks the current version.
- The git hash referenced in `RELEASE.md` and embedded in Docker image labels is the implementation commit hash, not the release commit hash.

## Acceptance Criteria

- Given a sprint is completing, when the spec is generated, then it asks the user for the version increment type (major, minor, or patch) and whether `README.md` needs updating, and records both decisions in `SPEC.md`.
- Given the release pipeline runs, when `make release-{patch,minor,major}` executes, then it runs tests, bumps `VERSION`, builds the Go binary, and builds the Docker image as an atomic sequence.
- Given the release pipeline succeeds, when `RELEASE.md` is updated, then the new entry is appended without reading or modifying any existing entries.
- Given `RELEASE.md` references a git hash, when that hash is checked, then it matches the implementation commit (not the release commit).
- Given the release is complete, when `make deploy-local` runs, then the k3d cluster in the `eve-realm` namespace is updated with the new Docker image.
- Given no marketplace registration exists for the MCP Server, when the release completes, then no marketplace registration step is attempted.

## Notes

- The git hash in `RELEASE.md` and Docker image labels both reference the implementation commit, not the release commit. This ensures traceability to the actual code changes.
- Cross-cutting policy loaded via REQ-005 lazy-load catalog (not pinned directly).
- Complements REQ-001 (TDD and mandatory test patterns) — tests must pass before the release pipeline proceeds.
