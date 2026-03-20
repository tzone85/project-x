package monitor

import (
	"context"
	"sync"
	"testing"
	"time"
)

// --- Mocks ---

type mockHealthChecker struct {
	mu       sync.Mutex
	statuses map[string]AgentStatus
}

func newMockHealthChecker() *mockHealthChecker {
	return &mockHealthChecker{statuses: make(map[string]AgentStatus)}
}

func (m *mockHealthChecker) SetStatus(session string, status AgentStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[session] = status
}

func (m *mockHealthChecker) CheckHealth(_ context.Context, session string) AgentStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.statuses[session]; ok {
		return s
	}
	return AgentRunning
}

type mockPipeline struct {
	mu          sync.Mutex
	completions []TrackedAgent
}

func (m *mockPipeline) HandleCompletion(_ context.Context, agent TrackedAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completions = append(m.completions, agent)
	return nil
}

func (m *mockPipeline) Completions() []TrackedAgent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]TrackedAgent, len(m.completions))
	copy(result, m.completions)
	return result
}

type mockWaveDispatcher struct {
	mu              sync.Mutex
	dispatchedWaves []int
}

func (m *mockWaveDispatcher) DispatchNextWave(_ context.Context, completedWave int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatchedWaves = append(m.dispatchedWaves, completedWave)
	return nil
}

func (m *mockWaveDispatcher) DispatchedWaves() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int, len(m.dispatchedWaves))
	copy(result, m.dispatchedWaves)
	return result
}

type mockRecovery struct {
	mu      sync.Mutex
	dead    []string
	stale   []string
	missing []string
}

func (m *mockRecovery) HandleDead(_ context.Context, agent TrackedAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dead = append(m.dead, agent.SessionName)
	return nil
}

func (m *mockRecovery) HandleStale(_ context.Context, agent TrackedAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stale = append(m.stale, agent.SessionName)
	return nil
}

func (m *mockRecovery) HandleMissing(_ context.Context, agent TrackedAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.missing = append(m.missing, agent.SessionName)
	return nil
}

// --- Tests ---

func TestPollerTrackAndUntrack(t *testing.T) {
	p := NewPoller(DefaultPollerConfig(), newMockHealthChecker(), &mockPipeline{}, &mockWaveDispatcher{}, &mockRecovery{}, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	p.Track(TrackedAgent{SessionName: "a2", StoryID: "s2", WaveNum: 1})

	agents := p.TrackedAgents()
	if len(agents) != 2 {
		t.Errorf("got %d agents, want 2", len(agents))
	}

	p.Untrack("a1")
	agents = p.TrackedAgents()
	if len(agents) != 1 {
		t.Errorf("got %d agents after untrack, want 1", len(agents))
	}
}

func TestPollerFinishedAgentTriggersPipeline(t *testing.T) {
	health := newMockHealthChecker()
	pipeline := &mockPipeline{}
	waves := &mockWaveDispatcher{}
	recovery := &mockRecovery{}

	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, pipeline, waves, recovery, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentFinished)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	completions := pipeline.Completions()
	if len(completions) == 0 {
		t.Fatal("expected pipeline completion, got none")
	}
	if completions[0].StoryID != "s1" {
		t.Errorf("story = %q, want s1", completions[0].StoryID)
	}

	// Agent should be untracked after finishing
	if len(p.TrackedAgents()) != 0 {
		t.Error("expected agent to be untracked after completion")
	}
}

func TestPollerDeadAgentTriggersRecovery(t *testing.T) {
	health := newMockHealthChecker()
	recovery := &mockRecovery{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, &mockPipeline{}, &mockWaveDispatcher{}, recovery, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentDead)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if len(recovery.dead) == 0 {
		t.Fatal("expected dead recovery, got none")
	}
}

func TestPollerStaleAgentTriggersRecovery(t *testing.T) {
	health := newMockHealthChecker()
	recovery := &mockRecovery{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, &mockPipeline{}, &mockWaveDispatcher{}, recovery, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentStale)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if len(recovery.stale) == 0 {
		t.Fatal("expected stale recovery, got none")
	}
}

func TestPollerMissingAgentCleansUp(t *testing.T) {
	health := newMockHealthChecker()
	recovery := &mockRecovery{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, &mockPipeline{}, &mockWaveDispatcher{}, recovery, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentMissing)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if len(recovery.missing) == 0 {
		t.Fatal("expected missing cleanup, got none")
	}

	if len(p.TrackedAgents()) != 0 {
		t.Error("expected missing agent to be untracked")
	}
}

func TestPollerWaveCompletionTriggersDispatch(t *testing.T) {
	health := newMockHealthChecker()
	waves := &mockWaveDispatcher{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, &mockPipeline{}, waves, &mockRecovery{}, nil)

	// Single agent in wave 1
	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentFinished)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	dispatched := waves.DispatchedWaves()
	if len(dispatched) == 0 {
		t.Fatal("expected wave dispatch, got none")
	}
	if dispatched[0] != 1 {
		t.Errorf("dispatched wave = %d, want 1", dispatched[0])
	}
}

func TestPollerPartialWaveDoesNotDispatch(t *testing.T) {
	health := newMockHealthChecker()
	waves := &mockWaveDispatcher{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, &mockPipeline{}, waves, &mockRecovery{}, nil)

	// Two agents in wave 1, only one finishes
	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	p.Track(TrackedAgent{SessionName: "a2", StoryID: "s2", WaveNum: 1})
	health.SetStatus("a1", AgentFinished)
	health.SetStatus("a2", AgentRunning)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	dispatched := waves.DispatchedWaves()
	if len(dispatched) != 0 {
		t.Errorf("expected no wave dispatch with partial completion, got %v", dispatched)
	}
}

func TestPollerRunningAgentNoAction(t *testing.T) {
	health := newMockHealthChecker()
	pipeline := &mockPipeline{}
	recovery := &mockRecovery{}
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, health, pipeline, &mockWaveDispatcher{}, recovery, nil)

	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1", WaveNum: 1})
	health.SetStatus("a1", AgentRunning)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	p.Run(ctx)

	if len(pipeline.Completions()) != 0 {
		t.Error("expected no completions for running agent")
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if len(recovery.dead) != 0 || len(recovery.stale) != 0 {
		t.Error("expected no recovery for running agent")
	}
}

func TestPollerEmptyAgents(t *testing.T) {
	p := NewPoller(PollerConfig{PollInterval: 10 * time.Millisecond}, newMockHealthChecker(), &mockPipeline{}, &mockWaveDispatcher{}, &mockRecovery{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// Should not panic with empty agents
	p.Run(ctx)
}

func TestDefaultPollerConfig(t *testing.T) {
	cfg := DefaultPollerConfig()
	if cfg.PollInterval != 10*time.Second {
		t.Errorf("poll interval = %v, want 10s", cfg.PollInterval)
	}
}

func TestPollerTrackSetsRunningStatus(t *testing.T) {
	p := NewPoller(DefaultPollerConfig(), newMockHealthChecker(), &mockPipeline{}, &mockWaveDispatcher{}, &mockRecovery{}, nil)
	p.Track(TrackedAgent{SessionName: "a1", StoryID: "s1"})

	agents := p.TrackedAgents()
	if agents[0].Status != AgentRunning {
		t.Errorf("status = %q, want running", agents[0].Status)
	}
}
