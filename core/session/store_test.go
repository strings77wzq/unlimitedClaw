package session

import (
	"sync"
	"testing"

	"github.com/strings77wzq/unlimitedClaw/core/providers"
)

func TestMemoryStoreSaveAndGet(t *testing.T) {
	store := NewMemoryStore()
	session := NewSession("test-id")
	session.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})

	if err := store.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	retrieved, ok := store.Get("test-id")
	if !ok {
		t.Fatal("expected session to be found")
	}

	if retrieved.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", retrieved.ID)
	}

	if retrieved.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", retrieved.MessageCount())
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryStore()

	session, ok := store.Get("non-existent")
	if ok {
		t.Error("expected session to not be found")
	}

	if session != nil {
		t.Error("expected nil session for non-existent ID")
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()
	session := NewSession("test-id")

	if err := store.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, ok := store.Get("test-id")
	if !ok {
		t.Fatal("expected session to be found before delete")
	}

	if err := store.Delete("test-id"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok = store.Get("test-id")
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestMemoryStoreList(t *testing.T) {
	store := NewMemoryStore()

	session1 := NewSession("id1")
	session2 := NewSession("id2")
	session3 := NewSession("id3")

	if err := store.Save(session1); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := store.Save(session2); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := store.Save(session3); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sessions := store.List()

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
	}

	if !ids["id1"] || !ids["id2"] || !ids["id3"] {
		t.Error("expected all three session IDs in list")
	}
}

func TestMemoryStoreConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			session := NewSession("concurrent-test")
			session.AddMessage(providers.Message{
				Role:    providers.RoleUser,
				Content: "message",
			})
			if err := store.Save(session); err != nil {
				t.Errorf("Save failed: %v", err)
			}
			_, _ = store.Get("concurrent-test")
		}(i)
	}

	wg.Wait()

	session, ok := store.Get("concurrent-test")
	if !ok {
		t.Fatal("expected session to be found after concurrent access")
	}

	if session.ID != "concurrent-test" {
		t.Errorf("expected ID 'concurrent-test', got %q", session.ID)
	}
}
