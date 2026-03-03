package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
)

// MockProvider is a configurable LLM provider for testing.
type MockProvider struct {
	mu           sync.Mutex
	responses    []*LLMResponse
	responseIdx  int
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
