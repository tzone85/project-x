package state

import (
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// EventType identifies the kind of domain event.
type EventType string

// Requirement lifecycle events.
const (
	EventReqSubmitted EventType = "req.submitted"
	EventReqAnalyzed  EventType = "req.analyzed"
	EventReqPlanned   EventType = "req.planned"
	EventReqPaused    EventType = "req.paused"
	EventReqResumed   EventType = "req.resumed"
	EventReqCompleted EventType = "req.completed"
)

// Story lifecycle events.
const (
	EventStoryCreated   EventType = "story.created"
	EventStoryEstimated EventType = "story.estimated"
	EventStoryAssigned  EventType = "story.assigned"
	EventStoryStarted   EventType = "story.started"
	EventStoryProgress  EventType = "story.progress"
	EventStoryCompleted EventType = "story.completed"
)

// Story review events.
const (
	EventStoryReviewRequested EventType = "story.review_requested"
	EventStoryReviewPassed    EventType = "story.review_passed"
	EventStoryReviewFailed    EventType = "story.review_failed"
)

// Story QA events.
const (
	EventStoryQAStarted EventType = "story.qa_started"
	EventStoryQAPassed  EventType = "story.qa_passed"
	EventStoryQAFailed  EventType = "story.qa_failed"
)

// Story merge events.
const (
	EventStoryPRCreated EventType = "story.pr_created"
	EventStoryMerged    EventType = "story.merged"
)

// Agent lifecycle events.
const (
	EventAgentSpawned EventType = "agent.spawned"
	EventAgentStuck   EventType = "agent.stuck"
	EventAgentDied    EventType = "agent.died"
	EventAgentStale   EventType = "agent.stale"
	EventAgentLost    EventType = "agent.lost"
)

// Escalation events.
const (
	EventEscalationCreated EventType = "escalation.created"
)

// Budget events.
const (
	EventBudgetWarning   EventType = "budget.warning"
	EventBudgetExhausted EventType = "budget.exhausted"
)

// Event is the atomic unit of the event-sourced architecture.
// All state changes are expressed as immutable events.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	AgentID   string          `json:"agent_id"`
	StoryID   string          `json:"story_id"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp string          `json:"timestamp"`
}

// NewEvent creates an Event with a generated ULID, current timestamp,
// and the given payload serialized to JSON. A nil payload results in
// a nil Payload field.
func NewEvent(typ EventType, agentID, storyID string, payload map[string]any) Event {
	var raw json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			// payload is map[string]any — marshal failure is a programming error.
			panic("state.NewEvent: failed to marshal payload: " + err.Error())
		}
		raw = data
	}

	return Event{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Type:      typ,
		AgentID:   agentID,
		StoryID:   storyID,
		Payload:   raw,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

// NewTypedEvent creates an Event whose payload is a typed struct marshaled
// to JSON. Returns an error if the payload cannot be serialized.
func NewTypedEvent(typ EventType, agentID, storyID string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}

	return Event{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Type:      typ,
		AgentID:   agentID,
		StoryID:   storyID,
		Payload:   data,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

// --- Typed payload structs ---

// ReqSubmittedPayload carries data for a requirement submission event.
type ReqSubmittedPayload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	RepoPath    string `json:"repo_path"`
}

// StoryCreatedPayload carries data for a story creation event.
type StoryCreatedPayload struct {
	ID                 string   `json:"id"`
	ReqID              string   `json:"req_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
	Complexity         int      `json:"complexity"`
	OwnedFiles         []string `json:"owned_files"`
	WaveHint           string   `json:"wave_hint"`
	DependsOn          []string `json:"depends_on"`
}

// StoryAssignedPayload carries data for a story assignment event.
type StoryAssignedPayload struct {
	AgentID string `json:"agent_id"`
	Wave    int    `json:"wave"`
}

// StoryPRCreatedPayload carries data for a story PR creation event.
type StoryPRCreatedPayload struct {
	PRUrl    string `json:"pr_url"`
	PRNumber int    `json:"pr_number"`
}

// AgentSpawnedPayload carries data for an agent spawned event.
type AgentSpawnedPayload struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Model       string `json:"model"`
	Runtime     string `json:"runtime"`
	SessionName string `json:"session_name"`
	StoryID     string `json:"story_id"`
}

// EscalationCreatedPayload carries data for an escalation creation event.
type EscalationCreatedPayload struct {
	ID        string `json:"id"`
	StoryID   string `json:"story_id"`
	FromAgent string `json:"from_agent"`
	Reason    string `json:"reason"`
}

// ReqStatusPayload carries the req_id for requirement status update events.
type ReqStatusPayload struct {
	ReqID string `json:"req_id"`
}

// BudgetWarningPayload carries data for a budget warning event.
type BudgetWarningPayload struct {
	ReqID      string  `json:"req_id"`
	StoryID    string  `json:"story_id"`
	UsedUSD    float64 `json:"used_usd"`
	LimitUSD   float64 `json:"limit_usd"`
	Percentage int     `json:"percentage"`
}
