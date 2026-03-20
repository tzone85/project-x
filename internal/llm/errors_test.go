package llm

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{"retryable", APIError{429, "rate limited", true}, "API error (status 429): rate limited"},
		{"fatal", APIError{401, "unauthorized", false}, "API error (status 401): unauthorized"},
		{"server error", APIError{500, "internal error", true}, "API error (status 500): internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBudgetExhaustedError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  BudgetExhaustedError
		want string
	}{
		{"story", BudgetExhaustedError{"story", 5.50, 5.00}, "budget exhausted (story): $5.50 / $5.00"},
		{"daily", BudgetExhaustedError{"daily", 10.00, 10.00}, "budget exhausted (daily): $10.00 / $10.00"},
		{"requirement", BudgetExhaustedError{"requirement", 0.50, 25.00}, "budget exhausted (requirement): $0.50 / $25.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsFatalAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"retryable API error", &APIError{429, "rate limit", true}, false},
		{"non-retryable API error", &APIError{401, "auth", false}, true},
		{"budget exhausted", &BudgetExhaustedError{"story", 5.50, 5.00}, true},
		{"generic error", errors.New("connection timeout"), false},
		{"wrapped API error", errors.Join(errors.New("wrap"), &APIError{403, "forbidden", false}), true},
		{"wrapped retryable", errors.Join(errors.New("wrap"), &APIError{500, "server", true}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFatalAPIError(tt.err)
			if got != tt.want {
				t.Errorf("IsFatalAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}
