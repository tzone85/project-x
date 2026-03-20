package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultAnthropicURL     = "https://api.anthropic.com/v1/messages"
	defaultAnthropicVersion = "2023-06-01"
	defaultModel            = "claude-sonnet-4-20250514"
	defaultMaxTokens        = 4096
)

// AnthropicClient implements Client using the Anthropic Messages API directly.
// This is the fallback client for when Claude CLI can't handle a request
// (e.g., large prompts, specific model requirements). It uses real API credits,
// making token tracking in CompletionResponse critical for cost management.
type AnthropicClient struct {
	apiKey     string
	baseURL    string
	apiVersion string
	httpClient *http.Client
}

// NewAnthropicClient creates a client configured with the given API key
// and default Anthropic API settings.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		apiKey:     apiKey,
		baseURL:    defaultAnthropicURL,
		apiVersion: defaultAnthropicVersion,
		httpClient: http.DefaultClient,
	}
}

// WithBaseURL returns a new copy with a custom base URL (for testing).
// The original client is not modified (immutability).
func (c *AnthropicClient) WithBaseURL(url string) *AnthropicClient {
	return &AnthropicClient{
		apiKey:     c.apiKey,
		baseURL:    url,
		apiVersion: c.apiVersion,
		httpClient: c.httpClient,
	}
}

// Compile-time interface check.
var _ Client = (*AnthropicClient)(nil)

// anthropicRequest is the request body for the Anthropic Messages API.
type anthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

// anthropicContentBlock represents a single content block in the API response.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicUsage represents token usage from the API response.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicResponse is the success response from the Anthropic Messages API.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Model   string                 `json:"model"`
	Usage   anthropicUsage         `json:"usage"`
}

// anthropicErrorBody is the error response from the Anthropic Messages API.
type anthropicErrorBody struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends a completion request to the Anthropic Messages API and returns
// the parsed response. It extracts the first text block from the content array
// and populates token usage for cost tracking.
func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	httpReq, err := c.buildHTTPRequest(ctx, req)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("building request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("reading response body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return CompletionResponse{}, parseErrorResponse(httpResp.StatusCode, body)
	}

	return parseSuccessResponse(body)
}

// buildHTTPRequest constructs the HTTP request for the Anthropic Messages API.
func (c *AnthropicClient) buildHTTPRequest(ctx context.Context, req CompletionRequest) (*http.Request, error) {
	apiReq := anthropicRequest{
		Model:     resolveModel(req.Model),
		MaxTokens: resolveMaxTokens(req.MaxTokens),
		System:    req.System,
		Messages:  req.Messages,
	}

	payload, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.apiVersion)
	httpReq.Header.Set("content-type", "application/json")

	return httpReq, nil
}

// resolveModel returns the given model if non-empty, otherwise the default.
func resolveModel(model string) string {
	if model != "" {
		return model
	}
	return defaultModel
}

// resolveMaxTokens returns the given value if positive, otherwise the default.
func resolveMaxTokens(maxTokens int) int {
	if maxTokens > 0 {
		return maxTokens
	}
	return defaultMaxTokens
}

// parseSuccessResponse extracts content and usage from a successful API response.
func parseSuccessResponse(body []byte) (CompletionResponse, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	content := extractFirstTextBlock(resp.Content)

	return CompletionResponse{
		Content:      content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// extractFirstTextBlock returns the text from the first "text" content block,
// or empty string if no text blocks are found.
func extractFirstTextBlock(blocks []anthropicContentBlock) string {
	for _, block := range blocks {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}

// parseErrorResponse creates an APIError from a non-2xx response.
// Status codes 429, 500, and 503 are marked as retryable.
func parseErrorResponse(statusCode int, body []byte) *APIError {
	message := extractErrorMessage(statusCode, body)
	retryable := isRetryableStatus(statusCode)

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Retryable:  retryable,
	}
}

// extractErrorMessage attempts to parse the Anthropic error JSON format.
// Falls back to the raw body string if parsing fails.
func extractErrorMessage(statusCode int, body []byte) string {
	var errBody anthropicErrorBody
	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error.Message != "" {
		return errBody.Error.Message
	}

	raw := string(body)
	if raw == "" {
		return fmt.Sprintf("API error with status %d", statusCode)
	}
	return raw
}

// isRetryableStatus returns true for status codes that indicate a transient error
// where retrying may succeed (429 rate limit, 500 server error, 503 overloaded).
func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}
