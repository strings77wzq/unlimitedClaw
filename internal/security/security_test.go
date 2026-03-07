package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthMiddlewareValidKey(t *testing.T) {
	cfg := AuthConfig{
		APIKeys: []string{"test-key-123"},
		Enabled: true,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareBearerToken(t *testing.T) {
	cfg := AuthConfig{
		APIKeys: []string{"bearer-token-456"},
		Enabled: true,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bearer-token-456")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareInvalidKey(t *testing.T) {
	cfg := AuthConfig{
		APIKeys: []string{"valid-key"},
		Enabled: true,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Errorf("expected error message to contain 'unauthorized', got %s", rec.Body.String())
	}
}

func TestAuthMiddlewareMissingKey(t *testing.T) {
	cfg := AuthConfig{
		APIKeys: []string{"valid-key"},
		Enabled: true,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareDisabled(t *testing.T) {
	cfg := AuthConfig{
		APIKeys: []string{"valid-key"},
		Enabled: false,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 when disabled, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAllowFrom(t *testing.T) {
	cfg := AuthConfig{
		APIKeys:   []string{"test-key"},
		AllowFrom: []string{"192.168.1.0/24"},
		Enabled:   true,
	}

	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for allowed IP, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-API-Key", "test-key")
	req2.Header.Set("X-Forwarded-For", "10.0.0.1")
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for denied IP, got %d", rec2.Code)
	}
}

func TestRateLimitAllow(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    10,
		Burst:   5,
		Enabled: true,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimitDeny(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    10,
		Burst:   2,
		Enabled: true,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if i < 2 {
			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected status 200, got %d", i, rec.Code)
			}
		} else {
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("request %d: expected status 429, got %d", i, rec.Code)
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Error("expected Retry-After header")
			}
			if !strings.Contains(rec.Body.String(), "rate limit exceeded") {
				t.Errorf("expected error message to contain 'rate limit exceeded', got %s", rec.Body.String())
			}
		}
	}
}

func TestRateLimitPerIP(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    10,
		Burst:   1,
		Enabled: true,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 first request: expected status 200, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 first request: expected status 200, got %d", rec2.Code)
	}
}

func TestRateLimitDisabled(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    1,
		Burst:   1,
		Enabled: false,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200 when disabled, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitRefill(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    10,
		Burst:   1,
		Enabled: true,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request (immediate): expected status 429, got %d", rec2.Code)
	}

	time.Sleep(150 * time.Millisecond)

	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("third request (after refill): expected status 200, got %d", rec3.Code)
	}
}

func TestSandboxValidatePath(t *testing.T) {
	sandbox := NewSandbox(SandboxConfig{
		AllowedPaths: []string{"/home/user/workspace"},
		DeniedPaths:  []string{"/home/user/workspace/secrets"},
	})

	err := sandbox.ValidatePath("/home/user/workspace/project/file.txt")
	if err != nil {
		t.Errorf("expected nil error for allowed path, got %v", err)
	}
}

func TestSandboxDeniedPath(t *testing.T) {
	sandbox := NewSandbox(SandboxConfig{
		AllowedPaths: []string{"/home/user/workspace"},
		DeniedPaths:  []string{"/home/user/workspace/secrets"},
	})

	err := sandbox.ValidatePath("/home/user/workspace/secrets/key.txt")
	if err == nil {
		t.Error("expected error for denied path, got nil")
	}
}

func TestSandboxValidateCommand(t *testing.T) {
	sandbox := NewSandbox(SandboxConfig{
		DeniedCommands: []string{"rm", "shutdown"},
	})

	err := sandbox.ValidateCommand("ls -la")
	if err != nil {
		t.Errorf("expected nil error for allowed command, got %v", err)
	}
}

func TestSandboxDeniedCommand(t *testing.T) {
	sandbox := NewSandbox(SandboxConfig{
		DeniedCommands: []string{"rm", "shutdown"},
	})

	err := sandbox.ValidateCommand("rm -rf /")
	if err == nil {
		t.Error("expected error for denied command, got nil")
	}

	err2 := sandbox.ValidateCommand("RM -rf /")
	if err2 == nil {
		t.Error("expected error for denied command (case insensitive), got nil")
	}
}

func TestSandboxPathTraversal(t *testing.T) {
	sandbox := NewSandbox(SandboxConfig{
		AllowedPaths: []string{"/home/user/workspace"},
	})

	err := sandbox.ValidatePath("/home/user/workspace/../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal escape, got nil")
	}
}

func TestRateLimitConcurrent(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:    100,
		Burst:   10,
		Enabled: true,
	}

	handler := RateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:1234"
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
