package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockClient is a test helper that returns predefined responses in sequence.
type mockClient struct {
	calls     int
	responses []mockResult
}

type mockResult struct {
	resp CompletionResponse
	err  error
}

func (m *mockClient) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	if m.calls >= len(m.responses) {
		return CompletionResponse{}, errors.New("mockClient: no more responses configured")
	}
	result := m.responses[m.calls]
	m.calls++
	return result.resp, result.err
}

func TestRetryClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = (*RetryClient)(nil)
}

func TestRetryClient_SucceedsOnFirstTry(t *testing.T) {
	mock := &mockClient{
		responses: []mockResult{
			{resp: CompletionResponse{Content: "ok", InputTokens: 10, OutputTokens: 5}},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	resp, err := rc.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected content 'ok', got %q", resp.Content)
	}
	if resp.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", resp.OutputTokens)
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryClient_RetriesOnRetryableError_SucceedsSecondTry(t *testing.T) {
	mock := &mockClient{
		responses: []mockResult{
			{err: &APIError{StatusCode: 429, Message: "rate limited", Retryable: true}},
			{resp: CompletionResponse{Content: "success", InputTokens: 20, OutputTokens: 10}},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	resp, err := rc.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "success" {
		t.Errorf("expected content 'success', got %q", resp.Content)
	}
	if resp.InputTokens != 20 {
		t.Errorf("expected 20 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 10 {
		t.Errorf("expected 10 output tokens, got %d", resp.OutputTokens)
	}
	if mock.calls != 2 {
		t.Errorf("expected 2 calls, got %d", mock.calls)
	}
}

func TestRetryClient_GivesUpAfterMaxAttempts(t *testing.T) {
	retryableErr := &APIError{StatusCode: 500, Message: "server error", Retryable: true}
	mock := &mockClient{
		responses: []mockResult{
			{err: retryableErr},
			{err: retryableErr},
			{err: retryableErr},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	_, err := rc.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error after max attempts, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls (max attempts), got %d", mock.calls)
	}
}

func TestRetryClient_DoesNotRetryNonRetryableError(t *testing.T) {
	mock := &mockClient{
		responses: []mockResult{
			{err: &APIError{StatusCode: 401, Message: "unauthorized", Retryable: false}},
			{resp: CompletionResponse{Content: "should not reach"}},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	_, err := rc.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("expected exactly 1 call (no retry), got %d", mock.calls)
	}
}

func TestRetryClient_DoesNotRetryBudgetExhaustedError(t *testing.T) {
	mock := &mockClient{
		responses: []mockResult{
			{err: &BudgetExhaustedError{BudgetType: "story", UsedUSD: 5.0, LimitUSD: 5.0}},
			{resp: CompletionResponse{Content: "should not reach"}},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	_, err := rc.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var budgetErr *BudgetExhaustedError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("expected BudgetExhaustedError, got %T: %v", err, err)
	}
	if mock.calls != 1 {
		t.Errorf("expected exactly 1 call (no retry), got %d", mock.calls)
	}
}

func TestRetryClient_RespectsContextCancellation(t *testing.T) {
	retryableErr := &APIError{StatusCode: 500, Message: "server error", Retryable: true}
	mock := &mockClient{
		responses: []mockResult{
			{err: retryableErr},
			{err: retryableErr},
			{err: retryableErr},
			{err: retryableErr},
			{err: retryableErr},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	rc := NewRetryClient(mock, 5, 50*time.Millisecond)

	// Cancel the context after a short delay — should interrupt between retries.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := rc.Complete(ctx, CompletionRequest{})
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
	// Should have made at least 1 call but fewer than max attempts.
	if mock.calls < 1 {
		t.Error("expected at least 1 call")
	}
	if mock.calls >= 5 {
		t.Errorf("expected fewer than 5 calls due to cancellation, got %d", mock.calls)
	}
}

func TestRetryClient_TokenCountsFromSuccessfulResponse(t *testing.T) {
	mock := &mockClient{
		responses: []mockResult{
			{err: &APIError{StatusCode: 503, Message: "overloaded", Retryable: true}},
			{err: &APIError{StatusCode: 429, Message: "rate limited", Retryable: true}},
			{resp: CompletionResponse{
				Content:      "final answer",
				Model:        "gpt-4o",
				InputTokens:  100,
				OutputTokens: 50,
			}},
		},
	}

	rc := NewRetryClient(mock, 5, time.Millisecond)
	resp, err := rc.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", resp.OutputTokens)
	}
	if resp.Content != "final answer" {
		t.Errorf("expected content 'final answer', got %q", resp.Content)
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls, got %d", mock.calls)
	}
}

func TestRetryClient_DoesNotRetryGenericErrors(t *testing.T) {
	// A non-APIError, non-BudgetExhaustedError should not be retried.
	mock := &mockClient{
		responses: []mockResult{
			{err: errors.New("network timeout")},
			{resp: CompletionResponse{Content: "should not reach"}},
		},
	}

	rc := NewRetryClient(mock, 3, time.Millisecond)
	_, err := rc.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.calls != 1 {
		t.Errorf("expected exactly 1 call (no retry for generic error), got %d", mock.calls)
	}
}
