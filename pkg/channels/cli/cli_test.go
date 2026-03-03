package cli

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/strin/unlimitedclaw/pkg/bus"
)

// TestRunOnce verifies one-shot message mode works
func TestRunOnce(t *testing.T) {
	b := bus.New()
	defer b.Close()

	ready := make(chan struct{})

	// Start a goroutine to respond to messages
	go func() {
		ch := b.Subscribe(TopicInbound)
		defer b.Unsubscribe(TopicInbound, ch)
		close(ready)

		for raw := range ch {
			msg := raw.(bus.InboundMessage)
			if msg.SessionID != "test-session" {
				continue
			}
			// Echo back with a response
			b.Publish(TopicOutbound, bus.OutboundMessage{
				SessionID: msg.SessionID,
				Content:   "Response: " + msg.Content,
				Role:      bus.RoleAssistant,
				Done:      true,
			})
		}
	}()

	// Wait for responder to be ready
	<-ready

	var buf bytes.Buffer
	adapter := New(b, WithSessionID("test-session"), WithWriter(&buf))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.RunOnce(ctx, "Hello")
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Response: Hello") {
		t.Errorf("Expected response in output, got: %q", output)
	}
}

// TestRunInteractive simulates multi-line input
func TestRunInteractive(t *testing.T) {
	b := bus.New()
	defer b.Close()

	// Set up responder
	go func() {
		ch := b.Subscribe(TopicInbound)
		defer b.Unsubscribe(TopicInbound, ch)

		for raw := range ch {
			msg := raw.(bus.InboundMessage)
			b.Publish(TopicOutbound, bus.OutboundMessage{
				SessionID: msg.SessionID,
				Content:   "Ack: " + msg.Content,
				Role:      bus.RoleAssistant,
				Done:      true,
			})
		}
	}()

	// Create adapter with mocked input
	input := "hello\nworld\nexit\n"
	var output bytes.Buffer

	adapter := New(b,
		WithSessionID("test-session"),
		WithReader(strings.NewReader(input)),
		WithWriter(&output),
		WithPrompt("> "))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.RunInteractive(ctx)
	if err != nil {
		t.Fatalf("RunInteractive failed: %v", err)
	}

	outStr := output.String()
	// Check that prompts appear
	if !strings.Contains(outStr, "> ") {
		t.Errorf("Expected prompt in output")
	}
	// Check that "Goodbye!" message appears
	if !strings.Contains(outStr, "Goodbye!") {
		t.Errorf("Expected 'Goodbye!' message in output")
	}
}

// TestExitCommand verifies exit command ends interactive loop
func TestExitCommand(t *testing.T) {
	b := bus.New()
	defer b.Close()

	input := "exit\n"
	var output bytes.Buffer

	adapter := New(b,
		WithSessionID("test-session"),
		WithReader(strings.NewReader(input)),
		WithWriter(&output))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.RunInteractive(ctx)
	if err != nil {
		t.Fatalf("RunInteractive with exit failed: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Goodbye!") {
		t.Errorf("Expected 'Goodbye!' in output")
	}
}

// TestQuitCommand verifies quit command also ends interactive loop
func TestQuitCommand(t *testing.T) {
	b := bus.New()
	defer b.Close()

	input := "quit\n"
	var output bytes.Buffer

	adapter := New(b,
		WithSessionID("test-session"),
		WithReader(strings.NewReader(input)),
		WithWriter(&output))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.RunInteractive(ctx)
	if err != nil {
		t.Fatalf("RunInteractive with quit failed: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Goodbye!") {
		t.Errorf("Expected 'Goodbye!' in output")
	}
}

// TestEmptyLineSkipped verifies empty lines are not published
func TestEmptyLineSkipped(t *testing.T) {
	b := bus.New()
	defer b.Close()

	var publishedMessages []bus.InboundMessage
	var mu sync.Mutex
	ready := make(chan struct{})

	go func() {
		ch := b.Subscribe(TopicInbound)
		defer b.Unsubscribe(TopicInbound, ch)
		close(ready)

		for raw := range ch {
			if msg, ok := raw.(bus.InboundMessage); ok {
				mu.Lock()
				publishedMessages = append(publishedMessages, msg)
				mu.Unlock()
			}
		}
	}()

	<-ready

	input := "\n\n  \nmessage\n\nexit\n"
	var output bytes.Buffer

	adapter := New(b,
		WithSessionID("test-session"),
		WithReader(strings.NewReader(input)),
		WithWriter(&output))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	adapter.RunInteractive(ctx)

	// Sleep briefly to allow goroutine to collect messages
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(publishedMessages) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(publishedMessages))
	}
	if len(publishedMessages) > 0 && publishedMessages[0].Content != "message" {
		t.Errorf("Expected 'message', got %q", publishedMessages[0].Content)
	}
}

// TestSessionIDOption verifies custom session ID is used
func TestSessionIDOption(t *testing.T) {
	b := bus.New()
	defer b.Close()

	var receivedSessionIDs []string
	var sessionMu sync.Mutex
	ready := make(chan struct{})

	go func() {
		ch := b.Subscribe(TopicInbound)
		defer b.Unsubscribe(TopicInbound, ch)
		close(ready)

		for raw := range ch {
			if msg, ok := raw.(bus.InboundMessage); ok {
				sessionMu.Lock()
				receivedSessionIDs = append(receivedSessionIDs, msg.SessionID)
				sessionMu.Unlock()
			}
		}
	}()

	<-ready

	input := "test\nexit\n"
	var output bytes.Buffer
	customSessionID := "custom-session-123"

	adapter := New(b,
		WithSessionID(customSessionID),
		WithReader(strings.NewReader(input)),
		WithWriter(&output))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	adapter.RunInteractive(ctx)

	// Sleep briefly to allow goroutine to collect messages
	time.Sleep(100 * time.Millisecond)

	sessionMu.Lock()
	defer sessionMu.Unlock()

	if len(receivedSessionIDs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(receivedSessionIDs))
	}
	if len(receivedSessionIDs) > 0 && receivedSessionIDs[0] != customSessionID {
		t.Errorf("Expected session ID %q, got %q", customSessionID, receivedSessionIDs[0])
	}
}
