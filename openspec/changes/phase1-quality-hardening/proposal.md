## Why

Phase 1 targets three high-risk gaps in the current project: the `cmd/golem` composition root has no test protection, the MCP feature has critical lifecycle paths that are effectively unverified, and the Bubble Tea TUI cannot scroll through long conversations. These gaps reduce confidence in releases and leave a core interactive workflow visibly incomplete.

## What Changes

- Add command and integration test coverage for `cmd/golem` so root command wiring, config/session flows, and provider registration are exercised through realistic command execution.
- Expand `feature/mcp` test coverage across transport, client error paths, manager lifecycle, and MCP tool proxy execution to raise confidence from partial protocol coverage to implementation-level reliability.
- Add scrollable conversation history to the TUI using Bubble Tea viewport patterns, including resize handling, bottom-follow behavior, and keyboard navigation for long sessions.
- Add verification steps and supporting test helpers so these improvements remain maintainable as new commands and features are added.

## Capabilities

### New Capabilities
- `cli-command-testing`: Defines required automated coverage for the Cobra composition root, command execution paths, and config/session command behavior.
- `mcp-runtime-hardening`: Defines required verification for MCP transport, client lifecycle, manager orchestration, and tool proxy execution paths.
- `tui-scrollable-history`: Defines required TUI behavior for scrollable transcript viewing, viewport resizing, and follow-bottom interaction during streaming output.

### Modified Capabilities
- None.

## Impact

- Affected code: `cmd/golem/*.go`, `feature/mcp/*.go`, `internal/channels/tui/tui.go`, related `_test.go` files, and new OpenSpec change artifacts.
- Affected user flows: CLI command execution, MCP-enabled agent startup, and long-running TUI conversations.
- Dependencies/systems: Cobra command tests, MCP JSON-RPC transport behavior, Bubble Tea viewport integration, Go test coverage targets in CI.
