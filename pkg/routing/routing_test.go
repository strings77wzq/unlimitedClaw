package routing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/strin/unlimitedclaw/pkg/providers"
	"github.com/strin/unlimitedclaw/pkg/tools"
)

func TestProviderError(t *testing.T) {
	baseErr := errors.New("connection timeout")
	provErr := &ProviderError{
		Provider:   "openai",
		StatusCode: 503,
		Message:    "service unavailable",
		Retryable:  true,
		Err:        baseErr,
	}

	expected := "provider openai (status 503): service unavailable"
	if provErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, provErr.Error())
	}

	if errors.Unwrap(provErr) != baseErr {
		t.Error("Unwrap should return base error")
	}

	provErrNoStatus := &ProviderError{
		Provider: "anthropic",
		Message:  "rate limit",
	}
	expectedNoStatus := "provider anthropic: rate limit"
	if provErrNoStatus.Error() != expectedNoStatus {
		t.Errorf("expected %q, got %q", expectedNoStatus, provErrNoStatus.Error())
	}
}

func TestToolError(t *testing.T) {
	baseErr := errors.New("file not found")
	toolErr := &ToolError{
		ToolName: "read_file",
		Message:  "invalid path",
		Err:      baseErr,
	}

	expected := "tool read_file: invalid path"
	if toolErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, toolErr.Error())
	}

	if errors.Unwrap(toolErr) != baseErr {
		t.Error("Unwrap should return base error")
	}
}

func TestConfigError(t *testing.T) {
	cfgErr := &ConfigError{
		Field:   "api_key",
		Message: "missing required field",
	}

	expected := "config error [api_key]: missing required field"
	if cfgErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, cfgErr.Error())
	}
}

func TestIsRetryable(t *testing.T) {
	retryableErr := &ProviderError{
		Provider:  "openai",
		Message:   "timeout",
		Retryable: true,
	}

	nonRetryableErr := &ProviderError{
		Provider:  "openai",
		Message:   "invalid key",
		Retryable: false,
	}

	otherErr := errors.New("some error")

	if !IsRetryable(retryableErr) {
		t.Error("expected IsRetryable to return true for retryable ProviderError")
	}

	if IsRetryable(nonRetryableErr) {
		t.Error("expected IsRetryable to return false for non-retryable ProviderError")
	}

	if IsRetryable(otherErr) {
		t.Error("expected IsRetryable to return false for non-ProviderError")
	}
}

func TestFallbackChainNext(t *testing.T) {
	models := []string{"openai/gpt-4", "anthropic/claude", "google/gemini"}
	chain := NewFallbackChain(models, time.Second)

	model, ok := chain.Next()
	if !ok || model != "openai/gpt-4" {
		t.Errorf("expected first model openai/gpt-4, got %s", model)
	}
}

func TestFallbackChainCooldown(t *testing.T) {
	models := []string{"openai/gpt-4", "anthropic/claude", "google/gemini"}
	chain := NewFallbackChain(models, 100*time.Millisecond)

	chain.MarkFailed("openai/gpt-4")

	model, ok := chain.Next()
	if !ok || model != "anthropic/claude" {
		t.Errorf("expected second model anthropic/claude (first is in cooldown), got %s", model)
	}

	chain.MarkFailed("anthropic/claude")
	model, ok = chain.Next()
	if !ok || model != "google/gemini" {
		t.Errorf("expected third model google/gemini, got %s", model)
	}

	chain.MarkFailed("google/gemini")
	model, ok = chain.Next()
	if ok {
		t.Errorf("expected no available model (all in cooldown), got %s", model)
	}
}

func TestFallbackChainCooldownExpiry(t *testing.T) {
	models := []string{"openai/gpt-4", "anthropic/claude"}
	chain := NewFallbackChain(models, 50*time.Millisecond)

	chain.MarkFailed("openai/gpt-4")

	model, ok := chain.Next()
	if !ok || model != "anthropic/claude" {
		t.Errorf("expected anthropic/claude during cooldown, got %s", model)
	}

	time.Sleep(60 * time.Millisecond)

	model, ok = chain.Next()
	if !ok || model != "openai/gpt-4" {
		t.Errorf("expected openai/gpt-4 after cooldown expiry, got %s", model)
	}
}

func TestFallbackChainMarkSuccess(t *testing.T) {
	models := []string{"openai/gpt-4", "anthropic/claude"}
	chain := NewFallbackChain(models, time.Second)

	chain.MarkFailed("openai/gpt-4")

	model, ok := chain.Next()
	if !ok || model != "anthropic/claude" {
		t.Errorf("expected anthropic/claude (first is in cooldown), got %s", model)
	}

	chain.MarkSuccess("openai/gpt-4")

	model, ok = chain.Next()
	if !ok || model != "openai/gpt-4" {
		t.Errorf("expected openai/gpt-4 after MarkSuccess, got %s", model)
	}
}

func TestRouterDirectRoute(t *testing.T) {
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("openai")
	factory.Register("openai", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "response from gpt-4",
		Model:   "gpt-4",
	})

	router := NewRouter(factory)
	router.AddRoute("default", "openai/gpt-4")

	resp, err := router.Chat(context.Background(), "default", []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "response from gpt-4" {
		t.Errorf("expected 'response from gpt-4', got %q", resp.Content)
	}

	if mockProvider.CallCount != 1 {
		t.Errorf("expected 1 call, got %d", mockProvider.CallCount)
	}
}

func TestRouterFallback(t *testing.T) {
	factory := providers.NewFactory()

	mockOpenAI := &errorMockProvider{
		name: "openai",
		err: &ProviderError{
			Provider:  "openai",
			Message:   "service temporarily unavailable",
			Retryable: true,
		},
	}
	mockAnthropic := providers.NewMockProvider("anthropic")

	factory.Register("openai", mockOpenAI)
	factory.Register("anthropic", mockAnthropic)

	mockAnthropic.AddResponse(&providers.LLMResponse{
		Content: "response from claude",
		Model:   "claude",
	})

	router := NewRouter(factory)
	router.AddRoute("default", "openai/gpt-4", "anthropic/claude")

	resp, err := router.Chat(context.Background(), "default", []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "response from claude" {
		t.Errorf("expected 'response from claude', got %q", resp.Content)
	}

	if mockOpenAI.callCount != 1 {
		t.Errorf("expected openai to be called once (and fail), got %d", mockOpenAI.callCount)
	}

	if mockAnthropic.CallCount != 1 {
		t.Errorf("expected anthropic to be called once (fallback), got %d", mockAnthropic.CallCount)
	}
}

func TestRouterNoRoute(t *testing.T) {
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("google")
	factory.Register("google", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "response from gemini",
		Model:   "gemini",
	})

	router := NewRouter(factory)

	resp, err := router.Chat(context.Background(), "google/gemini", []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "response from gemini" {
		t.Errorf("expected 'response from gemini', got %q", resp.Content)
	}

	if mockProvider.CallCount != 1 {
		t.Errorf("expected 1 call, got %d", mockProvider.CallCount)
	}
}

func TestRouterConcurrency(t *testing.T) {
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("openai")
	factory.Register("openai", mockProvider)

	for i := 0; i < 10; i++ {
		mockProvider.AddResponse(&providers.LLMResponse{
			Content: fmt.Sprintf("response %d", i),
			Model:   "gpt-4",
		})
	}

	router := NewRouter(factory)
	router.AddRoute("default", "openai/gpt-4")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := router.Chat(context.Background(), "default", []providers.Message{
				{Role: providers.RoleUser, Content: "test"},
			}, nil, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	if mockProvider.CallCount != 10 {
		t.Errorf("expected 10 calls, got %d", mockProvider.CallCount)
	}
}

func TestRouterFallbackStopsOnNonRetryable(t *testing.T) {
	factory := providers.NewFactory()

	mockOpenAI := &errorMockProvider{
		name: "openai",
		err: &ProviderError{
			Provider:  "openai",
			Message:   "invalid API key",
			Retryable: false,
		},
	}
	mockAnthropic := providers.NewMockProvider("anthropic")

	factory.Register("openai", mockOpenAI)
	factory.Register("anthropic", mockAnthropic)

	mockAnthropic.AddResponse(&providers.LLMResponse{
		Content: "should not be called",
	})

	router := NewRouter(factory)
	router.AddRoute("default", "openai/gpt-4", "anthropic/claude")

	_, err := router.Chat(context.Background(), "default", []providers.Message{
		{Role: providers.RoleUser, Content: "test"},
	}, nil, nil)

	if err == nil {
		t.Fatal("expected error from non-retryable failure")
	}

	if mockOpenAI.callCount != 1 {
		t.Errorf("expected openai to be called once, got %d", mockOpenAI.callCount)
	}

	if mockAnthropic.CallCount != 0 {
		t.Errorf("expected anthropic NOT to be called (non-retryable error), got %d calls", mockAnthropic.CallCount)
	}
}

type errorMockProvider struct {
	mu        sync.Mutex
	name      string
	err       error
	callCount int
}

func (e *errorMockProvider) Chat(ctx context.Context, messages []providers.Message, toolDefs []tools.ToolDefinition, model string, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.callCount++
	return nil, e.err
}

func (e *errorMockProvider) Name() string {
	return e.name
}
