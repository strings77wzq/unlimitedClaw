package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strings77wzq/golem/foundation/logger"
)

// benchAgentHandler is a minimal mock for benchmark tests
type benchAgentHandler struct{}

func (b *benchAgentHandler) HandleMessage(ctx context.Context, sessionID, message string) (string, error) {
	return "benchmark response", nil
}

func BenchmarkHealthCheck(b *testing.B) {
	log := logger.NopLogger()
	agent := &benchAgentHandler{}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", rec.Code)
		}
	}
}

func BenchmarkChatEndpoint(b *testing.B) {
	log := logger.NopLogger()
	agent := &benchAgentHandler{}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{
		SessionID: "bench-session",
		Message:   "benchmark test message",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", rec.Code)
		}
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	log := logger.NopLogger()
	agent := &benchAgentHandler{}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", rec.Code)
		}
	}
}

func BenchmarkConcurrentChat(b *testing.B) {
	log := logger.NopLogger()
	agent := &benchAgentHandler{}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	reqBody := chatRequest{
		SessionID: "bench-session",
		Message:   "concurrent benchmark test",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("unexpected status code: %d", rec.Code)
			}
		}
	})
}
