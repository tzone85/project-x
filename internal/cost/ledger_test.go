package cost

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Create token_usage table matching migration 003
	_, err = db.Exec(`CREATE TABLE token_usage (
		id TEXT PRIMARY KEY,
		req_id TEXT NOT NULL,
		story_id TEXT NOT NULL DEFAULT '',
		agent_id TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL,
		input_tokens INTEGER NOT NULL,
		output_tokens INTEGER NOT NULL,
		cost_usd REAL NOT NULL DEFAULT 0.0,
		stage TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLedger_RecordAndQueryByStory(t *testing.T) {
	db := setupTestDB(t)
	ledger := NewSQLiteLedger(db, DefaultPricing)

	err := ledger.Record(TokenUsage{
		ReqID: "r1", StoryID: "s1", AgentID: "a1",
		Model: "claude-sonnet-4-20250514",
		InputTokens: 1000, OutputTokens: 500,
		Stage: "review",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	total, err := ledger.QueryByStory("s1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if total < 0.01 {
		t.Errorf("expected non-zero cost, got %f", total)
	}
}

func TestLedger_QueryByRequirement(t *testing.T) {
	db := setupTestDB(t)
	ledger := NewSQLiteLedger(db, DefaultPricing)

	ledger.Record(TokenUsage{ReqID: "r1", StoryID: "s1", Model: "gpt-4o-mini", InputTokens: 1000, OutputTokens: 500})
	ledger.Record(TokenUsage{ReqID: "r1", StoryID: "s2", Model: "gpt-4o-mini", InputTokens: 2000, OutputTokens: 1000})
	ledger.Record(TokenUsage{ReqID: "r2", StoryID: "s3", Model: "gpt-4o-mini", InputTokens: 500, OutputTokens: 250})

	total, err := ledger.QueryByRequirement("r1")
	if err != nil {
		t.Fatalf("query r1: %v", err)
	}

	r2Total, err := ledger.QueryByRequirement("r2")
	if err != nil {
		t.Fatalf("query r2: %v", err)
	}

	if total <= r2Total {
		t.Errorf("r1 should have higher cost than r2: r1=%f, r2=%f", total, r2Total)
	}
}

func TestLedger_QueryByDay(t *testing.T) {
	db := setupTestDB(t)
	ledger := NewSQLiteLedger(db, DefaultPricing)

	ledger.Record(TokenUsage{ReqID: "r1", StoryID: "s1", Model: "gpt-4o-mini", InputTokens: 1000, OutputTokens: 500})

	total, err := ledger.QueryByDay(time.Now().Format("2006-01-02"))
	if err != nil {
		t.Fatalf("query by day: %v", err)
	}
	if total == 0 {
		t.Error("expected non-zero daily cost")
	}
}

func TestLedger_QueryEmpty(t *testing.T) {
	db := setupTestDB(t)
	ledger := NewSQLiteLedger(db, DefaultPricing)

	total, err := ledger.QueryByStory("nonexistent")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 for nonexistent story, got %f", total)
	}
}
