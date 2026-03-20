package llm

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ModelPrice holds per-million-token pricing for a model.
type ModelPrice struct {
	InputPer1M  float64 `yaml:"input_per_1m"`
	OutputPer1M float64 `yaml:"output_per_1m"`
}

// PricingTable maps model identifiers to their pricing.
type PricingTable struct {
	prices map[string]ModelPrice
}

// NewPricingTable creates a PricingTable from a map of model prices.
func NewPricingTable(prices map[string]ModelPrice) PricingTable {
	copied := make(map[string]ModelPrice, len(prices))
	for k, v := range prices {
		copied[k] = v
	}
	return PricingTable{prices: copied}
}

// DefaultPricingTable returns the built-in pricing for known models.
func DefaultPricingTable() PricingTable {
	return NewPricingTable(map[string]ModelPrice{
		"anthropic/claude-opus-4-20250514":   {InputPer1M: 15.00, OutputPer1M: 75.00},
		"anthropic/claude-sonnet-4-20250514": {InputPer1M: 3.00, OutputPer1M: 15.00},
		"openai/gpt-4o-mini":                {InputPer1M: 0.15, OutputPer1M: 0.60},
	})
}

// GetPrice returns the pricing for a model. The second return value
// indicates whether the model was found.
func (pt PricingTable) GetPrice(model string) (ModelPrice, bool) {
	p, ok := pt.prices[model]
	return p, ok
}

// ComputeCost calculates the USD cost for a given model and token counts.
// Formula: (inputTokens / 1M * inputPrice) + (outputTokens / 1M * outputPrice)
func (pt PricingTable) ComputeCost(model string, inputTokens, outputTokens int) (float64, error) {
	price, ok := pt.prices[model]
	if !ok {
		return 0, fmt.Errorf("unknown model %q: no pricing available", model)
	}

	inputCost := float64(inputTokens) / 1_000_000 * price.InputPer1M
	outputCost := float64(outputTokens) / 1_000_000 * price.OutputPer1M

	return inputCost + outputCost, nil
}

// Merge returns a new PricingTable with entries from both tables.
// Entries in other override entries in the receiver. Neither table is mutated.
func (pt PricingTable) Merge(other PricingTable) PricingTable {
	merged := make(map[string]ModelPrice, len(pt.prices)+len(other.prices))
	for k, v := range pt.prices {
		merged[k] = v
	}
	for k, v := range other.prices {
		merged[k] = v
	}
	return PricingTable{prices: merged}
}

// PricingTableFromYAML parses a pricing table from YAML bytes.
// Expected format:
//
//	model/name:
//	  input_per_1m: 15.00
//	  output_per_1m: 75.00
func PricingTableFromYAML(data []byte) (PricingTable, error) {
	var raw map[string]ModelPrice
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return PricingTable{}, fmt.Errorf("parsing pricing YAML: %w", err)
	}
	return NewPricingTable(raw), nil
}
