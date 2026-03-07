// Package providers defines the core LLM abstraction interfaces ([LLMProvider]
// and [StreamingProvider]) shared across all vendor adapters, plus a [Factory]
// for registering and retrieving provider instances by vendor name.
// The interface signatures in this file MUST NOT be changed — all adapters
// depend on them.
package providers

import (
	"context"

	"github.com/strings77wzq/unlimitedClaw/core/tools"
)

// Role represents the message sender role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a chat message.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMResponse is the response from an LLM provider.
type LLMResponse struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Usage      TokenUsage `json:"usage"`
	Model      string     `json:"model"`
	StopReason string     `json:"stop_reason"`
}

// ChatOptions contains optional parameters for Chat.
type ChatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// LLMProvider is the interface all LLM providers must implement.
type LLMProvider interface {
	// Chat sends messages to the LLM and returns a response.
	// tools parameter contains the available tool definitions for the LLM.
	// model is the model identifier (without vendor prefix).
	Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions) (*LLMResponse, error)

	// Name returns the provider name (e.g., "openai", "anthropic").
	Name() string
}

// StreamingProvider is an optional interface for providers that support
// token-by-token streaming. Use Go type assertion to check support:
//
//	if sp, ok := provider.(StreamingProvider); ok {
//	    resp, err := sp.ChatStream(ctx, msgs, tools, model, opts, onToken)
//	}
type StreamingProvider interface {
	LLMProvider

	// ChatStream sends messages and streams the response token-by-token.
	// onToken is called for each text delta as it arrives.
	// Returns the complete LLMResponse (with Usage) after streaming finishes.
	ChatStream(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition,
		model string, opts *ChatOptions, onToken func(token string)) (*LLMResponse, error)
}
