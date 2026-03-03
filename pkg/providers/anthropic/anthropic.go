package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/strings77wzq/unlimitedClaw/pkg/providers"
	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
)

const (
	defaultAPIBase    = "https://api.anthropic.com"
	defaultAPIVersion = "2023-06-01"
	defaultMaxRetries = 3
)

// Provider implements the LLMProvider interface for Anthropic's Claude API.
type Provider struct {
	apiKey     string
	apiBase    string
	apiVersion string
	httpClient *http.Client
	maxRetries int
}

// Option is a functional option for configuring the Provider.
type Option func(*Provider)

// WithAPIBase sets a custom API base URL.
func WithAPIBase(base string) Option {
	return func(p *Provider) {
		p.apiBase = base
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.httpClient = client
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(p *Provider) {
		p.maxRetries = n
	}
}

// WithAPIVersion sets the anthropic-version header.
func WithAPIVersion(version string) Option {
	return func(p *Provider) {
		p.apiVersion = version
	}
}

// New creates a new Anthropic provider.
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    defaultAPIBase,
		apiVersion: defaultAPIVersion,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		maxRetries: defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "anthropic"
}

// Chat implements the LLMProvider interface.
func (p *Provider) Chat(ctx context.Context, messages []providers.Message, toolDefs []tools.ToolDefinition, model string, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	reqBody, err := p.buildRequest(messages, toolDefs, model, opts)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := p.doRequest(ctx, reqBody)
		if err != nil {
			lastErr = err
			// Retry on network errors
			continue
		}

		// Check for retryable status codes
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			continue
		}

		// Non-retryable error
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		// Success
		defer resp.Body.Close()
		return p.parseResponse(resp.Body)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// buildRequest constructs the Anthropic API request body.
func (p *Provider) buildRequest(messages []providers.Message, toolDefs []tools.ToolDefinition, model string, opts *providers.ChatOptions) (map[string]interface{}, error) {
	req := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096, // default
	}

	// Extract system messages
	var systemMessages []string
	var conversationMessages []providers.Message
	for _, msg := range messages {
		if msg.Role == providers.RoleSystem {
			systemMessages = append(systemMessages, msg.Content)
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	// Set system field if present
	if len(systemMessages) > 0 {
		// Combine all system messages
		combined := ""
		for i, s := range systemMessages {
			if i > 0 {
				combined += "\n\n"
			}
			combined += s
		}
		req["system"] = combined
	}

	// Convert messages to Anthropic format
	anthropicMessages, err := p.convertMessages(conversationMessages)
	if err != nil {
		return nil, err
	}
	req["messages"] = anthropicMessages

	// Add tools if present
	if len(toolDefs) > 0 {
		req["tools"] = p.convertTools(toolDefs)
	}

	// Apply options
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
			req["stop_sequences"] = opts.Stop
		}
	}

	return req, nil
}

// convertMessages converts our message format to Anthropic's format.
func (p *Provider) convertMessages(messages []providers.Message) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	for _, msg := range messages {
		switch msg.Role {
		case providers.RoleUser:
			if msg.ToolCallID != "" {
				// This is a tool result - send as user message with tool_result content
				result = append(result, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": msg.ToolCallID,
							"content":     msg.Content,
						},
					},
				})
			} else {
				result = append(result, map[string]interface{}{
					"role":    "user",
					"content": msg.Content,
				})
			}

		case providers.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				var contentBlocks []map[string]interface{}
				if msg.Content != "" {
					contentBlocks = append(contentBlocks, map[string]interface{}{
						"type": "text",
						"text": msg.Content,
					})
				}
				for _, tc := range msg.ToolCalls {
					contentBlocks = append(contentBlocks, map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Name,
						"input": tc.Arguments,
					})
				}
				result = append(result, map[string]interface{}{
					"role":    "assistant",
					"content": contentBlocks,
				})
			} else {
				result = append(result, map[string]interface{}{
					"role":    "assistant",
					"content": msg.Content,
				})
			}

		case providers.RoleTool:
			// Tool results are sent as user messages with tool_result content
			result = append(result, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type":        "tool_result",
						"tool_use_id": msg.ToolCallID,
						"content":     msg.Content,
					},
				},
			})

		default:
			return nil, fmt.Errorf("unsupported role: %s", msg.Role)
		}
	}

	return result, nil
}

// convertTools converts our tool definitions to Anthropic's format.
func (p *Provider) convertTools(toolDefs []tools.ToolDefinition) []map[string]interface{} {
	var result []map[string]interface{}

	for _, td := range toolDefs {
		tool := map[string]interface{}{
			"name":        td.Name,
			"description": td.Description,
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		}

		// Build properties and required fields
		properties := make(map[string]interface{})
		var required []string

		for _, param := range td.Parameters {
			properties[param.Name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}

		inputSchema := tool["input_schema"].(map[string]interface{})
		inputSchema["properties"] = properties
		if len(required) > 0 {
			inputSchema["required"] = required
		}

		result = append(result, tool)
	}

	return result
}

// doRequest sends the HTTP request to Anthropic API.
func (p *Provider) doRequest(ctx context.Context, reqBody map[string]interface{}) (*http.Response, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.apiBase + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", p.apiVersion)
	req.Header.Set("Content-Type", "application/json")

	return p.httpClient.Do(req)
}

// parseResponse parses the Anthropic API response.
func (p *Provider) parseResponse(body io.Reader) (*providers.LLMResponse, error) {
	var apiResp struct {
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text,omitempty"`
			ID    string                 `json:"id,omitempty"`
			Name  string                 `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Model      string `json:"model"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.NewDecoder(body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Parse content blocks
	var textContent string
	var toolCalls []providers.ToolCall

	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, providers.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	// Map stop reason
	stopReason := apiResp.StopReason
	switch apiResp.StopReason {
	case "end_turn":
		stopReason = "stop"
	case "tool_use":
		stopReason = "tool_calls"
	case "max_tokens":
		stopReason = "length"
	}

	return &providers.LLMResponse{
		Content:   textContent,
		ToolCalls: toolCalls,
		Usage: providers.TokenUsage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
		Model:      apiResp.Model,
		StopReason: stopReason,
	}, nil
}
