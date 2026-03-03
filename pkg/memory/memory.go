package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Entry represents a single memory entry
type Entry struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Memory defines the interface for memory storage and retrieval
type Memory interface {
	Store(ctx context.Context, entry *Entry) error
	Recall(ctx context.Context, query string, limit int) ([]*Entry, error)
	Forget(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Entry, error)
}

// NewEntry creates a new Entry with generated ID and timestamps
func NewEntry(content string, tags ...string) *Entry {
	now := time.Now()
	return &Entry{
		ID:        generateID(),
		Content:   content,
		Tags:      tags,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateID generates a UUID-like ID
func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}
