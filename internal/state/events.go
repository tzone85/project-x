package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of event.
type EventType string

const (
	EventStoryCreated       EventType = "story_created"
	EventStoryUpdated       EventType = "story_updated"
	EventStoryStatusChanged EventType = "story_status_changed"
	EventAgentAssigned      EventType = "agent_assigned"
	EventAgentStatusChanged EventType = "agent_status_changed"
	EventAgentDied          EventType = "agent_died"
	EventAgentStale         EventType = "agent_stale"
	EventAgentLost          EventType = "agent_lost"
	EventRequirementCreated EventType = "requirement_created"
	EventRequirementUpdated EventType = "requirement_updated"
	EventEscalationCreated  EventType = "escalation_created"
	EventPipelineRunStarted EventType = "pipeline_run_started"
	EventPipelineRunUpdated EventType = "pipeline_run_updated"
	EventBudgetWarning      EventType = "budget_warning"
	EventTokenUsageRecorded EventType = "token_usage_recorded"
	EventSessionHealthChanged EventType = "session_health_changed"
)

// Event is the core event sourcing record. It is immutable once created.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent creates a new event with a generated UUID and current timestamp.
func NewEvent(eventType EventType, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal payload for %s: %w", eventType, err)
	}
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   data,
	}, nil
}

// DecodePayload decodes the event's raw JSON payload into the given typed struct.
// Returns an error if the JSON does not match the target type's schema.
func DecodePayload[T any](e Event) (T, error) {
	var result T
	if err := json.Unmarshal(e.Payload, &result); err != nil {
		return result, fmt.Errorf("decode %s payload: %w", e.Type, err)
	}
	return result, nil
}

// --- Typed Payloads ---

type StoryCreatedPayload struct {
	StoryID            string   `json:"story_id"`
	RequirementID      string   `json:"requirement_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
	OwnedFiles         []string `json:"owned_files"`
	Complexity         int      `json:"complexity"`
	DependsOn          []string `json:"depends_on"`
}

type StoryUpdatedPayload struct {
	StoryID string         `json:"story_id"`
	Fields  map[string]any `json:"fields"`
}

type StoryStatusChangedPayload struct {
	StoryID   string `json:"story_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Reason    string `json:"reason,omitempty"`
}

type AgentAssignedPayload struct {
	AgentID   string `json:"agent_id"`
	StoryID   string `json:"story_id"`
	Role      string `json:"role"`
	Runtime   string `json:"runtime"`
	Session   string `json:"session"`
}

type AgentStatusChangedPayload struct {
	AgentID   string `json:"agent_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type AgentDiedPayload struct {
	AgentID   string `json:"agent_id"`
	Session   string `json:"session"`
	ExitCode  int    `json:"exit_code"`
}

type AgentStalePayload struct {
	AgentID         string `json:"agent_id"`
	Session         string `json:"session"`
	StaleDurationMs int64  `json:"stale_duration_ms"`
}

type AgentLostPayload struct {
	AgentID string `json:"agent_id"`
	Session string `json:"session"`
}

type RequirementCreatedPayload struct {
	RequirementID string `json:"requirement_id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Source        string `json:"source"`
}

type RequirementUpdatedPayload struct {
	RequirementID string         `json:"requirement_id"`
	Fields        map[string]any `json:"fields"`
}

type EscalationCreatedPayload struct {
	EscalationID string `json:"escalation_id"`
	StoryID      string `json:"story_id"`
	Reason       string `json:"reason"`
	FromRole     string `json:"from_role"`
	ToRole       string `json:"to_role"`
}

type PipelineRunStartedPayload struct {
	StoryID string `json:"story_id"`
	Stage   string `json:"stage"`
	Attempt int    `json:"attempt"`
}

type PipelineRunUpdatedPayload struct {
	StoryID  string `json:"story_id"`
	Stage    string `json:"stage"`
	Status   string `json:"status"`
	Attempt  int    `json:"attempt"`
	Error    string `json:"error,omitempty"`
	DurationMs int64 `json:"duration_ms"`
}

type BudgetWarningPayload struct {
	StoryID       string  `json:"story_id,omitempty"`
	RequirementID string  `json:"requirement_id,omitempty"`
	CurrentCost   float64 `json:"current_cost"`
	BudgetLimit   float64 `json:"budget_limit"`
	Percentage    float64 `json:"percentage"`
}

type TokenUsageRecordedPayload struct {
	StoryID       string  `json:"story_id"`
	RequirementID string  `json:"requirement_id"`
	AgentID       string  `json:"agent_id"`
	Model         string  `json:"model"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	CostUSD       float64 `json:"cost_usd"`
	Stage         string  `json:"stage"`
}

type SessionHealthChangedPayload struct {
	SessionName    string `json:"session_name"`
	OldStatus      string `json:"old_status"`
	NewStatus      string `json:"new_status"`
	PanePID        int    `json:"pane_pid"`
	RecoveryAttempt int   `json:"recovery_attempt"`
}

// payloadRegistry maps event types to their expected payload type for validation.
var payloadRegistry = map[EventType]func() any{
	EventStoryCreated:         func() any { return &StoryCreatedPayload{} },
	EventStoryUpdated:         func() any { return &StoryUpdatedPayload{} },
	EventStoryStatusChanged:   func() any { return &StoryStatusChangedPayload{} },
	EventAgentAssigned:        func() any { return &AgentAssignedPayload{} },
	EventAgentStatusChanged:   func() any { return &AgentStatusChangedPayload{} },
	EventAgentDied:            func() any { return &AgentDiedPayload{} },
	EventAgentStale:           func() any { return &AgentStalePayload{} },
	EventAgentLost:            func() any { return &AgentLostPayload{} },
	EventRequirementCreated:   func() any { return &RequirementCreatedPayload{} },
	EventRequirementUpdated:   func() any { return &RequirementUpdatedPayload{} },
	EventEscalationCreated:    func() any { return &EscalationCreatedPayload{} },
	EventPipelineRunStarted:   func() any { return &PipelineRunStartedPayload{} },
	EventPipelineRunUpdated:   func() any { return &PipelineRunUpdatedPayload{} },
	EventBudgetWarning:        func() any { return &BudgetWarningPayload{} },
	EventTokenUsageRecorded:   func() any { return &TokenUsageRecordedPayload{} },
	EventSessionHealthChanged: func() any { return &SessionHealthChangedPayload{} },
}

// ValidatePayload checks if the event's payload matches the expected schema for its type.
func ValidatePayload(e Event) error {
	factory, ok := payloadRegistry[e.Type]
	if !ok {
		return fmt.Errorf("unknown event type: %s", e.Type)
	}
	target := factory()
	if err := json.Unmarshal(e.Payload, target); err != nil {
		return fmt.Errorf("payload validation failed for %s: %w", e.Type, err)
	}
	return nil
}
