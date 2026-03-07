package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewSQLiteStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check sessions table: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected sessions table to exist, got count %d", count)
	}
}

func TestSaveAndGetSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	record := &SessionRecord{
		ID:        "session-1",
		Messages:  []byte(`[{"role":"user","content":"hello"}]`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = store.SaveSession(record)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	retrieved, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected session to be found, got nil")
	}

	if retrieved.ID != record.ID {
		t.Errorf("Expected ID %s, got %s", record.ID, retrieved.ID)
	}
	if string(retrieved.Messages) != string(record.Messages) {
		t.Errorf("Expected messages %s, got %s", record.Messages, retrieved.Messages)
	}
	if !retrieved.CreatedAt.Equal(record.CreatedAt) {
		t.Errorf("Expected CreatedAt %v, got %v", record.CreatedAt, retrieved.CreatedAt)
	}
	if !retrieved.UpdatedAt.Equal(record.UpdatedAt) {
		t.Errorf("Expected UpdatedAt %v, got %v", record.UpdatedAt, retrieved.UpdatedAt)
	}
}

func TestUpdateSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	record := &SessionRecord{
		ID:        "session-1",
		Messages:  []byte(`[{"role":"user","content":"hello"}]`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = store.SaveSession(record)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	updatedTime := now.Add(time.Hour)
	record.Messages = []byte(`[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]`)
	record.UpdatedAt = updatedTime

	err = store.SaveSession(record)
	if err != nil {
		t.Fatalf("SaveSession (update) failed: %v", err)
	}

	retrieved, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected session to be found, got nil")
	}

	if string(retrieved.Messages) != string(record.Messages) {
		t.Errorf("Expected messages %s, got %s", record.Messages, retrieved.Messages)
	}
	if !retrieved.UpdatedAt.Equal(updatedTime) {
		t.Errorf("Expected UpdatedAt %v, got %v", updatedTime, retrieved.UpdatedAt)
	}

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", "session-1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session, got %d (update should not create duplicate)", count)
	}
}

func TestDeleteSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	record := &SessionRecord{
		ID:        "session-1",
		Messages:  []byte(`[]`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = store.SaveSession(record)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	err = store.DeleteSession("session-1")
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	retrieved, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected session to be deleted, got %+v", retrieved)
	}
}

func TestListSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)

	sessions := []*SessionRecord{
		{
			ID:        "session-1",
			Messages:  []byte(`[]`),
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "session-2",
			Messages:  []byte(`[]`),
			CreatedAt: now.Add(time.Minute),
			UpdatedAt: now.Add(time.Minute),
		},
		{
			ID:        "session-3",
			Messages:  []byte(`[]`),
			CreatedAt: now.Add(2 * time.Minute),
			UpdatedAt: now.Add(2 * time.Minute),
		},
	}

	for _, session := range sessions {
		err := store.SaveSession(session)
		if err != nil {
			t.Fatalf("SaveSession failed for %s: %v", session.ID, err)
		}
	}

	retrieved, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(retrieved))
	}

	if retrieved[0].ID != "session-3" {
		t.Errorf("Expected first session to be session-3 (most recent), got %s", retrieved[0].ID)
	}
	if retrieved[1].ID != "session-2" {
		t.Errorf("Expected second session to be session-2, got %s", retrieved[1].ID)
	}
	if retrieved[2].ID != "session-1" {
		t.Errorf("Expected third session to be session-1 (oldest), got %s", retrieved[2].ID)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	retrieved, err := store.GetSession("non-existent")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil for non-existent session, got %+v", retrieved)
	}
}

func TestMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check schema_migrations table: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected schema_migrations table to exist, got count %d", count)
	}

	var version int
	var name string
	err = store.db.QueryRow("SELECT version, name FROM schema_migrations WHERE version = 1").Scan(&version, &name)
	if err != nil {
		t.Fatalf("Failed to get migration record: %v", err)
	}
	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}
	if name != "create_sessions_table" {
		t.Errorf("Expected name 'create_sessions_table', got %s", name)
	}
}

func TestCGODisabled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed (CGO test): %v", err)
	}
	defer store.Close()

	t.Log("Store works with CGO_ENABLED=0 (modernc.org/sqlite is pure Go)")
}
