package monitor

import (
	"context"
	"sync"

	"github.com/tzone85/project-x/internal/state"
)

// mockSessionChecker is a test double for SessionChecker.
type mockSessionChecker struct {
	mu      sync.Mutex
	results map[string]SessionHealth
	err     error
	calls   []string // tracks agentIDs checked
}

func newMockChecker() *mockSessionChecker {
	return &mockSessionChecker{results: make(map[string]SessionHealth)}
}

func (m *mockSessionChecker) SetResult(agentID string, status HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[agentID] = SessionHealth{AgentID: agentID, Status: status}
}

func (m *mockSessionChecker) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockSessionChecker) CheckSession(_ context.Context, agentID, session string) (SessionHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, agentID)
	if m.err != nil {
		return SessionHealth{}, m.err
	}
	if h, ok := m.results[agentID]; ok {
		return h, nil
	}
	return SessionHealth{AgentID: agentID, Status: StatusHealthy}, nil
}

func (m *mockSessionChecker) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// mockPipelineDispatcher is a test double for PipelineDispatcher.
type mockPipelineDispatcher struct {
	mu        sync.Mutex
	err       error
	dispatched []state.Story
}

func newMockDispatcher() *mockPipelineDispatcher {
	return &mockPipelineDispatcher{}
}

func (m *mockPipelineDispatcher) Dispatch(_ context.Context, story state.Story) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.dispatched = append(m.dispatched, story)
	return nil
}

func (m *mockPipelineDispatcher) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockPipelineDispatcher) DispatchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.dispatched)
}

// mockWaveTracker is a test double for WaveTracker.
type mockWaveTracker struct {
	mu      sync.Mutex
	stories map[string]*state.Story            // storyID -> story
	byReq   map[string][]state.Story           // reqID -> stories
}

func newMockWaveTracker() *mockWaveTracker {
	return &mockWaveTracker{
		stories: make(map[string]*state.Story),
		byReq:   make(map[string][]state.Story),
	}
}

func (m *mockWaveTracker) AddStory(s state.Story) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stories[s.ID] = &s
	m.byReq[s.ReqID] = append(m.byReq[s.ReqID], s)
}

func (m *mockWaveTracker) GetStory(id string) (*state.Story, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stories[id], nil
}

func (m *mockWaveTracker) ListStoriesByRequirement(reqID string, _ state.PageParams) ([]state.Story, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byReq[reqID], nil
}

// mockEventEmitter is a test double for EventEmitter.
type mockEventEmitter struct {
	mu     sync.Mutex
	events []state.Event
	err    error
}

func newMockEmitter() *mockEventEmitter {
	return &mockEventEmitter{}
}

func (m *mockEventEmitter) Emit(event state.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventEmitter) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func (m *mockEventEmitter) LastEventType() state.EventType {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return ""
	}
	return m.events[len(m.events)-1].Type
}
