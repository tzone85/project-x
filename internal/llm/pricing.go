package llm

import "github.com/tzone85/project-x/internal/config"

// ComputeCost calculates the USD cost for a completion based on the pricing table.
func ComputeCost(model string, inputTokens, outputTokens int, pricing config.PricingMap) float64 {
	p, ok := pricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * p.InputPer1M
	outputCost := float64(outputTokens) / 1_000_000 * p.OutputPer1M
	return inputCost + outputCost
}
