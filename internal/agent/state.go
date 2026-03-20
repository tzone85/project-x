package agent

import "fmt"

// Status represents an agent's current operational state.
type Status string

const (
	StatusIdle       Status = "idle"
	StatusWorking    Status = "working"
	StatusBlocked    Status = "blocked"
	StatusTerminated Status = "terminated"
)

// validStatuses lists all known statuses for validation.
var validStatuses = map[Status]bool{
	StatusIdle:       true,
	StatusWorking:    true,
	StatusBlocked:    true,
	StatusTerminated: true,
}

// validTransitions defines the allowed state transitions.
var validTransitions = map[Status]map[Status]bool{
	StatusIdle: {
		StatusWorking:    true,
		StatusTerminated: true,
	},
	StatusWorking: {
		StatusIdle:       true,
		StatusBlocked:    true,
		StatusTerminated: true,
	},
	StatusBlocked: {
		StatusWorking:    true,
		StatusTerminated: true,
	},
	StatusTerminated: {},
}

// Agent represents an agent instance with its current state.
type Agent struct {
	ID      string
	Role    Role
	Status  Status
	StoryID string
	Memory  map[string]string
}

// NewAgent creates a new agent in idle status with the given role.
func NewAgent(id string, role Role) Agent {
	return Agent{
		ID:     id,
		Role:   role,
		Status: StatusIdle,
		Memory: make(map[string]string),
	}
}

// TransitionTo returns a new Agent with the target status if the transition
// is valid. Returns an error for invalid transitions.
func (a Agent) TransitionTo(target Status) (Agent, error) {
	allowed, ok := validTransitions[a.Status]
	if !ok || !allowed[target] {
		return a, fmt.Errorf("invalid transition: %s -> %s", a.Status, target)
	}

	result := a.clone()
	result.Status = target
	return result, nil
}

// AssignStory transitions the agent to working and sets the story ID.
// The agent must be idle.
func (a Agent) AssignStory(storyID string) (Agent, error) {
	if a.Status != StatusIdle {
		return a, fmt.Errorf("cannot assign story: agent is %s, must be idle", a.Status)
	}

	result := a.clone()
	result.Status = StatusWorking
	result.StoryID = storyID
	return result, nil
}

// CompleteStory transitions the agent back to idle and clears the story ID.
// The agent must be working.
func (a Agent) CompleteStory() (Agent, error) {
	if a.Status != StatusWorking {
		return a, fmt.Errorf("cannot complete story: agent is %s, must be working", a.Status)
	}

	result := a.clone()
	result.Status = StatusIdle
	result.StoryID = ""
	return result, nil
}

// WithMemory returns a new Agent with the given key-value pair added to memory.
func (a Agent) WithMemory(key, value string) Agent {
	result := a.clone()
	result.Memory[key] = value
	return result
}

// withStatus returns a copy of the agent with the given status (used in tests).
func (a Agent) withStatus(s Status) Agent {
	result := a.clone()
	result.Status = s
	return result
}

// clone returns a deep copy of the agent.
func (a Agent) clone() Agent {
	mem := make(map[string]string, len(a.Memory))
	for k, v := range a.Memory {
		mem[k] = v
	}
	return Agent{
		ID:      a.ID,
		Role:    a.Role,
		Status:  a.Status,
		StoryID: a.StoryID,
		Memory:  mem,
	}
}

// ParseStatus converts a string to a Status, returning an error if invalid.
func ParseStatus(s string) (Status, error) {
	st := Status(s)
	if !validStatuses[st] {
		return "", fmt.Errorf("unknown status: %q", s)
	}
	return st, nil
}
