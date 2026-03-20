package llm

import (
	"context"
	"errors"
	"time"
)

// RetryClient wraps any Client and automatically retries transient failures
// with exponential backoff. It only retries errors that are explicitly marked
// as retryable (APIError with Retryable=true). Non-retryable errors like auth
// failures, billing issues, and budget exhaustion are returned immediately.
type RetryClient struct {
	inner       Client
	maxAttempts int
	baseDelay   time.Duration
}

// NewRetryClient creates a retry wrapper around the given client.
// maxAttempts is the total number of attempts (including the first try).
// baseDelay is the initial delay before the first retry; subsequent retries
// double the delay (exponential backoff).
func NewRetryClient(inner Client, maxAttempts int, baseDelay time.Duration) *RetryClient {
	return &RetryClient{
		inner:       inner,
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
	}
}

// Compile-time interface check.
var _ Client = (*RetryClient)(nil)

// Complete delegates to the inner client, retrying on retryable errors with
// exponential backoff. It respects context cancellation between retries and
// never retries non-retryable errors or budget exhaustion.
func (c *RetryClient) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	var lastErr error

	for attempt := range c.maxAttempts {
		resp, err := c.inner.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			return CompletionResponse{}, err
		}

		// Don't sleep after the last attempt.
		if attempt == c.maxAttempts-1 {
			break
		}

		delay := c.backoffDelay(attempt)
		if err := sleepWithContext(ctx, delay); err != nil {
			return CompletionResponse{}, lastErr
		}
	}

	return CompletionResponse{}, lastErr
}

// isRetryableError returns true only for APIError with Retryable=true.
// BudgetExhaustedError, non-retryable APIError, and all other errors return false.
func isRetryableError(err error) bool {
	var budgetErr *BudgetExhaustedError
	if errors.As(err, &budgetErr) {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable
	}

	return false
}

// backoffDelay calculates the delay for the given attempt using exponential backoff.
// delay = baseDelay * 2^attempt
func (c *RetryClient) backoffDelay(attempt int) time.Duration {
	delay := c.baseDelay
	for range attempt {
		delay *= 2
	}
	return delay
}

// sleepWithContext blocks for the given duration or until the context is cancelled.
// Returns the context error if cancelled, nil otherwise.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
