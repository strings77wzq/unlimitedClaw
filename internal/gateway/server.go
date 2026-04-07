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

	coreproviders "github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/internal/security"
)

type HealthStatusProvider interface {
	GetAllStatuses() map[string]*coreproviders.HealthStatus
}

// SessionStore provides access to sessions for import/export.
type SessionStore interface {
	Get(id string) (*session.Session, bool)
	Save(s *session.Session) error
}

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
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
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

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	EnableAuth      bool
	AuthToken       string
	EnableRateLimit bool
	RateLimitRPS    float64
	RateLimitBurst  int
	CORS            CORSConfig
}

// DefaultSecurityConfig returns security defaults (auth disabled, rate limit enabled)
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableAuth:      false,
		EnableRateLimit: true,
		RateLimitRPS:    100,
		RateLimitBurst:  200,
		CORS:            DefaultCORSConfig(),
	}
}

// Server represents the HTTP gateway server
type Server struct {
	httpServer      *http.Server
	mux             *http.ServeMux
	logger          logger.Logger
	agent           AgentHandler
	healthChecker   HealthStatusProvider
	sessionStore    SessionStore
	shutdownTimeout time.Duration
}

// NewServer creates a new HTTP gateway server
func NewServer(cfg ServerConfig, agentHandler AgentHandler, log logger.Logger) *Server {
	return NewServerWithSecurity(cfg, DefaultSecurityConfig(), agentHandler, log)
}

// NewServerWithSecurity creates a new HTTP gateway server with security configuration
func NewServerWithSecurity(cfg ServerConfig, secCfg SecurityConfig, agentHandler AgentHandler, log logger.Logger) *Server {
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

	s.registerRoutes()

	middlewares := []Middleware{
		RequestIDMiddleware(),
		LoggingMiddleware(log),
		RecoveryMiddleware(log),
	}

	if secCfg.EnableAuth && secCfg.AuthToken != "" {
		middlewares = append(middlewares, security.AuthMiddleware(security.AuthConfig{
			Enabled: true,
			APIKeys: []string{secCfg.AuthToken},
		}))
		log.Info("gateway auth enabled")
	}

	if secCfg.EnableRateLimit {
		middlewares = append(middlewares, security.RateLimitMiddleware(security.RateLimitConfig{
			Enabled: true,
			Rate:    secCfg.RateLimitRPS,
			Burst:   secCfg.RateLimitBurst,
		}))
		log.Info("gateway rate limit enabled", "rps", secCfg.RateLimitRPS, "burst", secCfg.RateLimitBurst)
	}

	middlewares = append(middlewares, CORSMiddleware(secCfg.CORS))

	handler := Chain(middlewares...)(mux)

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

// SetHealthChecker sets the health checker for the provider health endpoint.
func (s *Server) SetHealthChecker(hc HealthStatusProvider) {
	s.healthChecker = hc
}

// SetSessionStore sets the session store for import/export endpoints.
func (s *Server) SetSessionStore(store SessionStore) {
	s.sessionStore = store
}

// MountHandler registers an HTTP handler at the given path on the server mux.
// Call this before Start() to add custom endpoints.
func (s *Server) MountHandler(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}
