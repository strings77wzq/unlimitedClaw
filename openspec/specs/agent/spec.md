# Agent Behavior Specification

## Overview
The Agent component implements the ReAct (Reasoning + Acting) loop for unlimitedClaw.

## Definitions
- **Agent**: The core component that orchestrates LLM interactions and tool execution.
- **ReAct Loop**: A cycle of reasoning (LLM call) and acting (tool execution) until a final response is produced.

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
