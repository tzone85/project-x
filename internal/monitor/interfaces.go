package monitor

import (
	"context"

	"github.com/tzone85/project-x/internal/state"
)

// HealthStatus represents the health of an agent session.
type HealthStatus string

const (
	StatusHealthy HealthStatus = "healthy"
	StatusStale   HealthStatus = "stale"
	StatusDead    HealthStatus = "dead"
	StatusMissing HealthStatus = "missing"
)

// SessionHealth is the health result for a single agent session.
type SessionHealth struct {
	AgentID string
	Session string
	Status  HealthStatus
}

// SessionChecker checks agent session health.
// Implemented by tmux.Watchdog (or a mock in tests).
type SessionChecker interface {
	CheckSession(ctx context.Context, agentID, session string) (SessionHealth, error)
}

// PipelineDispatcher hands a finished story off to the pipeline runner.
// Implemented by pipeline.Runner (or a mock in tests).
type PipelineDispatcher interface {
	Dispatch(ctx context.Context, story state.Story) error
}

// WaveTracker queries story state for wave orchestration.
// Implemented by state.ProjectionStore or ReadOnlyQueries.
type WaveTracker interface {
	ListStoriesByRequirement(reqID string, page state.PageParams) ([]state.Story, error)
	GetStory(id string) (*state.Story, error)
}

// EventEmitter appends events to the event store and projection channel.
type EventEmitter interface {
	Emit(event state.Event) error
}
