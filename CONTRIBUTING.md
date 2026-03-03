# Contributing to unlimitedClaw

Thank you for your interest in contributing to unlimitedClaw!

## Development Setup

1. **Prerequisites**: Go 1.25+, Git
2. **Clone**: `git clone https://github.com/strings77wzq/unlimitedClaw.git`
3. **Build**: `make build`
4. **Test**: `make test`

## Code Standards

- Run `go vet ./...` before committing
- Run `go test -race ./...` to verify no race conditions
- Follow existing code patterns and naming conventions
- Keep `CGO_ENABLED=0` — no CGO dependencies allowed
- Add tests for new features (target >70% coverage)

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Write your changes with tests
4. Ensure all tests pass: `go test -race ./...`
5. Commit with a clear message: `git commit -m "feat(pkg): add feature description"`
6. Push and open a Pull Request

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

Types: feat, fix, docs, refactor, test, build, ci
Scope: package name (agent, tools, providers, etc.)
```

Examples:
- `feat(agent): add multi-step planning support`
- `fix(providers): handle API timeout gracefully`
- `docs(study): add RAG pipeline guide`

## Architecture Guidelines

- **Hexagonal Architecture**: Define interfaces (ports) in the core package, implementations (adapters) separately
- **No circular imports**: Use interfaces to break dependency cycles
- **Pure Go**: All dependencies must work with `CGO_ENABLED=0`
- **Error handling**: Return errors, don't panic. Use `fmt.Errorf("context: %w", err)` for wrapping

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include Go version, OS, and reproduction steps for bugs

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
