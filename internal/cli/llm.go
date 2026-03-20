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
// If ANTHROPIC_API_KEY is set in the environment, an AnthropicClient is used.
// Otherwise, a ClaudeCLIClient is used (routes through the user's Claude subscription).
func buildLLMClient() llm.Client {
	var base llm.Client

	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		base = llm.NewAnthropicClient(apiKey)
	} else {
		base = llm.NewClaudeCLIClient()
	}

	return llm.NewRetryClient(base, retryMaxAttempts, retryBaseDelay)
}
