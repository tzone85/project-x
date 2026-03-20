package llm

import (
	"testing"

	"github.com/tzone85/project-x/internal/config"
)

func TestComputeCost(t *testing.T) {
	pricing := config.PricingMap{
		"anthropic/claude-sonnet-4-20250514": {InputPer1M: 3.00, OutputPer1M: 15.00},
		"openai/gpt-4o-mini":                {InputPer1M: 0.15, OutputPer1M: 0.60},
	}

	tests := []struct {
		name     string
		model    string
		input    int
		output   int
		expected float64
	}{
		{
			"sonnet basic",
			"anthropic/claude-sonnet-4-20250514",
			1000, 500,
			(1000.0/1_000_000)*3.00 + (500.0/1_000_000)*15.00,
		},
		{
			"gpt-4o-mini",
			"openai/gpt-4o-mini",
			10000, 2000,
			(10000.0/1_000_000)*0.15 + (2000.0/1_000_000)*0.60,
		},
		{
			"unknown model returns 0",
			"unknown/model",
			1000, 500,
			0,
		},
		{
			"zero tokens",
			"anthropic/claude-sonnet-4-20250514",
			0, 0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCost(tt.model, tt.input, tt.output, pricing)
			if got != tt.expected {
				t.Errorf("ComputeCost(%s, %d, %d) = %f, want %f",
					tt.model, tt.input, tt.output, got, tt.expected)
			}
		})
	}
}
