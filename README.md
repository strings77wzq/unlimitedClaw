# unlimitedClaw

[![CI](https://github.com/strings77wzq/unlimitedClaw/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/unlimitedClaw/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/unlimitedClaw)](https://goreportcard.com/report/github.com/strings77wzq/unlimitedClaw)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)

A lightweight, cloud-native AI assistant built with pure Go — inspired by [PicoClaw](https://github.com/sipeed/picoclaw).

unlimitedClaw is a progressive learning project that implements a full AI agent system from scratch, covering Agent core, Tool system, LLM integration, MCP protocol, RAG pipeline, and cloud-native deployment.

## Features

- **Agent ReAct Loop** — Think → Act → Observe reasoning cycle with configurable max iterations
- **Tool System** — Pluggable tool registry with built-in exec, file operations, and web search
- **LLM Providers** — OpenAI, Anthropic, DeepSeek, Kimi, GLM, MiniMax, and Qwen adapters with streaming support
- **MCP Client** — Model Context Protocol client for external tool integration
- **RAG Pipeline** — Retrieval-Augmented Generation with TF-IDF indexing, similarity search, and OpenAI embedding support
- **Skills System** — Composable skill registry with built-in skills (summarize, code-review)
- **Long-term Memory** — Persistent memory with importance scoring and exponential decay
- **Multiple Channels** — CLI, interactive TUI (Bubble Tea, auto-detected on TTY), HTTP Gateway (with SSE streaming), and Telegram bot adapters
- **First-run Wizard** — `unlimitedclaw init` interactive setup with 7 provider presets
- **Message Bus** — Async pub/sub event system for decoupled communication
- **Session Management** — Conversation history with SQLite persistence
- **Security** — Auth middleware, rate limiting, and command sandboxing
- **Concurrency** — Worker pool, semaphore, and rate limiter primitives
- **Prometheus Metrics** — Pure Go metrics (counter/gauge/histogram) with exposition endpoint
- **Cloud-Native** — Docker, Kubernetes, Helm, CI/CD, monitoring stack, and config hot reload (SIGHUP)

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Channels                          │
│              CLI / Gateway / Telegram                │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Agent Core                          │
│            ReAct Loop (Think→Act→Observe)            │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  Tools   │  │  Skills  │  │   LLM Providers   │  │
│  │  Registry│  │  Registry│  │  OpenAI / Anthropic│  │
│  └──────────┘  └──────────┘  └───────────────────┘  │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │   MCP    │  │   RAG    │  │     Memory        │  │
│  │  Client  │  │ Pipeline │  │   Long-term       │  │
│  └──────────┘  └──────────┘  └───────────────────┘  │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                Infrastructure                        │
│  Session / Store(SQLite) / Bus / Config / Logger     │
│  Security / Concurrency / Metrics / Routing          │
└─────────────────────────────────────────────────────┘
```

## Installation

### From Source (go install)

```bash
go install github.com/strings77wzq/unlimitedClaw/cmd/unlimitedclaw@latest
```

This installs the `unlimitedclaw` binary to your `$GOPATH/bin` (or `$HOME/go/bin`). Make sure it's in your `PATH`.

### From Release Binaries

Download pre-built binaries from the [Releases](https://github.com/strings77wzq/unlimitedClaw/releases) page. Available for Linux, macOS, and Windows (amd64/arm64).

### Build from Source

```bash
git clone https://github.com/strings77wzq/unlimitedClaw.git
cd unlimitedClaw

# Build binary (pure Go, no CGO)
CGO_ENABLED=0 go build -ldflags "-s -w" -o build/unlimitedclaw ./cmd/unlimitedclaw

# Or use Makefile
make build
```

### On Android/Termux (ARM64)

unlimitedClaw builds and runs natively on Android via [Termux](https://termux.dev/) — no root required.

```bash
# Install Go in Termux
pkg install golang

# Install directly via go install
go install github.com/strings77wzq/unlimitedClaw/cmd/unlimitedclaw@latest
# Binary lands at $HOME/go/bin/unlimitedclaw

# Or build from source
git clone https://github.com/strings77wzq/unlimitedClaw.git
cd unlimitedClaw
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath \
    -o ~/bin/unlimitedclaw ./cmd/unlimitedclaw
```

> **Termux notes:**
> - The TUI auto-activates when stdin is a TTY (standard Termux terminal); pipe/redirect falls back to plain output automatically.
> - Mouse input is disabled by default — compatible with all Termux terminal emulators and Android keyboards.
> - Alt+key shortcuts are not used; all keybindings work with standard terminal key sequences.
> - Use `unlimitedclaw init` for the first-run setup wizard to configure your API key.

## Quick Start

### Prerequisites

- Go 1.25+
- (Optional) Docker for containerized deployment

### First-run Setup

```bash
# Interactive setup wizard — configures provider, API key, and default model
unlimitedclaw init
```

The wizard supports 7 provider presets: OpenAI, Anthropic, DeepSeek, Moonshot/Kimi, Zhipu/GLM, MiniMax, and DashScope/Qwen. It writes to `~/.unlimitedclaw/config.json`.

### Usage

```bash
# Show help
unlimitedclaw --help

# Print version
unlimitedclaw version

# Start agent (auto-detects TTY → opens Bubble Tea TUI; pipe/redirect → plain output)
unlimitedclaw agent

# Start agent with an initial message (one-shot, no TUI)
unlimitedclaw agent -m "Hello, what can you do?"

# Force plain interactive mode (no TUI)
unlimitedclaw agent --no-tui

# Start HTTP gateway (port 18790)
unlimitedclaw gateway

# Use a specific model
unlimitedclaw agent -M deepseek/deepseek-chat -m "Hello"

# Resume last session
unlimitedclaw agent -C last

# Resume specific session
unlimitedclaw agent -C <session-id>

# Pipe input from another command
echo "Summarize this" | unlimitedclaw agent
```

### Configuration Management

unlimitedClaw stores config at `~/.unlimitedclaw/config.json`. Manage it via CLI:

```bash
# Set a config value
unlimitedclaw config set default_model openai/gpt-4

# Get a config value
unlimitedclaw config get default_model

# List all config values
unlimitedclaw config list

# Use a custom config file
unlimitedclaw --config /path/to/config.json agent -m "hello"
```

### Status & Health Check

```bash
# Show system status (version, config, model info, gateway health)
unlimitedclaw status
```

### Shell Completion

Generate shell completion scripts for your shell:

```bash
# Bash
unlimitedclaw completion bash > /etc/bash_completion.d/unlimitedclaw

# Zsh
unlimitedclaw completion zsh > "${fpath[1]}/_unlimitedclaw"

# Fish
unlimitedclaw completion fish > ~/.config/fish/completions/unlimitedclaw.fish

# PowerShell
unlimitedclaw completion powershell > unlimitedclaw.ps1
```

### Docker

```bash
# Build image
docker build -f docker/Dockerfile -t unlimitedclaw .

# Run with Docker Compose (gateway mode)
docker compose -f docker/docker-compose.yml --profile gateway up

# Run with monitoring stack (Prometheus + Grafana)
docker compose -f docker/monitoring/docker-compose.monitoring.yml up
```

### Environment Variables

Set API keys via environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Chinese LLM providers
export DEEPSEEK_API_KEY="sk-..."
export MOONSHOT_API_KEY="sk-..."
export ZHIPU_API_KEY="..."
export MINIMAX_API_KEY="..."
export DASHSCOPE_API_KEY="sk-..."
```

Or use the config file approach:

```bash
cp config/config.example.json ~/.unlimitedclaw/config.json
# Edit with your API keys, or use: unlimitedclaw config set ...
```

## Project Structure

```
unlimitedClaw/
├── cmd/unlimitedclaw/              # CLI entry point (cobra)
├── core/                           # Core domain logic
│   ├── agent/                      # ReAct loop engine
│   ├── bus/                        # Message bus (pub/sub)
│   ├── config/                     # Configuration system with hot reload
│   ├── providers/                  # LLM provider interface
│   │   ├── openai/                 # OpenAI adapter
│   │   └── anthropic/              # Anthropic adapter
│   ├── session/                    # Session + history management
│   ├── tools/                      # Tool interface + registry
│   │   ├── exec/                   # Command execution tool
│   │   ├── fileops/                # File operations tool
│   │   └── websearch/              # Web search tool
│   └── usage/                      # Token usage tracking & pricing
├── foundation/                     # Infrastructure primitives
│   ├── concurrency/                # Pool, semaphore, rate limiter
│   ├── logger/                     # Structured logging (slog)
│   ├── store/                      # SQLite persistence (pure Go)
│   └── term/                       # Terminal detection
├── feature/                        # Reference implementations (not wired into main.go)
│   │                               # These exist as standalone learning modules only.
│   ├── mcp/                        # MCP protocol client
│   ├── memory/                     # Long-term memory with importance decay
│   ├── rag/                        # RAG pipeline with OpenAI embedder
│   ├── routing/                    # Error handling + fallback
│   └── skills/                     # Skills registry + built-in skills
│       └── builtins/               # Built-in skills (summarize, code-review)
├── internal/                       # Internal-only packages
│   ├── channels/                   # I/O adapters
│   │   ├── cli/                    # CLI adapter
│   │   ├── tui/                    # Bubble Tea TUI (auto-detected on TTY)
│   │   └── telegram/               # Telegram bot adapter
│   ├── gateway/                    # HTTP gateway server with SSE streaming
│   ├── metrics/                    # Prometheus-compatible metrics
│   └── security/                   # Auth, rate limiting, sandbox
├── openspec/                       # OpenSpec SDD specifications
├── docs/study/                     # Learning guides (Chinese)
├── docker/                         # Dockerfile + Compose
│   └── monitoring/                 # Prometheus + Grafana configs
├── k8s/                            # Kubernetes manifests
├── helm/unlimitedclaw/             # Helm chart
├── .github/workflows/              # CI/CD pipelines
├── scripts/                        # Utility scripts
├── Makefile                        # Build automation
└── .golangci.yaml                  # Linter configuration
```

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./internal/gateway/...
```

**Test coverage: 79.2%** across 28 packages (200+ tests, 9 Example functions for godoc).

## Kubernetes Deployment

```bash
# Apply manifests directly
kubectl apply -f k8s/

# Or use Helm
helm install unlimitedclaw helm/unlimitedclaw/
```

## Learning Resources

The `docs/study/` directory contains Chinese learning guides:

1. **Architecture Overview** — Hexagonal architecture and design patterns
2. **Agent ReAct Loop** — How the Think→Act→Observe cycle works
3. **Tool System** — Building a pluggable tool registry
4. **Provider System** — LLM provider abstraction and adapters
5. **Message Bus** — Async event-driven communication
6. **Streaming & Chinese Providers** — SSE streaming, Chinese LLM integration, session resume

## Design Principles

- **Pure Go** — Zero CGO dependencies (`CGO_ENABLED=0`), single static binary
- **Layered Architecture** — 4-layer structure (core/foundation/feature/internal) with clean dependency flow
- **Cloud-Native** — Docker, Kubernetes, Helm, Prometheus metrics
- **Security First** — Auth middleware, rate limiting, command sandboxing
- **Test-Driven** — 79.2% coverage, race-detector clean, benchmark suite

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ |
| CLI | [cobra](https://github.com/spf13/cobra) |
| TUI | [Bubble Tea v1.3.10](https://github.com/charmbracelet/bubbletea) + lipgloss |
| Database | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go) |
| Metrics | Custom Prometheus-compatible (no external deps) |
| Container | Docker multi-stage build |
| Orchestration | Kubernetes + Helm |
| CI/CD | GitHub Actions |
| Monitoring | Prometheus + Grafana |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

[MIT License](LICENSE)

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
- Built following [OpenSpec SDD](https://github.com/Fission-AI/OpenSpec) workflow
