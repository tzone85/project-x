package llm

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// anthropicAPIResponse mirrors the Anthropic Messages API response format.
type anthropicAPIResponse struct {
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

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func successHandler(model string, content string, inputTokens, outputTokens int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicAPIResponse{
			ID:    "msg_test_123",
			Type:  "message",
			Role:  "assistant",
			Model: model,
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: content},
			},
			StopReason: "end_turn",
		}
		resp.Usage.InputTokens = inputTokens
		resp.Usage.OutputTokens = outputTokens

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestAnthropicClientComplete(t *testing.T) {
	srv := newTestServer(t, successHandler(
		"claude-sonnet-4-20250514", "Hello, world!", 10, 5,
	))

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	resp, err := client.Complete(context.Background(), "Say hello", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.InputTokens != 10 {
		t.Errorf("input tokens = %d, want 10", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("output tokens = %d, want 5", resp.OutputTokens)
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", resp.Model, "claude-sonnet-4-20250514")
	}
	if resp.FinishReason != FinishReasonStop {
		t.Errorf("finish reason = %q, want %q", resp.FinishReason, FinishReasonStop)
	}

	// Verify cost: (10/1M * 3.00) + (5/1M * 15.00) = 0.00003 + 0.000075 = 0.000105
	wantCost := 0.000105
	if math.Abs(resp.CostUSD-wantCost) > 1e-9 {
		t.Errorf("cost = %f, want %f", resp.CostUSD, wantCost)
	}
}

func TestAnthropicClientSendsCorrectHeaders(t *testing.T) {
	var capturedHeaders http.Header
	var capturedBody map[string]any

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		json.NewDecoder(r.Body).Decode(&capturedBody)
		successHandler("claude-sonnet-4-20250514", "ok", 1, 1)(w, r)
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "sk-test-123",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 50,
		SystemMsg: "You are helpful.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := capturedHeaders.Get("X-Api-Key"); got != "sk-test-123" {
		t.Errorf("X-Api-Key = %q, want %q", got, "sk-test-123")
	}
	if got := capturedHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := capturedHeaders.Get("Anthropic-Version"); got != "2023-06-01" {
		t.Errorf("Anthropic-Version = %q, want %q", got, "2023-06-01")
	}

	// Verify request body includes system message
	if system, ok := capturedBody["system"].(string); !ok || system != "You are helpful." {
		t.Errorf("system = %v, want %q", capturedBody["system"], "You are helpful.")
	}
}

func TestAnthropicClientContextCancellation(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
			successHandler("claude-sonnet-4-20250514", "slow", 1, 1)(w, r)
		}
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Complete(ctx, "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error = %q, want context canceled", err.Error())
	}
}

func TestAnthropicClientTimeout(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		successHandler("claude-sonnet-4-20250514", "slow", 1, 1)(w, r)
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Timeout: 50 * time.Millisecond,
		Pricing: DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestAnthropicClientRetry(t *testing.T) {
	var attempts atomic.Int32

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}
		successHandler("claude-sonnet-4-20250514", "retry success", 5, 3)(w, r)
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:     "test-key",
		BaseURL:    srv.URL,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
		Pricing:    DefaultPricingTable(),
	})

	resp, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if resp.Content != "retry success" {
		t.Errorf("content = %q, want %q", resp.Content, "retry success")
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestAnthropicClientRetryExhausted(t *testing.T) {
	var attempts atomic.Int32

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:     "test-key",
		BaseURL:    srv.URL,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
		Pricing:    DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error after exhausted retries, got nil")
	}
	// 1 initial + 2 retries = 3 total
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestAnthropicClientAPIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens must be positive"}}`))
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: -1,
	})
	if err == nil {
		t.Fatal("expected error for bad request, got nil")
	}
}

func TestAnthropicClientMaxTokensFinishReason(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicAPIResponse{
			ID:    "msg_test",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-20250514",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "truncated"},
			},
			StopReason: "max_tokens",
		}
		resp.Usage.InputTokens = 10
		resp.Usage.OutputTokens = 100

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	resp, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FinishReason != FinishReasonMaxTokens {
		t.Errorf("finish reason = %q, want %q", resp.FinishReason, FinishReasonMaxTokens)
	}
}

func TestAnthropicClientDefaultMaxTokens(t *testing.T) {
	var capturedBody map[string]any

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		successHandler("claude-sonnet-4-20250514", "ok", 1, 1)(w, r)
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Pricing: DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model: "anthropic/claude-sonnet-4-20250514",
		// MaxTokens not set
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	maxTokens, ok := capturedBody["max_tokens"].(float64)
	if !ok || maxTokens != 4096 {
		t.Errorf("max_tokens = %v, want 4096", capturedBody["max_tokens"])
	}
}

func TestAnthropicClientNoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid api key"}}`))
	})

	client := NewAnthropicClient(AnthropicConfig{
		APIKey:     "bad-key",
		BaseURL:    srv.URL,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
		Pricing:    DefaultPricingTable(),
	})

	_, err := client.Complete(context.Background(), "test", CompletionOptions{
		Model:     "anthropic/claude-sonnet-4-20250514",
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error for auth failure, got nil")
	}
	// Should not retry 4xx errors (except 429)
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retries for 4xx)", got)
	}
}
