## ADDED Requirements

### Requirement: Token estimation SHALL account for CJK characters
The system SHALL detect CJK runes in message content and apply a higher per-character token weight than ASCII characters.

#### Scenario: Pure ASCII content
- **WHEN** a message contains only ASCII characters
- **THEN** the token estimate uses the standard 4-chars-per-token approximation

#### Scenario: CJK-heavy content
- **WHEN** a message contains CJK (Chinese/Japanese/Korean) characters
- **THEN** the token estimate applies a roughly 2-chars-per-token weight for CJK runes

#### Scenario: Mixed ASCII and CJK content
- **WHEN** a message contains both ASCII and CJK characters
- **THEN** the token estimate combines ASCII-weighted and CJK-weighted counts

### Requirement: Context window truncation behavior SHALL remain compatible
The system SHALL preserve the existing `GetContextWindow()` behavior: system prompt is always included, recent messages are prioritized, and the signature does not change.

#### Scenario: System prompt is never truncated
- **WHEN** context window truncation runs with a system prompt that alone exceeds the budget
- **THEN** the system prompt is still included in the output

#### Scenario: Existing tests pass
- **WHEN** the token estimation improvement is deployed
- **THEN** existing tests in `core/session/history_test.go` pass (updated expectations if needed)
