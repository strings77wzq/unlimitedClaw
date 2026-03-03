package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/strin/unlimitedclaw/pkg/bus"
	"github.com/strin/unlimitedclaw/pkg/logger"
)

func TestGetUpdates(t *testing.T) {
	updates := []Update{
		{
			UpdateID: 123,
			Message: &TGMessage{
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
						Message: &TGMessage{
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

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}
