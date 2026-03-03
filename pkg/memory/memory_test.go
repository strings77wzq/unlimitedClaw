package memory

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStoreAndRecall(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("learn about golang concurrency patterns", "golang", "concurrency")
	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	results, err := mem.Recall(ctx, "golang", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].ID != entry.ID {
		t.Errorf("Expected entry ID %s, got %s", entry.ID, results[0].ID)
	}
}

func TestRecallByTags(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("important project deadline", "work", "deadline")
	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	results, err := mem.Recall(ctx, "deadline", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].ID != entry.ID {
		t.Errorf("Expected entry ID %s, got %s", entry.ID, results[0].ID)
	}
}

func TestRecallRelevance(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry1 := NewEntry("golang is great", "golang")
	entry2 := NewEntry("golang golang golang is amazing", "golang")

	if err := mem.Store(ctx, entry1); err != nil {
		t.Fatalf("Store entry1 failed: %v", err)
	}
	if err := mem.Store(ctx, entry2); err != nil {
		t.Fatalf("Store entry2 failed: %v", err)
	}

	results, err := mem.Recall(ctx, "golang", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].ID != entry2.ID {
		t.Errorf("Expected entry2 to rank higher, got %s first", results[0].ID)
	}
}

func TestRecallLimit(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		entry := NewEntry("test entry with keyword", "test")
		if err := mem.Store(ctx, entry); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	results, err := mem.Recall(ctx, "keyword", 3)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestRecallNoMatch(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("golang concurrency patterns", "golang")
	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	results, err := mem.Recall(ctx, "python", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestForget(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("temporary note", "temp")
	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if err := mem.Forget(ctx, entry.ID); err != nil {
		t.Fatalf("Forget failed: %v", err)
	}

	results, err := mem.Recall(ctx, "temporary", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results after forget, got %d", len(results))
	}
}

func TestList(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry1 := NewEntry("first entry", "test")
	time.Sleep(10 * time.Millisecond)
	entry2 := NewEntry("second entry", "test")
	time.Sleep(10 * time.Millisecond)
	entry3 := NewEntry("third entry", "test")

	if err := mem.Store(ctx, entry1); err != nil {
		t.Fatalf("Store entry1 failed: %v", err)
	}
	if err := mem.Store(ctx, entry2); err != nil {
		t.Fatalf("Store entry2 failed: %v", err)
	}
	if err := mem.Store(ctx, entry3); err != nil {
		t.Fatalf("Store entry3 failed: %v", err)
	}

	results, err := mem.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if results[0].ID != entry3.ID {
		t.Errorf("Expected entry3 first (most recent), got %s", results[0].ID)
	}
	if results[2].ID != entry1.ID {
		t.Errorf("Expected entry1 last (oldest), got %s", results[2].ID)
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "memory.json")
	ctx := context.Background()

	mem1, err := NewFileMemory(filePath)
	if err != nil {
		t.Fatalf("NewFileMemory failed: %v", err)
	}

	entry1 := NewEntry("persistent entry 1", "test")
	entry2 := NewEntry("persistent entry 2", "test")

	if err := mem1.Store(ctx, entry1); err != nil {
		t.Fatalf("Store entry1 failed: %v", err)
	}
	if err := mem1.Store(ctx, entry2); err != nil {
		t.Fatalf("Store entry2 failed: %v", err)
	}

	mem2, err := NewFileMemory(filePath)
	if err != nil {
		t.Fatalf("NewFileMemory (second instance) failed: %v", err)
	}

	results, err := mem2.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 persisted entries, got %d", len(results))
	}
}

func TestStoreUpdate(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("original content", "test")
	entry.ID = "test-id"

	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	updatedEntry := NewEntry("updated content", "test")
	updatedEntry.ID = "test-id"

	time.Sleep(10 * time.Millisecond)
	if err := mem.Store(ctx, updatedEntry); err != nil {
		t.Fatalf("Store update failed: %v", err)
	}

	results, err := mem.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 entry (not duplicated), got %d", len(results))
	}

	if results[0].Content != "updated content" {
		t.Errorf("Expected updated content, got %s", results[0].Content)
	}

	if results[0].CreatedAt.Equal(results[0].UpdatedAt) {
		t.Error("Expected UpdatedAt to be different from CreatedAt")
	}
}

func TestNewEntry(t *testing.T) {
	entry := NewEntry("test content", "tag1", "tag2")

	if entry.ID == "" {
		t.Error("Expected ID to be generated")
	}

	if entry.Content != "test content" {
		t.Errorf("Expected content 'test content', got %s", entry.Content)
	}

	if len(entry.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(entry.Tags))
	}

	if entry.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if entry.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	if !entry.CreatedAt.Equal(entry.UpdatedAt) {
		t.Error("Expected CreatedAt and UpdatedAt to be equal for new entry")
	}

	if entry.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "memory.json")
	ctx := context.Background()

	mem, err := NewFileMemory(filePath)
	if err != nil {
		t.Fatalf("NewFileMemory failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entry := NewEntry("concurrent entry", "test")
			if err := mem.Store(ctx, entry); err != nil {
				t.Errorf("Concurrent store failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) == 0 {
		t.Error("File is empty, atomic write may have failed")
	}

	mem2, err := NewFileMemory(filePath)
	if err != nil {
		t.Fatalf("NewFileMemory after concurrent writes failed: %v", err)
	}

	results, err := mem2.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 entries, got %d - file may be corrupted", len(results))
	}
}

func TestInMemoryStore(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("in-memory test", "test")
	if err := mem.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	results, err := mem.Recall(ctx, "in-memory", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if err := mem.Forget(ctx, entry.ID); err != nil {
		t.Fatalf("Forget failed: %v", err)
	}

	results, err = mem.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results after forget, got %d", len(results))
	}
}

func TestConcurrentAccess(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entry := NewEntry("concurrent test", "test")
			if err := mem.Store(ctx, entry); err != nil {
				t.Errorf("Concurrent store failed: %v", err)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mem.Recall(ctx, "concurrent", 5)
			if err != nil {
				t.Errorf("Concurrent recall failed: %v", err)
			}
		}()
	}

	wg.Wait()

	results, err := mem.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 entries after concurrent operations, got %d", len(results))
	}
}
