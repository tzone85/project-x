package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

var testStory = StoryContext{
	StoryID:       "story-1",
	RequirementID: "req-1",
	Title:         "Test Story",
	AgentID:       "agent-1",
	Wave:          1,
}

func TestRunAllStagesPass(t *testing.T) {
	stages := []Stage{
		newMockStage("review", Passed, nil),
		newMockStage("qa", Passed, nil),
		newMockStage("merge", Passed, nil),
	}
	emitter := newMockEmitter()
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), emitter, slog.Default())

	result := r.Run(context.Background(), testStory)

	if !result.Completed {
		t.Errorf("expected completed, got %+v", result)
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}
	if emitter.StartedCount() != 3 {
		t.Errorf("expected 3 started events, got %d", emitter.StartedCount())
	}
	if emitter.UpdatedCount() != 3 {
		t.Errorf("expected 3 updated events, got %d", emitter.UpdatedCount())
	}
}

func TestRunStageFailsAndRetries(t *testing.T) {
	// Review stage fails once then passes (max_retries=2, so this should succeed)
	reviewStage := newFailThenPassStage("review", 1, errors.New("review issue"))
	stages := []Stage{reviewStage}
	emitter := newMockEmitter()
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), emitter, slog.Default())

	result := r.Run(context.Background(), testStory)

	if !result.Completed {
		t.Errorf("expected completed after retry, got %+v", result)
	}
	if reviewStage.CallCount() != 2 {
		t.Errorf("expected 2 calls (1 fail + 1 pass), got %d", reviewStage.CallCount())
	}
}

func TestRunStageExhaustsRetries(t *testing.T) {
	// QA stage always fails (max_retries=3 → 4 attempts total)
	qaStage := newMockStage("qa", Failed, errors.New("qa failed"))
	stages := []Stage{qaStage}
	emitter := newMockEmitter()
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), emitter, slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.Completed {
		t.Error("expected incomplete after exhaustion")
	}
	if result.ExhaustStage != "qa" {
		t.Errorf("expected exhaust stage 'qa', got %q", result.ExhaustStage)
	}
	if qaStage.CallCount() != 4 {
		t.Errorf("expected 4 calls (1 + 3 retries), got %d", qaStage.CallCount())
	}
}

func TestRunStageFatal(t *testing.T) {
	fatalStage := newMockStage("rebase", Fatal, errors.New("cannot rebase"))
	stages := []Stage{
		newMockStage("review", Passed, nil),
		fatalStage,
	}
	emitter := newMockEmitter()
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), emitter, slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.Completed {
		t.Error("expected incomplete after fatal")
	}
	if result.FatalStage != "rebase" {
		t.Errorf("expected fatal stage 'rebase', got %q", result.FatalStage)
	}
	// Fatal should not retry — only 1 call
	if fatalStage.CallCount() != 1 {
		t.Errorf("expected 1 call for fatal stage, got %d", fatalStage.CallCount())
	}
}

func TestRunBudgetExhausted(t *testing.T) {
	budget := newMockBudget()
	budget.SetError(errors.New("budget exhausted"))
	stages := []Stage{newMockStage("review", Passed, nil)}
	r := NewRunner(stages, DefaultStageConfigs(), budget, newMockEmitter(), slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.Completed {
		t.Error("expected incomplete when budget exhausted")
	}
	if result.FailedStage != "review" {
		t.Errorf("expected failed stage 'review', got %q", result.FailedStage)
	}
}

func TestRunContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before running

	stages := []Stage{newMockStage("review", Passed, nil)}
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), newMockEmitter(), slog.Default())

	result := r.Run(ctx, testStory)

	if result.Completed {
		t.Error("expected incomplete when context cancelled")
	}
	if result.Err == nil {
		t.Error("expected error from context cancellation")
	}
}

func TestRunEmptyStages(t *testing.T) {
	r := NewRunner(nil, DefaultStageConfigs(), newMockBudget(), newMockEmitter(), slog.Default())
	result := r.Run(context.Background(), testStory)

	if !result.Completed {
		t.Error("expected completed with empty stages")
	}
}

func TestRunNilLogger(t *testing.T) {
	r := NewRunner(nil, nil, newMockBudget(), newMockEmitter(), nil)
	if r == nil {
		t.Fatal("expected non-nil runner with nil logger")
	}
}

func TestRunDefaultConfigs(t *testing.T) {
	r := NewRunner(nil, nil, newMockBudget(), newMockEmitter(), slog.Default())
	// Should use defaults when nil config passed
	if r.configs == nil {
		t.Error("expected default configs to be set")
	}
}

func TestRunUnknownStageUsesDefaultConfig(t *testing.T) {
	// A stage with a name not in the config map gets zero retries
	unknownStage := newMockStage("custom_stage", Failed, errors.New("failed"))
	stages := []Stage{unknownStage}
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), newMockEmitter(), slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.Completed {
		t.Error("expected incomplete")
	}
	// Zero retries = 1 attempt total
	if unknownStage.CallCount() != 1 {
		t.Errorf("expected 1 call for unknown stage (0 retries), got %d", unknownStage.CallCount())
	}
}

func TestRunMultipleStagesFirstFails(t *testing.T) {
	secondStage := newMockStage("qa", Passed, nil)
	stages := []Stage{
		newMockStage("review", Fatal, errors.New("fatal")),
		secondStage,
	}
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), newMockEmitter(), slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.Completed {
		t.Error("expected incomplete after first stage fatal")
	}
	if secondStage.CallCount() != 0 {
		t.Error("second stage should not have been called after first stage fatal")
	}
}

func TestRunEscalatePolicy(t *testing.T) {
	// Review has escalate policy; exhaust retries
	reviewStage := newMockStage("review", Failed, errors.New("review fail"))
	stages := []Stage{reviewStage}
	configs := map[string]StageConfig{
		"review": {MaxRetries: 1, OnExhaust: PolicyEscalate},
	}
	r := NewRunner(stages, configs, newMockBudget(), newMockEmitter(), slog.Default())

	result := r.Run(context.Background(), testStory)

	if result.ExhaustStage != "review" {
		t.Errorf("expected exhaust stage 'review', got %q", result.ExhaustStage)
	}
	// MaxRetries=1 → 2 attempts
	if reviewStage.CallCount() != 2 {
		t.Errorf("expected 2 calls, got %d", reviewStage.CallCount())
	}
}

func TestStageResultString(t *testing.T) {
	tests := []struct {
		result StageResult
		want   string
	}{
		{Passed, "passed"},
		{Failed, "failed"},
		{Fatal, "fatal"},
		{StageResult(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.result.String(); got != tt.want {
			t.Errorf("StageResult(%d).String() = %q, want %q", tt.result, got, tt.want)
		}
	}
}

func TestDefaultStageConfigs(t *testing.T) {
	configs := DefaultStageConfigs()
	if len(configs) != 4 {
		t.Errorf("expected 4 default configs, got %d", len(configs))
	}
	if configs["review"].OnExhaust != PolicyEscalate {
		t.Error("review should have escalate policy")
	}
	if configs["qa"].MaxRetries != 3 {
		t.Errorf("qa should have 3 retries, got %d", configs["qa"].MaxRetries)
	}
}

func TestRunEventEmissionOrder(t *testing.T) {
	stages := []Stage{
		newMockStage("review", Passed, nil),
		newMockStage("qa", Passed, nil),
	}
	emitter := newMockEmitter()
	r := NewRunner(stages, DefaultStageConfigs(), newMockBudget(), emitter, slog.Default())

	r.Run(context.Background(), testStory)

	if emitter.StartedCount() != 2 {
		t.Errorf("expected 2 started events, got %d", emitter.StartedCount())
	}
	if emitter.UpdatedCount() != 2 {
		t.Errorf("expected 2 updated events, got %d", emitter.UpdatedCount())
	}
	if emitter.LastUpdatedStatus() != "passed" {
		t.Errorf("expected last updated status 'passed', got %q", emitter.LastUpdatedStatus())
	}
}
