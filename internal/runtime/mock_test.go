package runtime

import (
	"context"
	"sync"
)

// mockRuntime is a test double for the Runtime interface.
type mockRuntime struct {
	mu           sync.Mutex
	name         string
	version      string
	health       HealthStatus
	healthErr    error
	caps         RuntimeCapabilities
	spawnCalls   int
	killCalls    int
	inputCalls   int
}

func newMockRuntime(name string, health HealthStatus, caps RuntimeCapabilities) *mockRuntime {
	return &mockRuntime{
		name:    name,
		version: "1.0.0",
		health:  health,
		caps:    caps,
	}
}

func (m *mockRuntime) Name() string { return m.name }

func (m *mockRuntime) Version(_ context.Context) (string, error) {
	return m.version, nil
}

func (m *mockRuntime) Spawn(_ context.Context, _ SessionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spawnCalls++
	return nil
}

func (m *mockRuntime) Kill(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killCalls++
	return nil
}

func (m *mockRuntime) DetectStatus(_ context.Context, _ string) (AgentStatus, error) {
	return StatusRunning, nil
}

func (m *mockRuntime) ReadOutput(_ context.Context, _ string, _ int) (string, error) {
	return "output", nil
}

func (m *mockRuntime) Health(_ context.Context, _ string) (HealthStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.health, m.healthErr
}

func (m *mockRuntime) SendInput(_ context.Context, _ string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputCalls++
	return nil
}

func (m *mockRuntime) Capabilities() RuntimeCapabilities {
	return m.caps
}

func (m *mockRuntime) SetHealth(h HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.health = h
}

func (m *mockRuntime) SetHealthErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthErr = err
}
