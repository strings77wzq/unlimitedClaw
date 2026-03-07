package tools

import "context"

// MockTool is a configurable tool for testing.
type MockTool struct {
	ToolName        string
	ToolDescription string
	ToolParameters  []ToolParameter
	ExecuteFn       func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

func (m *MockTool) Name() string {
	return m.ToolName
}

func (m *MockTool) Description() string {
	return m.ToolDescription
}

func (m *MockTool) Parameters() []ToolParameter {
	return m.ToolParameters
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, args)
	}
	return &ToolResult{
		ForLLM:  "mock result",
		Silent:  true,
		IsError: false,
	}, nil
}
