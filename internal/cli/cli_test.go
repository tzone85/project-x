package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tzone85/project-x/internal/config"
)

func TestNewInitCmd(t *testing.T) {
	cmd := NewInitCmd()
	if cmd.Use != "init" {
		t.Errorf("expected use 'init', got %q", cmd.Use)
	}
}

func TestRunInit(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "px-data")

	// Run in a temp working dir so config file doesn't pollute cwd
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runInit(dataDir)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check data dir was created
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("data dir was not created")
	}

	// Check logs subdir
	if _, err := os.Stat(filepath.Join(dataDir, "logs")); os.IsNotExist(err) {
		t.Error("logs dir was not created")
	}

	// Check events.jsonl
	if _, err := os.Stat(filepath.Join(dataDir, "events.jsonl")); os.IsNotExist(err) {
		t.Error("events.jsonl was not created")
	}

	// Check config file
	if _, err := os.Stat(filepath.Join(tmpDir, "px.config.yaml")); os.IsNotExist(err) {
		t.Error("px.config.yaml was not created")
	}
}

func TestRunInitIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "px-data")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Run twice — should not error
	if err := runInit(dataDir); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if err := runInit(dataDir); err != nil {
		t.Fatalf("second init failed: %v", err)
	}
}

func testCfgFn() func() config.Config {
	return func() config.Config {
		cfg := config.Default()
		cfg.Workspace.DataDir = "/tmp/px-test-nonexistent"
		return cfg
	}
}

func TestNewMigrateCmd(t *testing.T) {
	cmd := NewMigrateCmd(testCfgFn())
	if cmd.Use != "migrate" {
		t.Errorf("expected use 'migrate', got %q", cmd.Use)
	}
}

func TestNewCostCmd(t *testing.T) {
	cmd := NewCostCmd(testCfgFn())
	if cmd.Use != "cost" {
		t.Errorf("expected use 'cost', got %q", cmd.Use)
	}

	// Verify flags exist
	if cmd.Flags().Lookup("story") == nil {
		t.Error("expected --story flag")
	}
	if cmd.Flags().Lookup("requirement") == nil {
		t.Error("expected --requirement flag")
	}
	if cmd.Flags().Lookup("today") == nil {
		t.Error("expected --today flag")
	}
}

func TestNewStatusCmd(t *testing.T) {
	cmd := NewStatusCmd(testCfgFn())
	if cmd.Use != "status" {
		t.Errorf("expected use 'status', got %q", cmd.Use)
	}
}

func TestNewPlanCmd(t *testing.T) {
	cmd := NewPlanCmd()
	if cmd.Use != "plan [requirement-file]" {
		t.Errorf("expected use 'plan [requirement-file]', got %q", cmd.Use)
	}
	if cmd.Flags().Lookup("review") == nil {
		t.Error("expected --review flag")
	}
	if cmd.Flags().Lookup("refine") == nil {
		t.Error("expected --refine flag")
	}
}

func TestNewResumeCmd(t *testing.T) {
	cmd := NewResumeCmd()
	if cmd.Use != "resume <req-id>" {
		t.Errorf("expected use 'resume <req-id>', got %q", cmd.Use)
	}
}

func TestRunPlanNoArgs(t *testing.T) {
	cmd := NewPlanCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to plan")
	}
}

func TestRunPlanWithFile(t *testing.T) {
	cmd := NewPlanCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"test-req.md"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRunPlanReview(t *testing.T) {
	cmd := NewPlanCmd()
	cmd.SetArgs([]string{"--review", "req-123"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRunPlanRefine(t *testing.T) {
	cmd := NewPlanCmd()
	cmd.SetArgs([]string{"--refine", "req-123"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRunResumeNoArgs(t *testing.T) {
	cmd := NewResumeCmd()
	cmd.SetArgs([]string{})
	// Silence usage on error
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to resume")
	}
}

func TestRunResume(t *testing.T) {
	cmd := NewResumeCmd()
	cmd.SetArgs([]string{"req-123"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRunMigrateWithRealDB(t *testing.T) {
	tmpDir := t.TempDir()
	err := runMigrate(tmpDir)
	if err != nil {
		t.Fatalf("runMigrate failed: %v", err)
	}

	// Verify the DB was created
	dbPath := filepath.Join(tmpDir, "projections.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("projections.db was not created")
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := defaultDataDir()
	if dir == "" {
		t.Error("expected non-empty default data dir")
	}
}
