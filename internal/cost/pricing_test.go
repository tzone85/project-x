package cost

import "testing"

func TestComputeCost_KnownModel(t *testing.T) {
	cost := ComputeCost("claude-sonnet-4-20250514", 1000, 500, DefaultPricing)
	// Input: 1000 tokens * $3.00/1M = $0.003
	// Output: 500 tokens * $15.00/1M = $0.0075
	// Total: $0.0105
	expected := 0.0105
	if diff := cost - expected; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("expected ~%.4f, got %.4f", expected, cost)
	}
}

func TestComputeCost_UnknownModel(t *testing.T) {
	cost := ComputeCost("unknown-model", 1000, 500, DefaultPricing)
	if cost != 0 {
		t.Errorf("expected 0 for unknown model, got %f", cost)
	}
}

func TestComputeCost_ZeroTokens(t *testing.T) {
	cost := ComputeCost("claude-sonnet-4-20250514", 0, 0, DefaultPricing)
	if cost != 0 {
		t.Errorf("expected 0 for zero tokens, got %f", cost)
	}
}

func TestDefaultPricing_HasExpectedModels(t *testing.T) {
	expected := []string{
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-haiku-4-5-20251001",
		"gpt-4o-mini",
	}
	for _, model := range expected {
		if _, ok := DefaultPricing[model]; !ok {
			t.Errorf("missing model in DefaultPricing: %s", model)
		}
	}
}
