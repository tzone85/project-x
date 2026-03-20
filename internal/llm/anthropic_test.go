package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAnthropicClient(t *testing.T) {
	c := NewAnthropicClient("test-key")
	if c.apiKey != "test-key" {
		t.Errorf("expected api key 'test-key', got %q", c.apiKey)
	}
	if c.baseURL != defaultAnthropicURL {
		t.Errorf("expected default base URL, got %q", c.baseURL)
	}
	if c.apiVersion != defaultAnthropicVersion {
		t.Errorf("expected default API version, got %q", c.apiVersion)
	}
	if c.httpClient == nil {
		t.Error("expected non-nil http client")
	}
}

func TestAnthropicClient_WithBaseURL_Immutability(t *testing.T) {
	original := NewAnthropicClient("test-key")
	modified := original.WithBaseURL("http://localhost:1234")

	if original.baseURL != defaultAnthropicURL {
		t.Error("original should not be modified")
	}
	if modified.baseURL != "http://localhost:1234" {
		t.Errorf("copy should have new base URL, got %q", modified.baseURL)
	}
	if modified.apiKey != original.apiKey {
		t.Error("copy should preserve apiKey")
	}
	if modified.apiVersion != original.apiVersion {
		t.Error("copy should preserve apiVersion")
	}
}

func TestAnthropicClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = (*AnthropicClient)(nil)
}

func TestAnthropicClient_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != defaultAnthropicVersion {
			t.Errorf("expected anthropic-version header, got %q", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("expected content-type application/json, got %q", r.Header.Get("content-type"))
		}

		// Verify request body structure.
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["model"] != "claude-sonnet-4-20250514" {
			t.Errorf("expected model claude-sonnet-4-20250514, got %v", reqBody["model"])
		}
		if reqBody["system"] != "Be helpful." {
			t.Errorf("expected system 'Be helpful.', got %v", reqBody["system"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Hello from Claude!"},
			},
			"model": "claude-sonnet-4-20250514",
			"usage": map[string]int{
				"input_tokens":  150,
				"output_tokens": 42,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		System: "Be helpful.",
		Messages: []Message{
			{Role: RoleUser, Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Claude!" {
		t.Errorf("expected content 'Hello from Claude!', got %q", resp.Content)
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514', got %q", resp.Model)
	}
	if resp.InputTokens != 150 {
		t.Errorf("expected 150 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 42 {
		t.Errorf("expected 42 output tokens, got %d", resp.OutputTokens)
	}
}

func TestAnthropicClient_Complete_CustomModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["model"] != "claude-opus-4-20250514" {
			t.Errorf("expected custom model, got %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "response"},
			},
			"model": "claude-opus-4-20250514",
			"usage": map[string]int{"input_tokens": 10, "output_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "claude-opus-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "claude-opus-4-20250514" {
		t.Errorf("expected model 'claude-opus-4-20250514', got %q", resp.Model)
	}
}

func TestAnthropicClient_Complete_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "max_tokens must be a positive integer",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Retryable {
		t.Error("400 error should not be retryable")
	}
	if apiErr.Message == "" {
		t.Error("error message should not be empty")
	}
}

func TestAnthropicClient_Complete_AuthFailure401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("bad-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Retryable {
		t.Error("auth error should not be retryable")
	}
}

func TestAnthropicClient_Complete_RateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "rate_limit_error",
				"message": "rate limit exceeded",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
	if !apiErr.Retryable {
		t.Error("rate limit error should be retryable")
	}
}

func TestAnthropicClient_Complete_ServerError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "api_error",
				"message": "internal server error",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if !apiErr.Retryable {
		t.Error("server error should be retryable")
	}
}

func TestAnthropicClient_Complete_ServiceUnavailable503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		resp := map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "overloaded_error",
				"message": "service temporarily unavailable",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", apiErr.StatusCode)
	}
	if !apiErr.Retryable {
		t.Error("503 error should be retryable")
	}
}

func TestAnthropicClient_Complete_TokenExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "token test"},
			},
			"model": "claude-sonnet-4-20250514",
			"usage": map[string]int{
				"input_tokens":  1234,
				"output_tokens": 5678,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "count tokens"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.InputTokens != 1234 {
		t.Errorf("expected 1234 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 5678 {
		t.Errorf("expected 5678 output tokens, got %d", resp.OutputTokens)
	}
}

func TestAnthropicClient_Complete_EmptyContentArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{},
			"model":   "claude-sonnet-4-20250514",
			"usage":   map[string]int{"input_tokens": 10, "output_tokens": 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for empty content array, got %q", resp.Content)
	}
}

func TestAnthropicClient_Complete_MultipleContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "First block"},
				{"type": "text", "text": "Second block"},
			},
			"model": "claude-sonnet-4-20250514",
			"usage": map[string]int{"input_tokens": 10, "output_tokens": 20},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should extract first text block.
	if resp.Content != "First block" {
		t.Errorf("expected 'First block', got %q", resp.Content)
	}
}

func TestAnthropicClient_Complete_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler should not be reached if context is already cancelled.
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := client.Complete(ctx, CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestAnthropicClient_Complete_NonJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", apiErr.StatusCode)
	}
}

func TestAnthropicClient_Complete_RequestBody_MaxTokensDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Default max_tokens should be 4096.
		maxTokens, ok := reqBody["max_tokens"].(float64)
		if !ok {
			t.Fatal("max_tokens not found in request body")
		}
		if int(maxTokens) != 4096 {
			t.Errorf("expected default max_tokens 4096, got %v", maxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "ok"}},
			"model":   "claude-sonnet-4-20250514",
			"usage":   map[string]int{"input_tokens": 1, "output_tokens": 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnthropicClient_Complete_RequestBody_CustomMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		maxTokens, ok := reqBody["max_tokens"].(float64)
		if !ok {
			t.Fatal("max_tokens not found in request body")
		}
		if int(maxTokens) != 8192 {
			t.Errorf("expected custom max_tokens 8192, got %v", maxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "ok"}},
			"model":   "claude-sonnet-4-20250514",
			"usage":   map[string]int{"input_tokens": 1, "output_tokens": 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		MaxTokens: 8192,
		Messages:  []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
