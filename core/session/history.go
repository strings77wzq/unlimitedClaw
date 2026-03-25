package session

import "github.com/strings77wzq/golem/core/providers"

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
// Uses chars/4 approximation for ASCII and chars/2 for CJK characters.
func (h *HistoryManager) EstimateTokens(msg providers.Message) int {
	var cjkCount, asciiCount int
	for _, r := range msg.Content {
		if isCJKRune(r) {
			cjkCount++
		} else {
			asciiCount++
		}
	}
	// CJK: ~2 chars per token, ASCII: ~4 chars per token
	return (cjkCount+1)/2 + (asciiCount+3)/4
}

// isCJKRune returns true if the rune is a CJK character.
// Covers Chinese, Japanese (Hiragana/Katakana), and Korean (Hangul).
func isCJKRune(r rune) bool {
	// CJK Unified Ideographs (Chinese + Kanji)
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	// CJK Unified Ideographs Extension A
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	// CJK Compatibility Ideographs
	if r >= 0xF900 && r <= 0xFAFF {
		return true
	}
	// Hiragana
	if r >= 0x3040 && r <= 0x309F {
		return true
	}
	// Katakana
	if r >= 0x30A0 && r <= 0x30FF {
		return true
	}
	// Hangul Syllables (Korean)
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	// Hangul Jamo (Korean)
	if r >= 0x1100 && r <= 0x11FF {
		return true
	}
	return false
}
