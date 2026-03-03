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
- **LLM Providers** — OpenAI and Anthropic adapters with streaming support
- **MCP Client** — Model Context Protocol client for external tool integration
- **RAG Pipeline** — Retrieval-Augmented Generation with TF-IDF indexing, similarity search, and OpenAI embedding support
- **Skills System** — Composable skill registry with built-in skills (summarize, code-review)
- **Long-term Memory** — Persistent memory with importance scoring and exponential decay
- **Multiple Channels** — CLI, HTTP Gateway (with SSE streaming), and Telegram bot adapters
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

## Quick Start

### Prerequisites

- Go 1.25+
- (Optional) Docker for containerized deployment

### Build

```bash
# Build binary (pure Go, no CGO)
CGO_ENABLED=0 go build -ldflags "-s -w" -o build/unlimitedclaw ./cmd/unlimitedclaw

# Or use Makefile
make build
```

### Run

```bash
# Show help
./build/unlimitedclaw --help

# Print version
./build/unlimitedclaw version

# Start HTTP gateway (port 18790)
./build/unlimitedclaw gateway

# Start CLI agent
./build/unlimitedclaw agent -m "Hello, what can you do?"
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

### Configuration

Copy and edit the example config:

```bash
cp config/config.example.json config/config.json
```

Set API keys via environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

## Project Structure

```
unlimitedClaw/
├── cmd/unlimitedclaw/          # CLI entry point (cobra)
├── pkg/
│   ├── agent/                  # ReAct loop engine
│   ├── bus/                    # Message bus (pub/sub)
│   ├── channels/
│   │   ├── cli/                # CLI adapter
│   │   └── telegram/           # Telegram bot adapter
│   ├── concurrency/            # Pool, semaphore, rate limiter
│   ├── config/                 # Configuration system with hot reload
│   ├── gateway/                # HTTP gateway server with SSE streaming
│   ├── logger/                 # Structured logging (slog)
│   ├── mcp/                    # MCP protocol client
│   ├── memory/                 # Long-term memory with importance decay
│   ├── metrics/                # Prometheus-compatible metrics
│   ├── providers/              # LLM provider interface
│   │   ├── openai/             # OpenAI adapter
│   │   └── anthropic/          # Anthropic adapter
│   ├── rag/                    # RAG pipeline with OpenAI embedder
│   ├── routing/                # Error handling + fallback
│   ├── security/               # Auth, rate limiting, sandbox
│   ├── session/                # Session + history management
│   ├── skills/                 # Skills registry + built-in skills
│   ├── store/                  # SQLite persistence (pure Go)
│   └── tools/                  # Tool interface + registry
│       ├── exec/               # Command execution tool
│       ├── fileops/            # File operations tool
│       └── websearch/          # Web search tool
├── openspec/                   # OpenSpec SDD specifications
├── docs/study/                 # Learning guides (Chinese)
├── docker/                     # Dockerfile + Compose
│   └── monitoring/             # Prometheus + Grafana configs
├── k8s/                        # Kubernetes manifests
├── helm/unlimitedclaw/         # Helm chart
├── .github/workflows/          # CI/CD pipelines
├── scripts/                    # Utility scripts
├── Makefile                    # Build automation
└── .golangci.yaml              # Linter configuration
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
go test -bench=. -benchmem ./pkg/gateway/...
```

**Test coverage: 76.9%** across 25 packages (200+ tests).

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

## Design Principles

- **Pure Go** — Zero CGO dependencies (`CGO_ENABLED=0`), single static binary
- **Hexagonal Architecture** — Ports (interfaces) and adapters (implementations)
- **Cloud-Native** — Docker, Kubernetes, Helm, Prometheus metrics
- **Security First** — Auth middleware, rate limiting, command sandboxing
- **Test-Driven** — 76.9% coverage, race-detector clean, benchmark suite

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ |
| CLI | [cobra](https://github.com/spf13/cobra) |
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
