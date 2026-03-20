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
	defaultOpenAIURL   = "https://api.openai.com/v1"
	defaultOpenAIModel = "gpt-4o"
)

// OpenAIClient implements Client using the OpenAI Chat Completions API.
// It supports any OpenAI-compatible endpoint (including Azure, local proxies, etc.)
// by allowing the base URL to be overridden.
type OpenAIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIClient creates a client configured with the given API key
// and default OpenAI API settings.
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:     apiKey,
		baseURL:    defaultOpenAIURL,
		httpClient: http.DefaultClient,
	}
}

// WithBaseURL returns a new copy with a custom base URL (for testing).
// The original client is not modified (immutability).
func (c *OpenAIClient) WithBaseURL(url string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:     c.apiKey,
		baseURL:    url,
		httpClient: c.httpClient,
	}
}

// Compile-time interface check.
var _ Client = (*OpenAIClient)(nil)

// openaiMessage represents a single message in the OpenAI Chat Completions format.
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiRequest is the request body for the OpenAI Chat Completions API.
type openaiRequest struct {
	Model     string          `json:"model"`
	Messages  []openaiMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

// openaiChoice represents a single choice in the OpenAI API response.
type openaiChoice struct {
	Message openaiMessage `json:"message"`
}

// openaiUsage represents token usage from the OpenAI API response.
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// openaiResponse is the success response from the OpenAI Chat Completions API.
type openaiResponse struct {
	Choices []openaiChoice `json:"choices"`
	Model   string         `json:"model"`
	Usage   openaiUsage    `json:"usage"`
}

// openaiErrorBody is the error response from the OpenAI API.
type openaiErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Complete sends a completion request to the OpenAI Chat Completions API and
// returns the parsed response. It extracts content from choices[0].message.content
// and maps prompt_tokens/completion_tokens to InputTokens/OutputTokens.
func (c *OpenAIClient) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
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
		return CompletionResponse{}, parseOpenAIErrorResponse(httpResp.StatusCode, body)
	}

	return parseOpenAISuccessResponse(body)
}

// buildHTTPRequest constructs the HTTP request for the OpenAI Chat Completions API.
func (c *OpenAIClient) buildHTTPRequest(ctx context.Context, req CompletionRequest) (*http.Request, error) {
	messages := buildOpenAIMessages(req)

	apiReq := openaiRequest{
		Model:     resolveOpenAIModel(req.Model),
		Messages:  messages,
		MaxTokens: resolveMaxTokens(req.MaxTokens),
	}

	payload, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

// buildOpenAIMessages converts the CompletionRequest into OpenAI message format.
// If a System prompt is provided, it is prepended as a system message.
func buildOpenAIMessages(req CompletionRequest) []openaiMessage {
	var messages []openaiMessage

	if req.System != "" {
		messages = append(messages, openaiMessage{
			Role:    string(RoleSystem),
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	return messages
}

// resolveOpenAIModel returns the given model if non-empty, otherwise the default.
func resolveOpenAIModel(model string) string {
	if model != "" {
		return model
	}
	return defaultOpenAIModel
}

// parseOpenAISuccessResponse extracts content and usage from a successful API response.
func parseOpenAISuccessResponse(body []byte) (CompletionResponse, error) {
	var resp openaiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	content := extractFirstChoice(resp.Choices)

	return CompletionResponse{
		Content:      content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}, nil
}

// extractFirstChoice returns the content from the first choice,
// or empty string if no choices are present.
func extractFirstChoice(choices []openaiChoice) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[0].Message.Content
}

// parseOpenAIErrorResponse creates an APIError from a non-2xx response.
// Status codes 429, 500, and 503 are marked as retryable.
func parseOpenAIErrorResponse(statusCode int, body []byte) *APIError {
	message := extractOpenAIErrorMessage(statusCode, body)
	retryable := isRetryableStatus(statusCode)

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Retryable:  retryable,
	}
}

// extractOpenAIErrorMessage attempts to parse the OpenAI error JSON format.
// Falls back to the raw body string if parsing fails.
func extractOpenAIErrorMessage(statusCode int, body []byte) string {
	var errBody openaiErrorBody
	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error.Message != "" {
		return errBody.Error.Message
	}

	raw := string(body)
	if raw == "" {
		return fmt.Sprintf("API error with status %d", statusCode)
	}
	return raw
}
