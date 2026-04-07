package providers

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeHealthChecker struct{}

func (f *fakeHealthChecker) HealthCheck(_ context.Context) (*HealthStatus, error) {
	return &HealthStatus{Provider: "fake", Status: "healthy", Latency: 1, CheckedAt: 1}, nil
}

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

func TestHealthStatusJSONTags(t *testing.T) {
	status := HealthStatus{
		Provider:  "openai",
		Status:    "healthy",
		Latency:   12,
		Error:     "",
		CheckedAt: 1710000000,
	}

	b, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal health status: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal health status json: %v", err)
	}

	if _, ok := decoded["provider"]; !ok {
		t.Fatalf("expected provider field in json")
	}
	if _, ok := decoded["status"]; !ok {
		t.Fatalf("expected status field in json")
	}
	if _, ok := decoded["latency"]; !ok {
		t.Fatalf("expected latency field in json")
	}
	if _, ok := decoded["error"]; !ok {
		t.Fatalf("expected error field in json")
	}
	if _, ok := decoded["checked_at"]; !ok {
		t.Fatalf("expected checked_at field in json")
	}
}

func TestHealthCheckerInterfaceContract(t *testing.T) {
	var _ HealthChecker = (*fakeHealthChecker)(nil)

	checker := &fakeHealthChecker{}
	status, err := checker.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Provider != "fake" {
		t.Errorf("expected provider fake, got %q", status.Provider)
	}
}
