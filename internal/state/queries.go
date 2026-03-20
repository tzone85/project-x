package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// --- Requirement Queries ---

func (ps *ProjectionStore) GetRequirement(id string) (*Requirement, error) {
	return getRequirement(ps.writer, id)
}

func (ps *ProjectionStore) ListRequirements(page PageParams) ([]Requirement, error) {
	return listRequirements(ps.writer, page)
}

// --- Story Queries ---

func (ps *ProjectionStore) GetStory(id string) (*Story, error) {
	return getStory(ps.writer, id)
}

func (ps *ProjectionStore) ListStories(page PageParams) ([]Story, error) {
	return listStories(ps.writer, page)
}

func (ps *ProjectionStore) ListStoriesByRequirement(reqID string, page PageParams) ([]Story, error) {
	return listStoriesByRequirement(ps.writer, reqID, page)
}

// --- Agent Queries ---

func (ps *ProjectionStore) GetAgent(id string) (*Agent, error) {
	return getAgent(ps.writer, id)
}

func (ps *ProjectionStore) ListAgents(page PageParams) ([]Agent, error) {
	return listAgents(ps.writer, page)
}

// --- Escalation Queries ---

func (ps *ProjectionStore) ListEscalations(page PageParams) ([]Escalation, error) {
	return listEscalations(ps.writer, page)
}

// --- TokenUsage Queries ---

func (ps *ProjectionStore) ListTokenUsage(page PageParams) ([]TokenUsage, error) {
	return listTokenUsage(ps.writer, page)
}

func (ps *ProjectionStore) GetStoryTotalCost(storyID string) (float64, error) {
	return getStoryTotalCost(ps.writer, storyID)
}

func (ps *ProjectionStore) GetRequirementTotalCost(reqID string) (float64, error) {
	return getRequirementTotalCost(ps.writer, reqID)
}

func (ps *ProjectionStore) GetDailyTotalCost(date time.Time) (float64, error) {
	return getDailyTotalCost(ps.writer, date)
}

// --- PipelineRun Queries ---

func (ps *ProjectionStore) ListPipelineRuns(page PageParams) ([]PipelineRun, error) {
	return listPipelineRuns(ps.writer, page)
}

// --- Event Queries ---

func (ps *ProjectionStore) ListEvents(page PageParams) ([]Event, error) {
	return listEvents(ps.writer, page)
}

// --- Read-only query functions (used by both writer and read-only connections) ---

func getRequirement(db *sql.DB, id string) (*Requirement, error) {
	r := &Requirement{}
	err := db.QueryRow(
		"SELECT id, title, description, source, status, created_at, updated_at FROM requirements WHERE id = ?", id,
	).Scan(&r.ID, &r.Title, &r.Description, &r.Source, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get requirement: %w", err)
	}
	return r, nil
}

func listRequirements(db *sql.DB, page PageParams) ([]Requirement, error) {
	page = page.Normalize()
	rows, err := db.Query(
		"SELECT id, title, description, source, status, created_at, updated_at FROM requirements ORDER BY created_at DESC LIMIT ? OFFSET ?",
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list requirements: %w", err)
	}
	defer rows.Close()

	var result []Requirement
	for rows.Next() {
		var r Requirement
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Source, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan requirement: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func getStory(db *sql.DB, id string) (*Story, error) {
	s := &Story{}
	var ownedFiles, dependsOn string
	err := db.QueryRow(
		`SELECT id, req_id, title, description, acceptance_criteria, owned_files, complexity,
		        depends_on, status, COALESCE(agent_id,''), wave, created_at, updated_at
		 FROM stories WHERE id = ?`, id,
	).Scan(&s.ID, &s.ReqID, &s.Title, &s.Description, &s.AcceptanceCriteria,
		&ownedFiles, &s.Complexity, &dependsOn, &s.Status, &s.AgentID, &s.Wave,
		&s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get story: %w", err)
	}
	json.Unmarshal([]byte(ownedFiles), &s.OwnedFiles)
	json.Unmarshal([]byte(dependsOn), &s.DependsOn)
	return s, nil
}

func listStories(db *sql.DB, page PageParams) ([]Story, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, req_id, title, description, acceptance_criteria, owned_files, complexity,
		        depends_on, status, COALESCE(agent_id,''), wave, created_at, updated_at
		 FROM stories ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list stories: %w", err)
	}
	defer rows.Close()
	return scanStories(rows)
}

func listStoriesByRequirement(db *sql.DB, reqID string, page PageParams) ([]Story, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, req_id, title, description, acceptance_criteria, owned_files, complexity,
		        depends_on, status, COALESCE(agent_id,''), wave, created_at, updated_at
		 FROM stories WHERE req_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		reqID, page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list stories by req: %w", err)
	}
	defer rows.Close()
	return scanStories(rows)
}

func scanStories(rows *sql.Rows) ([]Story, error) {
	var result []Story
	for rows.Next() {
		var s Story
		var ownedFiles, dependsOn string
		if err := rows.Scan(&s.ID, &s.ReqID, &s.Title, &s.Description, &s.AcceptanceCriteria,
			&ownedFiles, &s.Complexity, &dependsOn, &s.Status, &s.AgentID, &s.Wave,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan story: %w", err)
		}
		json.Unmarshal([]byte(ownedFiles), &s.OwnedFiles)
		json.Unmarshal([]byte(dependsOn), &s.DependsOn)
		result = append(result, s)
	}
	return result, rows.Err()
}

func getAgent(db *sql.DB, id string) (*Agent, error) {
	a := &Agent{}
	err := db.QueryRow(
		`SELECT id, role, status, COALESCE(current_story,''), COALESCE(session_name,''),
		        COALESCE(runtime,''), created_at, updated_at
		 FROM agents WHERE id = ?`, id,
	).Scan(&a.ID, &a.Role, &a.Status, &a.CurrentStory, &a.SessionName, &a.Runtime,
		&a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

func listAgents(db *sql.DB, page PageParams) ([]Agent, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, role, status, COALESCE(current_story,''), COALESCE(session_name,''),
		        COALESCE(runtime,''), created_at, updated_at
		 FROM agents ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var result []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Role, &a.Status, &a.CurrentStory, &a.SessionName, &a.Runtime,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func listEscalations(db *sql.DB, page PageParams) ([]Escalation, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, story_id, reason, from_role, to_role, status, created_at, resolved_at
		 FROM escalations ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list escalations: %w", err)
	}
	defer rows.Close()

	var result []Escalation
	for rows.Next() {
		var e Escalation
		if err := rows.Scan(&e.ID, &e.StoryID, &e.Reason, &e.FromRole, &e.ToRole, &e.Status,
			&e.CreatedAt, &e.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan escalation: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func listTokenUsage(db *sql.DB, page PageParams) ([]TokenUsage, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, story_id, req_id, agent_id, model, input_tokens, output_tokens, cost_usd, stage, created_at
		 FROM token_usage ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list token usage: %w", err)
	}
	defer rows.Close()

	var result []TokenUsage
	for rows.Next() {
		var t TokenUsage
		if err := rows.Scan(&t.ID, &t.StoryID, &t.ReqID, &t.AgentID, &t.Model,
			&t.InputTokens, &t.OutputTokens, &t.CostUSD, &t.Stage, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan token usage: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func getStoryTotalCost(db *sql.DB, storyID string) (float64, error) {
	var total float64
	err := db.QueryRow("SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE story_id = ?", storyID).Scan(&total)
	return total, err
}

func getRequirementTotalCost(db *sql.DB, reqID string) (float64, error) {
	var total float64
	err := db.QueryRow("SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE req_id = ?", reqID).Scan(&total)
	return total, err
}

func getDailyTotalCost(db *sql.DB, date time.Time) (float64, error) {
	dateStr := date.Format("2006-01-02")
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE date(created_at) = ?", dateStr,
	).Scan(&total)
	return total, err
}

func listPipelineRuns(db *sql.DB, page PageParams) ([]PipelineRun, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, story_id, stage, status, attempt, error, started_at, ended_at, duration_ms
		 FROM pipeline_runs ORDER BY started_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline runs: %w", err)
	}
	defer rows.Close()

	var result []PipelineRun
	for rows.Next() {
		var p PipelineRun
		if err := rows.Scan(&p.ID, &p.StoryID, &p.Stage, &p.Status, &p.Attempt,
			&p.Error, &p.StartedAt, &p.EndedAt, &p.DurationMs); err != nil {
			return nil, fmt.Errorf("scan pipeline run: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func listEvents(db *sql.DB, page PageParams) ([]Event, error) {
	page = page.Normalize()
	rows, err := db.Query(
		`SELECT id, type, payload, created_at FROM events ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var result []Event
	for rows.Next() {
		var e Event
		var payload string
		if err := rows.Scan(&e.ID, &e.Type, &payload, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Payload = json.RawMessage(payload)
		result = append(result, e)
	}
	return result, rows.Err()
}

// ReadOnlyQueries provides read-only query methods against a read-only connection.
type ReadOnlyQueries struct {
	db *sql.DB
}

// NewReadOnlyQueries wraps a read-only connection with query methods.
func NewReadOnlyQueries(db *sql.DB) *ReadOnlyQueries {
	return &ReadOnlyQueries{db: db}
}

func (ro *ReadOnlyQueries) GetRequirement(id string) (*Requirement, error) {
	return getRequirement(ro.db, id)
}

func (ro *ReadOnlyQueries) ListRequirements(page PageParams) ([]Requirement, error) {
	return listRequirements(ro.db, page)
}

func (ro *ReadOnlyQueries) GetStory(id string) (*Story, error) {
	return getStory(ro.db, id)
}

func (ro *ReadOnlyQueries) ListStories(page PageParams) ([]Story, error) {
	return listStories(ro.db, page)
}

func (ro *ReadOnlyQueries) ListStoriesByRequirement(reqID string, page PageParams) ([]Story, error) {
	return listStoriesByRequirement(ro.db, reqID, page)
}

func (ro *ReadOnlyQueries) GetAgent(id string) (*Agent, error) {
	return getAgent(ro.db, id)
}

func (ro *ReadOnlyQueries) ListAgents(page PageParams) ([]Agent, error) {
	return listAgents(ro.db, page)
}

func (ro *ReadOnlyQueries) ListEscalations(page PageParams) ([]Escalation, error) {
	return listEscalations(ro.db, page)
}

func (ro *ReadOnlyQueries) ListTokenUsage(page PageParams) ([]TokenUsage, error) {
	return listTokenUsage(ro.db, page)
}

func (ro *ReadOnlyQueries) GetStoryTotalCost(storyID string) (float64, error) {
	return getStoryTotalCost(ro.db, storyID)
}

func (ro *ReadOnlyQueries) GetRequirementTotalCost(reqID string) (float64, error) {
	return getRequirementTotalCost(ro.db, reqID)
}

func (ro *ReadOnlyQueries) GetDailyTotalCost(date time.Time) (float64, error) {
	return getDailyTotalCost(ro.db, date)
}

func (ro *ReadOnlyQueries) ListPipelineRuns(page PageParams) ([]PipelineRun, error) {
	return listPipelineRuns(ro.db, page)
}

func (ro *ReadOnlyQueries) ListEvents(page PageParams) ([]Event, error) {
	return listEvents(ro.db, page)
}

func (ro *ReadOnlyQueries) Close() error {
	return ro.db.Close()
}
