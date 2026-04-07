package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreproviders "github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/foundation/logger"
)

type mockAgentHandler struct {
	response string
	err      error
}

type mockHealthStatusProvider struct {
	statuses map[string]*coreproviders.HealthStatus
}

func (m *mockHealthStatusProvider) GetAllStatuses() map[string]*coreproviders.HealthStatus {
	return m.statuses
}

func (m *mockAgentHandler) HandleMessage(ctx context.Context, sessionID, message string) (string, error) {
	return m.response, m.err
}

func TestHealthCheck(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}

	if resp.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}

	if _, err := time.Parse(time.RFC3339, resp.Timestamp); err != nil {
		t.Errorf("timestamp not in RFC3339 format: %v", err)
	}
}

func TestProvidersHealth_NotConfigured(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/health/providers", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "not_configured" {
		t.Errorf("expected status 'not_configured', got %q", resp["status"])
	}
	if resp["message"] != "health checker not configured" {
		t.Errorf("expected not-configured message, got %q", resp["message"])
	}
}

func TestProvidersHealth_WithChecker(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	provider := &mockHealthStatusProvider{statuses: map[string]*coreproviders.HealthStatus{
		"openai": {
			Provider:  "openai",
			Status:    "healthy",
			Latency:   42,
			CheckedAt: 1710000000,
		},
		"anthropic": {
			Provider:  "anthropic",
			Status:    "degraded",
			Latency:   2501,
			Error:     "slow upstream",
			CheckedAt: 1710000001,
		},
	}}
	server.SetHealthChecker(provider)

	req := httptest.NewRequest(http.MethodGet, "/health/providers", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]coreproviders.HealthStatus
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 provider statuses, got %d", len(resp))
	}
	if resp["openai"].Status != "healthy" {
		t.Errorf("expected openai healthy, got %q", resp["openai"].Status)
	}
	if resp["anthropic"].Status != "degraded" {
		t.Errorf("expected anthropic degraded, got %q", resp["anthropic"].Status)
	}
}

func TestVersionEndpoint(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp versionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Version != "dev" {
		t.Errorf("expected version 'dev', got '%s'", resp.Version)
	}
}

func TestChatEndpoint(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "Hello, how can I help you?"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{
		SessionID: "test-session",
		Message:   "Hello",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp chatResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.SessionID != "test-session" {
		t.Errorf("expected session_id 'test-session', got '%s'", resp.SessionID)
	}

	if resp.Response != "Hello, how can I help you?" {
		t.Errorf("expected response 'Hello, how can I help you?', got '%s'", resp.Response)
	}
}

func TestChatEndpointEmptyMessage(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{
		SessionID: "test-session",
		Message:   "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Error, "message is required") {
		t.Errorf("expected error to contain 'message is required', got '%s'", resp.Error)
	}
}

func TestChatEndpointAgentError(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{err: errors.New("agent processing failed")}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{
		SessionID: "test-session",
		Message:   "Hello",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Error, "internal server error") {
		t.Errorf("expected error to contain 'internal server error', got '%s'", resp.Error)
	}
}

func TestChatEndpointInvalidJSON(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Error, "invalid JSON") {
		t.Errorf("expected error to contain 'invalid JSON', got '%s'", resp.Error)
	}
}

func TestCORSHeaders(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	// Test with localhost origin (should be allowed by default)
	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:3000', got '%s'", origin)
	}

	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}

	if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}
}

func TestCORSHeadersBlocked(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	// Test with unknown origin (should be blocked by default)
	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", rec.Code)
	}

	// Origin should not be echoed back for blocked origins
	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for blocked origin, got '%s'", origin)
	}
}

func TestCORSHeadersWildcard(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	secCfg := DefaultSecurityConfig()
	secCfg.CORS = CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}
	server := NewServerWithSecurity(cfg, secCfg, agent, log)

	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	req.Header.Set("Origin", "https://any-site.com")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	log := logger.NopLogger()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := RecoveryMiddleware(log)
	handler := middleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic, got %d", rec.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestIDMiddleware()
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	log := logger.NopLogger()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(log)
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "m1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "m1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "m2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "m2-after")
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(middleware1, middleware2)(finalHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionOrder))
	}

	for i, v := range expected {
		if executionOrder[i] != v {
			t.Errorf("execution order[%d]: expected '%s', got '%s'", i, v, executionOrder[i])
		}
	}
}

type mockStreamingHandler struct {
	tokens []string
	err    error
}

func (m *mockStreamingHandler) HandleMessage(ctx context.Context, sessionID, message string) (string, error) {
	return strings.Join(m.tokens, ""), m.err
}

func (m *mockStreamingHandler) HandleMessageStream(ctx context.Context, sessionID, message string, ch chan<- string) error {
	defer close(ch)
	for _, tok := range m.tokens {
		ch <- tok
	}
	return m.err
}

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{httptest.NewRecorder()}
}

func TestChatStreamEndpoint(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockStreamingHandler{tokens: []string{"Hello", " ", "world"}}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{SessionID: "s1", Message: "hi"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}

	respBody := rec.Body.String()
	if !strings.Contains(respBody, "data: Hello") {
		t.Errorf("expected 'data: Hello' in response, got %q", respBody)
	}
	if !strings.Contains(respBody, "data: world") {
		t.Errorf("expected 'data: world' in response, got %q", respBody)
	}
	if !strings.Contains(respBody, "event: done") {
		t.Errorf("expected 'event: done' in response, got %q", respBody)
	}
}

func TestChatStreamFallbackToSync(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "sync response"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{SessionID: "s1", Message: "hi"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	respBody := rec.Body.String()
	if !strings.Contains(respBody, "data: sync response") {
		t.Errorf("expected sync fallback data, got %q", respBody)
	}
	if !strings.Contains(respBody, "event: done") {
		t.Errorf("expected 'event: done', got %q", respBody)
	}
}

func TestChatStreamEmptyMessage(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockStreamingHandler{tokens: []string{"x"}}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{SessionID: "s1", Message: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
