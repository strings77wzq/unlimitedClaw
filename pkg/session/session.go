package session

import (
	"sync"
	"time"

	"github.com/strings77wzq/unlimitedClaw/pkg/providers"
)

// Session holds a conversation's state.
type Session struct {
	ID        string
	Messages  []providers.Message
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
}

// NewSession creates a new session with the given ID.
func NewSession(id string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		Messages:  []providers.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage appends a message and updates timestamp.
func (s *Session) AddMessage(msg providers.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

// GetMessages returns a copy of all messages (thread-safe).
func (s *Session) GetMessages() []providers.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages := make([]providers.Message, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}

// MessageCount returns the number of messages.
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Messages)
}

// Clear removes all messages.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = []providers.Message{}
	s.UpdatedAt = time.Now()
}
