package llm

import (
	"math"
	"testing"
)

func TestDefaultPricingTable(t *testing.T) {
	table := DefaultPricingTable()

	tests := []struct {
		name  string
		model string
	}{
		{"claude-opus", "anthropic/claude-opus-4-20250514"},
		{"claude-sonnet", "anthropic/claude-sonnet-4-20250514"},
		{"gpt-4o-mini", "openai/gpt-4o-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := table.GetPrice(tt.model)
			if !ok {
				t.Fatalf("expected pricing for model %q, got none", tt.model)
			}
			if price.InputPer1M <= 0 {
				t.Errorf("expected positive input price, got %f", price.InputPer1M)
			}
			if price.OutputPer1M <= 0 {
				t.Errorf("expected positive output price, got %f", price.OutputPer1M)
			}
		})
	}
}

func TestComputeCost(t *testing.T) {
	table := DefaultPricingTable()

	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantCost     float64
	}{
		{
			name:         "claude-opus 1000 in 500 out",
			model:        "anthropic/claude-opus-4-20250514",
			inputTokens:  1000,
			outputTokens: 500,
			// (1000/1M * 15.00) + (500/1M * 75.00) = 0.015 + 0.0375 = 0.0525
			wantCost: 0.0525,
		},
		{
			name:         "claude-sonnet 10000 in 2000 out",
			model:        "anthropic/claude-sonnet-4-20250514",
			inputTokens:  10000,
			outputTokens: 2000,
			// (10000/1M * 3.00) + (2000/1M * 15.00) = 0.03 + 0.03 = 0.06
			wantCost: 0.06,
		},
		{
			name:         "gpt-4o-mini 5000 in 1000 out",
			model:        "openai/gpt-4o-mini",
			inputTokens:  5000,
			outputTokens: 1000,
			// (5000/1M * 0.15) + (1000/1M * 0.60) = 0.00075 + 0.0006 = 0.00135
			wantCost: 0.00135,
		},
		{
			name:         "zero tokens",
			model:        "anthropic/claude-opus-4-20250514",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := table.ComputeCost(tt.model, tt.inputTokens, tt.outputTokens)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(cost-tt.wantCost) > 1e-9 {
				t.Errorf("cost = %f, want %f", cost, tt.wantCost)
			}
		})
	}
}

func TestComputeCostUnknownModel(t *testing.T) {
	table := DefaultPricingTable()

	_, err := table.ComputeCost("unknown/model", 100, 100)
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
}

func TestGetPriceUnknownModel(t *testing.T) {
	table := DefaultPricingTable()

	_, ok := table.GetPrice("unknown/model")
	if ok {
		t.Fatal("expected false for unknown model, got true")
	}
}

func TestPricingTableFromYAML(t *testing.T) {
	yamlData := []byte(`
anthropic/claude-opus-4-20250514:
  input_per_1m: 20.00
  output_per_1m: 100.00
custom/model:
  input_per_1m: 1.00
  output_per_1m: 5.00
`)

	table, err := PricingTableFromYAML(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	price, ok := table.GetPrice("anthropic/claude-opus-4-20250514")
	if !ok {
		t.Fatal("expected pricing for claude-opus")
	}
	if price.InputPer1M != 20.00 {
		t.Errorf("input price = %f, want 20.00", price.InputPer1M)
	}

	price, ok = table.GetPrice("custom/model")
	if !ok {
		t.Fatal("expected pricing for custom/model")
	}
	if price.OutputPer1M != 5.00 {
		t.Errorf("output price = %f, want 5.00", price.OutputPer1M)
	}
}

func TestPricingTableMerge(t *testing.T) {
	base := DefaultPricingTable()

	override := PricingTable{
		prices: map[string]ModelPrice{
			// Override existing
			"anthropic/claude-opus-4-20250514": {InputPer1M: 20.00, OutputPer1M: 100.00},
			// Add new
			"custom/model": {InputPer1M: 1.00, OutputPer1M: 5.00},
		},
	}

	merged := base.Merge(override)

	// Overridden price
	price, ok := merged.GetPrice("anthropic/claude-opus-4-20250514")
	if !ok {
		t.Fatal("expected pricing for claude-opus")
	}
	if price.InputPer1M != 20.00 {
		t.Errorf("input price = %f, want 20.00", price.InputPer1M)
	}

	// Preserved default
	price, ok = merged.GetPrice("anthropic/claude-sonnet-4-20250514")
	if !ok {
		t.Fatal("expected pricing for claude-sonnet from base")
	}
	if price.InputPer1M != 3.00 {
		t.Errorf("input price = %f, want 3.00", price.InputPer1M)
	}

	// New model
	price, ok = merged.GetPrice("custom/model")
	if !ok {
		t.Fatal("expected pricing for custom/model")
	}
	if price.InputPer1M != 1.00 {
		t.Errorf("input price = %f, want 1.00", price.InputPer1M)
	}

	// Original base not mutated
	price, _ = base.GetPrice("anthropic/claude-opus-4-20250514")
	if price.InputPer1M != 15.00 {
		t.Errorf("base was mutated: input price = %f, want 15.00", price.InputPer1M)
	}
}

func TestPricingTableFromYAMLInvalid(t *testing.T) {
	_, err := PricingTableFromYAML([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
