package cost

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/llm"
	"github.com/tzone85/project-x/internal/state"
)

// Breaker wraps an llm.Client with pre-call budget checks.
// It checks story, requirement, and daily budgets before each LLM call.
// On breach: returns BudgetExhaustedError (non-retryable).
// At warning threshold: emits EventBudgetWarning.
type Breaker struct {
	inner   llm.Client
	ledger  *Ledger
	budget  config.BudgetConfig
	pricing config.PricingMap
	emitter EventEmitter
	logger  *slog.Logger
}

// BreakerOption configures the Breaker.
type BreakerOption func(*Breaker)

// WithBreakerLogger sets a custom logger.
func WithBreakerLogger(logger *slog.Logger) BreakerOption {
	return func(b *Breaker) {
		b.logger = logger
	}
}

// NewBreaker creates a circuit breaker that wraps the given LLM client.
func NewBreaker(
	inner llm.Client,
	ledger *Ledger,
	budget config.BudgetConfig,
	pricing config.PricingMap,
	emitter EventEmitter,
	opts ...BreakerOption,
) *Breaker {
	b := &Breaker{
		inner:   inner,
		ledger:  ledger,
		budget:  budget,
		pricing: pricing,
		emitter: emitter,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Complete performs budget checks, then delegates to the wrapped client.
// After completion, it records the usage in the ledger.
func (b *Breaker) Complete(ctx context.Context, prompt string, opts llm.CompletionOptions) (llm.CompletionResponse, error) {
	// Pre-call budget checks
	if err := b.checkBudgets(opts.StoryID, opts.ReqID); err != nil {
		return llm.CompletionResponse{}, err
	}

	// Delegate to inner client
	resp, err := b.inner.Complete(ctx, prompt, opts)
	if err != nil {
		return resp, err
	}

	// Compute cost if not already set
	if resp.CostUSD == 0 && resp.Model != "" {
		resp.CostUSD = llm.ComputeCost(resp.Model, resp.InputTokens, resp.OutputTokens, b.pricing)
	}

	// Record usage
	recordErr := b.ledger.RecordUsage(UsageRecord{
		StoryID:      opts.StoryID,
		ReqID:        opts.ReqID,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
		Stage:        opts.Stage,
	})
	if recordErr != nil {
		b.logger.Error("failed to record usage",
			"story_id", opts.StoryID,
			"error", recordErr,
		)
	}

	return resp, nil
}

func (b *Breaker) checkBudgets(storyID, reqID string) error {
	// Check story budget
	if storyID != "" && b.budget.MaxCostPerStoryUSD > 0 {
		total, err := b.ledger.GetStoryTotal(storyID)
		if err != nil {
			return fmt.Errorf("check story budget: %w", err)
		}
		if err := b.checkLimit("story", storyID, total, b.budget.MaxCostPerStoryUSD); err != nil {
			return err
		}
	}

	// Check requirement budget
	if reqID != "" && b.budget.MaxCostPerRequirementUSD > 0 {
		total, err := b.ledger.GetRequirementTotal(reqID)
		if err != nil {
			return fmt.Errorf("check requirement budget: %w", err)
		}
		if err := b.checkLimit("requirement", reqID, total, b.budget.MaxCostPerRequirementUSD); err != nil {
			return err
		}
	}

	// Check daily budget
	if b.budget.MaxCostPerDayUSD > 0 {
		total, err := b.ledger.GetDailyTotal(time.Now().UTC())
		if err != nil {
			return fmt.Errorf("check daily budget: %w", err)
		}
		dateStr := time.Now().UTC().Format("2006-01-02")
		if err := b.checkLimit("daily", dateStr, total, b.budget.MaxCostPerDayUSD); err != nil {
			return err
		}
	}

	return nil
}

func (b *Breaker) checkLimit(scope, id string, current, limit float64) error {
	if current >= limit && b.budget.HardStop {
		return &BudgetExhaustedError{
			Scope:   scope,
			Current: current,
			Limit:   limit,
			ID:      id,
		}
	}

	// Check warning threshold
	warningThreshold := limit * float64(b.budget.WarningThresholdPct) / 100.0
	if current >= warningThreshold {
		b.emitWarning(scope, id, current, limit)
	}

	return nil
}

func (b *Breaker) emitWarning(scope, id string, current, limit float64) {
	percentage := (current / limit) * 100.0

	payload := state.BudgetWarningPayload{
		CurrentCost: current,
		BudgetLimit: limit,
		Percentage:  percentage,
	}

	switch scope {
	case "story":
		payload.StoryID = id
	case "requirement":
		payload.RequirementID = id
	}

	event, err := state.NewEvent(state.EventBudgetWarning, payload)
	if err != nil {
		b.logger.Error("failed to create budget warning event", "error", err)
		return
	}

	if err := b.emitter.Emit(event); err != nil {
		b.logger.Error("failed to emit budget warning",
			"scope", scope,
			"id", id,
			"error", err,
		)
	}

	b.logger.Warn("budget warning",
		"scope", scope,
		"id", id,
		"current", current,
		"limit", limit,
		"percentage", percentage,
	)
}
