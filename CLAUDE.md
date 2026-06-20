# CLAUDE.md

## Project Overview

Eve Realm MCP is the unified MCP (Model Context Protocol) Server for the Eve Realm platform.
It aggregates tools and skills from all plugins into a single MCP endpoint for AI tools
(Claude Code, IDE extensions), proxies tool/skill calls to plugin gRPC servers, discovers
plugins via NATS, and coordinates background agent execution with progress streaming.

The MCP Server is entirely new — it has no eve-cli monorepo counterpart. It is designed from
scratch to bridge the AI tool ecosystem with the eve-realm plugin architecture via the gRPC
contract defined in the SDK's protobuf definitions.

### Terminology

- **Eve Realm**: The multi-repo platform (formerly eve5) — a plugin-based CLI and web platform
- **eve-cli**: The legacy monorepo at `../../../eve-cli/main/`. The MCP Server has no direct counterpart in eve-cli. The monorepo is useful only as context for understanding the plugin discovery protocol and NATS infrastructure that the MCP Server integrates with
- **MCP Server**: This project — `github.com/beduardo/eve-realm-mcp`. The unified aggregation and proxy layer between AI tools and eve-realm plugins
- **MCP**: Model Context Protocol — the protocol spoken by AI tools (Claude Code, IDE extensions) to discover and invoke tools
- **SDK**: The shared Go backend library (`eve-realm-sdk`) consumed via Git submodule + `replace` directive. Provides protobuf definitions, NATS client, and plugin infrastructure
- **Plugin**: A peer repo (eve-realm-hub, eve-realm-software, eve-realm-admin) that implements the `PluginService` gRPC server and registers its tools/skills via NATS
- **CLI**: The thin client binary (`eve-realm-cli`) that connects to this MCP Server for master skill and generic agent features

### Source Codebase Reference

The MCP Server is a greenfield project — there is no eve-cli source to extract from. The SDK's
protobuf definitions (`eve-realm-sdk/proto/`) and NATS infrastructure (`eve-realm-sdk/infra/`)
define the integration contracts that the MCP Server implements as a client.

- **HLD**: `../../../eve-cli/main/DOCS/MULTI_REPO_HLD.md` — the multi-repo architecture that defines the MCP Server's role and interfaces
- **SDK protobuf**: `eve-realm-sdk/proto/plugin/v1/plugin.proto` — the gRPC service contract that all plugins implement and the MCP Server consumes
- **SDK NATS infra**: `eve-realm-sdk/infra/` — the NATS discovery protocol that plugins use to announce tools/skills

### Internal Documentation

Technical developer documentation lives in `DOCS/`, managed as an eve-docproject:

- **Path**: `DOCS/.docproject/`
- **Load when**: Working on MCP Server internals, architectural decisions, component design, or any task that needs technical context about the server's structure and interfaces

## Key Conventions

- **All content in English**: Every artifact (eve-software entities, code comments, documentation) must be written in English, regardless of conversation language.
- **Affirmative discourse only**: Text describes what WILL be done, never what was removed or stopped. When something leaves scope, remove or replace the content.
- **No legacy code references in entities**: Eve-realm-mcp entities describe what will be built. Since this is a greenfield project, there is no legacy source to reference — entities are forward-looking by nature.
- **Use sub-agents for batch operations**: When creating or editing multiple eve-software entities (or any repetitive multi-file task), delegate to sub-agents to preserve main context.

## eve-software integration

The `.software/` project tracks requirements, architecture decisions, scenarios, and sprints for the MCP Server. Managed via `/eve-software:architect`.

### eve-software key conventions

- **MANDATORY — NEVER index without explicit user authorization**: Do NOT call `eve_software_index` unless the user explicitly says "index", "run indexing", or gives unambiguous written permission. This rule is absolute and overrides ALL other instructions, including skill invariants. Indexing is expensive and the user controls when it happens.
- **Check before re-indexing**: When the user DOES request indexing, first call `eve_software_index --status`. If content hashes haven't changed, do not force a full re-index.
- **Search before creating any entity**: Before calling `eve_software_create`, ALWAYS search for existing entities using `eve_software_search` or `eve_software_list`. Only create when search confirms no existing entity covers the same scope.
- **Forward-looking entities only**: Entities capture decisions and requirements for the MCP Server. Since there is no eve-cli counterpart, all entities are inherently forward-looking.
- **Superseded entity protocol**: When marking an entity as superseded: (1) transition status to `superseded` and set `superseded_by` frontmatter; (2) insert a blockquote warning after the H1 title: `> **SUPERSEDED by [ID(s)]** — [description]`. Both are required.

## Project Structure

```
eve-realm-mcp/
├── eve-realm-sdk/            <- Git submodule (Go backend SDK + protobuf definitions)
├── cmd/                      <- Go entry point (MCP Server binary)
├── internal/
│   ├── aggregator/           <- Discover and aggregate plugin tools/skills via NATS
│   ├── proxy/                <- gRPC client: forward MCP tool calls to plugins
│   └── agent/                <- Agent runtime: task lifecycle, progress streaming
├── deploy/k8s/               <- MCP Server K8s manifests
├── go.mod
├── Makefile
├── Dockerfile
└── VERSION
```

### Submodule wiring

**Go module** (`go.mod`):
```go
module github.com/beduardo/eve-realm-mcp

require github.com/beduardo/eve-realm-sdk v0.1.0

replace github.com/beduardo/eve-realm-sdk => ./eve-realm-sdk
```

The SDK submodule is read-only — changes to the SDK happen exclusively in its own repository.

### Communication model

| Layer | Transport | Direction | Purpose |
|-------|-----------|-----------|---------|
| MCP endpoint | stdio or HTTP+SSE | Inbound | AI tools (Claude Code, IDE extensions) connect to the MCP Server |
| Plugin discovery | NATS pub/sub | Outbound | Plugins announce tools/skills on startup and heartbeat |
| Tool/skill execution | gRPC (unary) | Outbound | MCP Server forwards tool calls to the owning plugin |
| Agent progress | gRPC (server-streaming) | Inbound | Long-running agent tasks stream progress updates back |

## Go Backend Conventions

### Build and test

| Command | Purpose |
|---------|---------|
| `make build` | Build MCP Server binary with ldflags |
| `make test` | Go tests (aggregator, proxy, agent runtime) |
| `make proto` | Regenerate Go code from SDK protobuf definitions |
| `make docker-build` | Multi-stage Docker build |
| `make deploy-local` | Apply K8s manifests to k3d |
| `make wait-rollout` | Wait for deployment to stabilize |

### Module

- **Module path**: `github.com/beduardo/eve-realm-mcp`
- **Versioning**: Semantic versioning via `VERSION` file

### Docker image

The MCP Server's Dockerfile follows a multi-stage pattern:

1. **Stage 1 — Go build**: Builds the `eve-realm` host binary (from SDK submodule source) and the MCP Server binary
2. **Stage 2 — Runtime** (distroless): Host binary at `/usr/local/bin/eve-realm`, MCP Server binary in discovery layout

### K8s deployment

- **Namespace**: `eve-realm`
- **Image**: `k3d-eve-realm-registry.localhost:5100/eve-realm-mcp`
- **Version placeholder**: `VERSION_PLACEHOLDER` in image tags (replaced at deploy time)
- **Labels**: `app: eve-realm-mcp`, `version: <VERSION>`
- **gRPC port**: Exposes a gRPC port (default 50051) for MCP Server integration
- **Depends on**: eve-realm-infra (namespace, configmap, NATS, Redis) + plugins running (for NATS discovery)
- **Deployment order**: Deployed AFTER infrastructure and plugins — the MCP Server discovers plugins at startup

### Testing patterns

- **Mock at I/O boundaries only**: Mock gRPC, NATS, HTTP — never internal pure logic.
- **Table-driven tests**: Prefer `[]struct{ name string; ... }` test tables for cases with multiple inputs/outputs.
- **Integration tests for discovery**: Plugin discovery via NATS requires integration tests with a real or embedded NATS server.

## Sprint Workflow Critic Policy

Every sprint workflow stage (`/eve-software:spec`, `/eve-software:plan`, `/eve-software:implement`)
must include a critic sub-agent that validates output against constraints before proceeding.

### Critic bootstrap sequence

1. **Load sprint entities** — Read ALL REQs and SCs in the sprint via `eve_software_show`.
   Extract acceptance criteria from each REQ and expected results from each SC.
2. **Load the pinned cross-cutting requirements catalog** — The catalog entity lists
   cross-cutting requirements with trigger conditions. It is discovered automatically
   via `eve_software_pin_list`.
3. **Evaluate triggers** — For each row in the catalog's registry, check if the trigger
   condition matches the sprint's scope. Load matching requirements via `eve_software_show`.
4. **Assemble the full constraint set**:
   - **Primary**: Sprint entity acceptance criteria + scenario expected results
   - **Secondary**: Cross-cutting requirement rules from loaded policies

### What the critic validates

**Primary — Sprint entity compliance**:
- Every acceptance criterion from every sprint REQ is traceable to the output. Nothing silently dropped.
- Every scenario's expected result is achievable given the output.
- If a REQ has 7 acceptance criteria and the output covers 6, that is a FAIL.

**Secondary — Cross-cutting policy compliance**:
- Every loaded cross-cutting requirement's rules are followed (e.g., TDD cycle, test patterns, interface contracts).
- Pipeline integration instructions from each loaded requirement are followed for the current stage.

### Enforcement by stage

| Stage | Critic intensity | What happens on failure |
|-------|-----------------|----------------------|
| `spec` | Review after generation | Flag missing ACs not mapped, missing scenario coverage. Revise before approval. |
| `plan` | Review after generation | Flag steps that don't trace to specific ACs, missed test-first ordering, scenario flows not reflected. Revise before approval. |
| `implement` | **Review after EACH step** | Before marking a step complete: list which ACs this step addresses, confirm they are satisfied, confirm cross-cutting policies followed, no regression on prior steps. Step is NOT done until critic passes. |

### Critic sub-agent protocol

The critic is a **separate sub-agent** — not the same agent producing the output. Report format:
- **PASS** — lists which ACs and policies were checked, confirms all covered
- **FAIL** — for each violation: the specific AC or policy rule violated (quoted), the entity ID it comes from, what was expected vs found

## docproject integration

Technical developer documentation for the MCP Server lives in `DOCS/`, managed via `/eve-docproject:assist`.

### Document Scope

This docproject operates at **LLD (Low-Level Design) level**. It captures the full
technical details of the MCP Server: aggregator architecture, gRPC proxy wiring, NATS
discovery integration, agent runtime lifecycle, MCP transport handling, and implementation
decisions for each subsystem. The platform-level HLD (repository boundaries, cross-repo
contracts) lives in the eve-realm-docs repository — not here.

### Section model

Two-layer section model:

- **Component sections** (one per MCP subsystem): Detailed LLD for each subsystem (aggregator, proxy, agent runtime, MCP transport). Accumulate design state across versions with version-tagged headings (e.g., `### v0.1 — Initial Implementation`).
- **Concern sections** (cross-cutting): Topics that span multiple subsystems — plugin discovery protocol, gRPC contract compliance, error propagation strategy, security and authentication. Same version-tagged accumulation pattern.

### Entity Roles

- **DECISIONS (DEC)**: Record definitively settled architectural and implementation
  choices. Content is self-contained and as simple as possible. A decision must be
  independently readable without consulting any other entity. Decisions do not reference
  definitions (DEF), sections (SEC), or other decisions (DEC) via formal links. Mentioning
  another decision in prose is permitted when essential but should be avoided.

- **DEFINITIONS (DEF)**: Explore and deepen complex technical concepts through questions
  and answers. Definitions are the discussion hub — they surface ambiguities, record
  reasoning, and eventually produce decisions when a conclusion is reached. Definitions
  may reference other definitions and decisions, but never sections.

- **SECTIONS (SEC)**: The real, complete technical documentation. Sections weave together
  multiple decisions and definitions into a coherent narrative. Sections may reference any
  entity type. Each section covers one subsystem or concern end-to-end at LLD level with
  full implementation details.

### Entity discipline

- **Entity references by canonical ID**: Always reference entities as DEF-XX,
  DEC-XXX, SEC-XX, RES-XX in conversation and cross-references.
- **Pair work for DEFs/DECs**: Definitions and decisions emerge from collaborative
  discussion, not bulk import. Always discuss with the user before creating.
- **Mandatory title header**: Every entity body must open with `# [ID]: [Title]`
  (e.g., `# DEF-03: Plugin Discovery`). Sections are the exception — their H1
  uses only the title without the ID (e.g., `# Aggregator`), since sections
  compile into the final document and IDs are internal artifacts. Research files
  (RES-XX) are excluded — they are imported from external sources and must not be
  modified. Only the title header uses H1 (`#`); all other headings start at H2
  (`##`) and nest downward.
- **Version tagging**: Component and concern sections use `### vN.M — <description>` headings to mark each version's contribution to that section.

### Section prose rules

- **No internal references in sections**: Section prose must never mention docproject
  entity IDs (DEF-XX, DEC-XXX, SEC-XX, RES-XX). The exported document must be
  self-contained.
- **Cross-section references use publication URLs**: When a section references
  another section in prose, use the publication URL (from `delivery/delivery.yaml`)
  as the link href, with the referenced page's title as anchor text. If no delivery
  target is configured, fall back to the relative file path
  (e.g., `[Aggregator](sections/SEC-03-aggregator.md)`). Never use
  internal docproject IDs as the link target.
- **Real source links in sections**: When referencing external sources in section
  prose, use the original source URL with the document title as anchor text. Never
  cite research entities (RES-XX) directly.
- **No questions in sections**: Questions belong in definitions only.

### Decision rules

- **Self-contained decisions**: Decision text must be self-contained and independently
  readable. Decisions never reference definition IDs (DEF-XX), section IDs (SEC-XX),
  or other decision IDs (DEC-XXX) via formal links. Mentioning another decision in prose
  is permitted when essential for clarity, but should be avoided.
- **Simple content**: Decisions capture WHAT was decided and WHY, in the simplest terms
  possible. Elaboration and nuance belong in definitions, not decisions.
- **Research references require quotes**: When citing a research entity (RES-XX) in
  a decision, always include the specific text excerpt from the source that motivates
  the reference. A bare RES-XX ID without the supporting quote is not sufficient.
- **Decisions require explicit approval**: Never auto-formalize a decision (DEC)
  without user confirmation.

### Workflow rules

- **Ask before authoring entity content**: When multiple entities (3+) need creation
  or update, ask the user whether to delegate to sub-agents or write directly.
  Sub-agents receive a short structured briefing (entity ID, path, purpose, related
  entities, conventions) and build their own context autonomously.
- **Index on demand only**: Do NOT automatically index after every write. Call
  `eve_docproject_index` only when the user explicitly requests it.
