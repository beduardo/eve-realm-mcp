---
content_hash: 23c5ed6518429ce12d7e777f8d014f43ff2b14cbbca80a41d33b7dca3be5f4e6
created: "2026-06-20"
id: REQ-001
priority: high
related_adrs: []
related_changes: []
related_scenarios: []
related_testcases: []
related_userstories: []
source: manual
status: draft
tags:
    - tdd
    - testing
    - cross-cutting
testing_strategy: tdd
title: Test-Driven Development Strategy
updated: "2026-06-20"
---

# REQ-001: Test-Driven Development Strategy

## Description

Defines the mandatory test-driven development discipline for the eve-realm-mcp project (`github.com/beduardo/eve-realm-mcp`). Every line of production code in this greenfield MCP Server must be preceded by a failing test. The strategy governs all packages — `cmd/`, `internal/aggregator/`, `internal/proxy/`, and `internal/agent/` — and applies uniformly across unit and integration tests.

### Core TDD Cycle

All implementation follows the **Red-Green-Refactor** cycle per acceptance criterion:

1. **Red** — Write a failing test that captures the desired behavior.
2. **Green** — Write the minimal production code to make the test pass.
3. **Refactor** — Improve code structure while keeping all tests green.

### Testing Standards

- **Standard library only**: Use Go's `testing` package exclusively. No testify, no external test frameworks.
- **Table-driven tests**: Prefer `[]struct{ name string; ... }` with `t.Run(tc.name, ...)` for functions with multiple input variations.
- **Temp directories**: Use `t.TempDir()` for any test that reads or writes files.
- **Interface-based mocking**: Define narrow interfaces at the consumption site and implement test doubles in the same test file. No external mock libraries (no mockgen, no gomock).
- **Process substitution**: Use the process substitution pattern for testing CLI binaries built from `cmd/`.
- **HTTP test servers**: Use `httptest.NewServer` for HTTP handler and transport tests.
- **YAML round-trip tests**: Validate config and manifest parsing with serialize-deserialize round trips.
- **White-box and black-box**: White-box tests live in the same package as the source; black-box tests use the `_test` suffix package.
- **Naming convention**: `TestFunctionName_Scenario` (e.g., `TestAggregator_DiscoverPluginsViaHeartbeat`).
- **No global state**: Tests must not depend on global variables or external service availability.
- **File placement**: All test files (`*_test.go`) reside in the same directory as the source they test.

### Integration Test Specifics

The MCP Server integrates with gRPC plugin backends, NATS discovery, and MCP transports. Integration tests mock at I/O boundaries only — never internal pure logic:

- **gRPC proxy testing**: Mock gRPC clients that implement the `PluginService` interface for `internal/proxy/` tests.
- **NATS discovery testing**: Use an embedded NATS server for `internal/aggregator/` plugin discovery protocol tests.
- **MCP transport testing**: Mock stdio and HTTP+SSE transports for end-to-end MCP handler tests.
- **Boundary rule**: Interfaces are mocked only at I/O boundaries (gRPC, NATS, HTTP). Internal logic between packages is tested directly without mocks.

### Pipeline Integration

This cross-cutting requirement integrates into the sprint pipeline at every stage:

- **Spec writer**: Generates a "Test Expectations" subsection for each requirement, specifying what tests must exist before the requirement is considered met.
- **Plan generator**: Propagates test expectations into the implementation plan and enforces test-first step ordering — every step that produces code must begin with a failing test.
- **Step implementer**: Writes the failing test BEFORE writing production code, following the Red-Green-Refactor cycle within each step.
- **Step verifier**: Validates that `go test ./...` passes, confirms test coverage for new code, and verifies the TDD cycle was followed (test commit precedes or accompanies production code).

## Acceptance Criteria

- Given a sprint step requires Go code in any package under `github.com/beduardo/eve-realm-mcp`, when the implementer begins the step, then it writes a failing test first before any production code.
- Given this TDD strategy is loaded as a cross-cutting requirement, when tests are written for any MCP Server component, then they use only Go's standard `testing` library with no external test frameworks.
- Given a function in `internal/aggregator/`, `internal/proxy/`, `internal/agent/`, or `cmd/` has multiple input variations, when tests are written for that function, then the table-driven pattern (`[]struct{ name string; ... }` with `t.Run`) is used.
- Given a component interacts with gRPC, NATS, or HTTP boundaries, when tests are written for that component, then narrow interfaces are defined at the consumption site and test doubles are implemented in the same test file.
- Given the step verifier checks a completed implementation step, when it validates tests, then `go test ./...` passes and the new code has corresponding test coverage.
- Given the spec writer generates test expectations for a requirement, when the plan generator creates the implementation plan, then test-first step ordering is preserved so that each step begins with a failing test.

## Notes

- This is a greenfield project with zero existing test files — TDD applies from the very first line of code.
- The Makefile provides a `test` target (`make test`) that runs the full test suite.
- This requirement is a cross-cutting policy loaded via the REQ-005 lazy-load catalog. It is not pinned directly but is triggered when sprint scope includes Go implementation work.
