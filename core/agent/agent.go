// Package agent implements the ReAct (Reason + Act) loop that drives the AI
// assistant. It receives user messages from the message bus, calls the LLM,
// dispatches tool calls, and publishes responses back onto the bus.
// The Agent type is the central coordinator; use [New] to construct one and
// [Agent.Start] to run its event loop.
package agent

import (
	"context"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

const (
	DefaultMaxToolIterations = 25
	TopicInbound             = "inbound"
	TopicOutbound            = "outbound"
)

// Runner abstracts the agent lifecycle for dependency injection and testing.
type Runner interface {
	Start(ctx context.Context)
}

// MessageHandler provides direct request/response handling for gateways and TUIs.
type MessageHandler interface {
	HandleMessage(ctx context.Context, sessionID string, message string) (string, error)
	HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
}

// Agent is the core orchestrator that runs the ReAct loop
type Agent struct {
	bus               bus.Bus
	toolRegistry      *tools.Registry
	providerFactory   *providers.Factory
	sessionStore      session.SessionStore
	historyManager    *session.HistoryManager
	logger            logger.Logger
	config            *config.Config
	systemPrompt      string
	maxToolIterations int
}

// Option is a functional option for configuring Agent
type Option func(*Agent)

// WithMaxToolIterations sets the maximum number of ReAct loop iterations
func WithMaxToolIterations(n int) Option {
	return func(a *Agent) {
		a.maxToolIterations = n
	}
}

// WithSystemPrompt sets the default system prompt
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// New creates a new Agent with the given dependencies
func New(
	b bus.Bus,
	registry *tools.Registry,
	factory *providers.Factory,
	store session.SessionStore,
	history *session.HistoryManager,
	log logger.Logger,
	cfg *config.Config,
	opts ...Option,
) *Agent {
	a := &Agent{
		bus:               b,
		toolRegistry:      registry,
		providerFactory:   factory,
		sessionStore:      store,
		historyManager:    history,
		logger:            log,
		config:            cfg,
		systemPrompt:      cfg.Agents.Defaults.SystemPrompt,
		maxToolIterations: DefaultMaxToolIterations,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Start begins listening for inbound messages and processing them
func (a *Agent) Start(ctx context.Context) {
	a.logger.Info("agent started")
	defer a.logger.Info("agent stopped")

	// Subscribe to inbound messages
	ch := a.bus.Subscribe(TopicInbound)
	defer a.bus.Unsubscribe(TopicInbound, ch)

	// Process messages until context is cancelled
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-ch:
			msg, ok := raw.(bus.InboundMessage)
			if !ok {
				a.logger.Error("invalid inbound message type", nil)
				continue
			}
			a.handleMessage(ctx, msg)
		}
	}
}

// Compile-time interface compliance check.
var _ Runner = (*Agent)(nil)
var _ MessageHandler = (*Agent)(nil)
