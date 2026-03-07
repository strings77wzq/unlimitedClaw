package telegram

import "encoding/json"

// Update represents a Telegram API update
type Update struct {
	UpdateID int        `json:"update_id"`
	Message  *TGMessage `json:"message,omitempty"`
}

// TGMessage represents a Telegram message
type TGMessage struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int    `json:"date"`
}

// User represents a Telegram user
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"` // "private", "group", etc.
}

// SendMessageRequest represents a sendMessage API request
type SendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

// APIResponse represents a generic Telegram API response
type APIResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
}
