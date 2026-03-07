package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

func TestChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       reqBody["model"],
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Hello! How can I help you today?"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hello"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello! How can I help you today?")
	}
	if resp.StopReason != "stop" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "stop")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 15 {
		t.Errorf("CompletionTokens = %d, want 15", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 25 {
		t.Errorf("TotalTokens = %d, want 25", resp.Usage.TotalTokens)
	}
}

func TestSystemMessageExtraction(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "You are helpful"},
		{Role: providers.RoleUser, Content: "Hi"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if receivedRequest["system"] != "You are helpful" {
		t.Errorf("system = %v, want %q", receivedRequest["system"], "You are helpful")
	}

	msgs, ok := receivedRequest["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages not array")
	}
	if len(msgs) != 1 {
		t.Errorf("len(messages) = %d, want 1 (system should be extracted)", len(msgs))
	}
}

func TestToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if _, ok := reqBody["tools"]; !ok {
			t.Error("expected tools in request")
		}

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "tool_use",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Let me check the weather"},
				{
					"type":  "tool_use",
					"id":    "toolu_1234",
					"name":  "get_weather",
					"input": map[string]interface{}{"city": "San Francisco"},
				},
			},
			"usage": map[string]interface{}{
				"input_tokens":  20,
				"output_tokens": 30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))

	toolDefs := []tools.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get weather for a city",
			Parameters: []tools.ToolParameter{
				{Name: "city", Type: "string", Description: "City name", Required: true},
			},
		},
	}

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What's the weather in SF?"},
	}

	resp, err := p.Chat(context.Background(), messages, toolDefs, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.Content != "Let me check the weather" {
		t.Errorf("Content = %q, want %q", resp.Content, "Let me check the weather")
	}

	if resp.StopReason != "tool_calls" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "tool_calls")
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_1234" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "toolu_1234")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Arguments["city"] != "San Francisco" {
		t.Errorf("ToolCall.Arguments[city] = %v, want %q", tc.Arguments["city"], "San Francisco")
	}
}

func TestToolResultFormat(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What's the weather?"},
		{
			Role:    providers.RoleAssistant,
			Content: "",
			ToolCalls: []providers.ToolCall{
				{ID: "toolu_1", Name: "get_weather", Arguments: map[string]interface{}{"city": "SF"}},
			},
		},
		{Role: providers.RoleTool, Content: `{"temp": 72}`, ToolCallID: "toolu_1"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	msgs, ok := receivedRequest["messages"].([]interface{})
	if !ok || len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	lastMsg := msgs[2].(map[string]interface{})
	if lastMsg["role"] != "user" {
		t.Errorf("tool result message role = %v, want user", lastMsg["role"])
	}

	content := lastMsg["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}

	block := content[0].(map[string]interface{})
	if block["type"] != "tool_result" {
		t.Errorf("content block type = %v, want tool_result", block["type"])
	}
	if block["tool_use_id"] != "toolu_1" {
		t.Errorf("tool_use_id = %v, want toolu_1", block["tool_use_id"])
	}
}

func TestRetryOn429(t *testing.T) {
	attempts := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)

		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit"}`))
			return
		}

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Success"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL), WithMaxRetries(3))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.Content != "Success" {
		t.Errorf("Content = %q, want Success", resp.Content)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("attempts = %d, want 2", atomic.LoadInt32(&attempts))
	}
}

func TestNoRetryOnClientError(t *testing.T) {
	attempts := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid API key"}`))
	}))
	defer server.Close()

	p := New("bad-key", WithAPIBase(server.URL), WithMaxRetries(3))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want 401", err)
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 401)", atomic.LoadInt32(&attempts))
	}
}

func TestCustomAPIBase(t *testing.T) {
	var requestURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if requestURL != "/v1/messages" {
		t.Errorf("requestURL = %q, want /v1/messages", requestURL)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := p.Chat(ctx, messages, nil, "claude-sonnet-4-20250514", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("error = %v, want context cancellation error", err)
	}
}

func TestChatOptions(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	temp := 0.7
	maxTokens := 2048
	topP := 0.9
	opts := &providers.ChatOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
		Stop:        []string{"STOP"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", opts)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if receivedRequest["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", receivedRequest["temperature"])
	}
	if receivedRequest["max_tokens"] != float64(2048) {
		t.Errorf("max_tokens = %v, want 2048", receivedRequest["max_tokens"])
	}
	if receivedRequest["top_p"] != 0.9 {
		t.Errorf("top_p = %v, want 0.9", receivedRequest["top_p"])
	}

	stopSeqs, ok := receivedRequest["stop_sequences"].([]interface{})
	if !ok || len(stopSeqs) != 1 || stopSeqs[0] != "STOP" {
		t.Errorf("stop_sequences = %v, want [STOP]", receivedRequest["stop_sequences"])
	}
}

func TestHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-api-key", WithAPIBase(server.URL), WithAPIVersion("2024-01-01"))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if receivedHeaders.Get("x-api-key") != "test-api-key" {
		t.Errorf("x-api-key header = %q, want test-api-key", receivedHeaders.Get("x-api-key"))
	}

	if receivedHeaders.Get("anthropic-version") != "2024-01-01" {
		t.Errorf("anthropic-version header = %q, want 2024-01-01", receivedHeaders.Get("anthropic-version"))
	}

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header = %q, want application/json", receivedHeaders.Get("Content-Type"))
	}
}

func TestName(t *testing.T) {
	p := New("test-key")
	if p.Name() != "anthropic" {
		t.Errorf("Name() = %q, want anthropic", p.Name())
	}
}

func TestMultipleSystemMessages(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)

		resp := map[string]interface{}{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-sonnet-4-20250514",
			"stop_reason": "end_turn",
			"content": []map[string]interface{}{
				{"type": "text", "text": "OK"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  5,
				"output_tokens": 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "You are helpful"},
		{Role: providers.RoleSystem, Content: "You are concise"},
		{Role: providers.RoleUser, Content: "Hi"},
	}

	_, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	expected := "You are helpful\n\nYou are concise"
	if receivedRequest["system"] != expected {
		t.Errorf("system = %v, want %q", receivedRequest["system"], expected)
	}
}

func TestStopReasonMapping(t *testing.T) {
	tests := []struct {
		name           string
		apiStopReason  string
		wantStopReason string
	}{
		{"end_turn", "end_turn", "stop"},
		{"tool_use", "tool_use", "tool_calls"},
		{"max_tokens", "max_tokens", "length"},
		{"other", "stop_sequence", "stop_sequence"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]interface{}{
					"id":          "msg_123",
					"type":        "message",
					"role":        "assistant",
					"model":       "claude-sonnet-4-20250514",
					"stop_reason": tt.apiStopReason,
					"content": []map[string]interface{}{
						{"type": "text", "text": "OK"},
					},
					"usage": map[string]interface{}{
						"input_tokens":  5,
						"output_tokens": 2,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			p := New("test-key", WithAPIBase(server.URL))
			messages := []providers.Message{
				{Role: providers.RoleUser, Content: "Hi"},
			}

			resp, err := p.Chat(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil)
			if err != nil {
				t.Fatalf("Chat() error: %v", err)
			}

			if resp.StopReason != tt.wantStopReason {
				t.Errorf("StopReason = %q, want %q", resp.StopReason, tt.wantStopReason)
			}
		})
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

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"model\":\"claude-sonnet-4-20250514\",\"usage\":{\"input_tokens\":25}}}\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":15}}\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n",
		}

		for _, ev := range events {
			fmt.Fprint(w, ev)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Hi"},
	}

	var tokens []string
	result, err := p.ChatStream(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil, func(token string) {
		tokens = append(tokens, token)
	})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}

	if result.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello world")
	}
	if result.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want claude-sonnet-4-20250514", result.Model)
	}
	if result.StopReason != "stop" {
		t.Errorf("StopReason = %q, want stop", result.StopReason)
	}
	if result.Usage.PromptTokens != 25 {
		t.Errorf("PromptTokens = %d, want 25", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 15 {
		t.Errorf("CompletionTokens = %d, want 15", result.Usage.CompletionTokens)
	}
	if result.Usage.TotalTokens != 40 {
		t.Errorf("TotalTokens = %d, want 40", result.Usage.TotalTokens)
	}
	if len(tokens) != 2 || tokens[0] != "Hello" || tokens[1] != " world" {
		t.Errorf("tokens = %v, want [Hello, ' world']", tokens)
	}
}

func TestChatStreamToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_2\",\"model\":\"claude-sonnet-4-20250514\",\"usage\":{\"input_tokens\":20}}}\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Let me check\"}}\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_abc\",\"name\":\"get_weather\"}}\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"city\\\"\"}}\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\": \\\"Tokyo\\\"}\"}}\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":1}\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":30}}\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n",
		}

		for _, ev := range events {
			fmt.Fprint(w, ev)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithAPIBase(server.URL))
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "Weather in Tokyo?"},
	}

	result, err := p.ChatStream(context.Background(), messages, nil, "claude-sonnet-4-20250514", nil, nil)
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}

	if result.Content != "Let me check" {
		t.Errorf("Content = %q, want 'Let me check'", result.Content)
	}
	if result.StopReason != "tool_calls" {
		t.Errorf("StopReason = %q, want tool_calls", result.StopReason)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.ID != "toolu_abc" {
		t.Errorf("ToolCall.ID = %q, want toolu_abc", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want get_weather", tc.Name)
	}
	if tc.Arguments["city"] != "Tokyo" {
		t.Errorf("ToolCall.Arguments[city] = %v, want Tokyo", tc.Arguments["city"])
	}
}

func TestChatStreamHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	p := New("bad-key", WithAPIBase(server.URL))
	_, err := p.ChatStream(context.Background(), []providers.Message{{Role: providers.RoleUser, Content: "Hi"}}, nil, "claude-sonnet-4-20250514", nil, nil)
	if err == nil {
		t.Fatal("Expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Error = %v, want 401", err)
	}
}

func TestStreamingProviderInterface(t *testing.T) {
	p := New("test-key")
	var _ providers.StreamingProvider = p
}
