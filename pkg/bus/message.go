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

// OutboundMessage represents a message going out from the system
type OutboundMessage struct {
	SessionID string
	Content   string
	Role      Role
	Done      bool // signals end of response stream
}
