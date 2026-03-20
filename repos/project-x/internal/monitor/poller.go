// Package monitor implements the agent poller and wave orchestration.
// The poller is a slim (~150 lines) single-responsibility component that:
// - Polls active tmux sessions at configurable intervals
// - Checks health before fingerprinting
// - Hands off finished agents to the pipeline runner
// - Triggers next wave dispatch when current wave completes
package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// AgentStatus represents the detected status of an agent session.
type AgentStatus string

const (
	AgentRunning  AgentStatus = "running"
	AgentFinished AgentStatus = "finished"
	AgentDead     AgentStatus = "dead"
	AgentStale    AgentStatus = "stale"
	AgentMissing  AgentStatus = "missing"
)

// TrackedAgent represents an agent being monitored.
type TrackedAgent struct {
	SessionName string
	StoryID     string
	WaveNum     int
	Status      AgentStatus
}

// HealthChecker checks the health of a tmux session.
type HealthChecker interface {
	CheckHealth(ctx context.Context, sessionName string) AgentStatus
}

// PipelineHandler is called when an agent finishes its work.
type PipelineHandler interface {
	HandleCompletion(ctx context.Context, agent TrackedAgent) error
}

// WaveDispatcher dispatches the next wave of stories.
type WaveDispatcher interface {
	DispatchNextWave(ctx context.Context, completedWave int) error
}

// RecoveryHandler handles dead/stale agent recovery.
type RecoveryHandler interface {
	HandleDead(ctx context.Context, agent TrackedAgent) error
	HandleStale(ctx context.Context, agent TrackedAgent) error
	HandleMissing(ctx context.Context, agent TrackedAgent) error
}

// PollerConfig configures the agent poller.
type PollerConfig struct {
	PollInterval time.Duration
}

// DefaultPollerConfig returns sensible poller defaults.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		PollInterval: 10 * time.Second,
	}
}

// Poller polls active tmux sessions and orchestrates wave completion.
type Poller struct {
	config   PollerConfig
	health   HealthChecker
	pipeline PipelineHandler
	waves    WaveDispatcher
	recovery RecoveryHandler
	logger   *slog.Logger

	mu     sync.Mutex
	agents map[string]TrackedAgent // session name → agent
}

// NewPoller creates a new agent poller.
func NewPoller(
	config PollerConfig,
	health HealthChecker,
	pipeline PipelineHandler,
	waves WaveDispatcher,
	recovery RecoveryHandler,
	logger *slog.Logger,
) *Poller {
	if logger == nil {
		logger = slog.Default()
	}
	return &Poller{
		config:   config,
		health:   health,
		pipeline: pipeline,
		waves:    waves,
		recovery: recovery,
		logger:   logger.With("component", "poller"),
		agents:   make(map[string]TrackedAgent),
	}
}

// Track registers an agent for polling.
func (p *Poller) Track(agent TrackedAgent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	agent.Status = AgentRunning
	p.agents[agent.SessionName] = agent
}

// Untrack removes an agent from polling.
func (p *Poller) Untrack(sessionName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.agents, sessionName)
}

// TrackedAgents returns a snapshot of all tracked agents.
func (p *Poller) TrackedAgents() []TrackedAgent {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]TrackedAgent, 0, len(p.agents))
	for _, a := range p.agents {
		result = append(result, a)
	}
	return result
}

// Run starts the polling loop. It blocks until the context is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	p.logger.Info("poller started", "interval", p.config.PollInterval)

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poller stopped")
			return ctx.Err()
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context) {
	agents := p.TrackedAgents()
	if len(agents) == 0 {
		return
	}

	completedWaves := make(map[int]bool)

	for _, agent := range agents {
		status := p.health.CheckHealth(ctx, agent.SessionName)

		switch status {
		case AgentFinished:
			p.logger.Info("agent finished", "session", agent.SessionName, "story", agent.StoryID)
			if err := p.pipeline.HandleCompletion(ctx, agent); err != nil {
				p.logger.Error("pipeline error", "session", agent.SessionName, "error", err)
			}
			p.Untrack(agent.SessionName)
			completedWaves[agent.WaveNum] = true

		case AgentDead:
			p.logger.Warn("agent dead", "session", agent.SessionName, "story", agent.StoryID)
			if err := p.recovery.HandleDead(ctx, agent); err != nil {
				p.logger.Error("recovery error", "session", agent.SessionName, "error", err)
			}
			p.updateStatus(agent.SessionName, AgentDead)

		case AgentStale:
			p.logger.Warn("agent stale", "session", agent.SessionName, "story", agent.StoryID)
			if err := p.recovery.HandleStale(ctx, agent); err != nil {
				p.logger.Error("recovery error", "session", agent.SessionName, "error", err)
			}
			p.updateStatus(agent.SessionName, AgentStale)

		case AgentMissing:
			p.logger.Warn("agent missing", "session", agent.SessionName, "story", agent.StoryID)
			if err := p.recovery.HandleMissing(ctx, agent); err != nil {
				p.logger.Error("cleanup error", "session", agent.SessionName, "error", err)
			}
			p.Untrack(agent.SessionName)

		case AgentRunning:
			// Still working, nothing to do
		}
	}

	// Check if any wave completed
	for waveNum := range completedWaves {
		if p.isWaveComplete(waveNum) {
			p.logger.Info("wave complete", "wave", waveNum)
			if err := p.waves.DispatchNextWave(ctx, waveNum); err != nil {
				p.logger.Error("wave dispatch error", "wave", waveNum, "error", err)
			}
		}
	}
}

func (p *Poller) updateStatus(sessionName string, status AgentStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if agent, ok := p.agents[sessionName]; ok {
		agent.Status = status
		p.agents[sessionName] = agent
	}
}

func (p *Poller) isWaveComplete(waveNum int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, a := range p.agents {
		if a.WaveNum == waveNum {
			return false // still have tracked agents in this wave
		}
	}
	return true
}
