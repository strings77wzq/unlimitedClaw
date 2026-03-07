package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/unlimitedClaw/core/providers"
	"github.com/strings77wzq/unlimitedClaw/core/tools"
)

func TestChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header with Bearer test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if reqBody["model"] != "gpt-4" {
			t.Errorf("Expected model gpt-4, got %v", reqBody["model"])
		}

		resp := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello! How can I help you?",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hello"},
	}

	result, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if result.Content != "Hello! How can I help you?" {
		t.Errorf("Expected content 'Hello! How can I help you?', got %s", result.Content)
	}
	if result.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", result.Model)
	}
	if result.StopReason != "stop" {
		t.Errorf("Expected stop reason 'stop', got %s", result.StopReason)
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 20 {
		t.Errorf("Expected 20 completion tokens, got %d", result.Usage.CompletionTokens)
	}
	if result.Usage.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		toolsRaw, ok := reqBody["tools"]
		if !ok {
			t.Fatalf("Expected tools in request body")
		}
		toolsArr := toolsRaw.([]interface{})
		if len(toolsArr) != 1 {
			t.Fatalf("Expected 1 tool, got %d", len(toolsArr))
		}

		tool := toolsArr[0].(map[string]interface{})
		if tool["type"] != "function" {
			t.Errorf("Expected tool type 'function', got %v", tool["type"])
		}

		function := tool["function"].(map[string]interface{})
		if function["name"] != "get_weather" {
			t.Errorf("Expected function name 'get_weather', got %v", function["name"])
		}

		params := function["parameters"].(map[string]interface{})
		if params["type"] != "object" {
			t.Errorf("Expected parameters type 'object', got %v", params["type"])
		}

		resp := map[string]interface{}{
			"id":      "chatcmpl-456",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": `{"location":"Tokyo"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     15,
				"completion_tokens": 10,
				"total_tokens":      25,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What's the weather in Tokyo?"},
	}
	toolDefs := []tools.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			Parameters: []tools.ToolParameter{
				{Name: "location", Type: "string", Description: "City name", Required: true},
			},
		},
	}

	result, err := provider.Chat(context.Background(), messages, toolDefs, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got %s", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got %s", tc.Name)
	}
	if tc.Arguments["location"] != "Tokyo" {
		t.Errorf("Expected location 'Tokyo', got %v", tc.Arguments["location"])
	}
	if result.StopReason != "tool_calls" {
		t.Errorf("Expected stop reason 'tool_calls', got %s", result.StopReason)
	}
}

func TestToolCallArgumentParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-789",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_456",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "complex_tool",
									"arguments": `{"count":42,"items":["a","b"],"nested":{"key":"value"}}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     5,
				"completion_tokens": 5,
				"total_tokens":      10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	result, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.Arguments["count"].(float64) != 42 {
		t.Errorf("Expected count 42, got %v", tc.Arguments["count"])
	}

	items := tc.Arguments["items"].([]interface{})
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Errorf("Expected items [a,b], got %v", items)
	}

	nested := tc.Arguments["nested"].(map[string]interface{})
	if nested["key"] != "value" {
		t.Errorf("Expected nested.key=value, got %v", nested["key"])
	}
}

func TestRetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		resp := map[string]interface{}{
			"id":      "chatcmpl-retry",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Success after retry"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	result, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if result.Content != "Success after retry" {
		t.Errorf("Expected 'Success after retry', got %s", result.Content)
	}
}

func TestRetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
			return
		}

		resp := map[string]interface{}{
			"id":      "chatcmpl-500",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Recovered"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	result, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if result.Content != "Recovered" {
		t.Errorf("Expected 'Recovered', got %s", result.Content)
	}
}

func TestNoRetryOnClientError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	_, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err == nil {
		t.Fatal("Expected error for 401, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
	}
}

func TestCustomAPIBase(t *testing.T) {
	customPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customPath = r.URL.Path
		resp := map[string]interface{}{
			"id":      "chatcmpl-custom",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Custom base"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	_, err := provider.Chat(context.Background(), messages, nil, "gpt-4", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if customPath != "/v1/chat/completions" {
		t.Errorf("Expected path /v1/chat/completions, got %s", customPath)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Chat(ctx, messages, nil, "gpt-4", nil)
	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}
}

func TestChatOptions(t *testing.T) {
	var receivedReq map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		resp := map[string]interface{}{
			"id":      "chatcmpl-opts",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Options test"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}

	temp := 0.7
	maxTokens := 1024
	topP := 0.9
	opts := &providers.ChatOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
		Stop:        []string{"STOP", "END"},
	}

	_, err := provider.Chat(context.Background(), messages, nil, "gpt-4", opts)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if receivedReq["temperature"].(float64) != 0.7 {
		t.Errorf("Expected temperature 0.7, got %v", receivedReq["temperature"])
	}
	if receivedReq["max_tokens"].(float64) != 1024 {
		t.Errorf("Expected max_tokens 1024, got %v", receivedReq["max_tokens"])
	}
	if receivedReq["top_p"].(float64) != 0.9 {
		t.Errorf("Expected top_p 0.9, got %v", receivedReq["top_p"])
	}

	stop := receivedReq["stop"].([]interface{})
	if len(stop) != 2 || stop[0] != "STOP" || stop[1] != "END" {
		t.Errorf("Expected stop [STOP, END], got %v", stop)
	}
}

func TestName(t *testing.T) {
	provider := New("test-key")
	if provider.Name() != "openai" {
		t.Errorf("Expected provider name 'openai', got %s", provider.Name())
	}
}

func TestChatStreamTextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["stream"] != true {
			t.Error("Expected stream=true in request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`{"id":"chatcmpl-1","model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-1","model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-1","model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-1","model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	var tokens []string
	result, err := provider.ChatStream(context.Background(), messages, nil, "gpt-4", nil, func(token string) {
		tokens = append(tokens, token)
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	if result.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello world")
	}
	if result.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", result.Model, "gpt-4")
	}
	if result.StopReason != "stop" {
		t.Errorf("StopReason = %q, want %q", result.StopReason, "stop")
	}
	if result.Usage.PromptTokens != 5 || result.Usage.CompletionTokens != 2 || result.Usage.TotalTokens != 7 {
		t.Errorf("Usage = %+v, want {5, 2, 7}", result.Usage)
	}
	if len(tokens) != 2 || tokens[0] != "Hello" || tokens[1] != " world" {
		t.Errorf("tokens = %v, want [Hello, ' world']", tokens)
	}
}

func TestChatStreamToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Tokyo\"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-2","model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":8,"total_tokens":18}}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Weather in Tokyo?"},
	}

	result, err := provider.ChatStream(context.Background(), messages, nil, "gpt-4", nil, nil)
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	if result.StopReason != "tool_calls" {
		t.Errorf("StopReason = %q, want tool_calls", result.StopReason)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want call_abc", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want get_weather", tc.Name)
	}
	if tc.Arguments["location"] != "Tokyo" {
		t.Errorf("ToolCall.Arguments[location] = %v, want Tokyo", tc.Arguments["location"])
	}
}

func TestChatStreamHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	provider := New("bad-key", WithAPIBase(server.URL))
	_, err := provider.ChatStream(context.Background(), []providers.Message{{Role: providers.RoleUser, Content: "Hi"}}, nil, "gpt-4", nil, nil)
	if err == nil {
		t.Fatal("Expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Error = %v, want 401", err)
	}
}

func TestStreamingProviderInterface(t *testing.T) {
	provider := New("test-key")
	var _ providers.StreamingProvider = provider
}
