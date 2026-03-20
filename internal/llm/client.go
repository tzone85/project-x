// Package llm defines the LLM client interface for internal operations
// such as planning, review, and conflict resolution.
package llm

import "context"

// Message represents a single message in an LLM conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse holds the result of an LLM completion request.
type CompletionResponse struct {
	Content      string  `json:"content"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	Model        string  `json:"model"`
}

// Client is the interface for making LLM completion requests.
// Implementations handle provider-specific details (Anthropic, OpenAI, etc.).
type Client interface {
	Complete(ctx context.Context, msgs []Message) (CompletionResponse, error)
}
