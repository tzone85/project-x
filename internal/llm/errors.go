package llm

import (
	"errors"
	"fmt"
)

// APIError represents a structured error from an LLM API.
type APIError struct {
	StatusCode int
	Message    string
	Retryable  bool
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// BudgetExhaustedError indicates a cost budget has been exceeded.
type BudgetExhaustedError struct {
	BudgetType string  // "story", "requirement", "daily"
	UsedUSD    float64
	LimitUSD   float64
}

func (e *BudgetExhaustedError) Error() string {
	return fmt.Sprintf("budget exhausted (%s): $%.2f / $%.2f", e.BudgetType, e.UsedUSD, e.LimitUSD)
}

// IsFatalAPIError returns true if the error is a non-retryable API error
// (e.g., auth failure, billing exhaustion, permission denied).
// These errors should trigger requirement pausing, not retry.
func IsFatalAPIError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return !apiErr.Retryable
	}
	var budgetErr *BudgetExhaustedError
	if errors.As(err, &budgetErr) {
		return true
	}
	return false
}
