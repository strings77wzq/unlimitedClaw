# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
