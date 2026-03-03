package session

import "github.com/strings77wzq/unlimitedClaw/pkg/providers"

// HistoryManager handles context window building with token budget.
type HistoryManager struct {
	MaxTokens int
}

// NewHistoryManager creates a new history manager with the given token budget.
func NewHistoryManager(maxTokens int) *HistoryManager {
	return &HistoryManager{
		MaxTokens: maxTokens,
	}
}

// GetContextWindow returns messages that fit within the token budget.
// System prompt (first message with Role=system) is NEVER truncated.
// Algorithm:
//  1. Always include system message (if present)
//  2. Subtract system message tokens from budget
//  3. Add messages from most recent to oldest until budget exhausted
//  4. Return in chronological order (system first, then selected messages)
func (h *HistoryManager) GetContextWindow(messages []providers.Message) []providers.Message {
	if len(messages) == 0 {
		return []providers.Message{}
	}

	var systemMsg *providers.Message
	var otherMsgs []providers.Message

	if messages[0].Role == providers.RoleSystem {
		systemMsg = &messages[0]
		otherMsgs = messages[1:]
	} else {
		otherMsgs = messages
	}

	budget := h.MaxTokens

	if systemMsg != nil {
		budget -= h.EstimateTokens(*systemMsg)
	}

	if budget < 0 {
		budget = 0
	}

	var selected []providers.Message
	for i := len(otherMsgs) - 1; i >= 0; i-- {
		msgTokens := h.EstimateTokens(otherMsgs[i])
		if budget-msgTokens < 0 {
			break
		}
		budget -= msgTokens
		selected = append([]providers.Message{otherMsgs[i]}, selected...)
	}

	var result []providers.Message
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}
	result = append(result, selected...)

	return result
}

// EstimateTokens gives a rough token estimate for a message.
// Uses chars/4 approximation (industry standard rough estimate).
func (h *HistoryManager) EstimateTokens(msg providers.Message) int {
	return len(msg.Content) / 4
}
