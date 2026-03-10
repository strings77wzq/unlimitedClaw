# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.5.1] - 2026-03-10
### Security Fixes
#### Exec Tool Command Injection Prevention
- Rewrote `core/tools/exec/exec.go` with default direct command execution (no shell interpretation) to prevent command injection attacks
- Implemented 4-tier security mode system: Sandbox (default), Allowlist, Denylist, Unrestricted
- Added 70+ default safe command allowlist and dangerous operation denylist
- Added `WithAllowShell()` option for explicit shell access (only for trusted environments)
- Updated all unit tests with comprehensive security test cases
#### Gateway Security Enhancements
- Added built-in Bearer token authentication for HTTP gateway (via `GOLEM_AUTH_TOKEN` environment variable or config)
- Implemented configurable rate limiting (default 100 req/s, 200 burst)
- Added configurable CORS origin restriction (default: only allow localhost/127.0.0.1)
- Updated all gateway middleware to apply security policies by default
#### Error Handling Fixes
- Fixed ignored JSON marshal error in `core/providers/openai/openai.go`
- Added error logging for ignored `processMessage()` errors in `core/agent/loop.go`
### Changed
- All existing tests pass, coverage maintained at 79.3%
- Updated `docs/SECURITY.md` with new security feature documentation
- Updated openspec specifications to align with new security capabilities

## [0.5.0] - 2026-03-09

### Added
#### Feature Module Wiring
- **MCP Client Integration**: Added `--mcp` flag to load external Model Context Protocol servers via STDIO
  - Auto-registers all tools from configured MCP servers into the global tool registry
  - Supports multiple MCP server configurations via JSON array
- **RAG Pipeline Integration**: Added `--rag` flag to enable Retrieval-Augmented Generation
  - Supports indexing directories of text files or explicit JSON document lists
  - Auto-registers `rag_retrieve` tool for semantic document search
  - Uses TF-IDF + cosine similarity by default, compatible with OpenAI embeddings
- **Skills System Enhancement**: Added `--skills` flag to dynamically load built-in skills
  - Auto-registers `summarize` and `code-review` built-in skills
  - Supports comma-separated skill selection: `--skills summarize,code-review`
  - Skills are available as LLM-callable tools with proper parameter schemas

### Changed
- `feature/` modules (MCP/RAG/Skills) are now officially wired into the main binary, no longer just reference implementations
- All existing tests pass, coverage maintained at 79.2%
- Updated documentation to reflect new CLI flags and features

## [0.3.0] - 2026-03-07

### Added

#### TUI Channel (`internal/channels/tui/`)
- Interactive Bubble Tea TUI with real-time token streaming display
- Auto-detection: opens TUI when stdin is a TTY; falls back to plain interactive mode on pipes/redirects
- `--no-tui` flag on `agent` command to force plain mode
- Termux-safe: no mouse options (default behaviour), no Alt+key shortcuts
- Key bindings: `ctrl+c`/`esc` to quit, `enter` to send, `backspace` to delete
- 11 unit tests covering model state transitions and command generation

#### First-run Setup Wizard (`golem init`)
- Interactive `init` command for first-time configuration
- 7 provider presets: OpenAI, Anthropic, DeepSeek, Moonshot/Kimi, Zhipu/GLM, MiniMax, DashScope/Qwen
- Prompts for API key, base URL override, and default model selection
- Writes `~/.golem/config.json` with safe file permissions (0600)
- Skips fields left blank (preserves defaults)

#### Streaming Fixes (`core/agent/loop.go`)
- Fixed `invokeProvider` double-call bug: tools present → always `Chat` (sync); no tools → `ChatStream`
- `HandleMessageStream` now emits a single-chunk fallback if no tokens were streamed (tool-assisted responses)
- `handleMessage` (bus path) uses `streamFinal=false` to avoid streaming into the message bus

### Changed
- `agent` command in `main.go` wires TUI automatically on TTY
- `main.go` registers `newInitCommand()` in the root cobra command
- 28 test packages now pass (added `internal/channels/tui` package with 11 tests)
- README updated: TUI in features list, `feature/` marked as reference implementations, `tui/` added to project structure, `init` command added to Quick Start, Bubble Tea added to tech stack table

## [0.2.1] - 2026-03-07

### Changed

#### Architecture Refactoring (Wave 2.5)
- Reorganized flat `pkg/` into 4-layer architecture: `core/`, `foundation/`, `feature/`, `internal/`
- `core/`: agent, bus, config, providers, session, tools, usage (7 packages — domain logic)
- `foundation/`: concurrency, logger, store, term (4 packages — infrastructure primitives)
- `feature/`: mcp, memory, rag, routing, skills (6 packages — optional modules)
- `internal/`: channels, gateway, metrics, security (5 packages — internal-only)
- Updated all import paths across 40+ source files (63 occurrences)
- Updated all documentation to reflect new directory structure
- Removed empty `pkg/` directory
- All 27 test packages continue to pass, 79.2% coverage maintained

## [0.2.0] - 2026-03-07

### Added

#### Streaming
- StreamingProvider interface with `ChatStream()` method for token-by-token streaming
- OpenAI SSE streaming implementation with tool call accumulation and usage tracking
- Anthropic SSE streaming implementation with event-based state machine

#### Chinese LLM Providers
- DeepSeek provider (`deepseek/deepseek-chat`, `deepseek/deepseek-reasoner`)
- Moonshot/Kimi provider (`moonshot/moonshot-v1-8k`, `moonshot/moonshot-v1-32k`, `moonshot/moonshot-v1-128k`)
- Zhipu/GLM provider (`zhipu/glm-4`, `zhipu/glm-4-flash`, `zhipu/glm-4-plus`)
- MiniMax provider (`minimax/MiniMax-Text-01`, `minimax/abab6.5s-chat`)
- DashScope/Qwen provider (`dashscope/qwen-plus`, `dashscope/qwen-turbo`, `dashscope/qwen-max`)
- All Chinese providers use OpenAI-compatible API format via `openai.New()` with `WithAPIBase()`

#### Session Management
- `-C` / `--continue` flag for session resume ("last" or explicit session ID)
- `resolveSessionID()` function with "last" keyword support

#### Token Usage
- `core/usage/` package with `Tracker`, `SessionUsage`, and `GetPricing()`
- Built-in pricing for 25+ models across all 7 providers
- Token usage display on stderr: `[tokens: X prompt + Y completion = Z total]`

#### CLI Enhancements
- `-M` / `--model` flag for runtime model override
- Stdin pipe support for non-TTY input

#### Documentation
- Chapter 06: Streaming and Chinese Providers learning guide
- Updated all existing documentation for Wave 2 features

## [0.1.0] - 2026-03-03

### Added

#### Core
- Agent ReAct loop with configurable max iterations and timeout
- Tool system with pluggable registry and built-in tools (exec, fileops, websearch)
- LLM provider interface with OpenAI and Anthropic adapters
- MCP (Model Context Protocol) client for external tool integration
- RAG pipeline with TF-IDF indexing and cosine similarity search
- Skills system with composable skill registry
- Long-term memory with importance scoring and time-based decay

#### Infrastructure
- Session management with conversation history
- SQLite persistence layer (pure Go, modernc.org/sqlite)
- Message bus with async pub/sub event system
- Configuration system with model_list protocol-based format
- Structured logging with slog (JSON/Text formats)
- Error handling with model fallback chain routing

#### Channels
- CLI adapter for terminal interaction
- HTTP Gateway server (port 18790) with REST API
- Telegram bot adapter

#### Security & Performance
- Auth middleware with API key validation
- Rate limiting (per-client and global)
- Command execution sandboxing
- Concurrency primitives (worker pool, semaphore, rate limiter)
- Prometheus-compatible metrics (counter, gauge, histogram)
- Gateway benchmark suite

#### Cloud-Native
- Multi-stage Dockerfile (2.5MB binary)
- Docker Compose with service profiles
- Kubernetes manifests (namespace, deployment, service, ingress, configmap, secret)
- Helm chart with parameterized templates
- GitHub Actions CI/CD (test, build, release)
- Prometheus + Grafana monitoring stack

#### Documentation
- OpenSpec SDD specifications
- 5 Chinese architecture learning guides
- Contributing guidelines
