package rag

import (
	"context"
	"fmt"
)

type RawDocument struct {
	ID       string
	Content  string
	Metadata map[string]string
}

type Retriever struct {
	embedder Embedder
	store    VectorStore
	chunker  *Chunker
	topK     int
}

func NewRetriever(embedder Embedder, store VectorStore, topK int) *Retriever {
	return &Retriever{
		embedder: embedder,
		store:    store,
		chunker:  NewChunker(ChunkerConfig{ChunkSize: 500, ChunkOverlap: 50}),
		topK:     topK,
	}
}

func (r *Retriever) AddDocuments(ctx context.Context, docs []RawDocument) error {
	var allChunks []Document

	for _, doc := range docs {
		chunks := r.chunker.Split(doc.Content, doc.Metadata)

		texts := make([]string, len(chunks))
		for i, chunk := range chunks {
			texts[i] = chunk.Content
		}

		embeddings, err := r.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return fmt.Errorf("failed to embed chunks: %w", err)
		}

		for i, chunk := range chunks {
			chunkDoc := Document{
				ID:       fmt.Sprintf("%s_chunk_%d", doc.ID, chunk.Index),
				Content:  chunk.Content,
				Metadata: chunk.Metadata,
				Vector:   embeddings[i],
			}
			chunkDoc.Metadata["source_doc_id"] = doc.ID
			allChunks = append(allChunks, chunkDoc)
		}
	}

	return r.store.Add(ctx, allChunks)
}

func (r *Retriever) Query(ctx context.Context, query string) ([]SearchResult, error) {
	queryVec, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return r.store.Search(ctx, queryVec, r.topK)
}

func (r *Retriever) SetChunker(chunker *Chunker) {
	r.chunker = chunker
}
