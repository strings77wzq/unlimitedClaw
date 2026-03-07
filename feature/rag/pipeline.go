// Package rag implements a Retrieval-Augmented Generation pipeline. Documents
// are indexed with TF-IDF for keyword search and optionally with dense
// embeddings (via OpenAI Embeddings API) for semantic search. Retrieved
// chunks are prepended to the LLM context to ground responses in facts.
// This is a reference implementation in the feature/ layer and is NOT wired
// into main.go by default.
package rag

import (
	"context"
	"fmt"
	"strings"
)

type Pipeline struct {
	retriever *Retriever
}

func NewPipeline(retriever *Retriever) *Pipeline {
	return &Pipeline{
		retriever: retriever,
	}
}

func (p *Pipeline) Augment(ctx context.Context, query string) (string, error) {
	results, err := p.retriever.Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf("retrieval failed: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("Relevant context:\n\n")

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, result.Document.Content))
	}

	return sb.String(), nil
}
