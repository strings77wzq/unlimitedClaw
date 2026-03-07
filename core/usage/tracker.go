package usage

import "sync"

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Tracker struct {
	mu       sync.Mutex
	sessions map[string]*SessionUsage
}

type SessionUsage struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Requests         int
	EstimatedCostUSD float64
}

func NewTracker() *Tracker {
	return &Tracker{
		sessions: make(map[string]*SessionUsage),
	}
}

func (t *Tracker) Record(sessionID, model string, u TokenUsage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	su, ok := t.sessions[sessionID]
	if !ok {
		su = &SessionUsage{Model: model}
		t.sessions[sessionID] = su
	}

	su.PromptTokens += u.PromptTokens
	su.CompletionTokens += u.CompletionTokens
	su.TotalTokens += u.TotalTokens
	su.Requests++
	su.Model = model

	pricing := GetPricing(model)
	su.EstimatedCostUSD += float64(u.PromptTokens) * pricing.InputPerToken
	su.EstimatedCostUSD += float64(u.CompletionTokens) * pricing.OutputPerToken
}

func (t *Tracker) GetSession(sessionID string) *SessionUsage {
	t.mu.Lock()
	defer t.mu.Unlock()

	su, ok := t.sessions[sessionID]
	if !ok {
		return &SessionUsage{}
	}
	cp := *su
	return &cp
}

func (t *Tracker) GetTotal() *SessionUsage {
	t.mu.Lock()
	defer t.mu.Unlock()

	total := &SessionUsage{}
	for _, su := range t.sessions {
		total.PromptTokens += su.PromptTokens
		total.CompletionTokens += su.CompletionTokens
		total.TotalTokens += su.TotalTokens
		total.Requests += su.Requests
		total.EstimatedCostUSD += su.EstimatedCostUSD
	}
	return total
}
