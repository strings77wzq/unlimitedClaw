## ADDED Requirements

### Requirement: Long-term memory SHALL be enabled via CLI flag
The system SHALL provide a `--memory` CLI flag on the agent command that enables the long-term memory feature. When omitted, the agent operates without memory.

#### Scenario: Agent starts with memory enabled
- **WHEN** the user runs `golem agent --memory`
- **THEN** the agent starts with the memory module active and registers `memory_store`/`memory_recall` tools

#### Scenario: Agent starts without memory flag
- **WHEN** the user runs `golem agent` without the `--memory` flag
- **THEN** the agent starts without any memory tools and the memory field on the agent is nil

### Requirement: The LLM SHALL be able to store and recall memories via tools
The system SHALL expose `memory_store` and `memory_recall` as registered tools that the LLM can invoke during conversation.

#### Scenario: LLM stores a memory
- **WHEN** the LLM invokes `memory_store` with a content string, optional importance (0-1), and optional tags
- **THEN** the memory is persisted and a confirmation result is returned

#### Scenario: LLM recalls memories
- **WHEN** the LLM invokes `memory_recall` with a query string and optional limit
- **THEN** matching memories ranked by decayed importance are returned

### Requirement: Memory persistence SHALL survive across sessions
The system SHALL persist memories to a file or database path so they survive across agent sessions.

#### Scenario: Memory file is created on first use
- **WHEN** the user starts the agent with `--memory` and stores a memory for the first time
- **THEN** a memory file is created at the configured path (default: `~/.golem/memory.json`)

#### Scenario: Memory file is loaded on subsequent starts
- **WHEN** the user starts the agent with `--memory` and a memory file already exists
- **THEN** previously stored memories are available for recall
