package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOpenAIClient(t *testing.T) {
	c := NewOpenAIClient("sk-test-key")
	if c.apiKey != "sk-test-key" {
		t.Errorf("expected api key 'sk-test-key', got %q", c.apiKey)
	}
	if c.baseURL != defaultOpenAIURL {
		t.Errorf("expected default base URL, got %q", c.baseURL)
	}
	if c.httpClient == nil {
		t.Error("expected non-nil http client")
	}
}

func TestOpenAIClient_WithBaseURL_Immutability(t *testing.T) {
	original := NewOpenAIClient("sk-test-key")
	modified := original.WithBaseURL("http://localhost:9999")

	if original.baseURL != defaultOpenAIURL {
		t.Error("original should not be modified")
	}
	if modified.baseURL != "http://localhost:9999" {
		t.Errorf("copy should have new base URL, got %q", modified.baseURL)
	}
	if modified.apiKey != original.apiKey {
		t.Error("copy should preserve apiKey")
	}
}

func TestOpenAIClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = (*OpenAIClient)(nil)
}

func TestOpenAIClient_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-test-key" {
			t.Errorf("expected Authorization 'Bearer sk-test-key', got %q", authHeader)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected content-type application/json, got %q", r.Header.Get("Content-Type"))
		}

		// Verify request body structure.
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["model"] != "gpt-4o" {
			t.Errorf("expected model gpt-4o, got %v", reqBody["model"])
		}

		// System message should be prepended to messages array.
		messages, ok := reqBody["messages"].([]interface{})
		if !ok {
			t.Fatal("messages not found in request body")
		}
		if len(messages) != 2 {
			t.Fatalf("expected 2 messages (system + user), got %d", len(messages))
		}
		sysMsg := messages[0].(map[string]interface{})
		if sysMsg["role"] != "system" || sysMsg["content"] != "Be helpful." {
			t.Errorf("expected system message, got %v", sysMsg)
		}
		userMsg := messages[1].(map[string]interface{})
		if userMsg["role"] != "user" || userMsg["content"] != "Say hello" {
			t.Errorf("expected user message, got %v", userMsg)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from GPT!",
					},
				},
			},
			"model": "gpt-4o-2024-05-13",
			"usage": map[string]int{
				"prompt_tokens":     150,
				"completion_tokens": 42,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		System: "Be helpful.",
		Model:  "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from GPT!" {
		t.Errorf("expected content 'Hello from GPT!', got %q", resp.Content)
	}
	if resp.Model != "gpt-4o-2024-05-13" {
		t.Errorf("expected model 'gpt-4o-2024-05-13', got %q", resp.Model)
	}
	if resp.InputTokens != 150 {
		t.Errorf("expected 150 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 42 {
		t.Errorf("expected 42 output tokens, got %d", resp.OutputTokens)
	}
}

func TestOpenAIClient_Complete_TokenExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "token test"}},
			},
			"model": "gpt-4o",
			"usage": map[string]int{
				"prompt_tokens":     1234,
				"completion_tokens": 5678,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_ModelPassthrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["model"] != "gpt-4-turbo" {
			t.Errorf("expected custom model 'gpt-4-turbo', got %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "response"}},
			},
			"model": "gpt-4-turbo-2024-04-09",
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "gpt-4-turbo-2024-04-09" {
		t.Errorf("expected model 'gpt-4-turbo-2024-04-09', got %q", resp.Model)
	}
}

func TestOpenAIClient_Complete_NoSystemMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		messages, ok := reqBody["messages"].([]interface{})
		if !ok {
			t.Fatal("messages not found in request body")
		}
		// Without system, should only have user message.
		if len(messages) != 1 {
			t.Fatalf("expected 1 message (user only), got %d", len(messages))
		}
		userMsg := messages[0].(map[string]interface{})
		if userMsg["role"] != "user" {
			t.Errorf("expected user role, got %v", userMsg["role"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
			"model": "gpt-4o",
			"usage": map[string]int{"prompt_tokens": 5, "completion_tokens": 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIClient_Complete_AuthFailure401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "Incorrect API key provided",
				"type":    "invalid_request_error",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("bad-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_RateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "Rate limit reached",
				"type":    "rate_limit_error",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_ServerError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "The server had an error",
				"type":    "server_error",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_ServiceUnavailable503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "Service temporarily unavailable",
				"type":    "server_error",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_ContentExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "extracted content"}},
			},
			"model": "gpt-4o",
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "extracted content" {
		t.Errorf("expected 'extracted content', got %q", resp.Content)
	}
}

func TestOpenAIClient_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
			"model":   "gpt-4o",
			"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	resp, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for empty choices, got %q", resp.Content)
	}
}

func TestOpenAIClient_Complete_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := client.Complete(ctx, CompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestOpenAIClient_Complete_NonJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4o",
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

func TestOpenAIClient_Complete_DefaultModelAndMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// When no model specified, should use default.
		if reqBody["model"] != defaultOpenAIModel {
			t.Errorf("expected default model %q, got %v", defaultOpenAIModel, reqBody["model"])
		}

		// Default max_tokens should be applied.
		maxTokens, ok := reqBody["max_tokens"].(float64)
		if !ok {
			t.Fatal("max_tokens not found in request body")
		}
		if int(maxTokens) != defaultMaxTokens {
			t.Errorf("expected default max_tokens %d, got %v", defaultMaxTokens, maxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
			"model": defaultOpenAIModel,
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("sk-test-key").WithBaseURL(server.URL)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
