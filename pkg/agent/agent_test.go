package agent

import (
	"context"
	"testing"
	"time"

	"github.com/strin/unlimitedclaw/pkg/bus"
	"github.com/strin/unlimitedclaw/pkg/config"
	"github.com/strin/unlimitedclaw/pkg/logger"
	"github.com/strin/unlimitedclaw/pkg/providers"
	"github.com/strin/unlimitedclaw/pkg/session"
	"github.com/strin/unlimitedclaw/pkg/tools"
)

func setupTestAgent(t *testing.T) (*Agent, bus.Bus, *providers.MockProvider, *tools.Registry) {
	t.Helper()

	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test-model"

	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	a := New(b, registry, factory, store, history, log, cfg)

	return a, b, mockProvider, registry
}

func startAgent(ctx context.Context, a *Agent) {
	go a.Start(ctx)
	time.Sleep(50 * time.Millisecond)
}

func TestAgentPureTextResponse(t *testing.T) {
	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello!",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "Hi",
		Role:      bus.RoleUser,
	})

	select {
	case raw := <-outCh:
		msg, ok := raw.(bus.OutboundMessage)
		if !ok {
			t.Fatal("expected OutboundMessage")
		}
		if msg.Content != "Hello!" {
			t.Errorf("expected content 'Hello!', got %q", msg.Content)
		}
		if !msg.Done {
			t.Error("expected Done to be true")
		}
		if msg.Role != bus.RoleAssistant {
			t.Errorf("expected role %v, got %v", bus.RoleAssistant, msg.Role)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestAgentSingleToolCall(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	echoTool := &tools.MockTool{
		ToolName:        "echo",
		ToolDescription: "echoes input",
	}
	echoTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{ForLLM: "echoed"}, nil
	}
	registry.Register(echoTool)

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{
				ID:        "call_1",
				Name:      "echo",
				Arguments: map[string]interface{}{"text": "hello"},
			},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "I echoed your message",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "echo hello",
		Role:      bus.RoleUser,
	})

	select {
	case raw := <-outCh:
		msg, ok := raw.(bus.OutboundMessage)
		if !ok {
			t.Fatal("expected OutboundMessage")
		}
		if msg.Content != "I echoed your message" {
			t.Errorf("expected content 'I echoed your message', got %q", msg.Content)
		}
		if !msg.Done {
			t.Error("expected Done to be true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestAgentMultipleToolCalls(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	tool1 := &tools.MockTool{
		ToolName:        "tool1",
		ToolDescription: "first tool",
	}
	tool1Executed := false
	tool1.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		tool1Executed = true
		return &tools.ToolResult{ForLLM: "result1"}, nil
	}
	registry.Register(tool1)

	tool2 := &tools.MockTool{
		ToolName:        "tool2",
		ToolDescription: "second tool",
	}
	tool2Executed := false
	tool2.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		tool2Executed = true
		return &tools.ToolResult{ForLLM: "result2"}, nil
	}
	registry.Register(tool2)

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "tool1", Arguments: map[string]interface{}{}},
			{ID: "call_2", Name: "tool2", Arguments: map[string]interface{}{}},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Both tools executed",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "run both tools",
		Role:      bus.RoleUser,
	})

	select {
	case raw := <-outCh:
		msg, ok := raw.(bus.OutboundMessage)
		if !ok {
			t.Fatal("expected OutboundMessage")
		}
		if msg.Content != "Both tools executed" {
			t.Errorf("expected content 'Both tools executed', got %q", msg.Content)
		}
		if !tool1Executed {
			t.Error("tool1 was not executed")
		}
		if !tool2Executed {
			t.Error("tool2 was not executed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestAgentMaxIterations(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	a.maxToolIterations = 3

	dummyTool := &tools.MockTool{
		ToolName:        "dummy",
		ToolDescription: "dummy tool",
	}
	dummyTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{ForLLM: "dummy result"}, nil
	}
	registry.Register(dummyTool)

	for i := 0; i < 3; i++ {
		mockProvider.AddResponse(&providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Name: "dummy", Arguments: map[string]interface{}{}},
			},
		})
	}

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "loop forever",
		Role:      bus.RoleUser,
	})

	select {
	case raw := <-outCh:
		msg, ok := raw.(bus.OutboundMessage)
		if !ok {
			t.Fatal("expected OutboundMessage")
		}
		if msg.Content != "max tool iterations reached" {
			t.Errorf("expected 'max tool iterations reached', got %q", msg.Content)
		}
		if !msg.Done {
			t.Error("expected Done to be true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestAgentToolNotFound(t *testing.T) {
	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "nonexistent", Arguments: map[string]interface{}{}},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Tool not found, continuing",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "use fake tool",
		Role:      bus.RoleUser,
	})

	select {
	case raw := <-outCh:
		msg, ok := raw.(bus.OutboundMessage)
		if !ok {
			t.Fatal("expected OutboundMessage")
		}
		if msg.Content != "Tool not found, continuing" {
			t.Errorf("expected 'Tool not found, continuing', got %q", msg.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestAgentSessionPersistence(t *testing.T) {
	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "First response",
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Second response",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "persistent-session",
		Content:   "First message",
		Role:      bus.RoleUser,
	})

	<-outCh

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "persistent-session",
		Content:   "Second message",
		Role:      bus.RoleUser,
	})

	<-outCh

	if mockProvider.CallCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockProvider.CallCount)
	}

	lastMessages := mockProvider.LastMessages
	if len(lastMessages) < 3 {
		t.Fatalf("expected at least 3 messages in history, got %d", len(lastMessages))
	}

	userMessageCount := 0
	for _, msg := range lastMessages {
		if msg.Role == providers.RoleUser {
			userMessageCount++
		}
	}
	if userMessageCount != 2 {
		t.Errorf("expected 2 user messages in history, got %d", userMessageCount)
	}
}

func TestAgentSystemPrompt(t *testing.T) {
	b := bus.New()
	defer b.Close()

	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test-model"

	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	customPrompt := "You are a helpful assistant"
	a := New(b, registry, factory, store, history, log, cfg, WithSystemPrompt(customPrompt))

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "Hi",
		Role:      bus.RoleUser,
	})

	<-outCh

	lastMessages := mockProvider.LastMessages
	if len(lastMessages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(lastMessages))
	}

	if lastMessages[0].Role != providers.RoleSystem {
		t.Errorf("expected first message to be system, got %v", lastMessages[0].Role)
	}
	if lastMessages[0].Content != customPrompt {
		t.Errorf("expected system prompt %q, got %q", customPrompt, lastMessages[0].Content)
	}
}

func TestAgentToolForUserOutput(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	verboseTool := &tools.MockTool{
		ToolName:        "verbose",
		ToolDescription: "verbose tool",
	}
	verboseTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{
			ForLLM:  "LLM sees this",
			ForUser: "User sees this",
			Silent:  false,
		}, nil
	}
	registry.Register(verboseTool)

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "verbose", Arguments: map[string]interface{}{}},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Final response",
	})

	outCh := b.Subscribe(TopicOutbound)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: "test-session",
		Content:   "run verbose tool",
		Role:      bus.RoleUser,
	})

	var messages []bus.OutboundMessage
	timeout := time.After(2 * time.Second)

	for len(messages) < 2 {
		select {
		case raw := <-outCh:
			msg, ok := raw.(bus.OutboundMessage)
			if !ok {
				t.Fatal("expected OutboundMessage")
			}
			messages = append(messages, msg)
		case <-timeout:
			t.Fatalf("timeout waiting for messages, got %d", len(messages))
		}
	}

	if messages[0].Content != "User sees this" {
		t.Errorf("expected first message 'User sees this', got %q", messages[0].Content)
	}
	if messages[0].Role != bus.RoleTool {
		t.Errorf("expected first message role %v, got %v", bus.RoleTool, messages[0].Role)
	}
	if messages[0].Done {
		t.Error("expected first message Done to be false")
	}

	if messages[1].Content != "Final response" {
		t.Errorf("expected second message 'Final response', got %q", messages[1].Content)
	}
	if !messages[1].Done {
		t.Error("expected second message Done to be true")
	}
}

func TestAgentContextCancellation(t *testing.T) {
	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello",
	})

	ctx, cancel := context.WithCancel(context.Background())

	go a.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	cancel()

	time.Sleep(100 * time.Millisecond)
}
