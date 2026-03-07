// Package store defines the [Store] interface for key-value persistence used
// by the session layer. The SQLite implementation (foundation/store/sqlite.go)
// is built with modernc.org/sqlite (CGO_ENABLED=0), making it compatible with
// static binaries on Linux amd64 and Android/Termux ARM64.
package store

import "time"

// Store defines the interface for session persistence operations.
type Store interface {
	// GetSession retrieves a session record by ID.
	// Returns nil if the session does not exist.
	GetSession(id string) (*SessionRecord, error)

	// SaveSession creates or updates a session record.
	// Uses upsert semantics (INSERT OR REPLACE).
	SaveSession(record *SessionRecord) error

	// DeleteSession removes a session record by ID.
	DeleteSession(id string) error

	// ListSessions retrieves all session records, ordered by updated_at DESC.
	ListSessions() ([]*SessionRecord, error)
}

// SessionRecord represents a stored session with JSON-encoded messages.
type SessionRecord struct {
	ID        string    `json:"id"`
	Messages  []byte    `json:"messages"` // JSON-encoded messages
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
