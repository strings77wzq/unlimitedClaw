package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func TestSQLiteAdapter_SaveAndGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter: %v", err)
	}
	defer adapter.Close()

	sess := session.NewSession("test-1")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "hi there"})

	if err := adapter.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok := adapter.Get("test-1")
	if !ok {
		t.Fatal("Get returned false for saved session")
	}
	if got.ID != "test-1" {
		t.Errorf("ID = %q, want %q", got.ID, "test-1")
	}
	msgs := got.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("messages count = %d, want 2", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Errorf("msgs[0].Content = %q, want %q", msgs[0].Content, "hello")
	}
	if msgs[1].Role != providers.RoleAssistant {
		t.Errorf("msgs[1].Role = %q, want %q", msgs[1].Role, providers.RoleAssistant)
	}
}

func TestSQLiteAdapter_List(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter: %v", err)
	}
	defer adapter.Close()

	ids := []string{"s1", "s2", "s3"}
	for _, id := range ids {
		s := session.NewSession(id)
		s.AddMessage(providers.Message{Role: providers.RoleUser, Content: "msg-" + id})
		if err := adapter.Save(s); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	list := adapter.List()
	if len(list) != 3 {
		t.Fatalf("List count = %d, want 3", len(list))
	}

	found := make(map[string]bool)
	for _, s := range list {
		found[s.ID] = true
	}
	for _, id := range ids {
		if !found[id] {
			t.Errorf("session %q not found in List()", id)
		}
	}
}

func TestSQLiteAdapter_Delete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter: %v", err)
	}
	defer adapter.Close()

	sess := session.NewSession("del-1")
	if err := adapter.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := adapter.Delete("del-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := adapter.Get("del-1")
	if ok {
		t.Error("Get returned true after Delete")
	}
}

func TestSQLiteAdapter_GetNotFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter: %v", err)
	}
	defer adapter.Close()

	_, ok := adapter.Get("nonexistent")
	if ok {
		t.Error("Get returned true for nonexistent session")
	}
}

func TestSQLiteAdapter_Persistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Write with one adapter instance
	adapter1, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter: %v", err)
	}
	sess := session.NewSession("persist-1")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "persisted"})
	if err := adapter1.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	adapter1.Close()

	// Read with a new adapter instance
	adapter2, err := session.NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteAdapter (reopen): %v", err)
	}
	defer adapter2.Close()

	got, ok := adapter2.Get("persist-1")
	if !ok {
		t.Fatal("session not found after reopen")
	}
	msgs := got.GetMessages()
	if len(msgs) != 1 || msgs[0].Content != "persisted" {
		t.Errorf("unexpected messages after reopen: %+v", msgs)
	}
}

func TestSQLiteAdapter_BadPath(t *testing.T) {
	_, err := session.NewSQLiteAdapter(filepath.Join(os.DevNull, "impossible", "path.db"))
	if err == nil {
		t.Error("expected error for bad path, got nil")
	}
}
