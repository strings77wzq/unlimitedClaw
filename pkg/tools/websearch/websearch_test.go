package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type redirectTransport struct {
	serverURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.serverURL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

const mockHTMLWithResults = `
<!DOCTYPE html>
<html>
<body>
	<div class="result">
		<a class="result__a" href="https://example.com/page1">Example Page 1</a>
		<a class="result__snippet">This is the first example snippet with useful information.</a>
	</div>
	<div class="result">
		<a class="result__a" href="https://example.com/page2">Example Page 2</a>
		<a class="result__snippet">This is the second example snippet with more details.</a>
	</div>
	<div class="result">
		<a class="result__a" href="https://example.com/page3">Example Page 3</a>
		<a class="result__snippet">This is the third example snippet.</a>
	</div>
</body>
</html>
`

const mockHTMLNoResults = `
<!DOCTYPE html>
<html>
<body>
	<p>No results found</p>
</body>
</html>
`

func TestSearchResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTMLWithResults))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &redirectTransport{serverURL: server.URL},
	}

	tool := New(WithHTTPClient(client))

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test query",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Result marked as error: %s", result.ForLLM)
	}

	if result.ForLLM == "" {
		t.Error("ForLLM is empty")
	}

	if result.ForUser == "" {
		t.Error("ForUser is empty")
	}

	if !strings.Contains(result.ForLLM, "Example Page 1") {
		t.Errorf("ForLLM doesn't contain expected title. Got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "https://example.com/page1") {
		t.Errorf("ForLLM doesn't contain expected URL. Got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "first example snippet") {
		t.Errorf("ForLLM doesn't contain expected snippet. Got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForUser, "🔍 Results for 'test query':") {
		t.Errorf("ForUser doesn't contain expected header. Got: %s", result.ForUser)
	}

	if !strings.Contains(result.ForUser, "• Example Page 1 - https://example.com/page1") {
		t.Errorf("ForUser doesn't contain expected result. Got: %s", result.ForUser)
	}
}

func TestEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTMLNoResults))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &redirectTransport{serverURL: server.URL},
	}

	tool := New(WithHTTPClient(client))

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "nonexistent query",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Result marked as error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "No results found") {
		t.Errorf("Expected 'No results found' message. Got: %s", result.ForLLM)
	}
}

func TestSearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &redirectTransport{serverURL: server.URL},
	}

	tool := New(WithHTTPClient(client))

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test query",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !result.IsError {
		t.Error("Result should be marked as error")
	}

	if !strings.Contains(result.ForLLM, "Search failed") {
		t.Errorf("Expected error message in ForLLM. Got: %s", result.ForLLM)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTMLWithResults))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &redirectTransport{serverURL: server.URL},
	}

	tool := New(WithHTTPClient(client))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test query",
	})

	if err == nil {
		t.Fatal("Expected error due to cancelled context, got nil")
	}

	if !result.IsError {
		t.Error("Result should be marked as error")
	}

	if !strings.Contains(result.ForLLM, "context canceled") {
		t.Errorf("Expected context cancellation error in ForLLM. Got: %s", result.ForLLM)
	}
}

func TestMaxResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTMLWithResults))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &redirectTransport{serverURL: server.URL},
	}

	tool := New(WithHTTPClient(client), WithMaxResults(2))

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test query",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Result marked as error: %s", result.ForLLM)
	}

	if strings.Contains(result.ForLLM, "Example Page 3") {
		t.Errorf("Should not contain third result when maxResults=2. Got: %s", result.ForLLM)
	}

	llmLines := strings.Split(result.ForLLM, "\n")
	resultCount := 0
	for _, line := range llmLines {
		if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") {
			resultCount++
		}
	}

	if resultCount != 2 {
		t.Errorf("Expected 2 results, found %d", resultCount)
	}
}

func TestParameterValidation(t *testing.T) {
	tool := New()

	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error for missing query parameter")
	}

	if !result.IsError {
		t.Error("Result should be marked as error")
	}

	if !strings.Contains(result.ForLLM, "query") {
		t.Errorf("Expected error message about missing query. Got: %s", result.ForLLM)
	}

	result2, err2 := tool.Execute(context.Background(), map[string]interface{}{
		"query": "",
	})

	if err2 == nil {
		t.Fatal("Expected error for empty query parameter")
	}

	if !result2.IsError {
		t.Error("Result should be marked as error for empty query")
	}

	result3, err3 := tool.Execute(context.Background(), map[string]interface{}{
		"query": 123,
	})

	if err3 == nil {
		t.Fatal("Expected error for non-string query parameter")
	}

	if !result3.IsError {
		t.Error("Result should be marked as error for non-string query")
	}
}
