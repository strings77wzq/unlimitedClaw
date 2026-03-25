package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://api.telegram.org/bot"

// Client is an HTTP client for the Telegram Bot API
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client)

// WithBaseURL overrides the default Telegram API base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a new Telegram Bot API client
func NewClient(token string, opts ...ClientOption) *Client {
	c := &Client{
		token:      token,
		baseURL:    defaultBaseURL + token + "/",
		httpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetUpdates retrieves updates using long polling
func (c *Client) GetUpdates(ctx context.Context, offset int, timeout int) ([]Update, error) {
	url := fmt.Sprintf("%sgetUpdates?offset=%d&timeout=%d", c.baseURL, offset, timeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("API returned not OK")
	}

	var updates []Update
	if err := json.Unmarshal(apiResp.Result, &updates); err != nil {
		return nil, fmt.Errorf("unmarshaling updates: %w", err)
	}

	return updates, nil
}

// doRequest performs a POST request to the given Telegram Bot API method
func (c *Client) doRequest(method string, body []byte) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, method)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request to %s: %w", method, err)
	}

	return resp, nil
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%ssendMessage", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("API returned not OK")
	}

	return nil
}
