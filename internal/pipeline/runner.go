package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RunResult summarizes the outcome of running the full pipeline for a story.
type RunResult struct {
	StoryID      string
	Completed    bool
	FailedStage  string
	FatalStage   string
	ExhaustStage string
	Err          error
}

// Runner executes pipeline stages sequentially for a story.
// It integrates per-stage retry budgets, cost breaker checks, and exhaust policies.
type Runner struct {
	stages  []Stage
	configs map[string]StageConfig
	budget  BudgetChecker
	emitter EventEmitter
	logger  *slog.Logger
}

// NewRunner creates a pipeline runner.
func NewRunner(
	stages []Stage,
	configs map[string]StageConfig,
	budget BudgetChecker,
	emitter EventEmitter,
	logger *slog.Logger,
) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if configs == nil {
		configs = DefaultStageConfigs()
	}
	return &Runner{
		stages:  stages,
		configs: configs,
		budget:  budget,
		emitter: emitter,
		logger:  logger,
	}
}

// Run executes all stages sequentially for the given story.
func (r *Runner) Run(ctx context.Context, story StoryContext) RunResult {
	for _, stage := range r.stages {
		result := r.runStage(ctx, stage, story)
		if !result.Completed {
			return result
		}
	}

	return RunResult{StoryID: story.StoryID, Completed: true}
}

// runStage executes a single stage with retries and budget checks.
func (r *Runner) runStage(ctx context.Context, stage Stage, story StoryContext) RunResult {
	name := stage.Name()
	cfg := r.configFor(name)

	for attempt := 1; attempt <= cfg.MaxRetries+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return RunResult{StoryID: story.StoryID, Err: fmt.Errorf("context cancelled: %w", err)}
		}

		if err := r.budget.CheckBudget(ctx, story.StoryID, story.RequirementID); err != nil {
			r.logger.Warn("budget exhausted before stage",
				"stage", name, "story_id", story.StoryID, "error", err)
			return RunResult{StoryID: story.StoryID, FailedStage: name, Err: err}
		}

		r.emitter.EmitRunStarted(story.StoryID, name, attempt)

		start := time.Now()
		result, err := stage.Execute(ctx, story)
		durationMs := time.Since(start).Milliseconds()

		switch result {
		case Passed:
			r.emitter.EmitRunUpdated(story.StoryID, name, "passed", attempt, "", durationMs)
			r.logger.Info("stage passed", "stage", name, "story_id", story.StoryID,
				"attempt", attempt, "duration_ms", durationMs)
			return RunResult{StoryID: story.StoryID, Completed: true}

		case Fatal:
			errMsg := errString(err)
			r.emitter.EmitRunUpdated(story.StoryID, name, "fatal", attempt, errMsg, durationMs)
			r.logger.Error("stage fatal", "stage", name, "story_id", story.StoryID,
				"attempt", attempt, "error", err)
			return RunResult{StoryID: story.StoryID, FatalStage: name, Err: err}

		case Failed:
			errMsg := errString(err)
			r.emitter.EmitRunUpdated(story.StoryID, name, "failed", attempt, errMsg, durationMs)
			r.logger.Warn("stage failed", "stage", name, "story_id", story.StoryID,
				"attempt", attempt, "max_retries", cfg.MaxRetries, "error", err)

			if attempt > cfg.MaxRetries {
				r.logger.Error("stage retries exhausted", "stage", name,
					"story_id", story.StoryID, "policy", cfg.OnExhaust)
				return RunResult{
					StoryID:      story.StoryID,
					ExhaustStage: name,
					Err:          fmt.Errorf("retries exhausted for %s: %w", name, err),
				}
			}
		}
	}

	// Should not reach here, but just in case
	return RunResult{StoryID: story.StoryID, FailedStage: stage.Name()}
}

// configFor returns the stage config, falling back to a zero-retry default.
func (r *Runner) configFor(name string) StageConfig {
	if cfg, ok := r.configs[name]; ok {
		return cfg
	}
	return StageConfig{MaxRetries: 0, OnExhaust: PolicyPauseRequirement}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
