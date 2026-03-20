package cost

// PricingEntry holds the per-million-token cost for a model.
type PricingEntry struct {
	InputPer1M  float64
	OutputPer1M float64
}

// DefaultPricing contains pricing for well-known models.
// Users can override via px.yaml configuration.
var DefaultPricing = map[string]PricingEntry{
	"claude-opus-4-20250514":    {InputPer1M: 15.0, OutputPer1M: 75.0},
	"claude-sonnet-4-20250514":  {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-haiku-4-5-20251001": {InputPer1M: 0.80, OutputPer1M: 4.0},
	"gpt-4o-mini":               {InputPer1M: 0.15, OutputPer1M: 0.60},
}

// ComputeCost calculates the dollar cost for the given token counts using
// the supplied pricing table. Returns 0 if the model is not found.
func ComputeCost(model string, inputTokens, outputTokens int, pricing map[string]PricingEntry) float64 {
	entry, ok := pricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) * entry.InputPer1M / 1_000_000
	outputCost := float64(outputTokens) * entry.OutputPer1M / 1_000_000
	return inputCost + outputCost
}
