package session

import (
	"encoding/json"
	"fmt"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/foundation/store"
)

// SQLiteAdapter bridges store.SQLiteStore to the SessionStore interface,
// wiring persistent SQLite storage into the agent's session system.
type SQLiteAdapter struct {
	db *store.SQLiteStore
}

// NewSQLiteAdapter creates a SQLite-backed session store at the given path.
func NewSQLiteAdapter(dbPath string) (*SQLiteAdapter, error) {
	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening session database: %w", err)
	}
	return &SQLiteAdapter{db: db}, nil
}

func (a *SQLiteAdapter) Get(id string) (*Session, bool) {
	record, err := a.db.GetSession(id)
	if err != nil || record == nil {
		return nil, false
	}
	sess, err := recordToSession(record)
	if err != nil {
		return nil, false
	}
	return sess, true
}

func (a *SQLiteAdapter) Save(sess *Session) error {
	record, err := sessionToRecord(sess)
	if err != nil {
		return err
	}
	return a.db.SaveSession(record)
}

func (a *SQLiteAdapter) Delete(id string) error {
	return a.db.DeleteSession(id)
}

func (a *SQLiteAdapter) List() []*Session {
	records, err := a.db.ListSessions()
	if err != nil {
		return nil
	}
	sessions := make([]*Session, 0, len(records))
	for _, r := range records {
		sess, err := recordToSession(r)
		if err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions
}

// Close releases the underlying database connection.
func (a *SQLiteAdapter) Close() error {
	return a.db.Close()
}

func recordToSession(r *store.SessionRecord) (*Session, error) {
	var msgs []providers.Message
	if len(r.Messages) > 0 {
		if err := json.Unmarshal(r.Messages, &msgs); err != nil {
			return nil, fmt.Errorf("unmarshaling messages: %w", err)
		}
	}
	return &Session{
		ID:        r.ID,
		Messages:  msgs,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func sessionToRecord(s *Session) (*store.SessionRecord, error) {
	msgs := s.GetMessages()
	data, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("marshaling messages: %w", err)
	}
	return &store.SessionRecord{
		ID:        s.ID,
		Messages:  data,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}, nil
}
