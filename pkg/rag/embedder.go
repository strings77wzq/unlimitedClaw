package rag

import (
	"context"
	"hash/fnv"
	"math"
)

// Embedder converts text into embedding vectors.
type Embedder interface {
	// Embed converts a single text into an embedding vector.
	Embed(ctx context.Context, text string) ([]float64, error)

	// EmbedBatch converts multiple texts into embedding vectors.
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)

	// Dimension returns the dimensionality of the embedding vectors.
	Dimension() int
}

// MockEmbedder is a deterministic mock embedder for testing.
// It generates embeddings by hashing the input text to seed a pseudo-random generator.
type MockEmbedder struct {
	dimension int
}

// NewMockEmbedder creates a new mock embedder with the specified dimension.
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		dimension: dimension,
	}
}

// Embed generates a deterministic embedding for the given text.
// The same text always produces the same embedding.
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// Hash the text to get a deterministic seed
	h := fnv.New64a()
	h.Write([]byte(text))
	seed := h.Sum64()

	// Generate pseudo-random vector using simple LCG
	vector := make([]float64, m.dimension)
	rng := seed
	for i := 0; i < m.dimension; i++ {
		// Linear congruential generator
		rng = rng*6364136223846793005 + 1442695040888963407
		// Map to [-1, 1]
		vector[i] = float64(int64(rng)) / float64(math.MaxInt64)
	}

	// Normalize to unit length
	var sumSquares float64
	for _, v := range vector {
		sumSquares += v * v
	}
	norm := math.Sqrt(sumSquares)

	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		emb, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// Dimension returns the dimensionality of the embedding vectors.
func (m *MockEmbedder) Dimension() int {
	return m.dimension
}
