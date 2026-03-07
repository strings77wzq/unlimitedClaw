// Package openai implements the [providers.LLMProvider] and
// [providers.StreamingProvider] interfaces for the OpenAI Chat Completions
// API. It also serves as the adapter for all OpenAI-compatible endpoints
// (DeepSeek, Kimi, GLM, MiniMax, Qwen) by accepting a custom API base URL
// via [WithAPIBase].
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// Compile-time check: Provider implements StreamingProvider.
var _ providers.StreamingProvider = (*Provider)(nil)

const (
	defaultAPIBase    = "https://api.openai.com"
	defaultMaxRetries = 3
)

// Provider implements providers.LLMProvider for OpenAI-compatible APIs.
type Provider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
	maxRetries int
}

// Option is a functional option for configuring the Provider.
type Option func(*Provider)

// WithAPIBase sets a custom API base URL (for OpenRouter, local models, etc.).
func WithAPIBase(base string) Option {
	return func(p *Provider) {
		p.apiBase = strings.TrimRight(base, "/")
	}
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.httpClient = client
	}
}

// WithMaxRetries sets the maximum number of retries for rate limits and server errors.
func WithMaxRetries(n int) Option {
	return func(p *Provider) {
		p.maxRetries = n
	}
}

// New creates a new OpenAI provider.
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    defaultAPIBase,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		maxRetries: defaultMaxRetries,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai"
}

// Chat implements providers.LLMProvider.Chat.
func (p *Provider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	model string,
	opts *providers.ChatOptions,
) (*providers.LLMResponse, error) {
	reqBody := p.buildRequest(messages, toolDefs, model, opts)

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		resp, retryAfter, shouldRetry, err := p.doRequest(ctx, reqBody)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !shouldRetry {
			return nil, err
		}

		// If Retry-After header is present, wait for that duration
		if retryAfter > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (p *Provider) ChatStream(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	model string,
	opts *providers.ChatOptions,
	onToken func(token string),
) (*providers.LLMResponse, error) {
	reqBody := p.buildRequest(messages, toolDefs, model, opts)
	reqBody["stream"] = true
	reqBody["stream_options"] = map[string]interface{}{"include_usage": true}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.apiBase + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed (status %d): %s", resp.StatusCode, string(body))
	}

	return p.parseSSEStream(resp.Body, onToken)
}

func (p *Provider) parseSSEStream(body io.Reader, onToken func(token string)) (*providers.LLMResponse, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var contentBuilder strings.Builder
	var responseModel string
	var finishReason string
	var usage providers.TokenUsage

	type streamToolCall struct {
		ID       string
		Name     string
		ArgsJSON strings.Builder
	}
	toolCallMap := make(map[int]*streamToolCall)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Model != "" {
			responseModel = chunk.Model
		}

		if chunk.Usage != nil {
			usage = providers.TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		if choice.FinishReason != nil {
			finishReason = *choice.FinishReason
		}

		if choice.Delta.Content != "" {
			contentBuilder.WriteString(choice.Delta.Content)
			if onToken != nil {
				onToken(choice.Delta.Content)
			}
		}

		for _, tc := range choice.Delta.ToolCalls {
			stc, ok := toolCallMap[tc.Index]
			if !ok {
				stc = &streamToolCall{}
				toolCallMap[tc.Index] = stc
			}
			if tc.ID != "" {
				stc.ID = tc.ID
			}
			if tc.Function.Name != "" {
				stc.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				stc.ArgsJSON.WriteString(tc.Function.Arguments)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SSE stream: %w", err)
	}

	toolCalls := make([]providers.ToolCall, 0, len(toolCallMap))
	for i := 0; i < len(toolCallMap); i++ {
		stc, ok := toolCallMap[i]
		if !ok {
			continue
		}
		var args map[string]interface{}
		raw := stc.ArgsJSON.String()
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				args = map[string]interface{}{"raw": raw}
			}
		} else {
			args = make(map[string]interface{})
		}
		toolCalls = append(toolCalls, providers.ToolCall{
			ID:        stc.ID,
			Name:      stc.Name,
			Arguments: args,
		})
	}

	return &providers.LLMResponse{
		Content:    contentBuilder.String(),
		ToolCalls:  toolCalls,
		Usage:      usage,
		StopReason: finishReason,
		Model:      responseModel,
	}, nil
}

func (p *Provider) buildRequest(
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	model string,
	opts *providers.ChatOptions,
) map[string]interface{} {
	req := map[string]interface{}{
		"model":    model,
		"messages": p.convertMessages(messages),
	}

	if len(toolDefs) > 0 {
		req["tools"] = p.convertTools(toolDefs)
	}

	if opts != nil {
		if opts.Temperature != nil {
			req["temperature"] = *opts.Temperature
		}
		if opts.MaxTokens != nil {
			req["max_tokens"] = *opts.MaxTokens
		}
		if opts.TopP != nil {
			req["top_p"] = *opts.TopP
		}
		if len(opts.Stop) > 0 {
			req["stop"] = opts.Stop
		}
	}

	return req
}

func (p *Provider) convertMessages(messages []providers.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		m := map[string]interface{}{
			"role":    string(msg.Role),
			"content": msg.Content,
		}

		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]interface{}, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(argsJSON),
					},
				})
			}
			m["tool_calls"] = toolCalls
		}

		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}

		result = append(result, m)
	}
	return result
}

func (p *Provider) convertTools(toolDefs []tools.ToolDefinition) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(toolDefs))
	for _, def := range toolDefs {
		properties := make(map[string]interface{})
		required := make([]string, 0)

		for _, param := range def.Parameters {
			properties[param.Name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}

		parameters := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			parameters["required"] = required
		}

		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        def.Name,
				"description": def.Description,
				"parameters":  parameters,
			},
		})
	}
	return result
}

func (p *Provider) doRequest(
	ctx context.Context,
	reqBody map[string]interface{},
) (*providers.LLMResponse, time.Duration, bool, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.apiBase + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Network errors are retryable
		return nil, 0, true, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		llmResp, err := p.parseResponse(body)
		return llmResp, 0, false, err
	}

	// Handle retryable errors
	shouldRetry := false
	var retryAfter time.Duration

	switch resp.StatusCode {
	case http.StatusTooManyRequests: // 429
		shouldRetry = true
		// Parse Retry-After header if present
		if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		// 500, 502, 503, 504
		shouldRetry = true
	}

	return nil, retryAfter, shouldRetry, fmt.Errorf("API request failed (status %d): %s", resp.StatusCode, string(body))
}

func (p *Provider) parseResponse(body []byte) (*providers.LLMResponse, error) {
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return &providers.LLMResponse{
			Content:    "",
			StopReason: "stop",
			Model:      apiResp.Model,
		}, nil
	}

	choice := apiResp.Choices[0]
	toolCalls := make([]providers.ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]interface{}
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				// If parsing fails, store raw string
				args = map[string]interface{}{"raw": tc.Function.Arguments}
			}
		} else {
			args = make(map[string]interface{})
		}

		toolCalls = append(toolCalls, providers.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return &providers.LLMResponse{
		Content:   choice.Message.Content,
		ToolCalls: toolCalls,
		Usage: providers.TokenUsage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
		StopReason: choice.FinishReason,
		Model:      apiResp.Model,
	}, nil
}
