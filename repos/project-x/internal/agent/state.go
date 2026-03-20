package agent

import (
	"fmt"
	"sync"
)

// Status represents the current state of an agent.
type Status string

const (
	StatusIdle       Status = "idle"
	StatusWorking    Status = "working"
	StatusBlocked    Status = "blocked"
	StatusTerminated Status = "terminated"
)

// State holds the current state of a single agent.
type State struct {
	SessionName  string
	Role         Role
	Status       Status
	CurrentStory string // story ID currently assigned, empty if idle
}

// StateManager tracks agent states. Thread-safe for concurrent access.
type StateManager struct {
	mu     sync.RWMutex
	agents map[string]State // keyed by session name
}

// NewStateManager creates a new agent state manager.
func NewStateManager() *StateManager {
	return &StateManager{
		agents: make(map[string]State),
	}
}

// Register adds a new agent in idle status.
func (sm *StateManager) Register(sessionName string, role Role) State {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := State{
		SessionName: sessionName,
		Role:        role,
		Status:      StatusIdle,
	}
	sm.agents[sessionName] = state
	return state
}

// Get returns the current state of an agent.
func (sm *StateManager) Get(sessionName string) (State, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, ok := sm.agents[sessionName]
	return state, ok
}

// AssignStory transitions an idle agent to working on a story.
// Returns an error if the agent is not idle.
func (sm *StateManager) AssignStory(sessionName, storyID string) (State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, ok := sm.agents[sessionName]
	if !ok {
		return State{}, fmt.Errorf("agent %q not found", sessionName)
	}
	if state.Status != StatusIdle {
		return State{}, fmt.Errorf("agent %q is %s, not idle", sessionName, state.Status)
	}

	state.Status = StatusWorking
	state.CurrentStory = storyID
	sm.agents[sessionName] = state
	return state, nil
}

// SetStatus updates an agent's status. Use for blocked/terminated transitions.
func (sm *StateManager) SetStatus(sessionName string, status Status) (State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, ok := sm.agents[sessionName]
	if !ok {
		return State{}, fmt.Errorf("agent %q not found", sessionName)
	}

	state.Status = status
	if status == StatusIdle || status == StatusTerminated {
		state.CurrentStory = ""
	}
	sm.agents[sessionName] = state
	return state, nil
}

// ReleaseStory transitions an agent back to idle and clears the story assignment.
func (sm *StateManager) ReleaseStory(sessionName string) (State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, ok := sm.agents[sessionName]
	if !ok {
		return State{}, fmt.Errorf("agent %q not found", sessionName)
	}

	state.Status = StatusIdle
	state.CurrentStory = ""
	sm.agents[sessionName] = state
	return state, nil
}

// ListByStatus returns all agents with the given status.
func (sm *StateManager) ListByStatus(status Status) []State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []State
	for _, state := range sm.agents {
		if state.Status == status {
			result = append(result, state)
		}
	}
	return result
}

// All returns all agent states.
func (sm *StateManager) All() []State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]State, 0, len(sm.agents))
	for _, state := range sm.agents {
		result = append(result, state)
	}
	return result
}
