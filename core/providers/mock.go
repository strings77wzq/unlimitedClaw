package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/strings77wzq/golem/core/tools"
)

// MockProvider is a configurable LLM provider for testing.
type MockProvider struct {
	mu           sync.Mutex
	responses    []*LLMResponse
	responseIdx  int
	streamDelay  int
	CallCount    int
	LastMessages []Message
	LastModel    string
	LastTools    []tools.ToolDefinition
	ProviderName string
}

// NewMockProvider creates a new mock provider with the given name.
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		ProviderName: name,
		responses:    make([]*LLMResponse, 0),
	}
}

// AddResponse queues a response to be returned by Chat.
func (m *MockProvider) AddResponse(resp *LLMResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, resp)
}

// Chat returns the next queued response, or error if queue exhausted.
func (m *MockProvider) Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions) (*LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastMessages = messages
	m.LastModel = model
	m.LastTools = toolDefs

	if m.responseIdx >= len(m.responses) {
		return nil, fmt.Errorf("mock provider: no more responses queued (call count: %d)", m.CallCount)
	}

	resp := m.responses[m.responseIdx]
	m.responseIdx++
	return resp, nil
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	return m.ProviderName
}

// ChatStream streams the next queued response token-by-token.
func (m *MockProvider) ChatStream(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions, onToken func(token string)) (*LLMResponse, error) {
	resp, err := m.Chat(ctx, messages, toolDefs, model, opts)
	if err != nil {
		return nil, err
	}

	if onToken != nil && resp.Content != "" {
		for _, token := range splitMockTokens(resp.Content) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			onToken(token)
		}
	}

	return resp, nil
}

func splitMockTokens(content string) []string {
	parts := strings.Fields(content)
	if len(parts) == 0 {
		if content == "" {
			return nil
		}
		return []string{content}
	}

	tokens := make([]string, 0, len(parts))
	for i, part := range parts {
		token := part
		if i < len(parts)-1 {
			token += " "
		}
		tokens = append(tokens, token)
	}
	return tokens
}
