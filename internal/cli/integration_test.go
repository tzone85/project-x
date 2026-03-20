package cli

import (
	"path/filepath"
	"testing"

	"github.com/tzone85/project-x/internal/state"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a temporary projection store and returns the data dir.
func setupTestDB(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "projections.db")
	store, err := state.NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("create projection store: %v", err)
	}
	store.Close()
	return tmpDir
}

func TestRunCostDefault(t *testing.T) {
	dataDir := setupTestDB(t)
	err := runCost(dataDir, "", "", false)
	if err != nil {
		t.Fatalf("runCost default failed: %v", err)
	}
}

func TestRunCostDaily(t *testing.T) {
	dataDir := setupTestDB(t)
	err := runCost(dataDir, "", "", true)
	if err != nil {
		t.Fatalf("runCost daily failed: %v", err)
	}
}

func TestRunCostByStory(t *testing.T) {
	dataDir := setupTestDB(t)
	err := runCost(dataDir, "story-1", "", false)
	if err != nil {
		t.Fatalf("runCost by story failed: %v", err)
	}
}

func TestRunCostByRequirement(t *testing.T) {
	dataDir := setupTestDB(t)
	err := runCost(dataDir, "", "req-1", false)
	if err != nil {
		t.Fatalf("runCost by requirement failed: %v", err)
	}
}

func TestRunCostBadDB(t *testing.T) {
	err := runCost("/nonexistent/dir", "", "", false)
	if err == nil {
		t.Error("expected error with bad DB path")
	}
}

func TestRunStatus(t *testing.T) {
	dataDir := setupTestDB(t)
	err := runStatus(dataDir)
	if err != nil {
		t.Fatalf("runStatus failed: %v", err)
	}
}

func TestRunStatusBadDB(t *testing.T) {
	// Using a non-existent path should fail when opening the DB
	err := runStatus("/nonexistent/dir")
	if err == nil {
		t.Error("expected error with bad DB path")
	}
}

func TestOpenReadOnly(t *testing.T) {
	dataDir := setupTestDB(t)
	dbPath := filepath.Join(dataDir, "projections.db")
	queries, err := openReadOnly(dbPath)
	if err != nil {
		t.Fatalf("openReadOnly failed: %v", err)
	}
	defer queries.Close()
}

func TestRunMigrateBadDir(t *testing.T) {
	// This should fail because the path component doesn't exist
	err := runMigrate("/nonexistent/deep/nested/dir")
	if err == nil {
		t.Error("expected error with bad data dir")
	}
}
