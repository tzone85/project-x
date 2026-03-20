// Package llm defines the LLM client interface and response types.
package llm

import "context"

// CompletionOptions configures an LLM completion request.
type CompletionOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	StoryID     string // For cost tracking
	ReqID       string // For cost tracking
	Stage       string // Pipeline stage context
}

// CompletionResponse contains the result of an LLM completion.
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Model        string
	FinishReason string
}

// Client is the interface for making LLM completions.
// Implementations handle API calls to specific providers (Anthropic, OpenAI, etc.).
type Client interface {
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (CompletionResponse, error)
}
