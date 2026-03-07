# AGENTS.md — Golem AI Collaboration Context

> This file is the **first thing AI tools should read** when opening this repository.
> It contains the project context, architecture rules, hard constraints, and working principles
> that govern all AI-assisted development in this codebase.

---

## 1. Project Overview

**Golem** is a progressive Go AI assistant learning project, inspired by
[PicoClaw](https://github.com/sipeed/picoclaw). Its purpose is two-fold:

1. **Functional**: A working AI agent CLI that supports multiple LLM providers, tool calling,
   streaming output, session persistence, and a Bubble Tea TUI.
2. **Educational**: A structured learning project for intermediate Go developers to understand
   hexagonal architecture, concurrency patterns, and AI agent design.

**Current stable version**: v0.3.0  
**Target platform**: Linux amd64 + Android/Termux ARM64  
**Build constraint**: `CGO_ENABLED=0` — pure Go, zero CGO dependencies, single static binary

---

## 2. Repository Layout

```
golem/
├── cmd/golem/         # Composition root — wires all layers; cobra CLI entry point
├── core/                  # Domain logic — agent, bus, config, providers, session, tools, usage
├── foundation/            # Infrastructure primitives — concurrency, logger, store, term
├── feature/               # Reference implementations (NOT wired into main.go — learning only)
│   ├── mcp/               # MCP protocol client (JSON-RPC over stdio)
│   ├── memory/            # Long-term memory with importance decay
│   ├── rag/               # RAG pipeline with TF-IDF + embeddings
│   ├── routing/           # Provider fallback routing
│   └── skills/            # Composable skill registry
├── internal/              # Internal adapters (only importable within this module)
│   ├── channels/cli/      # Plain readline-style interactive mode
│   ├── channels/tui/      # Bubble Tea TUI (auto-activated on TTY)
│   ├── channels/telegram/ # Telegram bot adapter
│   ├── gateway/           # HTTP gateway server with SSE streaming
│   ├── metrics/           # Prometheus-compatible metrics (no external deps)
│   └── security/          # Auth middleware, rate limiting, command sandbox
├── openspec/              # AI development specification (read this too)
└── docs/study/            # Chinese learning guides (Wave 1–3)
```

---

## 3. Layer Dependency Rules

These rules are **enforced by Go's import system** and must never be violated.

```
cmd/         → imports ALL layers (composition root only)
internal/    → imports core/ + foundation/ only
core/        → imports foundation/ only (never internal/ or feature/)
feature/     → imports core/ + foundation/ (standalone; not imported by main.go)
foundation/  → imports stdlib only (zero project dependencies)
```

**Forbidden cross-layer imports:**
- `foundation/` MUST NOT import `core/`, `internal/`, or `feature/`
- `core/` MUST NOT import `internal/`, `feature/`, or `cmd/`
- `feature/` MUST NOT import `internal/` or `cmd/`
- `internal/` MUST NOT import `feature/` or `cmd/`

**Circular imports are always wrong.** Use interfaces (ports) in `core/` to break any cycle.

---

## 4. Hard Constraints

| Constraint | Rule |
|---|---|
| **CGO** | `CGO_ENABLED=0` always. No CGO dependencies ever. Use `modernc.org/sqlite` for SQLite. |
| **LLMProvider interface** | MUST NOT modify the `LLMProvider` or `StreamingProvider` interface signatures in `core/providers/types.go`. All adapters must satisfy these interfaces without changes. |
| **Bubble Tea isolation** | Bubble Tea (`github.com/charmbracelet/bubbletea`) MUST be isolated to `internal/channels/tui/` only. No other package may import it. |
| **TUI options** | Use `tea.WithContext(ctx)` for context propagation. Do NOT use `WithoutMouseCellMotion` — it does not exist in v1.3.10. The default (no mouse option) is correct for Termux. |
| **No Alt+key shortcuts** | MUST NOT use Alt+key combinations in TUI keybindings — they break on Termux/Android. |
| **Error handling** | Return errors; never panic in library code. Always wrap: `fmt.Errorf("context: %w", err)`. |
| **No suppressed types** | Never use `as any` casts to silence type errors, `//nolint` without explanation, or empty catch-equivalent patterns. |

---

## 5. Key Interfaces (Do Not Modify Signatures)

```go
// core/providers/types.go — the LLM contract
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition,
        model string, opts *ChatOptions) (*LLMResponse, error)
    Name() string
}

type StreamingProvider interface {
    LLMProvider
    ChatStream(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition,
        model string, opts *ChatOptions, onToken func(token string)) (*LLMResponse, error)
}

// core/agent/agent.go — how channels talk to the agent
type MessageHandler interface {
    HandleMessage(ctx context.Context, sessionID string, message string) (string, error)
    HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
}

type Runner interface {
    Start(ctx context.Context)
}
```

---

## 6. Architecture Decision Log (Critical WHYs)

### Why `feature/` is not wired into `main.go`
`feature/` packages are **standalone learning modules** — complete, tested, but intentionally
disconnected from the running binary. They exist to demonstrate advanced patterns (MCP, RAG,
memory) without coupling the core agent to optional dependencies. Wire them in only if building
a production extension.

### Why tools are always sorted alphabetically in the registry
`ListTools()` / `ListDefinitions()` always return tools in alphabetical order. This is
**intentional and critical**: it maximizes LLM KV-cache reuse across requests. If tool order
changes between calls, the LLM cannot reuse its cached prefix, which increases latency and cost.

### Why streaming uses `canStream := ok && streamFinal && len(toolDefs) == 0`
Streaming is disabled when tools are available because mid-stream tool calls would require
buffering the entire stream to parse JSON tool-call arguments, eliminating any latency benefit.
The streaming contract only applies to final text responses. After any tool-use turn, the agent
falls back to `Chat()` (non-streaming) for the final answer, then delivers it as one token chunk.

### Why `waitNextToken` uses recursive Cmd in Bubble Tea
Bubble Tea follows the Elm architecture: side effects are values (`tea.Cmd`), not callbacks.
`waitNextToken` returns a `tea.Cmd` that blocks on `<-tokens`; when a token arrives, it emits
a `tokenMsg`, triggering `Update`, which then calls `waitNextToken` again. This creates a
**recursive Cmd chain** — the idiomatic Bubble Tea way to consume a channel without goroutines
or timers inside `Update`.

### Why `bufio.Reader` in `onboard.go` instead of `fmt.Scan`
`bufio.Reader.ReadString('\n')` handles the full line including spaces and special characters
robustly. `fmt.Scan` stops at whitespace, which would break API key input that contains spaces.

---

## 7. Streaming Architecture

```
TUI (tui.go)                    Agent (loop.go)
┌──────────────┐                ┌──────────────────────────────┐
│ handleKey    │                │ HandleMessageStream          │
│ (KeyEnter)   │                │  └─ processMessage           │
│   │          │                │       └─ invokeProvider      │
│   ├─ startStream ─────────────►         (StreamingProvider?) │
│   │  goroutine │  tokens chan  │           │                  │
│   │            ◄──────────────┤  onToken(tok) → tokens <- tok│
│   └─ waitNextToken            │                              │
│      (recursive Cmd)          │ close(tokens) on return      │
└──────────────┘                └──────────────────────────────┘
```

- `tokens` is a `chan string` with buffer 64 (TUI) or 32 (gateway)
- `HandleMessageStream` always `defer close(tokens)` — consumers range over it safely
- Non-streaming fallback: if `streamed==false` after processing, entire content is sent as one token

---

## 8. Current Project State

| Metric | Value |
|---|---|
| Version | v0.5.0 |
| Packages | 28 |
| Test coverage | 79.2% |
| CI status | ✅ Green |
| Go version | 1.25+ |
| Active channels | CLI, TUI, HTTP Gateway |
| Providers wired | OpenAI, Anthropic, DeepSeek, Kimi, GLM, MiniMax, Qwen |

---

## 9. Working Principles

- **First principles**: Understand *why* before *how*. If the motivation is unclear, stop and
  discuss rather than guessing.
- **Path optimization**: If you see a better path than what was described, say so explicitly.
  Don't silently implement an inferior approach.
- **One pass, no partial state**: Fix everything in one pass. Never leave the codebase in a
  broken or inconsistent intermediate state.
- **Chinese for conversation**: All conversational responses to the user in this project are
  in Chinese (Simplified). Code, comments, and commit messages remain in English.
- **Parallel agent delegation**: Use background `task()` calls with `run_in_background=true`
  for independent exploration (explore/librarian agents). Collect results via `background_output`
  after the system notification fires. Do NOT block on results — end the response and wait.

---

## 10. Common Commands

```bash
# Build (pure Go, Termux-compatible)
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o build/golem ./cmd/golem

# Or via Makefile
make build

# Run all tests
go test ./...

# Race detector
go test -race ./...

# Vet
go vet ./...

# Lint (requires golangci-lint)
golangci-lint run

# First-run setup wizard
./build/golem init

# Start TUI agent
./build/golem agent

# One-shot query
./build/golem agent -m "Hello"

# Start HTTP gateway (port 18790)
./build/golem gateway
```

---

## 11. How to Add a New LLM Provider

1. Create `core/providers/<vendor>/vendor.go` implementing `LLMProvider` (and optionally
   `StreamingProvider`).
2. Register in `cmd/golem/main.go`:`registerProviders()` — add a `case "vendor":` block.
3. Add a preset entry in `cmd/golem/onboard.go`:`providerPresets` if it has a
   well-known API base URL.
4. Add at least one test in `core/providers/<vendor>/vendor_test.go`.
5. Do **not** modify `LLMProvider` or `StreamingProvider` interface signatures.

## 12. How to Add a New Tool

1. Create `core/tools/<toolname>/<toolname>.go` implementing the `tools.Tool` interface.
2. Register in `cmd/golem/main.go`:`buildToolRegistry()`.
3. Return a meaningful `ToolResult.ForUser` string for user-visible output; put the LLM context
   in `ToolResult.ForLLM`.
4. Add tests — tools are pure functions and should be unit-tested without a real agent.
