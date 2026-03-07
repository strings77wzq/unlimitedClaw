package usage

import "testing"

func TestTrackerRecord(t *testing.T) {
	tr := NewTracker()

	tr.Record("sess1", "gpt-4o", TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	})
	tr.Record("sess1", "gpt-4o", TokenUsage{
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
	})

	su := tr.GetSession("sess1")
	if su.PromptTokens != 300 {
		t.Errorf("PromptTokens = %d, want 300", su.PromptTokens)
	}
	if su.CompletionTokens != 150 {
		t.Errorf("CompletionTokens = %d, want 150", su.CompletionTokens)
	}
	if su.TotalTokens != 450 {
		t.Errorf("TotalTokens = %d, want 450", su.TotalTokens)
	}
	if su.Requests != 2 {
		t.Errorf("Requests = %d, want 2", su.Requests)
	}
	if su.EstimatedCostUSD <= 0 {
		t.Error("EstimatedCostUSD should be > 0 for a known model")
	}
}

func TestTrackerGetSessionUnknown(t *testing.T) {
	tr := NewTracker()
	su := tr.GetSession("nonexistent")
	if su.TotalTokens != 0 {
		t.Errorf("expected zero usage for unknown session, got %d", su.TotalTokens)
	}
}

func TestTrackerGetTotal(t *testing.T) {
	tr := NewTracker()

	tr.Record("a", "gpt-4o", TokenUsage{TotalTokens: 100, PromptTokens: 60, CompletionTokens: 40})
	tr.Record("b", "gpt-4o", TokenUsage{TotalTokens: 200, PromptTokens: 120, CompletionTokens: 80})

	total := tr.GetTotal()
	if total.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", total.TotalTokens)
	}
	if total.Requests != 2 {
		t.Errorf("Requests = %d, want 2", total.Requests)
	}
}

func TestGetPricingKnown(t *testing.T) {
	p := GetPricing("gpt-4o")
	if p.InputPerToken <= 0 {
		t.Error("expected non-zero pricing for gpt-4o")
	}
}

func TestGetPricingUnknown(t *testing.T) {
	p := GetPricing("nonexistent-model-xyz")
	if p.InputPerToken != 0 || p.OutputPerToken != 0 {
		t.Error("expected zero pricing for unknown model")
	}
}

func TestGetPricingCaseInsensitive(t *testing.T) {
	p := GetPricing("GPT-4o")
	if p.InputPerToken <= 0 {
		t.Error("expected non-zero pricing for case-insensitive match")
	}
}

func TestTrackerUnknownModelZeroCost(t *testing.T) {
	tr := NewTracker()
	tr.Record("s", "unknown-model", TokenUsage{TotalTokens: 100, PromptTokens: 50, CompletionTokens: 50})
	su := tr.GetSession("s")
	if su.EstimatedCostUSD != 0 {
		t.Errorf("expected zero cost for unknown model, got %f", su.EstimatedCostUSD)
	}
	if su.TotalTokens != 100 {
		t.Errorf("TotalTokens = %d, want 100", su.TotalTokens)
	}
}
