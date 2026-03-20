package cost

import "fmt"

// BudgetExhaustedError indicates a budget limit has been exceeded.
// This error is non-retryable — the pipeline must pause the requirement.
type BudgetExhaustedError struct {
	Scope   string  // "story", "requirement", or "daily"
	Current float64 // Current total cost
	Limit   float64 // Budget limit
	ID      string  // Story ID, Requirement ID, or date string
}

func (e *BudgetExhaustedError) Error() string {
	return fmt.Sprintf("budget exhausted: %s %s cost $%.2f exceeds limit $%.2f",
		e.Scope, e.ID, e.Current, e.Limit)
}

// IsBudgetExhausted checks if an error is a BudgetExhaustedError.
func IsBudgetExhausted(err error) bool {
	_, ok := err.(*BudgetExhaustedError)
	return ok
}
