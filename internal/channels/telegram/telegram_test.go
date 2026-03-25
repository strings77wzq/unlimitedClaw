package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/foundation/logger"
)

func TestGetUpdates(t *testing.T) {
	updates := []Update{
		{
			UpdateID: 123,
			Message: &Message{
				MessageID: 456,
				From: &User{
					ID:        789,
					FirstName: "John",
					Username:  "johndoe",
				},
				Chat: Chat{
					ID:   789,
					Type: "private",
				},
				Text: "Hello",
				Date: 1234567890,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:     true,
			Result: mustMarshal(t, updates),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	ctx := context.Background()
	result, err := client.GetUpdates(ctx, 0, 30)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 update, got %d", len(result))
	}

	if result[0].UpdateID != 123 {
		t.Errorf("expected UpdateID 123, got %d", result[0].UpdateID)
	}

	if result[0].Message.Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", result[0].Message.Text)
	}
}

func TestGetUpdatesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage("[]"),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	ctx := context.Background()
	result, err := client.GetUpdates(ctx, 0, 30)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 updates, got %d", len(result))
	}
}

func TestSendMessage(t *testing.T) {
	var receivedReq SendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage("{}"),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	ctx := context.Background()
	err := client.SendMessage(ctx, 789, "Test message")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if receivedReq.ChatID != 789 {
		t.Errorf("expected ChatID 789, got %d", receivedReq.ChatID)
	}

	if receivedReq.Text != "Test message" {
		t.Errorf("expected text 'Test message', got %q", receivedReq.Text)
	}
}

func TestClientWithBaseURL(t *testing.T) {
	customURL := "https://custom.api.url/"
	client := NewClient("test-token", WithBaseURL(customURL))

	if client.baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, client.baseURL)
	}
}

func TestMessageParsing(t *testing.T) {
	jsonData := `{
		"update_id": 999,
		"message": {
			"message_id": 111,
			"from": {
				"id": 222,
				"first_name": "Alice",
				"username": "alice"
			},
			"chat": {
				"id": 222,
				"type": "private"
			},
			"text": "Test",
			"date": 1234567890
		}
	}`

	var update Update
	if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if update.UpdateID != 999 {
		t.Errorf("expected UpdateID 999, got %d", update.UpdateID)
	}

	if update.Message == nil {
		t.Fatal("expected Message to be non-nil")
	}

	if update.Message.Text != "Test" {
		t.Errorf("expected text 'Test', got %q", update.Message.Text)
	}
}

func TestAdapterStart(t *testing.T) {
	updateCount := 0
	var mu sync.Mutex
	sentMessageCh := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/getUpdates" {
			mu.Lock()
			var updates []Update
			if updateCount == 0 {
				updates = []Update{
					{
						UpdateID: 1,
						Message: &Message{
							MessageID: 100,
							From: &User{
								ID:        123,
								FirstName: "Test",
							},
							Chat: Chat{
								ID:   123,
								Type: "private",
							},
							Text: "Hello Bot",
							Date: 1234567890,
						},
					},
				}
				updateCount++
			}
			mu.Unlock()

			resp := APIResponse{
				OK:     true,
				Result: mustMarshal(t, updates),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else if r.URL.Path == "/sendMessage" {
			var req SendMessageRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.ChatID == 123 && req.Text == "Response" {
				select {
				case sentMessageCh <- true:
				default:
				}
			}

			resp := APIResponse{OK: true, Result: json.RawMessage("{}")}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	msgBus := bus.New()
	defer msgBus.Close()

	log := logger.NopLogger()
	cfg := AdapterConfig{
		Token:       "test-token",
		PollTimeout: 1,
	}

	adapter := NewAdapter(cfg, msgBus, log, WithBaseURL(server.URL+"/"))

	inbound := msgBus.Subscribe("inbound")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Stop()

	select {
	case msg := <-inbound:
		inMsg, ok := msg.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", msg)
		}

		if inMsg["text"] != "Hello Bot" {
			t.Errorf("expected text 'Hello Bot', got %v", inMsg["text"])
		}

		chatID, ok := inMsg["chat_id"].(int64)
		if !ok || chatID != 123 {
			t.Errorf("expected chat_id 123, got %v", inMsg["chat_id"])
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for inbound message")
	}

	msgBus.Publish("outbound", map[string]interface{}{
		"chat_id": int64(123),
		"text":    "Response",
	})

	select {
	case <-sentMessageCh:
	case <-time.After(2 * time.Second):
		t.Error("expected message to be sent")
	}
}

func TestAdapterGracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage("[]"),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	msgBus := bus.New()
	defer msgBus.Close()

	log := logger.NopLogger()
	cfg := AdapterConfig{
		Token:       "test-token",
		PollTimeout: 1,
	}

	adapter := NewAdapter(cfg, msgBus, log, WithBaseURL(server.URL+"/"))

	ctx, cancel := context.WithCancel(context.Background())

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	cancel()
	done := make(chan struct{})
	go func() {
		adapter.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("adapter did not shut down gracefully")
	}
}

func TestSendMessageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	ctx := context.Background()
	err := client.SendMessage(ctx, 789, "Test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "API returned not OK" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClientWithHTTPClient(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 5 * time.Second}
	client := NewClient("test-token", WithHTTPClient(customHTTPClient))

	if client.httpClient != customHTTPClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestWebhookHandler(t *testing.T) {
	msgBus := bus.New()
	defer msgBus.Close()

	log := logger.NopLogger()
	adapter := &Adapter{
		msgBus: msgBus,
		log:    log,
	}

	inbound := msgBus.Subscribe("inbound")

	handler := adapter.WebhookHandler("secret-token")

	reqBody := `{
		"update_id": 123,
		"message": {
			"message_id": 456,
			"from": {
				"id": 789,
				"first_name": "John",
				"username": "johndoe"
			},
			"chat": {
				"id": 789,
				"type": "private"
			},
			"text": "Hello webhook",
			"date": 1234567890
		}
	}`

	t.Run("valid webhook request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(reqBody))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		select {
		case msg := <-inbound:
			inMsg, ok := msg.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map[string]interface{}, got %T", msg)
			}
			if inMsg["text"] != "Hello webhook" {
				t.Errorf("expected text 'Hello webhook', got %v", inMsg["text"])
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("expected inbound message")
		}
	})

	t.Run("wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})

	t.Run("wrong secret token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(reqBody))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-token")

		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("no secret token configured", func(t *testing.T) {
		handlerNoSecret := adapter.WebhookHandler("")

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(reqBody))
		w := httptest.NewRecorder()
		handlerNoSecret(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("invalid json"))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-token")

		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestSetWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/setWebhook" {
			t.Errorf("expected path /setWebhook, got %s", r.URL.Path)
		}

		var params map[string]string
		json.NewDecoder(r.Body).Decode(&params)

		if params["url"] != "https://example.com/webhook" {
			t.Errorf("expected url 'https://example.com/webhook', got %s", params["url"])
		}

		resp := APIResponse{OK: true, Result: json.RawMessage("{}")}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	err := client.SetWebhook(context.Background(), "https://example.com/webhook", "secret")
	if err != nil {
		t.Fatalf("SetWebhook failed: %v", err)
	}
}

func TestSetWebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	err := client.SetWebhook(context.Background(), "https://example.com/webhook", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deleteWebhook" {
			t.Errorf("expected path /deleteWebhook, got %s", r.URL.Path)
		}

		var params map[string]bool
		json.NewDecoder(r.Body).Decode(&params)

		if !params["drop_pending_updates"] {
			t.Error("expected drop_pending_updates to be true")
		}

		resp := APIResponse{OK: true, Result: json.RawMessage("{}")}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	err := client.DeleteWebhook(context.Background(), true)
	if err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}
}

func TestDeleteWebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/"))

	err := client.DeleteWebhook(context.Background(), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOutboundLoopError(t *testing.T) {
	msgBus := bus.New()
	defer msgBus.Close()

	log := logger.NopLogger()
	cfg := AdapterConfig{
		Token:       "test-token",
		PollTimeout: 1,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{OK: false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewAdapter(cfg, msgBus, log, WithBaseURL(server.URL+"/"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer adapter.Stop()

	// Publish invalid message type
	msgBus.Publish("outbound", "invalid")

	// Publish missing chat_id
	msgBus.Publish("outbound", map[string]interface{}{"text": "hello"})

	// Publish missing text
	msgBus.Publish("outbound", map[string]interface{}{"chat_id": int64(123)})

	time.Sleep(100 * time.Millisecond)
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}
