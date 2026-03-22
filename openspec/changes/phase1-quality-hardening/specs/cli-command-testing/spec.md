## ADDED Requirements

### Requirement: Root command wiring SHALL be executable through automated tests
The system SHALL provide automated tests for the `cmd/golem` composition root that execute Cobra commands through `NewRootCommand()` and verify that core command wiring remains functional.

#### Scenario: Version command executes successfully
- **WHEN** a test constructs the root command and executes `version`
- **THEN** the command completes without error and writes version information to the captured output buffer

#### Scenario: Unknown command remains rejected
- **WHEN** a test constructs the root command and executes an unknown subcommand
- **THEN** command execution returns an error and preserves Cobra's failure semantics

### Requirement: Config and session command flows SHALL be testable with isolated filesystem state
The system SHALL support automated tests for `config` and `session` command flows using temporary config directories and temporary SQLite-backed session stores so tests do not depend on developer-local state.

#### Scenario: Config commands operate on a temporary config path
- **WHEN** a test executes `config set`, `config get`, or `config list` with a temporary config file path
- **THEN** the command reads and writes only that temporary path and produces deterministic output

#### Scenario: Session commands operate on a temporary session database
- **WHEN** a test executes `session list`, `session show`, or `session delete` against a temporary session database
- **THEN** the command reflects the seeded test data without touching the default user session store

### Requirement: Provider and feature wiring helpers SHALL have direct coverage
The system SHALL include focused tests for command-layer helpers that influence runtime composition, including provider registration, model listing, config loading, tool registry creation, and session resolution.

#### Scenario: Provider registration includes configured and fallback providers
- **WHEN** a test invokes provider registration with a config that includes explicit model entries
- **THEN** the factory contains the configured provider implementations and preserves the built-in mock fallback

#### Scenario: Session resolution handles explicit and implicit session targets
- **WHEN** a test resolves a session identifier using an explicit ID or the `last` selector
- **THEN** the helper returns the correct session ID or a deterministic error for missing state
