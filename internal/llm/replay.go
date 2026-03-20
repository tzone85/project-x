package llm

import (
	"context"
	"sync"
)

// ReplayClient returns pre-recorded responses for deterministic testing.
// It cycles through the provided responses in order, wrapping around to the
// start when all responses have been consumed. This enables tests to exercise
// LLM-dependent code without making real API calls.
type ReplayClient struct {
	responses []CompletionResponse
	index     int
	mu        sync.Mutex
}

// NewReplayClient creates a client that returns the given responses in sequence.
// When all responses have been returned, it wraps around to the beginning.
func NewReplayClient(responses ...CompletionResponse) *ReplayClient {
	// Copy the slice to prevent external mutation.
	copied := make([]CompletionResponse, len(responses))
	copy(copied, responses)

	return &ReplayClient{
		responses: copied,
	}
}

// Compile-time interface check.
var _ Client = (*ReplayClient)(nil)

// Complete returns the next pre-recorded response in sequence.
// It is safe for concurrent use. When all responses have been consumed,
// it wraps around to the first response.
func (c *ReplayClient) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.responses) == 0 {
		return CompletionResponse{}, nil
	}

	resp := c.responses[c.index]
	c.index = (c.index + 1) % len(c.responses)

	return resp, nil
}
