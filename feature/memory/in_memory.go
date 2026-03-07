package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type InMemoryStore struct {
	entries map[string]*Entry
	mu      sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]*Entry),
	}
}

func (s *InMemoryStore) Store(ctx context.Context, entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.entries[entry.ID]; ok {
		entry.CreatedAt = existing.CreatedAt
	}
	entry.UpdatedAt = time.Now()

	s.entries[entry.ID] = entry
	return nil
}

func (s *InMemoryStore) Recall(ctx context.Context, query string, limit int) ([]*Entry, error) {
	if query == "" {
		return nil, nil
	}

	queryWords := strings.Fields(strings.ToLower(query))
	if len(queryWords) == 0 {
		return nil, nil
	}

	now := time.Now()

	type scoredEntry struct {
		entry *Entry
		score float64
	}

	// Read phase: score all entries under RLock
	s.mu.RLock()
	var scored []scoredEntry
	for _, entry := range s.entries {
		wordScore := 0
		contentLower := strings.ToLower(entry.Content)

		for _, word := range queryWords {
			wordScore += strings.Count(contentLower, word)

			for _, tag := range entry.Tags {
				wordScore += strings.Count(strings.ToLower(tag), word)
			}
		}

		if wordScore > 0 {
			decayed := entry.DecayedImportance(now, DefaultDecayLambda)
			finalScore := float64(wordScore) * (1.0 + decayed)
			scored = append(scored, scoredEntry{entry: entry, score: finalScore})
		}
	}
	s.mu.RUnlock()

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]*Entry, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].entry)
	}

	// Write phase: update AccessedAt under write lock
	s.mu.Lock()
	for _, entry := range result {
		entry.AccessedAt = now
	}
	s.mu.Unlock()

	return result, nil
}

func (s *InMemoryStore) Forget(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, id)
	return nil
}

func (s *InMemoryStore) List(ctx context.Context) ([]*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}
