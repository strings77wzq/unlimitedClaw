## Context

Phase 1 addresses three separate but related quality gaps in the current codebase. First, `cmd/golem` is the composition root for provider registration, command wiring, config loading, session resolution, and feature flag integration, but it currently has no automated tests. Second, `feature/mcp` exposes a full MCP runtime path but only verifies a narrow subset of protocol serialization and happy-path client behavior; transport lifecycle, manager orchestration, and proxy execution remain largely untested. Third, `internal/channels/tui/tui.go` streams responses correctly but renders the transcript as a flat string without viewport state, which prevents users from reviewing long conversations.

These three tasks cross module boundaries (`cmd`, `feature`, and `internal`) but do not require new external dependencies. The implementation must preserve the existing pure-Go build, Bubble Tea v1.3.10 compatibility, and current agent/TUI streaming behavior.

## Goals / Non-Goals

**Goals:**
- Add deterministic automated tests for high-value `cmd/golem` behaviors without introducing flaky end-to-end network dependencies.
- Raise `feature/mcp` coverage by targeting currently unverified lifecycle and error-path code rather than padding easy helper tests.
- Add transcript scrolling to the TUI with viewport-based rendering, resize handling, and bottom-follow semantics that work with streamed tokens.
- Keep the resulting design maintainable by introducing focused helpers instead of broad refactors.

**Non-Goals:**
- Reworking the CLI command surface or changing public flags/command names.
- Changing the MCP protocol surface or adding new MCP features beyond better verification.
- Redesigning the TUI layout, theming, or input model beyond the minimum viewport changes needed for scrollability.
- Introducing browser-based UI, alternate terminal frameworks, or new runtime dependencies.

## Decisions

### 1. Test `cmd/golem` primarily through Cobra command execution and focused helpers
- Use `NewRootCommand()` and `cmd.SetArgs(...)` to verify command wiring and output for `version`, `config`, `session`, and status-oriented flows.
- Add small helper seams only where current code is otherwise difficult to observe deterministically, such as temp config paths, temporary session databases, and captured output writers.
- Keep tests in `cmd/golem/*_test.go` so the coverage applies to the actual composition root rather than to extracted wrappers.

**Alternatives considered:**
- Extracting large portions of `main.go` into new packages would improve testability but would expand scope from verification to architecture refactoring.
- Shelling out to built binaries for every command would be closer to black-box testing but slower and harder to keep deterministic across CI platforms.

### 2. Raise MCP coverage by layering tests from the protocol edge inward
- Add transport-level tests around `StdioTransport` lifecycle and message I/O using lightweight subprocess fixtures or pipe-backed helpers.
- Extend client tests to cover initialization guards, JSON-RPC error responses, transport failures, and close behavior.
- Add manager lifecycle tests for `Start`, `DiscoverTools`, `Close`, `callTool`, and `MCPToolProxy.Execute()` so the module is verified at its real integration boundaries.

**Alternatives considered:**
- Mocking everything below `Manager` would be fast but would miss exactly the untested transport and orchestration code that currently drives the coverage gap.
- Relying only on a full external MCP server process would provide realism but would make tests brittle and harder to debug.

### 3. Use Bubble Tea viewport as the sole scrolling abstraction in the TUI
- Introduce `viewport.Model` into the TUI model and render transcript content through `viewport.View()` rather than printing the raw joined transcript.
- Initialize viewport dimensions lazily on `tea.WindowSizeMsg`, then update width/height on later resize messages.
- Before appending new assistant/user content, capture whether the viewport is already at the bottom; after content refresh, auto-scroll only when the user was already following the bottom.
- Preserve existing non-Alt keybindings and add scroll navigation using viewport-compatible keys: arrow keys, `PgUp`, `PgDn`, `Ctrl+U`, `Ctrl+D`, `Home`, and `End`.

**Alternatives considered:**
- Manual offset bookkeeping inside the TUI model would duplicate viewport behavior and make resize math harder to maintain.
- Switching to a list component would force a more invasive redesign of message rendering and streaming updates.

### 4. Verify each Phase 1 stream independently, then run a combined regression pass
- `cmd/golem`: run targeted `go test ./cmd/golem`.
- `feature/mcp`: run targeted `go test ./feature/mcp` and confirm coverage materially improves.
- TUI: run `go test ./internal/channels/tui` and extend tests for resize/scroll/bottom-follow logic.
- Regression: run `go test ./...` after all three changes land to ensure the composition root, MCP module, and TUI changes do not break surrounding packages.

**Alternatives considered:**
- Running only full-suite tests at the end would make debugging slower and obscure which stream introduced a regression.

## Risks / Trade-offs

- **Transport tests may become flaky across platforms** -> Prefer deterministic helper processes/pipes and avoid timing-sensitive assertions.
- **Cobra tests can be noisy because commands write to stdout/stderr directly** -> Standardize captured buffers in test helpers and avoid asserting full output blobs when only key lines matter.
- **Viewport integration can break current transcript rendering or input flow** -> Keep message storage unchanged, restrict changes to render/update paths, and add regression tests for current send/receive behavior.
- **Auto-scroll can fight the user during manual review** -> Only call `GotoBottom()` when the viewport was already at bottom before the new content arrived.

## Migration Plan

1. Land test helpers and command tests in `cmd/golem`.
2. Land MCP transport/client/manager coverage improvements.
3. Land viewport integration and TUI tests.
4. Run targeted package tests, then `go test ./...`.
5. If TUI behavior regresses, revert the viewport-specific changes while keeping the new tests in place for the other two streams.

## Open Questions

- Whether `cmd/golem` needs an explicit command-output helper to reduce repeated stdout capture boilerplate.
- Whether `feature/mcp` transport tests should use a committed helper binary or a test-local Go subprocess launched via `go test`.
