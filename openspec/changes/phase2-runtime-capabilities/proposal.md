## Why

Golem has strong core infrastructure (agent loop, tool registry, provider abstraction, session management) but several runtime capabilities remain unbuilt or unconnected. The long-term memory module (`feature/memory`) is implemented but not wired into the agent, the Telegram adapter only supports polling and is not connected to the composition root, and token estimation uses a naive `/4` approximation that significantly underestimates CJK text. Phase 2 connects these capabilities and makes them production-useful.

## What Changes

- Wire `feature/memory` into the agent runtime as an optional feature with a `--memory` CLI flag, `memory_store`/`memory_recall` tools, and optional automatic memory injection into the agent context.
- Extend `internal/channels/telegram/adapter.go` to support both polling and webhook modes; add `TelegramConfig` to `core/config/config.go`; expose a webhook HTTP handler that can be mounted on the existing gateway server.
- Improve `core/session/history.go` token estimation to account for CJK runes and multi-byte characters, reducing context-window overflow risk for non-ASCII workloads while preserving backward compatibility.

## Capabilities

### New Capabilities

- `memory-runtime-wiring`: Defines how the optional long-term memory module is enabled, how the agent and LLM interact with it via tools, and how memory is persisted across sessions.
- `telegram-dual-channel`: Defines how the Telegram channel adapter supports polling and webhook modes, including configuration, gateway integration, and operational behavior.
- `token-estimation-accuracy`: Defines how the agent estimates message token counts for context-window truncation, including CJK-aware heuristics and backward-compatible fallbacks.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/golem/main.go`, `cmd/golem/memory_adapter.go` (new), `core/config/config.go`, `core/agent/agent.go`, `core/agent/loop.go`, `core/session/history.go`, `core/session/history_test.go`, `internal/channels/telegram/adapter.go`, `internal/gateway/server.go`, related test files.
- Affected user flows: agent startup with `--memory`, Telegram bot setup in polling/webhook mode, long-session context-window accuracy for CJK content.
- Dependencies: existing `feature/memory` module, `internal/gateway` HTTP server, `core/bus` event bus.
