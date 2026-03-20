// Package cost implements cost accounting, budget enforcement, and the circuit breaker.
package cost

import (
	"time"

	"github.com/tzone85/project-x/internal/state"
)

// UsageRecord represents a single token usage entry for cost tracking.
type UsageRecord struct {
	StoryID      string
	ReqID        string
	AgentID      string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Stage        string
}

// CostSummary provides cost breakdown for display (px cost command).
type CostSummary struct {
	ByStory       map[string]float64
	ByRequirement map[string]float64
	ByDay         map[string]float64 // "2006-01-02" → total
	Total         float64
}

// Ledger tracks token usage and cost via event emission and projection queries.
type Ledger struct {
	store   *state.ProjectionStore
	emitter EventEmitter
}

// EventEmitter emits events to the event store and projection system.
type EventEmitter interface {
	Emit(event state.Event) error
}

// NewLedger creates a Ledger backed by the given projection store.
func NewLedger(store *state.ProjectionStore, emitter EventEmitter) *Ledger {
	return &Ledger{store: store, emitter: emitter}
}

// RecordUsage records a token usage entry by emitting a TokenUsageRecorded event.
func (l *Ledger) RecordUsage(record UsageRecord) error {
	event, err := state.NewEvent(state.EventTokenUsageRecorded, state.TokenUsageRecordedPayload{
		StoryID:       record.StoryID,
		RequirementID: record.ReqID,
		AgentID:       record.AgentID,
		Model:         record.Model,
		InputTokens:   record.InputTokens,
		OutputTokens:  record.OutputTokens,
		CostUSD:       record.CostUSD,
		Stage:         record.Stage,
	})
	if err != nil {
		return err
	}
	return l.emitter.Emit(event)
}

// GetStoryTotal returns the total cost for a story.
func (l *Ledger) GetStoryTotal(storyID string) (float64, error) {
	return l.store.GetStoryTotalCost(storyID)
}

// GetRequirementTotal returns the total cost for a requirement.
func (l *Ledger) GetRequirementTotal(reqID string) (float64, error) {
	return l.store.GetRequirementTotalCost(reqID)
}

// GetDailyTotal returns the total cost for a given date.
func (l *Ledger) GetDailyTotal(date time.Time) (float64, error) {
	return l.store.GetDailyTotalCost(date)
}
