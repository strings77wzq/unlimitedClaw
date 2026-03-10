# Agent Behavior Specification

## Overview
The Agent component implements the ReAct (Reasoning + Acting) loop for Golem.

## Definitions
- **Agent**: The core component that orchestrates LLM interactions and tool execution.
- **ReAct Loop**: A cycle of reasoning (LLM call) and acting (tool execution) until a final response is produced.
- **MCP (Model Context Protocol)**: A standard protocol for integrating external tool servers with AI agents, enabling dynamic tool loading.
- **RAG (Retrieval-Augmented Generation)**: A technique that retrieves relevant external knowledge and injects it into the LLM context to improve response accuracy.
- **Skill**: A pre-defined, reusable LLM prompt template or workflow that encapsulates a specific capability (e.g., summarization, code review).

## Behavior

### Message Processing
- The Agent MUST subscribe to inbound messages via the Message Bus.
- The Agent MUST build a context from system prompt, session history, and current message.
- The Agent MUST call the LLM Provider with the built context and available tools.

### Tool Execution
- When the LLM response contains tool calls, the Agent MUST execute each tool via the Tool Registry.
- Tool results MUST be appended to the context as tool response messages.
- The Agent MUST re-invoke the LLM with updated context after tool execution.
- The Agent MUST repeat this cycle until the LLM returns a response without tool calls.

#### Tool Execution Security Constraints
- The `exec` tool MUST run in Sandbox mode by default (no shell interpretation, command allowlist enforcement)
- Shell interpretation for `exec` tool MUST ONLY be enabled via explicit `WithAllowShell()` configuration
- All tool executions MUST respect configured security policies (allowlist/denylist rules)
- Tool execution errors MUST be properly logged and returned to the user without exposing sensitive system information

### Iteration Control
- The Agent MUST enforce a maximum tool iteration limit (configurable, default: 25).
- When the iteration limit is reached, the Agent MUST return an error message to the user.
- The Agent SHOULD NOT silently drop tool calls.

### Context Window Management
- The Agent MUST track approximate token usage of the context.
- When context exceeds the model's max_tokens, the Agent MUST truncate oldest messages.
- The system prompt MUST NEVER be truncated.

### Output
- The Agent MUST publish the final response as an outbound message via the Message Bus.
- The ToolResult MUST separate ForLLM (always sent to LLM) and ForUser (displayed to user) channels.

### MCP Integration Behavior
- When MCP is enabled via configuration, the Agent MUST load all tools from configured MCP servers at startup.
- MCP tools MUST be registered in the global Tool Registry with a `mcp_` prefix to avoid name conflicts.
- The Agent MUST handle MCP tool execution identically to native tools, including error handling and result formatting.
- MCP servers MUST be terminated gracefully when the Agent shuts down.

### RAG Integration Behavior
- When RAG is enabled via configuration, the Agent MUST index all provided documents at startup.
- The Agent MUST register a `rag_retrieve` tool that allows the LLM to search the indexed knowledge base.
- RAG retrieval results MUST be added to the LLM context before processing the user query.
- The Agent MUST NOT modify the original indexed documents during runtime.

### Skills Integration Behavior
- When Skills are enabled via configuration, the Agent MUST register all selected skills as callable tools.
- Each Skill MUST expose a well-defined JSON schema for input parameters.
- Skill execution MUST be stateless and idempotent — repeated calls with the same input produce the same output.
- Skill results MUST be formatted as standard ToolResult objects, separating ForLLM and ForUser content.

## Scenarios

### Given a simple text message
- **Given**: A user sends "Hello"
- **When**: The Agent processes the message
- **Then**: The Agent calls the LLM and publishes the text response

### Given a message requiring tool use
- **Given**: A user asks "What's the weather?"
- **When**: The LLM returns a tool call for weather_search
- **Then**: The Agent executes the tool, feeds result back to LLM, and publishes final response

### Given too many tool iterations
- **Given**: A user triggers a chain of 30 tool calls
- **When**: The iteration limit (25) is reached
- **Then**: The Agent stops and returns an error message

### Given MCP is enabled
- **Given**: MCP is configured with a file system server
- **When**: The user asks "What files are in the current directory?"
- **Then**: The Agent calls the `mcp_list_files` tool, processes the result, and returns the list of files

### Given RAG is enabled with documentation
- **Given**: RAG is enabled with Golem architecture documentation
- **When**: The user asks "What is the layer dependency rule for Golem?"
- **Then**: The Agent uses `rag_retrieve` to get the relevant documentation, injects it into the context, and returns an accurate answer

### Given summarize skill is enabled
- **Given**: The summarize skill is enabled
- **When**: The user pastes a 1000-word article and asks "Summarize this"
- **Then**: The Agent invokes the summarize skill, processes the input, and returns a concise summary
