## 1. Token estimation improvement

- [x] 1.1 Modify `core/session/history.go` to add CJK-aware `EstimateTokens()` with rune detection and weighted counting
- [x] 1.2 Update `core/session/history_test.go` expectations to reflect the new estimation logic
- [x] 1.3 Run `go test ./core/session` and confirm tests pass with accurate CJK estimation

## 2. Memory runtime wiring

- [x] 2.1 Create `cmd/golem/memory_adapter.go` with `ParseMemoryConfig()` and `LoadMemoryTools()` following the RAG/MCP adapter pattern
- [x] 2.2 Implement `memoryStoreTool` and `memoryRecallTool` satisfying the `tools.Tool` interface
- [x] 2.3 Add `--memory` CLI flag to `cmd/golem/main.go` and wire memory tools into the agent registry
- [x] 2.6 Run `go test ./feature/memory` and `go test ./cmd/golem` and confirm tests pass

## 3. Telegram dual-channel

- [x] 3.1 Add `TelegramConfig` struct to `core/config/config.go` with mode, token, webhook URL, secret token
- [x] 3.2 Extend `internal/channels/telegram/adapter.go` with webhook handler method and mode switching
- [x] 3.3 Add `MountHandler(path, handler)` method to `internal/gateway/server.go`
- [x] 3.4 Create `cmd/golem/telegram_adapter.go` adapter wiring file following the MCP/RAG pattern
- [x] 3.5 Add `--telegram` CLI flag to `cmd/golem/main.go` and wire Telegram adapter
- [x] 3.6 Run `go test ./internal/channels/telegram` and `go test ./internal/gateway` and confirm tests pass

## 4. Regression verification

- [x] 4.1 Run `go test ./...` and confirm all targeted packages pass
- [x] 4.2 Manual QA: start `golem agent --memory` and verify memory tools are registered
- [x] 4.3 Manual QA: start `golem agent --telegram` in polling mode and verify adapter starts
- [x] 4.4 Manual QA: verify token estimation improvement by running CJK content through the agent
