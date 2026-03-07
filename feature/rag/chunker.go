package rag

import (
	"unicode"
)

type ChunkerConfig struct {
	ChunkSize    int
	ChunkOverlap int
}

type Chunk struct {
	Content  string
	Index    int
	Metadata map[string]string
}

type Chunker struct {
	config ChunkerConfig
}

func NewChunker(cfg ChunkerConfig) *Chunker {
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 500
	}
	if cfg.ChunkOverlap < 0 {
		cfg.ChunkOverlap = 50
	}
	if cfg.ChunkOverlap >= cfg.ChunkSize {
		cfg.ChunkOverlap = cfg.ChunkSize / 2
	}
	return &Chunker{config: cfg}
}

func (c *Chunker) Split(text string, metadata map[string]string) []Chunk {
	if len(text) == 0 {
		return []Chunk{}
	}

	if len(text) <= c.config.ChunkSize {
		chunk := Chunk{
			Content:  text,
			Index:    0,
			Metadata: copyMetadata(metadata),
		}
		chunk.Metadata["chunk_index"] = "0"
		return []Chunk{chunk}
	}

	chunks := []Chunk{}
	start := 0
	index := 0

	for start < len(text) {
		end := start + c.config.ChunkSize

		if end >= len(text) {
			chunk := Chunk{
				Content:  text[start:],
				Index:    index,
				Metadata: copyMetadata(metadata),
			}
			chunk.Metadata["chunk_index"] = string(rune('0' + index))
			chunks = append(chunks, chunk)
			break
		}

		actualEnd := findWordBoundary(text, end)

		chunk := Chunk{
			Content:  text[start:actualEnd],
			Index:    index,
			Metadata: copyMetadata(metadata),
		}
		chunk.Metadata["chunk_index"] = string(rune('0' + index))
		chunks = append(chunks, chunk)

		start = actualEnd - c.config.ChunkOverlap
		if start <= chunks[len(chunks)-1].Index {
			start = actualEnd
		}
		index++
	}

	return chunks
}

func findWordBoundary(text string, pos int) int {
	if pos >= len(text) {
		return len(text)
	}

	for i := pos; i < len(text) && i < pos+50; i++ {
		if unicode.IsSpace(rune(text[i])) {
			return i
		}
	}

	for i := pos; i > 0 && i > pos-50; i-- {
		if unicode.IsSpace(rune(text[i])) {
			return i
		}
	}

	return pos
}

func copyMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}

	copy := make(map[string]string, len(metadata)+1)
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}
