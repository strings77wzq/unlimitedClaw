// Package gateway implements the HTTP gateway server that exposes the AI agent
// over a REST API with Server-Sent Events (SSE) streaming. It listens on port
// 18790 by default and provides endpoints for chat, session management, and
// health checks. The [Server] wraps an [agent.MessageHandler] and handles
// concurrent requests via SSE.
package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/strings77wzq/golem/foundation/logger"
)

// AgentHandler decouples gateway from agent package
type AgentHandler interface {
	HandleMessage(ctx context.Context, sessionID string, message string) (string, error)
}

// StreamingAgentHandler extends AgentHandler with token-by-token streaming.
// If the agent implements this interface, the SSE endpoint will use it.
type StreamingAgentHandler interface {
	AgentHandler
	HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Addr            string        // default ":18790"
	ReadTimeout     time.Duration // default 30s
	WriteTimeout    time.Duration // default 30s
	ShutdownTimeout time.Duration // default 10s
}

// DefaultServerConfig returns sensible defaults
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:            ":18790",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Server represents the HTTP gateway server
type Server struct {
	httpServer      *http.Server
	mux             *http.ServeMux
	logger          logger.Logger
	agent           AgentHandler
	shutdownTimeout time.Duration
}

// NewServer creates a new HTTP gateway server
func NewServer(cfg ServerConfig, agentHandler AgentHandler, log logger.Logger) *Server {
	// Apply defaults if zero values
	if cfg.Addr == "" {
		cfg.Addr = ":18790"
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 30 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 30 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:             mux,
		logger:          log,
		agent:           agentHandler,
		shutdownTimeout: cfg.ShutdownTimeout,
	}

	// Register routes
	s.registerRoutes()

	// Apply middleware chain
	handler := Chain(
		RequestIDMiddleware(),
		LoggingMiddleware(log),
		RecoveryMiddleware(log),
		CORSMiddleware(),
	)(mux)

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return s
}

// Start starts the HTTP server (blocks until server stops)
func (s *Server) Start() error {
	s.logger.Info("starting HTTP gateway server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP gateway server")
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the HTTP handler (for testing)
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
