package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
)

func TestMockProviderTextResponse(t *testing.T) {
	mock := NewMockProvider("test")

	resp := &LLMResponse{
		Content: "Hello, world!",
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		Model:      "test-model",
		StopReason: "stop",
	}
	mock.AddResponse(resp)

	messages := []Message{{Role: RoleUser, Content: "Hi"}}
	result, err := mock.Chat(context.Background(), messages, nil, "test-model", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "Hello, world!" {
		t.Errorf("Content mismatch: got %s, want %s", result.Content, "Hello, world!")
	}

	if mock.CallCount != 1 {
		t.Errorf("CallCount mismatch: got %d, want 1", mock.CallCount)
	}

	if mock.LastModel != "test-model" {
		t.Errorf("LastModel mismatch: got %s, want test-model", mock.LastModel)
	}
}

func TestMockProviderToolCalls(t *testing.T) {
	mock := NewMockProvider("test")

	resp := &LLMResponse{
		Content: "I'll search for that.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Name: "search",
				Arguments: map[string]interface{}{
					"query": "golang testing",
				},
			},
		},
		Usage: TokenUsage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
		Model:      "test-model",
		StopReason: "tool_calls",
	}
	mock.AddResponse(resp)

	result, err := mock.Chat(context.Background(), []Message{{Role: RoleUser, Content: "Search"}}, nil, "model", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.Name != "search" {
		t.Errorf("Tool name mismatch: got %s, want search", tc.Name)
	}

	if tc.Arguments["query"] != "golang testing" {
		t.Errorf("Tool argument mismatch: got %v", tc.Arguments["query"])
	}
}

func TestMockProviderResponseQueue(t *testing.T) {
	mock := NewMockProvider("test")

	mock.AddResponse(&LLMResponse{Content: "Response 1", Model: "m1", StopReason: "stop"})
	mock.AddResponse(&LLMResponse{Content: "Response 2", Model: "m2", StopReason: "stop"})
	mock.AddResponse(&LLMResponse{Content: "Response 3", Model: "m3", StopReason: "stop"})

	messages := []Message{{Role: RoleUser, Content: "Test"}}

	resp1, err := mock.Chat(context.Background(), messages, nil, "model", nil)
	if err != nil {
		t.Fatalf("call 1 error: %v", err)
	}
	if resp1.Content != "Response 1" {
		t.Errorf("Response 1 mismatch: got %s", resp1.Content)
	}

	resp2, err := mock.Chat(context.Background(), messages, nil, "model", nil)
	if err != nil {
		t.Fatalf("call 2 error: %v", err)
	}
	if resp2.Content != "Response 2" {
		t.Errorf("Response 2 mismatch: got %s", resp2.Content)
	}

	resp3, err := mock.Chat(context.Background(), messages, nil, "model", nil)
	if err != nil {
		t.Fatalf("call 3 error: %v", err)
	}
	if resp3.Content != "Response 3" {
		t.Errorf("Response 3 mismatch: got %s", resp3.Content)
	}

	if mock.CallCount != 3 {
		t.Errorf("CallCount mismatch: got %d, want 3", mock.CallCount)
	}
}

func TestMockProviderExhaustedQueue(t *testing.T) {
	mock := NewMockProvider("test")

	mock.AddResponse(&LLMResponse{Content: "Only response", Model: "m", StopReason: "stop"})

	messages := []Message{{Role: RoleUser, Content: "Test"}}

	_, err := mock.Chat(context.Background(), messages, nil, "model", nil)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	_, err = mock.Chat(context.Background(), messages, nil, "model", nil)
	if err == nil {
		t.Fatal("second call should fail with exhausted queue")
	}

	if !strings.Contains(err.Error(), "no more responses queued") {
		t.Errorf("error message should mention queue exhaustion: %s", err.Error())
	}
}

func TestMockProviderTracking(t *testing.T) {
	mock := NewMockProvider("openai")

	if mock.Name() != "openai" {
		t.Errorf("Name mismatch: got %s, want openai", mock.Name())
	}

	mock.AddResponse(&LLMResponse{Content: "Test", Model: "m", StopReason: "stop"})

	messages := []Message{
		{Role: RoleSystem, Content: "You are helpful"},
		{Role: RoleUser, Content: "Hello"},
	}

	toolDefs := []tools.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			Parameters: []tools.ToolParameter{
				{
					Name:        "query",
					Type:        "string",
					Description: "Search query",
					Required:    true,
				},
			},
		},
	}

	_, err := mock.Chat(context.Background(), messages, toolDefs, "gpt-4", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.CallCount != 1 {
		t.Errorf("CallCount mismatch: got %d, want 1", mock.CallCount)
	}

	if len(mock.LastMessages) != 2 {
		t.Errorf("LastMessages count mismatch: got %d, want 2", len(mock.LastMessages))
	}

	if mock.LastMessages[0].Role != RoleSystem {
		t.Errorf("First message role mismatch: got %s", mock.LastMessages[0].Role)
	}

	if mock.LastModel != "gpt-4" {
		t.Errorf("LastModel mismatch: got %s, want gpt-4", mock.LastModel)
	}

	if len(mock.LastTools) != 1 {
		t.Fatalf("LastTools count mismatch: got %d, want 1", len(mock.LastTools))
	}

	if mock.LastTools[0].Name != "search" {
		t.Errorf("Tool name mismatch: got %s, want search", mock.LastTools[0].Name)
	}
}
