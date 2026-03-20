package cli

import (
	"os"
	"time"

	"github.com/tzone85/project-x/internal/llm"
)

const (
	retryMaxAttempts = 3
	retryBaseDelay   = 2 * time.Second
)

// buildLLMClient returns an LLM client wrapped with retry logic.
// Claude CLI is ALWAYS the primary client (uses subscription, no per-token cost).
// Anthropic API is only used as a fallback if PX_USE_API=true is explicitly set.
// This prevents accidental API spend — the #1 pain point from VXD.
func buildLLMClient() llm.Client {
	var base llm.Client

	if os.Getenv("PX_USE_API") == "true" {
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			base = llm.NewAnthropicClient(apiKey)
		} else {
			base = llm.NewClaudeCLIClient()
		}
	} else {
		base = llm.NewClaudeCLIClient()
	}

	return llm.NewRetryClient(base, retryMaxAttempts, retryBaseDelay)
}
