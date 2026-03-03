package session

import (
	"testing"

	"github.com/strin/unlimitedclaw/pkg/providers"
)

func TestGetContextWindowAllFit(t *testing.T) {
	manager := NewHistoryManager(1000)

	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "System prompt"},
		{Role: providers.RoleUser, Content: "Hello"},
		{Role: providers.RoleAssistant, Content: "Hi there"},
	}

	result := manager.GetContextWindow(messages)

	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}

	if result[0].Role != providers.RoleSystem {
		t.Error("expected first message to be system")
	}
	if result[1].Role != providers.RoleUser {
		t.Error("expected second message to be user")
	}
	if result[2].Role != providers.RoleAssistant {
		t.Error("expected third message to be assistant")
	}
}

func TestGetContextWindowTruncation(t *testing.T) {
	manager := NewHistoryManager(20)

	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "System"},
		{Role: providers.RoleUser, Content: "Message 1 with some content"},
		{Role: providers.RoleAssistant, Content: "Response 1 with some content"},
		{Role: providers.RoleUser, Content: "Message 2 with some content"},
		{Role: providers.RoleAssistant, Content: "Response 2 with some content"},
	}

	result := manager.GetContextWindow(messages)

	if result[0].Role != providers.RoleSystem {
		t.Error("system message should always be first")
	}

	if len(result) == len(messages) {
		t.Error("expected some messages to be truncated")
	}

	for i := 1; i < len(result); i++ {
		if result[i].Content != messages[len(messages)-(len(result)-i)].Content {
			t.Error("expected most recent messages to be kept")
		}
	}
}

func TestSystemPromptNeverTruncated(t *testing.T) {
	manager := NewHistoryManager(10)

	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "This is a very long system prompt that exceeds the token budget by a lot"},
		{Role: providers.RoleUser, Content: "Hello"},
		{Role: providers.RoleAssistant, Content: "Hi"},
	}

	result := manager.GetContextWindow(messages)

	if len(result) < 1 {
		t.Fatal("expected at least system message")
	}

	if result[0].Role != providers.RoleSystem {
		t.Error("system message should always be preserved")
	}

	if result[0].Content != messages[0].Content {
		t.Error("system message content should be unchanged")
	}
}

func TestEmptyMessages(t *testing.T) {
	manager := NewHistoryManager(100)

	result := manager.GetContextWindow([]providers.Message{})

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d messages", len(result))
	}
}

func TestEstimateTokens(t *testing.T) {
	manager := NewHistoryManager(100)

	msg := providers.Message{
		Role:    providers.RoleUser,
		Content: "1234567890123456",
	}

	tokens := manager.EstimateTokens(msg)

	expected := len(msg.Content) / 4
	if tokens != expected {
		t.Errorf("expected %d tokens, got %d", expected, tokens)
	}
}

func TestGetContextWindowOrderPreserved(t *testing.T) {
	manager := NewHistoryManager(200)

	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "System"},
		{Role: providers.RoleUser, Content: "First user message"},
		{Role: providers.RoleAssistant, Content: "First assistant message"},
		{Role: providers.RoleUser, Content: "Second user message"},
		{Role: providers.RoleAssistant, Content: "Second assistant message"},
	}

	result := manager.GetContextWindow(messages)

	if result[0].Role != providers.RoleSystem {
		t.Error("system message should be first")
	}

	for i := 1; i < len(result)-1; i++ {
		currentIdx := findMessageIndex(messages, result[i].Content)
		nextIdx := findMessageIndex(messages, result[i+1].Content)

		if currentIdx >= nextIdx {
			t.Error("messages should maintain chronological order")
		}
	}
}

func TestGetContextWindowNoSystemMessage(t *testing.T) {
	manager := NewHistoryManager(50)

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Message 1 with some content"},
		{Role: providers.RoleAssistant, Content: "Response 1 with some content"},
		{Role: providers.RoleUser, Content: "Message 2 with some content"},
		{Role: providers.RoleAssistant, Content: "Response 2 with some content"},
	}

	result := manager.GetContextWindow(messages)

	if len(result) == 0 {
		t.Fatal("expected some messages")
	}

	for _, msg := range result {
		if msg.Role == providers.RoleSystem {
			t.Error("should not have system message when none provided")
		}
	}

	for i := 0; i < len(result)-1; i++ {
		currentIdx := findMessageIndex(messages, result[i].Content)
		nextIdx := findMessageIndex(messages, result[i+1].Content)

		if currentIdx >= nextIdx {
			t.Error("messages should maintain chronological order")
		}
	}
}

func findMessageIndex(messages []providers.Message, content string) int {
	for i, msg := range messages {
		if msg.Content == content {
			return i
		}
	}
	return -1
}
