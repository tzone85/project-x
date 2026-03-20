package llm

import "context"

// Role represents a message participant role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest is the input to an LLM completion call.
type CompletionRequest struct {
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Model     string    `json:"model,omitempty"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// CompletionResponse is the output from an LLM completion call.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// Client is the interface for LLM completion providers.
type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
