## 1. Command-layer test coverage

- [x] 1.1 Add shared command test helpers for temporary config paths, captured stdout/stderr, and temporary session databases in `cmd/golem/*_test.go`.
- [x] 1.2 Add root command execution tests for `version` and invalid subcommand behavior via `NewRootCommand()`.
- [x] 1.3 Add focused tests for command-layer helpers in `cmd/golem/main_test.go`, including `registerProviders`, `listModelNames`, `loadConfig`, `buildToolRegistry`, and `resolveSessionID`.
- [x] 1.4 Add integration-style tests for `config` command flows in `cmd/golem/config_test.go` using a temporary config file.
- [x] 1.5 Add integration-style tests for `session` command flows in `cmd/golem/session_test.go` using a temporary SQLite session store.
- [x] 1.6 Run `go test ./cmd/golem` and confirm meaningful coverage is reported for the package.

## 2. MCP runtime hardening

- [x] 2.1 Add transport lifecycle tests in `feature/mcp/transport_test.go` for start, send, send-notification, receive, lifecycle misuse, and close behavior.
- [x] 2.2 Extend client tests in `feature/mcp/client_test.go` or `feature/mcp/mcp_test.go` to cover not-initialized guards, transport failures, JSON-RPC errors, context cancellation, and close delegation.
- [x] 2.3 Add manager lifecycle tests for `Start`, `DiscoverTools`, `Close`, and `callTool` using deterministic MCP fixtures.
- [x] 2.4 Add `MCPToolProxy.Execute()` tests that verify success formatting and error propagation into `ToolResult`.
- [x] 2.5 Run `go test ./feature/mcp -cover` and confirm coverage reaches the Phase 1 target range.

## 3. TUI scrollable history

- [x] 3.1 Refactor `internal/channels/tui/tui.go` to store transcript content in a Bubble Tea viewport while preserving the current input and streaming model.
- [x] 3.2 Add `tea.WindowSizeMsg` handling for viewport initialization and resize updates.
- [x] 3.3 Add scroll navigation bindings for arrow keys, `PgUp`, `PgDn`, `Ctrl+U`, `Ctrl+D`, `Home`, and `End`.
- [x] 3.4 Implement bottom-follow behavior that auto-scrolls only when the viewport was already at the bottom before new content arrived.
- [x] 3.5 Extend `internal/channels/tui/tui_test.go` to cover viewport sizing, scroll navigation, and preserved manual position during streamed updates.
- [x] 3.6 Run `go test ./internal/channels/tui` and manually exercise a long transcript in `golem agent` to verify scrolling behavior.

## 4. Regression verification

- [x] 4.1 Run `go test ./...` after all Phase 1 changes are complete.
- [x] 4.2 Review resulting coverage/output and document whether the three target gaps (`cmd/golem`, `feature/mcp`, `internal/channels/tui`) were closed as planned.
