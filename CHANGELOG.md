# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
