package session

import "sync"

// SessionStore defines session persistence operations.
type SessionStore interface {
	Get(id string) (*Session, bool)
	Save(session *Session) error
	Delete(id string) error
	List() []*Session
}

// MemoryStore is an in-memory session store (for development/testing).
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Get retrieves a session by ID.
func (m *MemoryStore) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Save stores a session.
func (m *MemoryStore) Save(session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

// Delete removes a session by ID.
func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

// List returns all sessions as a slice.
func (m *MemoryStore) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
