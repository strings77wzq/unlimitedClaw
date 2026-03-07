// Package cli implements the plain interactive channel for the AI agent.
// It uses bufio.Scanner to read lines from stdin and prints responses to
// stdout, making it suitable for piped input, scripts, and terminals where
// the Bubble Tea TUI is not desired (use --no-tui to force this mode).
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/strings77wzq/unlimitedClaw/core/bus"
)

const (
	TopicInbound     = "inbound"
	TopicOutbound    = "outbound"
	DefaultSessionID = "cli-session"
)

// Adapter bridges stdin/stdout with the message bus.
type Adapter struct {
	bus       bus.Bus
	reader    io.Reader
	writer    io.Writer
	mu        sync.Mutex
	sessionID string
	prompt    string
}

// Option is a functional option for configuring Adapter.
type Option func(*Adapter)

// WithSessionID sets a custom session ID.
func WithSessionID(id string) Option {
	return func(a *Adapter) {
		a.sessionID = id
	}
}

// WithPrompt sets a custom prompt string.
func WithPrompt(p string) Option {
	return func(a *Adapter) {
		a.prompt = p
	}
}

// WithReader sets a custom reader (for testing).
func WithReader(r io.Reader) Option {
	return func(a *Adapter) {
		a.reader = r
	}
}

// WithWriter sets a custom writer (for testing).
func WithWriter(w io.Writer) Option {
	return func(a *Adapter) {
		a.writer = w
	}
}

// New creates a new CLI adapter.
func New(b bus.Bus, opts ...Option) *Adapter {
	a := &Adapter{
		bus:       b,
		reader:    os.Stdin,
		writer:    os.Stdout,
		sessionID: DefaultSessionID,
		prompt:    "You: ",
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Adapter) writeln(s string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fmt.Fprintln(a.writer, s)
}

func (a *Adapter) writePrompt() {
	a.mu.Lock()
	defer a.mu.Unlock()
	fmt.Fprint(a.writer, a.prompt)
}

// RunOnce sends a single message and waits for the response.
// Used for `-m "message"` one-shot mode.
func (a *Adapter) RunOnce(ctx context.Context, message string) error {
	// Subscribe to outbound
	outCh := a.bus.Subscribe(TopicOutbound)
	defer a.bus.Unsubscribe(TopicOutbound, outCh)

	// Publish inbound message
	a.bus.Publish(TopicInbound, bus.InboundMessage{
		SessionID: a.sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	})

	// Wait for Done=true response or context cancellation
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case raw := <-outCh:
			msg, ok := raw.(bus.OutboundMessage)
			if !ok || msg.SessionID != a.sessionID {
				continue
			}
			a.writeln(msg.Content)
			if msg.Done {
				return nil
			}
		}
	}
}

// RunInteractive starts an interactive readline loop.
// Reads lines from reader, publishes to bus, prints responses.
func (a *Adapter) RunInteractive(ctx context.Context) error {
	outCh := a.bus.Subscribe(TopicOutbound)
	defer a.bus.Unsubscribe(TopicOutbound, outCh)

	childCtx, childCancel := context.WithCancel(ctx)
	defer childCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-childCtx.Done():
				return
			case raw := <-outCh:
				msg, ok := raw.(bus.OutboundMessage)
				if !ok || msg.SessionID != a.sessionID {
					continue
				}
				a.writeln(msg.Content)
			}
		}
	}()

	scanner := bufio.NewScanner(a.reader)
	var scanErr error
	for {
		a.writePrompt()
		if !scanner.Scan() {
			scanErr = scanner.Err()
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			a.writeln("Goodbye!")
			break
		}

		a.bus.Publish(TopicInbound, bus.InboundMessage{
			SessionID: a.sessionID,
			Content:   line,
			Role:      bus.RoleUser,
		})
	}

	childCancel()
	wg.Wait()
	return scanErr
}
