package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (a *Adapter) WebhookHandler(secretToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if secretToken != "" {
			if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secretToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			a.log.Error("failed to read webhook body", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var update Update
		if err := json.Unmarshal(body, &update); err != nil {
			a.log.Error("failed to decode webhook update", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

		if update.Message != nil && update.Message.Text != "" {
			a.log.Debug("received webhook message",
				"chat_id", update.Message.Chat.ID,
				"text", update.Message.Text)

			msg := map[string]interface{}{
				"chat_id": update.Message.Chat.ID,
				"text":    update.Message.Text,
				"user_id": int64(0),
			}
			if update.Message.From != nil {
				msg["user_id"] = update.Message.From.ID
			}

			a.msgBus.Publish("inbound", msg)
		}
	}
}

func (c *Client) SetWebhook(ctx context.Context, url string, secretToken string) error {
	params := map[string]string{"url": url}
	if secretToken != "" {
		params["secret_token"] = secretToken
	}

	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshaling webhook params: %w", err)
	}

	endpoint := fmt.Sprintf("%ssetWebhook", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("setWebhook failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) DeleteWebhook(ctx context.Context, dropPending bool) error {
	params := map[string]bool{"drop_pending_updates": dropPending}
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshaling delete params: %w", err)
	}

	endpoint := fmt.Sprintf("%sdeleteWebhook", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deleteWebhook failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
