## MODIFIED Requirements

### Requirement: Message Processing
- The Agent MUST subscribe to inbound messages via the Message Bus.
- The Agent MUST build a context from system prompt, session history, and current message.
- The Agent MUST call the LLM Provider with the built context and available tools.
- Core provider message and supporting contract types referenced by agent-adjacent integrations (including gateway health wiring) MUST remain compile-time resolvable in CI validation steps.

#### Scenario: Given a simple text message
- **Given**: A user sends "Hello"
- **When**: The Agent processes the message
- **Then**: The Agent calls the LLM and publishes the text response

#### Scenario: Contract regression is introduced in provider core types
- **WHEN** a change introduces unresolved provider contract symbols used by agent-adjacent runtime packages
- **THEN** CI static validation fails before test execution and the change is blocked
