package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tzone85/project-x/internal/state"
)

// PollerConfig holds configurable parameters for the agent poller.
type PollerConfig struct {
	PollInterval time.Duration
	StopTimeout  time.Duration
}

// DefaultPollerConfig returns reasonable defaults.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		PollInterval: 5 * time.Second,
		StopTimeout:  30 * time.Second,
	}
}

// Poller polls active agent sessions at a configurable interval.
// When an agent finishes, it hands the story to the pipeline runner.
// When all stories in a wave complete, it triggers the next wave.
type Poller struct {
	checker    SessionChecker
	dispatcher PipelineDispatcher
	waveCheck  *WaveChecker
	emitter    EventEmitter
	config     PollerConfig
	logger     *slog.Logger

	mu      sync.Mutex
	tracked map[string]trackedAgent // agentID -> tracked info
}

// trackedAgent holds minimal state about an in-flight agent.
type trackedAgent struct {
	storyID string
	reqID   string
	session string
	wave    int
}

// NewPoller creates a new agent poller.
func NewPoller(
	checker SessionChecker,
	dispatcher PipelineDispatcher,
	waveCheck *WaveChecker,
	emitter EventEmitter,
	config PollerConfig,
	logger *slog.Logger,
) *Poller {
	if logger == nil {
		logger = slog.Default()
	}
	return &Poller{
		checker:    checker,
		dispatcher: dispatcher,
		waveCheck:  waveCheck,
		emitter:    emitter,
		config:     config,
		logger:     logger,
		tracked:    make(map[string]trackedAgent),
	}
}

// Track registers an agent for health polling.
func (p *Poller) Track(agentID, storyID, reqID, session string, wave int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tracked[agentID] = trackedAgent{
		storyID: storyID,
		reqID:   reqID,
		session: session,
		wave:    wave,
	}
}

// Untrack removes an agent from health polling.
func (p *Poller) Untrack(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.tracked, agentID)
}

// TrackedCount returns the number of tracked agents.
func (p *Poller) TrackedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tracked)
}

// Run starts the polling loop. It blocks until the context is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

// pollOnce checks all tracked agents and handles finished/unhealthy ones.
func (p *Poller) pollOnce(ctx context.Context) {
	p.mu.Lock()
	snapshot := make(map[string]trackedAgent, len(p.tracked))
	for k, v := range p.tracked {
		snapshot[k] = v
	}
	p.mu.Unlock()

	for agentID, info := range snapshot {
		health, err := p.checker.CheckSession(ctx, agentID, info.session)
		if err != nil {
			p.logger.Error("health check failed", "agent_id", agentID, "error", err)
			continue
		}

		switch health.Status {
		case StatusHealthy, StatusStale:
			continue
		case StatusDead:
			p.handleFinished(ctx, agentID, info)
		case StatusMissing:
			p.handleMissing(agentID, info)
		}
	}
}

// handleFinished processes an agent whose session has exited.
func (p *Poller) handleFinished(ctx context.Context, agentID string, info trackedAgent) {
	p.logger.Info("agent finished", "agent_id", agentID, "story_id", info.storyID)
	p.Untrack(agentID)

	story, err := p.waveCheck.waves.GetStory(info.storyID)
	if err != nil || story == nil {
		p.logger.Error("failed to get story for pipeline dispatch",
			"story_id", info.storyID, "error", err)
		return
	}

	if err := p.dispatcher.Dispatch(ctx, *story); err != nil {
		p.logger.Error("pipeline dispatch failed", "story_id", info.storyID, "error", err)
		return
	}

	if p.waveCheck.CheckCompletion(info.reqID, info.wave) {
		p.logger.Info("wave complete", "req_id", info.reqID, "wave", info.wave)
		ready := p.waveCheck.NextWaveStories(info.reqID, info.wave)
		for _, s := range ready {
			p.logger.Info("story ready for dispatch",
				"story_id", s.ID, "wave", info.wave+1, "title", s.Title)
		}
	}
}

// handleMissing processes an agent whose session cannot be found.
func (p *Poller) handleMissing(agentID string, info trackedAgent) {
	p.logger.Warn("agent session missing", "agent_id", agentID, "session", info.session)

	evt, err := state.NewEvent(state.EventAgentLost, state.AgentLostPayload{
		AgentID: agentID,
		Session: info.session,
	})
	if err == nil {
		p.emitter.Emit(evt)
	}

	p.Untrack(agentID)
}
