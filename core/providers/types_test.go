package providers

import (
	"testing"
)

func TestMessageRoles(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleSystem, "system"},
		{RoleTool, "tool"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("Role constant mismatch: got %s, want %s", tt.role, tt.expected)
		}
	}
}

func TestLLMResponseWithToolCalls(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call_123",
		Name: "get_weather",
		Arguments: map[string]interface{}{
			"location": "San Francisco",
			"unit":     "celsius",
		},
	}

	resp := &LLMResponse{
		Content:   "Let me check the weather for you.",
		ToolCalls: []ToolCall{toolCall},
		Usage: TokenUsage{
			PromptTokens:     50,
			CompletionTokens: 20,
			TotalTokens:      70,
		},
		Model:      "gpt-4",
		StopReason: "tool_calls",
	}

	if resp.Content != "Let me check the weather for you." {
		t.Errorf("Content mismatch: got %s", resp.Content)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("ToolCall ID mismatch: got %s", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall Name mismatch: got %s", tc.Name)
	}
	if tc.Arguments["location"] != "San Francisco" {
		t.Errorf("ToolCall location arg mismatch: got %v", tc.Arguments["location"])
	}

	if resp.StopReason != "tool_calls" {
		t.Errorf("StopReason mismatch: got %s", resp.StopReason)
	}
}

func TestTokenUsage(t *testing.T) {
	usage := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Errorf("Total tokens should equal prompt + completion: %d != %d + %d",
			usage.TotalTokens, usage.PromptTokens, usage.CompletionTokens)
	}
}

func TestChatOptions(t *testing.T) {
	temp := 0.7
	maxTokens := 2000
	topP := 0.9

	opts := &ChatOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
		Stop:        []string{"STOP", "END"},
	}

	if opts.Temperature == nil || *opts.Temperature != 0.7 {
		t.Errorf("Temperature mismatch")
	}

	if opts.MaxTokens == nil || *opts.MaxTokens != 2000 {
		t.Errorf("MaxTokens mismatch")
	}

	if opts.TopP == nil || *opts.TopP != 0.9 {
		t.Errorf("TopP mismatch")
	}

	if len(opts.Stop) != 2 {
		t.Errorf("Expected 2 stop sequences, got %d", len(opts.Stop))
	}
}

func TestMessageWithToolCall(t *testing.T) {
	msg := Message{
		Role:    RoleAssistant,
		Content: "I'll help with that.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_abc",
				Name: "search",
				Arguments: map[string]interface{}{
					"query": "test",
				},
			},
		},
	}

	if msg.Role != RoleAssistant {
		t.Errorf("Role mismatch: got %s, want %s", msg.Role, RoleAssistant)
	}

	if len(msg.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].Name != "search" {
		t.Errorf("Tool name mismatch: got %s", msg.ToolCalls[0].Name)
	}
}

func TestToolResultMessage(t *testing.T) {
	msg := Message{
		Role:       RoleTool,
		Content:    "Search results: ...",
		ToolCallID: "call_abc",
	}

	if msg.Role != RoleTool {
		t.Errorf("Role mismatch: got %s, want %s", msg.Role, RoleTool)
	}

	if msg.ToolCallID != "call_abc" {
		t.Errorf("ToolCallID mismatch: got %s", msg.ToolCallID)
	}
}
