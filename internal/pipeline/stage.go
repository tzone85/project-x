package pipeline

import (
	"context"
	"fmt"
)

// StageResult represents the outcome of a pipeline stage execution.
type StageResult int

const (
	// Passed means the stage completed successfully — advance to next stage.
	Passed StageResult = iota
	// Failed means the stage failed but may be retried.
	Failed
	// Fatal means a non-recoverable error — pause the requirement immediately.
	Fatal
)

func (r StageResult) String() string {
	switch r {
	case Passed:
		return "passed"
	case Failed:
		return "failed"
	case Fatal:
		return "fatal"
	default:
		return fmt.Sprintf("unknown(%d)", int(r))
	}
}

// StoryContext holds the context for a story being processed through the pipeline.
type StoryContext struct {
	StoryID       string
	RequirementID string
	Title         string
	AgentID       string
	Wave          int
}

// Stage is the interface that all pipeline stages must implement.
// Each stage performs one step in the post-completion pipeline
// (e.g., auto-commit, diff check, review, QA, rebase, merge, cleanup).
type Stage interface {
	Name() string
	Execute(ctx context.Context, story StoryContext) (StageResult, error)
}

// ExhaustPolicy defines what happens when a stage exhausts its retry budget.
type ExhaustPolicy string

const (
	// PolicyEscalate re-assigns the story to a senior agent with a higher-capability model.
	PolicyEscalate ExhaustPolicy = "escalate"
	// PolicyPauseRequirement pauses the entire requirement.
	PolicyPauseRequirement ExhaustPolicy = "pause_requirement"
)

// StageConfig holds retry and exhaust policy for a single stage.
type StageConfig struct {
	MaxRetries int
	OnExhaust  ExhaustPolicy
}

// DefaultStageConfigs returns the spec-defined retry policies.
func DefaultStageConfigs() map[string]StageConfig {
	return map[string]StageConfig{
		"review": {MaxRetries: 2, OnExhaust: PolicyEscalate},
		"qa":     {MaxRetries: 3, OnExhaust: PolicyPauseRequirement},
		"rebase": {MaxRetries: 2, OnExhaust: PolicyPauseRequirement},
		"merge":  {MaxRetries: 1, OnExhaust: PolicyPauseRequirement},
	}
}

// BudgetChecker checks whether the story has remaining budget before a stage runs.
// Implemented by cost.Breaker (or a mock in tests).
type BudgetChecker interface {
	CheckBudget(ctx context.Context, storyID, reqID string) error
}

// EventEmitter emits pipeline events to the event store.
type EventEmitter interface {
	EmitRunStarted(storyID, stage string, attempt int) error
	EmitRunUpdated(storyID, stage, status string, attempt int, stageErr string, durationMs int64) error
}
