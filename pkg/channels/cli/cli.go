package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/strings77wzq/unlimitedClaw/pkg/bus"
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
			fmt.Fprintln(a.writer, msg.Content)
			if msg.Done {
				return nil
			}
		}
	}
}

// RunInteractive starts an interactive readline loop.
// Reads lines from reader, publishes to bus, prints responses.
func (a *Adapter) RunInteractive(ctx context.Context) error {
	// Subscribe to outbound
	outCh := a.bus.Subscribe(TopicOutbound)
	defer a.bus.Unsubscribe(TopicOutbound, outCh)

	scanner := bufio.NewScanner(a.reader)

	// Start goroutine to print outbound messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-outCh:
				msg, ok := raw.(bus.OutboundMessage)
				if !ok || msg.SessionID != a.sessionID {
					continue
				}
				fmt.Fprintln(a.writer, msg.Content)
			}
		}
	}()

	// Read loop
	for {
		fmt.Fprint(a.writer, a.prompt)
		if !scanner.Scan() {
			break // EOF or error
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Fprintln(a.writer, "Goodbye!")
			return nil
		}

		a.bus.Publish(TopicInbound, bus.InboundMessage{
			SessionID: a.sessionID,
			Content:   line,
			Role:      bus.RoleUser,
		})
	}

	return scanner.Err()
}
