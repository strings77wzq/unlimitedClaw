// Package memory implements long-term memory for the AI agent with importance
// scoring and exponential decay. Memories are stored persistently and pruned
// automatically when the store exceeds capacity. This is a reference
// implementation in the feature/ layer and is NOT wired into main.go by
// default.
package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"time"
)

const DefaultDecayLambda = 0.001

// Entry represents a single memory entry with importance scoring.
type Entry struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Importance float64           `json:"importance"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	AccessedAt time.Time         `json:"accessed_at"`
}

// DecayedImportance returns importance * exp(-lambda * hours_since_last_access).
func (e *Entry) DecayedImportance(now time.Time, lambda float64) float64 {
	dt := now.Sub(e.AccessedAt).Hours()
	if dt < 0 {
		dt = 0
	}
	return e.Importance * math.Exp(-lambda*dt)
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
		ID:         generateID(),
		Content:    content,
		Tags:       tags,
		Metadata:   make(map[string]string),
		Importance: 1.0,
		CreatedAt:  now,
		UpdatedAt:  now,
		AccessedAt: now,
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
