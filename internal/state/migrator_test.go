package state

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrator_AppliesAllMigrations(t *testing.T) {
	db := openTestDB(t)
	n, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 migrations applied, got %d", n)
	}

	// Verify core tables exist
	tables := []string{
		"requirements", "stories", "agents", "story_deps",
		"escalations", "agent_scores", "token_usage",
		"session_health", "pipeline_runs",
	}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestMigrator_SkipsAlreadyApplied(t *testing.T) {
	db := openTestDB(t)
	n1, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("first migration run: %v", err)
	}
	n2, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("second migration run: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0 migrations on second run, got %d (first run applied %d)", n2, n1)
	}
}

func TestMigrator_TracksVersions(t *testing.T) {
	db := openTestDB(t)
	_, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("query versions: %v", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan version: %v", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	expected := []int{1, 2, 3, 4, 5}
	if len(versions) != len(expected) {
		t.Fatalf("expected %d versions, got %d", len(expected), len(versions))
	}
	for i, v := range versions {
		if v != expected[i] {
			t.Errorf("version[%d]: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestMigrator_VerifiesIndexes(t *testing.T) {
	db := openTestDB(t)
	_, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	indexes := []string{
		"idx_stories_req_id",
		"idx_stories_status",
		"idx_stories_req_status",
		"idx_agents_status",
		"idx_escalations_story_id",
		"idx_token_usage_story_id",
		"idx_token_usage_req_id",
		"idx_token_usage_date",
		"idx_pipeline_runs_story_id",
	}
	for _, idx := range indexes {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}
}

func TestMigrator_SchemaColumnsExist(t *testing.T) {
	db := openTestDB(t)
	_, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Verify token_usage columns
	_, err = db.Exec(`INSERT INTO token_usage
		(id, req_id, story_id, agent_id, model, input_tokens, output_tokens, cost_usd, stage)
		VALUES ('t1', 'r1', 's1', 'a1', 'gpt-4', 100, 50, 0.01, 'plan')`)
	if err != nil {
		t.Errorf("insert token_usage: %v", err)
	}

	// Verify session_health columns
	_, err = db.Exec(`INSERT INTO session_health
		(session_name, status, pane_pid, last_output_hash, recovery_attempts)
		VALUES ('sess1', 'healthy', 1234, 'abc123', 0)`)
	if err != nil {
		t.Errorf("insert session_health: %v", err)
	}

	// Verify pipeline_runs columns
	_, err = db.Exec(`INSERT INTO pipeline_runs
		(id, story_id, stage, status, attempt, error_message)
		VALUES ('pr1', 's1', 'build', 'pending', 1, '')`)
	if err != nil {
		t.Errorf("insert pipeline_runs: %v", err)
	}
}
