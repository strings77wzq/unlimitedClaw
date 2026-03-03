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

	"github.com/strin/unlimitedclaw/pkg/logger"
)

type mockAgentHandler struct {
	response string
	err      error
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

	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}

	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}

	if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
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
