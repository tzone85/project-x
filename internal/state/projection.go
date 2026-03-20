package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tzone85/project-x/migrations"
)

// Requirement is the projection model for requirements.
type Requirement struct {
	ID          string
	Title       string
	Description string
	Source      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Story is the projection model for stories.
type Story struct {
	ID                 string
	ReqID              string
	Title              string
	Description        string
	AcceptanceCriteria string
	OwnedFiles         []string
	Complexity         int
	DependsOn          []string
	Status             string
	AgentID            string
	Wave               int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Agent is the projection model for agents.
type Agent struct {
	ID           string
	Role         string
	Status       string
	CurrentStory string
	SessionName  string
	Runtime      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Escalation is the projection model for escalations.
type Escalation struct {
	ID         string
	StoryID    string
	Reason     string
	FromRole   string
	ToRole     string
	Status     string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// TokenUsage is the projection model for token_usage.
type TokenUsage struct {
	ID           string
	StoryID      string
	ReqID        string
	AgentID      string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Stage        string
	CreatedAt    time.Time
}

// SessionHealth is the projection model for session_health.
type SessionHealth struct {
	SessionName     string
	Status          string
	PanePID         int
	LastOutputHash  string
	RecoveryAttempts int
	LastCheckAt     time.Time
	UpdatedAt       time.Time
}

// PipelineRun is the projection model for pipeline_runs.
type PipelineRun struct {
	ID         string
	StoryID    string
	Stage      string
	Status     string
	Attempt    int
	Error      string
	StartedAt  time.Time
	EndedAt    *time.Time
	DurationMs int64
}

// ProjectionStore manages SQLite projections derived from the event log.
// It uses WAL mode with a single writer connection. Read-only connections
// can be created separately for dashboard/web use.
type ProjectionStore struct {
	writer *sql.DB
}

// NewProjectionStore opens a SQLite database at the given path in WAL mode,
// runs pending migrations, and returns a ProjectionStore ready for writes.
func NewProjectionStore(dbPath string) (*ProjectionStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open projection db: %w", err)
	}

	// Single writer connection
	db.SetMaxOpenConns(1)

	// Run migrations
	migrator := NewMigrator(db, migrations.FS)
	if err := migrator.Migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &ProjectionStore{writer: db}, nil
}

// NewReadOnlyConnection opens a separate read-only connection for dashboard/web use.
func NewReadOnlyConnection(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&mode=ro&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open read-only db: %w", err)
	}
	return db, nil
}

// Close closes the writer connection.
func (ps *ProjectionStore) Close() error {
	return ps.writer.Close()
}

// DB returns the underlying writer database for testing or advanced use.
func (ps *ProjectionStore) DB() *sql.DB {
	return ps.writer
}

// --- Apply Event Projections ---

// ApplyEvent routes an event to the correct projection handler.
func (ps *ProjectionStore) ApplyEvent(event Event) error {
	switch event.Type {
	case EventStoryCreated:
		return ps.applyStoryCreated(event)
	case EventStoryStatusChanged:
		return ps.applyStoryStatusChanged(event)
	case EventAgentAssigned:
		return ps.applyAgentAssigned(event)
	case EventAgentStatusChanged:
		return ps.applyAgentStatusChanged(event)
	case EventRequirementCreated:
		return ps.applyRequirementCreated(event)
	case EventEscalationCreated:
		return ps.applyEscalationCreated(event)
	case EventPipelineRunStarted:
		return ps.applyPipelineRunStarted(event)
	case EventPipelineRunUpdated:
		return ps.applyPipelineRunUpdated(event)
	case EventTokenUsageRecorded:
		return ps.applyTokenUsageRecorded(event)
	case EventSessionHealthChanged:
		return ps.applySessionHealthChanged(event)
	default:
		// Store event in events table for unknown types — forward-compatible
		return ps.storeEvent(event)
	}
}

func (ps *ProjectionStore) storeEvent(event Event) error {
	_, err := ps.writer.Exec(
		"INSERT OR IGNORE INTO events (id, type, payload, created_at) VALUES (?, ?, ?, ?)",
		event.ID, string(event.Type), string(event.Payload), event.Timestamp,
	)
	return err
}

func (ps *ProjectionStore) applyStoryCreated(event Event) error {
	p, err := DecodePayload[StoryCreatedPayload](event)
	if err != nil {
		return err
	}
	ownedFiles, _ := json.Marshal(p.OwnedFiles)
	dependsOn, _ := json.Marshal(p.DependsOn)

	_, err = ps.writer.Exec(
		`INSERT OR IGNORE INTO stories (id, req_id, title, description, acceptance_criteria, owned_files, complexity, depends_on, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'planned', ?, ?)`,
		p.StoryID, p.RequirementID, p.Title, p.Description, p.AcceptanceCriteria,
		string(ownedFiles), p.Complexity, string(dependsOn),
		event.Timestamp, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert story: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyStoryStatusChanged(event Event) error {
	p, err := DecodePayload[StoryStatusChangedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		"UPDATE stories SET status = ?, updated_at = ? WHERE id = ?",
		p.NewStatus, event.Timestamp, p.StoryID,
	)
	if err != nil {
		return fmt.Errorf("update story status: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyAgentAssigned(event Event) error {
	p, err := DecodePayload[AgentAssignedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT INTO agents (id, role, status, current_story, session_name, runtime, created_at, updated_at)
		 VALUES (?, ?, 'working', ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   role = excluded.role, status = 'working', current_story = excluded.current_story,
		   session_name = excluded.session_name, runtime = excluded.runtime, updated_at = excluded.updated_at`,
		p.AgentID, p.Role, p.StoryID, p.Session, p.Runtime,
		event.Timestamp, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}

	// Update story's agent_id
	_, err = ps.writer.Exec(
		"UPDATE stories SET agent_id = ?, updated_at = ? WHERE id = ?",
		p.AgentID, event.Timestamp, p.StoryID,
	)
	if err != nil {
		return fmt.Errorf("update story agent: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyAgentStatusChanged(event Event) error {
	p, err := DecodePayload[AgentStatusChangedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		"UPDATE agents SET status = ?, updated_at = ? WHERE id = ?",
		p.NewStatus, event.Timestamp, p.AgentID,
	)
	if err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyRequirementCreated(event Event) error {
	p, err := DecodePayload[RequirementCreatedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT OR IGNORE INTO requirements (id, title, description, source, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'draft', ?, ?)`,
		p.RequirementID, p.Title, p.Description, p.Source,
		event.Timestamp, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert requirement: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyEscalationCreated(event Event) error {
	p, err := DecodePayload[EscalationCreatedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT OR IGNORE INTO escalations (id, story_id, reason, from_role, to_role, status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'open', ?)`,
		p.EscalationID, p.StoryID, p.Reason, p.FromRole, p.ToRole, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert escalation: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyPipelineRunStarted(event Event) error {
	p, err := DecodePayload[PipelineRunStartedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT INTO pipeline_runs (id, story_id, stage, status, attempt, started_at)
		 VALUES (?, ?, ?, 'running', ?, ?)`,
		event.ID, p.StoryID, p.Stage, p.Attempt, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline run: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyPipelineRunUpdated(event Event) error {
	p, err := DecodePayload[PipelineRunUpdatedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`UPDATE pipeline_runs SET status = ?, error = ?, ended_at = ?, duration_ms = ?
		 WHERE story_id = ? AND stage = ? AND attempt = ? AND status = 'running'`,
		p.Status, p.Error, event.Timestamp, p.DurationMs,
		p.StoryID, p.Stage, p.Attempt,
	)
	if err != nil {
		return fmt.Errorf("update pipeline run: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applyTokenUsageRecorded(event Event) error {
	p, err := DecodePayload[TokenUsageRecordedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT INTO token_usage (id, story_id, req_id, agent_id, model, input_tokens, output_tokens, cost_usd, stage, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, p.StoryID, p.RequirementID, p.AgentID, p.Model,
		p.InputTokens, p.OutputTokens, p.CostUSD, p.Stage, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert token usage: %w", err)
	}
	return ps.storeEvent(event)
}

func (ps *ProjectionStore) applySessionHealthChanged(event Event) error {
	p, err := DecodePayload[SessionHealthChangedPayload](event)
	if err != nil {
		return err
	}
	_, err = ps.writer.Exec(
		`INSERT INTO session_health (session_name, status, pane_pid, recovery_attempts, last_check_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(session_name) DO UPDATE SET
		   status = excluded.status, pane_pid = excluded.pane_pid,
		   recovery_attempts = excluded.recovery_attempts,
		   last_check_at = excluded.last_check_at, updated_at = excluded.updated_at`,
		p.SessionName, p.NewStatus, p.PanePID, p.RecoveryAttempt,
		event.Timestamp, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upsert session health: %w", err)
	}
	return ps.storeEvent(event)
}
