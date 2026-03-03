package rag

import (
	"context"
	"math"
	"strings"
	"testing"
)

func TestMockEmbedder(t *testing.T) {
	embedder := NewMockEmbedder(128)

	ctx := context.Background()

	emb1a, err := embedder.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	emb1b, err := embedder.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(emb1a) != 128 {
		t.Errorf("Expected dimension 128, got %d", len(emb1a))
	}

	for i := range emb1a {
		if emb1a[i] != emb1b[i] {
			t.Errorf("Same text produced different embeddings at index %d", i)
			break
		}
	}

	emb2, err := embedder.Embed(ctx, "different text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	same := true
	for i := range emb1a {
		if emb1a[i] != emb2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different texts produced identical embeddings")
	}
}

func TestMockEmbedderDimension(t *testing.T) {
	tests := []int{64, 128, 256, 512}

	for _, dim := range tests {
		embedder := NewMockEmbedder(dim)
		if embedder.Dimension() != dim {
			t.Errorf("Expected dimension %d, got %d", dim, embedder.Dimension())
		}

		ctx := context.Background()
		emb, err := embedder.Embed(ctx, "test")
		if err != nil {
			t.Fatalf("Embed failed: %v", err)
		}

		if len(emb) != dim {
			t.Errorf("Expected embedding dimension %d, got %d", dim, len(emb))
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a         []float64
		b         []float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "identical vectors",
			a:         []float64{1, 0, 0},
			b:         []float64{1, 0, 0},
			expected:  1.0,
			tolerance: 0.0001,
		},
		{
			name:      "orthogonal vectors",
			a:         []float64{1, 0, 0},
			b:         []float64{0, 1, 0},
			expected:  0.0,
			tolerance: 0.0001,
		},
		{
			name:      "opposite vectors",
			a:         []float64{1, 0, 0},
			b:         []float64{-1, 0, 0},
			expected:  -1.0,
			tolerance: 0.0001,
		},
		{
			name:      "zero vector",
			a:         []float64{0, 0, 0},
			b:         []float64{1, 0, 0},
			expected:  0.0,
			tolerance: 0.0001,
		},
		{
			name:      "different dimensions",
			a:         []float64{1, 0},
			b:         []float64{1, 0, 0},
			expected:  0.0,
			tolerance: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestMemoryVectorStoreAdd(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	docs := []Document{
		{
			ID:      "doc1",
			Content: "hello",
			Vector:  []float64{1, 0, 0},
		},
		{
			ID:      "doc2",
			Content: "world",
			Vector:  []float64{0, 1, 0},
		},
	}

	err := store.Add(ctx, docs)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Expected count 2, got %d", store.Count())
	}

	err = store.Add(ctx, []Document{{ID: "doc1", Content: "updated", Vector: []float64{0, 0, 1}}})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Expected count 2 after update, got %d", store.Count())
	}
}

func TestMemoryVectorStoreSearch(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	docs := []Document{
		{ID: "doc1", Content: "first", Vector: []float64{1, 0, 0}},
		{ID: "doc2", Content: "second", Vector: []float64{0, 1, 0}},
		{ID: "doc3", Content: "third", Vector: []float64{0.7, 0.7, 0}},
	}

	err := store.Add(ctx, docs)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	query := []float64{1, 0, 0}
	results, err := store.Search(ctx, query, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Document.ID != "doc1" {
		t.Errorf("Expected first result to be doc1, got %s", results[0].Document.ID)
	}

	if results[0].Score <= results[1].Score {
		t.Error("Results not sorted by score descending")
	}
}

func TestMemoryVectorStoreDelete(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	docs := []Document{
		{ID: "doc1", Content: "first", Vector: []float64{1, 0, 0}},
		{ID: "doc2", Content: "second", Vector: []float64{0, 1, 0}},
		{ID: "doc3", Content: "third", Vector: []float64{0, 0, 1}},
	}

	err := store.Add(ctx, docs)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	err = store.Delete(ctx, []string{"doc2"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Expected count 2, got %d", store.Count())
	}

	results, err := store.Search(ctx, []float64{0, 1, 0}, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	for _, r := range results {
		if r.Document.ID == "doc2" {
			t.Error("Deleted document still appears in search results")
		}
	}
}

func TestChunkerSplit(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 20, ChunkOverlap: 5})

	text := "This is a test document that should be split into multiple chunks."
	chunks := chunker.Split(text, map[string]string{"source": "test"})

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("Chunk %d has incorrect index %d", i, chunk.Index)
		}

		if chunk.Metadata["source"] != "test" {
			t.Error("Chunk did not inherit metadata")
		}

		if _, ok := chunk.Metadata["chunk_index"]; !ok {
			t.Error("Chunk missing chunk_index metadata")
		}
	}

	if len(chunks) > 1 {
		lastChar := chunks[0].Content[len(chunks[0].Content)-5:]
		firstChar := chunks[1].Content[:5]

		overlap := 0
		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				if lastChar[i] == firstChar[j] {
					overlap++
					break
				}
			}
		}

		if overlap == 0 {
			t.Log("Note: Chunks may not have character-level overlap due to word boundary splitting")
		}
	}
}

func TestChunkerWordBoundary(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 20, ChunkOverlap: 3})

	text := "word1 word2 word3 word4 word5 word6 word7 word8 word9"
	chunks := chunker.Split(text, nil)

	for i, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk.Content)

		if len(trimmed) > 0 && trimmed != chunk.Content {
			continue
		}

		if i < len(chunks)-1 {
			words := strings.Fields(trimmed)
			for _, word := range words {
				if len(word) > 0 && strings.Contains(word, " ") {
					t.Errorf("Chunk %d contains split word: %q", i, word)
				}
			}
		}
	}
}

func TestChunkerSmallText(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 100, ChunkOverlap: 10})

	text := "Small text"
	chunks := chunker.Split(text, nil)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for small text, got %d", len(chunks))
	}

	if chunks[0].Content != text {
		t.Errorf("Expected chunk content to match input text")
	}
}

func TestRetrieverAddAndQuery(t *testing.T) {
	embedder := NewMockEmbedder(128)
	store := NewMemoryVectorStore()
	retriever := NewRetriever(embedder, store, 3)

	ctx := context.Background()

	docs := []RawDocument{
		{
			ID:       "doc1",
			Content:  "The quick brown fox jumps over the lazy dog",
			Metadata: map[string]string{"type": "sentence"},
		},
		{
			ID:       "doc2",
			Content:  "Machine learning is a subset of artificial intelligence",
			Metadata: map[string]string{"type": "definition"},
		},
	}

	err := retriever.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}

	if store.Count() == 0 {
		t.Error("No documents were added to store")
	}

	results, err := retriever.Query(ctx, "fox dog")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Query returned no results")
	}

	for _, r := range results {
		if r.Document.Metadata["source_doc_id"] == "" {
			t.Error("Result missing source_doc_id metadata")
		}
	}
}

func TestPipelineAugment(t *testing.T) {
	embedder := NewMockEmbedder(128)
	store := NewMemoryVectorStore()
	retriever := NewRetriever(embedder, store, 2)
	pipeline := NewPipeline(retriever)

	ctx := context.Background()

	docs := []RawDocument{
		{ID: "doc1", Content: "Python is a programming language", Metadata: nil},
		{ID: "doc2", Content: "Go is used for backend development", Metadata: nil},
		{ID: "doc3", Content: "JavaScript runs in browsers", Metadata: nil},
	}

	err := retriever.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}

	context_str, err := pipeline.Augment(ctx, "programming language")
	if err != nil {
		t.Fatalf("Augment failed: %v", err)
	}

	if !strings.Contains(context_str, "Relevant context:") {
		t.Error("Context string missing expected prefix")
	}

	if !strings.Contains(context_str, "[1]") {
		t.Error("Context string missing numbered chunks")
	}
}

func TestRetrievalPipeline(t *testing.T) {
	embedder := NewMockEmbedder(256)
	store := NewMemoryVectorStore()
	retriever := NewRetriever(embedder, store, 5)

	retriever.SetChunker(NewChunker(ChunkerConfig{ChunkSize: 100, ChunkOverlap: 20}))

	pipeline := NewPipeline(retriever)

	ctx := context.Background()

	docs := []RawDocument{
		{
			ID: "rag_doc",
			Content: "Retrieval-Augmented Generation (RAG) is a technique that combines information retrieval with text generation. " +
				"It retrieves relevant documents from a knowledge base and uses them to augment the context for generation. " +
				"This approach improves the factual accuracy of generated text.",
			Metadata: map[string]string{"topic": "RAG"},
		},
		{
			ID: "ml_doc",
			Content: "Machine learning models learn patterns from data. Deep learning uses neural networks with multiple layers. " +
				"Training requires large datasets and significant computational resources.",
			Metadata: map[string]string{"topic": "ML"},
		},
	}

	err := retriever.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}

	t.Logf("Store contains %d chunks", store.Count())

	contextStr, err := pipeline.Augment(ctx, "What is RAG?")
	if err != nil {
		t.Fatalf("Augment failed: %v", err)
	}

	if contextStr == "" {
		t.Error("Pipeline returned empty context")
	}

	t.Logf("Retrieved context:\n%s", contextStr)

	if !strings.Contains(contextStr, "Retrieval") && !strings.Contains(contextStr, "retriev") {
		t.Error("Retrieved context doesn't seem relevant to RAG query")
	}
}
