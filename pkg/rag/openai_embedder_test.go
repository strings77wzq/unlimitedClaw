package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIEmbedderEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Input) != 1 || req.Input[0] != "hello world" {
			t.Errorf("unexpected input: %v", req.Input)
		}

		resp := embeddingResponse{
			Data: []embeddingData{
				{Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key",
		WithAPIBase(server.URL),
		WithDimension(3),
	)

	vec, err := embedder.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vec) != 3 {
		t.Fatalf("expected 3-dim vector, got %d", len(vec))
	}
}

func TestOpenAIEmbedderBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		data := make([]embeddingData, len(req.Input))
		for i := range req.Input {
			data[i] = embeddingData{Index: i, Embedding: []float64{float64(i), 0.5, 1.0}}
		}

		resp := embeddingResponse{Data: data}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", WithAPIBase(server.URL))

	vecs, err := embedder.EmbedBatch(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
}

func TestOpenAIEmbedderEmptyBatch(t *testing.T) {
	embedder := NewOpenAIEmbedder("test-key")
	vecs, err := embedder.EmbedBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vecs != nil {
		t.Errorf("expected nil for empty batch, got %v", vecs)
	}
}

func TestOpenAIEmbedderAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := embeddingResponse{
			Error: &apiError{Message: "invalid api key", Type: "invalid_request_error"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("bad-key", WithAPIBase(server.URL))

	_, err := embedder.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestOpenAIEmbedderDimension(t *testing.T) {
	embedder := NewOpenAIEmbedder("key", WithDimension(768))
	if embedder.Dimension() != 768 {
		t.Errorf("expected dimension 768, got %d", embedder.Dimension())
	}
}

func TestOpenAIEmbedderInterface(t *testing.T) {
	var _ Embedder = (*OpenAIEmbedder)(nil)
}
