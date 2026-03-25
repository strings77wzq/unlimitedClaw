## Context

Phase 2 wires three capabilities into Golem's runtime: long-term memory (already implemented but disconnected in `feature/memory/`), Telegram dual-channel (polling adapter exists but not connected to `cmd/golem/`), and token estimation accuracy (naive `/4` approximation ignores CJK/multi-byte characters).

The project has strict layering: `foundation/` imports stdlib only; `core/` imports `foundation/`; `feature/` imports `core/` + `foundation/`; `internal/` imports `core/` + `foundation/`; `cmd/` imports all layers and is the sole composition root. Optional features must be wired via CLI flags and adapter files in `cmd/golem/`. CGO_ENABLED=0 is mandatory.

Phase 1 established the pattern: CLI tests, MCP runtime tests, and TUI viewport scrolling are now verified. Phase 2 extends runtime capabilities rather than hardening quality.

## Goals / Non-Goals

**Goals:**
- Enable long-term memory as an optional runtime feature with `--memory` flag, tool-based LLM interaction, and optional automatic context injection.
- Extend the Telegram adapter to support both polling and webhook modes, with webhook handler mountable on the existing gateway server.
- Improve token estimation for CJK text in `core/session/history.go` without breaking context-window behavior or existing tests.

**Non-Goals:**
- Embedding-based memory retrieval or vector search (the memory module uses keyword + decay scoring).
- HTTPS certificate management or Telegram bot token provisioning.
- External tokenizer library integration (CGO_ENABLED=0 constraint prevents tiktoken).

## Decisions

### 1. Memory wiring via adapter pattern + tool registration
- Follow the same `cmd/golem/mcp_adapter.go` / `rag_adapter.go` pattern.
- Create `cmd/golem/memory_adapter.go` with `ParseMemoryConfig()` and `LoadMemoryTools()`.
- Register `memory_store` and `memory_recall` as tool implementations in the global tool registry.
- Optionally inject memory into agent context via `WithMemory()` Option on `core/agent.Agent`.

**Alternatives considered:**
- Hard-coding memory in the agent loop (rejected: violates optional-feature wiring pattern).
- Making memory a `feature/` import from `core/` (rejected: violates layer dependency rules).

### 2. Telegram dual-channel: gateway-mounted webhook handler
- Extend `internal/channels/telegram/adapter.go` with `WebhookHandler() http.HandlerFunc`.
- Add `TelegramConfig` to `core/config/config.go` with mode (polling/webhook), webhook URL, listen addr, secret token.
- Create `cmd/golem/telegram.go` adapter to wire Telegram into the composition root.
- For webhook mode: mount the handler on the existing gateway server via a new `MountHandler(path, handler)` method on `internal/gateway/server.go`.

**Alternatives considered:**
- Standalone Telegram HTTP server (rejected: adds another listen port, increases operational surface).
- Polling-only Telegram command (rejected: does not address webhook requirement).

### 3. CJK-aware token estimation
- Modify `EstimateTokens()` in `core/session/history.go` to detect CJK runes and apply a 2x weight adjustment (CJK characters are roughly 2 tokens each vs. 4 ASCII characters per token).
- Keep `/4` as the default path for pure ASCII content.
- Preserve `GetContextWindow()` signature and system-prompt-preservation behavior.

**Alternatives considered:**
- Model-specific metadata registry (deferred: no existing tokenizer metadata infrastructure, would add significant scope).
- External tokenizer library (rejected: CGO_ENABLED=0 constraint).

## Risks / Trade-offs

- **Token estimation heuristic is not exact** → Mitigate by documenting the approximation and keeping system prompts safe via always-include logic.
- **Memory recall quality depends on keyword overlap** → Mitigate by clearly scoping as keyword-based (not embedding-based) and documenting retrieval semantics.
- **Telegram webhook mode requires external HTTPS exposure** → Mitigate by documenting tunneling options for local development and providing polling fallback.
- **Gateway mount points could collide** → Mitigate by using a well-named path prefix (`/telegram/webhook`) and documenting routing.

## Migration Plan

1. Implement token estimation improvement (low risk, single file change + test update).
2. Wire memory adapter and tools.
3. Extend Telegram adapter with webhook handler and wire into composition root.
4. Run targeted and full regression: `go test ./...`.
5. Manual QA: run `golem agent --memory`, verify Telegram webhook health endpoint.

## Open Questions

- Should automatic memory recall be opt-in via config (`auto_recall: true`) or always-on when memory is enabled?
- What is the default memory persistence path? Suggestion: `~/.golem/memory.json`.
- Should the gateway mount Telegram webhook by default, or require explicit flag?
