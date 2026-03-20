// Package llm provides an abstraction layer for LLM API interactions
// with token tracking and cost computation.
package llm

import "context"

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonMaxTokens FinishReason = "max_tokens"
	FinishReasonError     FinishReason = "error"
)

// CompletionOptions configures a single LLM completion request.
type CompletionOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	SystemMsg   string
	StopSeqs    []string
}

// CompletionResponse holds the result of an LLM completion call.
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Model        string
	FinishReason FinishReason
}

// Client defines the interface for LLM API interactions.
type Client interface {
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (CompletionResponse, error)
}
