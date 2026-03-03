package rag

import (
	"context"
	"math"
	"sort"
	"sync"
)

type Document struct {
	ID       string
	Content  string
	Metadata map[string]string
	Vector   []float64
}

type SearchResult struct {
	Document Document
	Score    float64
}

type VectorStore interface {
	Add(ctx context.Context, docs []Document) error
	Search(ctx context.Context, query []float64, topK int) ([]SearchResult, error)
	Delete(ctx context.Context, ids []string) error
	Count() int
}

type MemoryVectorStore struct {
	mu        sync.RWMutex
	documents map[string]Document
	docList   []Document
}

func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{
		documents: make(map[string]Document),
		docList:   make([]Document, 0),
	}
}

func (m *MemoryVectorStore) Add(ctx context.Context, docs []Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, doc := range docs {
		if _, exists := m.documents[doc.ID]; exists {
			for i, d := range m.docList {
				if d.ID == doc.ID {
					m.docList[i] = doc
					break
				}
			}
		} else {
			m.docList = append(m.docList, doc)
		}
		m.documents[doc.ID] = doc
	}

	return nil
}

func (m *MemoryVectorStore) Search(ctx context.Context, query []float64, topK int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]SearchResult, 0, len(m.docList))

	for _, doc := range m.docList {
		score := cosineSimilarity(query, doc.Vector)
		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

func (m *MemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
		delete(m.documents, id)
	}

	newList := make([]Document, 0, len(m.docList))
	for _, doc := range m.docList {
		if !idSet[doc.ID] {
			newList = append(newList, doc)
		}
	}
	m.docList = newList

	return nil
}

func (m *MemoryVectorStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docList)
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}
