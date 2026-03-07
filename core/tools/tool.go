// Package tools defines the [Tool] interface and [Registry] that all agent
// tools must implement and register with. Tools are pure functions: they
// receive a map of arguments and return a [ToolResult] with separate strings
// for user-visible output and LLM context. The registry always returns tools
// in alphabetical order to maximise LLM KV-cache reuse across requests.
package tools

import "context"

// Tool defines the interface every tool must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() []ToolParameter
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// ToolParameter describes a single parameter for a tool.
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolResult contains the dual-channel output from tool execution.
// ForLLM is ALWAYS sent to the LLM as context.
// ForUser is immediately displayed to the user (optional).
type ToolResult struct {
	ForLLM  string // always sent to LLM
	ForUser string // displayed to user immediately (can be empty)
	IsError bool   // indicates tool execution failed
	Silent  bool   // if true, don't display ForUser even if non-empty
}

// ToolDefinition is the JSON-serializable representation of a tool for LLM consumption.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`
}

// ToDefinition converts a Tool to its JSON-serializable definition.
func ToDefinition(t Tool) ToolDefinition {
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}
