## ADDED Requirements

### Requirement: MCP transport lifecycle SHALL be verified by automated tests
The system SHALL provide automated tests for `feature/mcp/transport.go` covering transport construction, process startup, request sending, notification sending, response receiving, and close behavior.

#### Scenario: Stdio transport starts and exchanges JSON-RPC messages
- **WHEN** a test starts a transport against a deterministic subprocess or pipe-backed fixture and sends a JSON-RPC request
- **THEN** the transport writes the encoded message, receives a valid response, and exposes it to the caller without hanging

#### Scenario: Stdio transport rejects invalid lifecycle usage
- **WHEN** a test calls send, receive, or start on a closed or improperly initialized transport
- **THEN** the transport returns deterministic lifecycle errors instead of blocking indefinitely

### Requirement: MCP client error paths SHALL be covered
The system SHALL provide automated tests for MCP client initialization guards, JSON-RPC failures, transport failures, context cancellation, and close delegation.

#### Scenario: Client methods reject use before initialization
- **WHEN** a test calls `ListTools` or `CallTool` before the client is initialized
- **THEN** the client returns an explicit initialization error

#### Scenario: Client surfaces JSON-RPC and transport failures
- **WHEN** the transport returns an error response, an I/O failure, or a canceled context during a request
- **THEN** the client returns the corresponding failure to the caller and clears any pending request bookkeeping

### Requirement: MCP manager orchestration SHALL be covered end-to-end
The system SHALL provide automated tests for MCP manager startup, tool discovery, tool invocation routing, proxy execution, and graceful shutdown.

#### Scenario: Manager starts configured servers and discovers tools
- **WHEN** a test starts a manager with one or more configured MCP server fixtures
- **THEN** the manager initializes each connection, discovers available tools, and exposes prefixed tool proxies for registry use

#### Scenario: Tool proxy execution forwards MCP tool results correctly
- **WHEN** a test executes an `MCPToolProxy` against a manager-backed tool call
- **THEN** the proxy returns a `ToolResult` that preserves user-visible text and reports MCP errors deterministically
