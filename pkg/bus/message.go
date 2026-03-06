package bus

// Role represents the role of a message participant
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// InboundMessage represents a message coming into the system
type InboundMessage struct {
	SessionID string
	Content   string
	Role      Role
}

// TokenUsage tracks token consumption for display purposes.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// OutboundMessage represents a message going out from the system
type OutboundMessage struct {
	SessionID  string
	Content    string
	Role       Role
	Done       bool        // signals end of response stream
	TokenDelta string      // streaming token chunk (empty for non-streaming)
	Usage      *TokenUsage // token usage data (nil until response complete)
}
