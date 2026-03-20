package cost

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/llm"
	"github.com/tzone85/project-x/internal/state"

	_ "github.com/mattn/go-sqlite3"
)

// --- Test helpers ---

type directEmitter struct {
	store *state.ProjectionStore
}

func (e *directEmitter) Emit(event state.Event) error {
	return e.store.ApplyEvent(event)
}

func newTestStore(t *testing.T) (*state.ProjectionStore, *directEmitter) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ps, err := state.NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}
	t.Cleanup(func() { ps.Close() })
	emitter := &directEmitter{store: ps}
	return ps, emitter
}

type mockLLMClient struct {
	response llm.CompletionResponse
	err      error
	calls    int
}

func (m *mockLLMClient) Complete(_ context.Context, _ string, _ llm.CompletionOptions) (llm.CompletionResponse, error) {
	m.calls++
	return m.response, m.err
}

// --- Ledger tests ---

func TestLedger_RecordAndQuery(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	err := ledger.RecordUsage(UsageRecord{
		StoryID:      "story-1",
		ReqID:        "req-1",
		AgentID:      "agent-1",
		Model:        "claude-sonnet",
		InputTokens:  1000,
		OutputTokens: 500,
		CostUSD:      0.105,
		Stage:        "review",
	})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	// Query story total
	total, err := ledger.GetStoryTotal("story-1")
	if err != nil {
		t.Fatalf("GetStoryTotal: %v", err)
	}
	if total != 0.105 {
		t.Errorf("expected 0.105, got %f", total)
	}

	// Query requirement total
	reqTotal, err := ledger.GetRequirementTotal("req-1")
	if err != nil {
		t.Fatalf("GetRequirementTotal: %v", err)
	}
	if reqTotal != 0.105 {
		t.Errorf("expected 0.105, got %f", reqTotal)
	}

	// Query daily total
	dailyTotal, err := ledger.GetDailyTotal(time.Now().UTC())
	if err != nil {
		t.Fatalf("GetDailyTotal: %v", err)
	}
	if dailyTotal != 0.105 {
		t.Errorf("expected 0.105, got %f", dailyTotal)
	}
}

func TestLedger_MultiplRecords(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	for i := 0; i < 3; i++ {
		ledger.RecordUsage(UsageRecord{
			StoryID: "story-1", ReqID: "req-1",
			Model: "claude-sonnet", InputTokens: 1000, OutputTokens: 500,
			CostUSD: 1.00, Stage: "coding",
		})
	}

	total, _ := ledger.GetStoryTotal("story-1")
	if total != 3.00 {
		t.Errorf("expected 3.00, got %f", total)
	}
}

// --- BudgetExhaustedError tests ---

func TestBudgetExhaustedError(t *testing.T) {
	err := &BudgetExhaustedError{
		Scope:   "story",
		Current: 2.50,
		Limit:   2.00,
		ID:      "story-1",
	}

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	if !IsBudgetExhausted(err) {
		t.Error("IsBudgetExhausted should return true")
	}

	if IsBudgetExhausted(context.DeadlineExceeded) {
		t.Error("IsBudgetExhausted should return false for non-budget errors")
	}
}

// --- Breaker tests ---

func TestBreaker_PassesThrough(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	inner := &mockLLMClient{
		response: llm.CompletionResponse{
			Content:      "hello",
			InputTokens:  100,
			OutputTokens: 50,
			Model:        "anthropic/claude-sonnet-4-20250514",
		},
	}

	pricing := config.PricingMap{
		"anthropic/claude-sonnet-4-20250514": {InputPer1M: 3.00, OutputPer1M: 15.00},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:       2.00,
		MaxCostPerRequirementUSD: 20.00,
		MaxCostPerDayUSD:         50.00,
		WarningThresholdPct:      80,
		HardStop:                 true,
	}

	breaker := NewBreaker(inner, ledger, budget, pricing, emitter)

	resp, err := breaker.Complete(context.Background(), "test prompt", llm.CompletionOptions{
		StoryID: "story-1",
		ReqID:   "req-1",
		Stage:   "review",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("expected content 'hello', got %s", resp.Content)
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 call to inner, got %d", inner.calls)
	}

	// Verify usage was recorded
	total, _ := ledger.GetStoryTotal("story-1")
	if total == 0 {
		t.Error("expected non-zero cost recorded")
	}
}

func TestBreaker_BlocksOnStoryBudgetExhausted(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	// Pre-fill usage to exceed budget
	for i := 0; i < 3; i++ {
		ledger.RecordUsage(UsageRecord{
			StoryID: "story-1", ReqID: "req-1",
			Model: "claude-sonnet", CostUSD: 1.00, Stage: "coding",
		})
	}

	inner := &mockLLMClient{
		response: llm.CompletionResponse{Content: "should not reach"},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:       2.00,
		MaxCostPerRequirementUSD: 20.00,
		MaxCostPerDayUSD:         50.00,
		WarningThresholdPct:      80,
		HardStop:                 true,
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-1",
		ReqID:   "req-1",
	})
	if err == nil {
		t.Fatal("expected BudgetExhaustedError")
	}
	if !IsBudgetExhausted(err) {
		t.Errorf("expected BudgetExhaustedError, got %T: %v", err, err)
	}
	if inner.calls != 0 {
		t.Error("inner client should not have been called")
	}
}

func TestBreaker_BlocksOnRequirementBudget(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	// Exceed requirement budget
	for i := 0; i < 25; i++ {
		ledger.RecordUsage(UsageRecord{
			StoryID: "story-" + string(rune('a'+i%5)), ReqID: "req-1",
			Model: "claude-sonnet", CostUSD: 1.00, Stage: "coding",
		})
	}

	inner := &mockLLMClient{}
	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:       100.00, // High story budget
		MaxCostPerRequirementUSD: 20.00,
		MaxCostPerDayUSD:         100.00,
		WarningThresholdPct:      80,
		HardStop:                 true,
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-x",
		ReqID:   "req-1",
	})
	if !IsBudgetExhausted(err) {
		t.Errorf("expected BudgetExhaustedError for requirement, got %v", err)
	}
}

func TestBreaker_BlocksOnDailyBudget(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	// Exceed daily budget
	for i := 0; i < 60; i++ {
		ledger.RecordUsage(UsageRecord{
			StoryID: "story-x", ReqID: "req-x",
			Model: "claude-sonnet", CostUSD: 1.00, Stage: "coding",
		})
	}

	inner := &mockLLMClient{}
	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:       100.00,
		MaxCostPerRequirementUSD: 100.00,
		MaxCostPerDayUSD:         50.00,
		WarningThresholdPct:      80,
		HardStop:                 true,
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-y",
		ReqID:   "req-y",
	})
	if !IsBudgetExhausted(err) {
		t.Errorf("expected BudgetExhaustedError for daily, got %v", err)
	}
}

func TestBreaker_NoBlockWhenHardStopDisabled(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	// Exceed budget
	ledger.RecordUsage(UsageRecord{
		StoryID: "story-1", ReqID: "req-1",
		Model: "claude-sonnet", CostUSD: 3.00, Stage: "coding",
	})

	inner := &mockLLMClient{
		response: llm.CompletionResponse{Content: "ok", Model: "m"},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:  2.00,
		WarningThresholdPct: 80,
		HardStop:            false, // Disabled
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-1",
		ReqID:   "req-1",
	})
	if err != nil {
		t.Fatalf("should not block when hard_stop=false, got %v", err)
	}
	if inner.calls != 1 {
		t.Error("inner should have been called")
	}
}

func TestBreaker_EmitsWarningAtThreshold(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	// Add usage at 85% of budget
	ledger.RecordUsage(UsageRecord{
		StoryID: "story-1", ReqID: "req-1",
		Model: "claude-sonnet", CostUSD: 1.70, Stage: "coding",
	})

	inner := &mockLLMClient{
		response: llm.CompletionResponse{Content: "ok", Model: "m"},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:  2.00,
		WarningThresholdPct: 80,
		HardStop:            false,
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-1",
		ReqID:   "req-1",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Check that a budget warning event was stored
	events, _ := ps.ListEvents(state.DefaultPageParams())
	foundWarning := false
	for _, ev := range events {
		if ev.Type == state.EventBudgetWarning {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected EventBudgetWarning to be emitted")
	}
}

func TestBreaker_ComputesCostFromPricing(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	inner := &mockLLMClient{
		response: llm.CompletionResponse{
			Content:      "result",
			InputTokens:  1000,
			OutputTokens: 500,
			Model:        "anthropic/claude-sonnet-4-20250514",
			CostUSD:      0, // Not set by inner — breaker should compute
		},
	}

	pricing := config.PricingMap{
		"anthropic/claude-sonnet-4-20250514": {InputPer1M: 3.00, OutputPer1M: 15.00},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:  100.00,
		WarningThresholdPct: 80,
		HardStop:            true,
	}

	breaker := NewBreaker(inner, ledger, budget, pricing, emitter)

	resp, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{
		StoryID: "story-1",
		ReqID:   "req-1",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	expectedCost := (1000.0/1_000_000)*3.00 + (500.0/1_000_000)*15.00
	if resp.CostUSD != expectedCost {
		t.Errorf("expected cost %f, got %f", expectedCost, resp.CostUSD)
	}
}

func TestBreaker_WithLogger(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)
	inner := &mockLLMClient{}
	budget := config.BudgetConfig{}

	// Should not panic
	_ = NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter, WithBreakerLogger(nil))
}

func TestBreaker_SkipsCheckForEmptyIDs(t *testing.T) {
	ps, emitter := newTestStore(t)
	ledger := NewLedger(ps, emitter)

	inner := &mockLLMClient{
		response: llm.CompletionResponse{Content: "ok", Model: "m"},
	}

	budget := config.BudgetConfig{
		MaxCostPerStoryUSD:       2.00,
		MaxCostPerRequirementUSD: 20.00,
		MaxCostPerDayUSD:         0, // Disabled
		WarningThresholdPct:      80,
		HardStop:                 true,
	}

	breaker := NewBreaker(inner, ledger, budget, config.PricingMap{}, emitter)

	// Empty story and req IDs — should skip budget checks
	_, err := breaker.Complete(context.Background(), "test", llm.CompletionOptions{})
	if err != nil {
		t.Fatalf("should pass with empty IDs: %v", err)
	}
}
