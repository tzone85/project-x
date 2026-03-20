package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultTimeout    = 120 * time.Second
	defaultMaxRetries = 2
	defaultRetryDelay = 1 * time.Second
	defaultMaxTokens  = 4096
	anthropicVersion  = "2023-06-01"
)

// AnthropicConfig holds configuration for the Anthropic API client.
type AnthropicConfig struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Pricing    PricingTable
	Logger     *slog.Logger
}

// anthropicRequest is the request body for the Anthropic Messages API.
type anthropicRequest struct {
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	Messages    []anthropicMsg    `json:"messages"`
	System      string            `json:"system,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	StopSeqs    []string          `json:"stop_sequences,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response from the Anthropic Messages API.
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// anthropicErrorResponse is the error response from the Anthropic API.
type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// AnthropicClient implements Client using the Anthropic Messages API.
type AnthropicClient struct {
	config     AnthropicConfig
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAnthropicClient creates a new Anthropic API client with the given config.
func NewAnthropicClient(cfg AnthropicConfig) *AnthropicClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = defaultRetryDelay
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &AnthropicClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: cfg.Logger.With("component", "anthropic_client"),
	}
}

// Complete sends a completion request to the Anthropic Messages API.
func (c *AnthropicClient) Complete(ctx context.Context, prompt string, opts CompletionOptions) (CompletionResponse, error) {
	model := stripProvider(opts.Model)
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []anthropicMsg{
			{Role: "user", Content: prompt},
		},
		System:   opts.SystemMsg,
		StopSeqs: opts.StopSeqs,
	}
	if opts.Temperature != 0 {
		reqBody.Temperature = &opts.Temperature
	}

	c.logger.Debug("sending completion request",
		"model", model,
		"max_tokens", maxTokens,
		"prompt_len", len(prompt),
	)

	apiResp, err := c.doWithRetry(ctx, reqBody)
	if err != nil {
		return CompletionResponse{}, err
	}

	content := extractContent(apiResp)
	finishReason := mapFinishReason(apiResp.StopReason)

	cost, err := c.config.Pricing.ComputeCost(
		opts.Model,
		apiResp.Usage.InputTokens,
		apiResp.Usage.OutputTokens,
	)
	if err != nil {
		// Log but don't fail — pricing might be missing for the model
		c.logger.Warn("could not compute cost", "model", opts.Model, "error", err)
		cost = 0
	}

	c.logger.Debug("completion response",
		"model", apiResp.Model,
		"input_tokens", apiResp.Usage.InputTokens,
		"output_tokens", apiResp.Usage.OutputTokens,
		"cost_usd", cost,
		"finish_reason", finishReason,
	)

	return CompletionResponse{
		Content:      content,
		InputTokens:  apiResp.Usage.InputTokens,
		OutputTokens: apiResp.Usage.OutputTokens,
		CostUSD:      cost,
		Model:        apiResp.Model,
		FinishReason: finishReason,
	}, nil
}

func (c *AnthropicClient) doWithRetry(ctx context.Context, reqBody anthropicRequest) (anthropicResponse, error) {
	var lastErr error

	maxAttempts := 1 + c.config.MaxRetries
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return anthropicResponse{}, fmt.Errorf("request cancelled during retry: %w", ctx.Err())
			case <-time.After(c.config.RetryDelay):
			}
		}

		resp, statusCode, err := c.doRequest(ctx, reqBody)
		if err != nil {
			lastErr = err
			// Network errors are retryable
			if ctx.Err() != nil {
				return anthropicResponse{}, fmt.Errorf("request cancelled: %w", ctx.Err())
			}
			c.logger.Warn("request failed, retrying",
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			return resp, nil
		}

		// Don't retry client errors (4xx) except rate limits (429)
		if statusCode >= 400 && statusCode < 500 && statusCode != http.StatusTooManyRequests {
			return anthropicResponse{}, fmt.Errorf("API error (status %d): non-retryable", statusCode)
		}

		lastErr = fmt.Errorf("API error (status %d)", statusCode)
		c.logger.Warn("retryable API error",
			"attempt", attempt+1,
			"status", statusCode,
		)
	}

	return anthropicResponse{}, fmt.Errorf("request failed after %d attempts: %w", maxAttempts, lastErr)
}

func (c *AnthropicClient) doRequest(ctx context.Context, reqBody anthropicRequest) (anthropicResponse, int, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return anthropicResponse{}, 0, fmt.Errorf("marshaling request: %w", err)
	}

	url := strings.TrimRight(c.config.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return anthropicResponse{}, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.config.APIKey)
	req.Header.Set("Anthropic-Version", anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return anthropicResponse{}, 0, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return anthropicResponse{}, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr anthropicErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			c.logger.Error("API error",
				"status", resp.StatusCode,
				"type", apiErr.Error.Type,
				"message", apiErr.Error.Message,
			)
		}
		return anthropicResponse{}, resp.StatusCode, nil
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return anthropicResponse{}, resp.StatusCode, fmt.Errorf("decoding response: %w", err)
	}

	return apiResp, resp.StatusCode, nil
}

// stripProvider removes the "provider/" prefix from a model string.
// e.g., "anthropic/claude-sonnet-4-20250514" → "claude-sonnet-4-20250514"
func stripProvider(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

func extractContent(resp anthropicResponse) string {
	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}

func mapFinishReason(reason string) FinishReason {
	switch reason {
	case "end_turn", "stop_sequence":
		return FinishReasonStop
	case "max_tokens":
		return FinishReasonMaxTokens
	default:
		return FinishReasonError
	}
}
