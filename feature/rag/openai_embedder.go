package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const (
	defaultOpenAIBase       = "https://api.openai.com/v1"
	defaultEmbeddingModel   = "text-embedding-3-small"
	defaultEmbedDimension   = 1536
	embeddingMaxBatchSize   = 2048
	embeddingRequestTimeout = 30 * time.Second
)

type OpenAIEmbedder struct {
	apiKey     string
	apiBase    string
	model      string
	dimension  int
	httpClient *http.Client
}

type OpenAIEmbedderOption func(*OpenAIEmbedder)

func WithAPIBase(base string) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.apiBase = base }
}

func WithModel(model string) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.model = model }
}

func WithDimension(dim int) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.dimension = dim }
}

func WithHTTPClient(client *http.Client) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.httpClient = client }
}

func NewOpenAIEmbedder(apiKey string, opts ...OpenAIEmbedderOption) *OpenAIEmbedder {
	e := &OpenAIEmbedder{
		apiKey:     apiKey,
		apiBase:    defaultOpenAIBase,
		model:      defaultEmbeddingModel,
		dimension:  defaultEmbedDimension,
		httpClient: &http.Client{Timeout: embeddingRequestTimeout},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Error *apiError       `json:"error,omitempty"`
}

type embeddingData struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	batch, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return batch[0], nil
}

func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	result := make([][]float64, len(texts))

	for start := 0; start < len(texts); start += embeddingMaxBatchSize {
		end := start + embeddingMaxBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		chunk := texts[start:end]
		embeddings, err := e.requestEmbeddings(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("embedding batch [%d:%d]: %w", start, end, err)
		}

		for i, emb := range embeddings {
			result[start+i] = emb
		}
	}

	return result, nil
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

func (e *OpenAIEmbedder) requestEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: e.model,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := e.apiBase + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp embeddingResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(embResp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(embResp.Data))
	}

	embeddings := make([][]float64, len(texts))
	for _, d := range embResp.Data {
		normalized := normalize(d.Embedding)
		embeddings[d.Index] = normalized
	}

	return embeddings, nil
}

func normalize(vec []float64) []float64 {
	var sumSq float64
	for _, v := range vec {
		sumSq += v * v
	}
	norm := math.Sqrt(sumSq)
	if norm == 0 {
		return vec
	}
	out := make([]float64, len(vec))
	for i, v := range vec {
		out[i] = v / norm
	}
	return out
}
