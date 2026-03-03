package session

import (
	"sync"
	"testing"
	"time"

	"github.com/strings77wzq/unlimitedClaw/pkg/providers"
)

func TestNewSession(t *testing.T) {
	id := "test-session"
	session := NewSession(id)

	if session.ID != id {
		t.Errorf("expected ID %q, got %q", id, session.ID)
	}

	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}

	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	if !session.CreatedAt.Equal(session.UpdatedAt) {
		t.Error("expected CreatedAt and UpdatedAt to be equal for new session")
	}
}

func TestAddMessage(t *testing.T) {
	session := NewSession("test")

	msg1 := providers.Message{Role: providers.RoleUser, Content: "Hello"}
	msg2 := providers.Message{Role: providers.RoleAssistant, Content: "Hi"}

	session.AddMessage(msg1)
	if session.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", session.MessageCount())
	}

	session.AddMessage(msg2)
	if session.MessageCount() != 2 {
		t.Errorf("expected 2 messages, got %d", session.MessageCount())
	}

	messages := session.GetMessages()
	if messages[0].Content != "Hello" {
		t.Errorf("expected first message 'Hello', got %q", messages[0].Content)
	}
	if messages[1].Content != "Hi" {
		t.Errorf("expected second message 'Hi', got %q", messages[1].Content)
	}
}

func TestGetMessages(t *testing.T) {
	session := NewSession("test")

	msg := providers.Message{Role: providers.RoleUser, Content: "Test"}
	session.AddMessage(msg)

	messages := session.GetMessages()
	messages[0].Content = "Modified"

	origMessages := session.GetMessages()
	if origMessages[0].Content != "Test" {
		t.Error("modifying returned slice should not affect session")
	}
}

func TestClear(t *testing.T) {
	session := NewSession("test")

	session.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	session.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi"})

	if session.MessageCount() != 2 {
		t.Errorf("expected 2 messages before clear, got %d", session.MessageCount())
	}

	oldUpdatedAt := session.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	session.Clear()

	if session.MessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", session.MessageCount())
	}

	if !session.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated after clear")
	}
}

func TestConcurrentAddMessage(t *testing.T) {
	session := NewSession("test")

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := providers.Message{
				Role:    providers.RoleUser,
				Content: "message",
			}
			session.AddMessage(msg)
		}(i)
	}

	wg.Wait()

	if session.MessageCount() != numGoroutines {
		t.Errorf("expected %d messages, got %d", numGoroutines, session.MessageCount())
	}
}
