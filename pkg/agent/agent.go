package agent

import (
	"context"

	"github.com/strings77wzq/unlimitedClaw/pkg/bus"
	"github.com/strings77wzq/unlimitedClaw/pkg/config"
	"github.com/strings77wzq/unlimitedClaw/pkg/logger"
	"github.com/strings77wzq/unlimitedClaw/pkg/providers"
	"github.com/strings77wzq/unlimitedClaw/pkg/session"
	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
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
