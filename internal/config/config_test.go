package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Budget.MaxCostPerStoryUSD != 2.00 {
		t.Errorf("expected MaxCostPerStoryUSD=2.00, got %f", cfg.Budget.MaxCostPerStoryUSD)
	}
	if cfg.Budget.WarningThresholdPct != 80 {
		t.Errorf("expected WarningThresholdPct=80, got %d", cfg.Budget.WarningThresholdPct)
	}
	if cfg.Budget.HardStop != true {
		t.Error("expected HardStop=true")
	}
	if cfg.Sessions.StaleThresholdS != 180 {
		t.Errorf("expected StaleThresholdS=180, got %d", cfg.Sessions.StaleThresholdS)
	}
	if cfg.Routing.Strategy != "cost_optimized" {
		t.Errorf("expected Strategy=cost_optimized, got %s", cfg.Routing.Strategy)
	}
	if cfg.Workspace.LogLevel != "info" {
		t.Errorf("expected LogLevel=info, got %s", cfg.Workspace.LogLevel)
	}
	if len(cfg.Pricing) != 3 {
		t.Errorf("expected 3 pricing entries, got %d", len(cfg.Pricing))
	}
	if len(cfg.Pipeline.Stages) != 4 {
		t.Errorf("expected 4 pipeline stages, got %d", len(cfg.Pipeline.Stages))
	}
	if cfg.Planning.MaxStoryComplexity != 8 {
		t.Errorf("expected MaxStoryComplexity=8, got %d", cfg.Planning.MaxStoryComplexity)
	}
}

func TestDefaultValidates(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid, got: %v", err)
	}
}

func TestLoadNonExistentFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/px.config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Budget.MaxCostPerStoryUSD != 2.00 {
		t.Errorf("expected defaults for missing file, got MaxCostPerStoryUSD=%f", cfg.Budget.MaxCostPerStoryUSD)
	}
}

func TestLoadPartialOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.config.yaml")

	content := []byte(`
budget:
  max_cost_per_story_usd: 5.00
  max_cost_per_day_usd: 100.00
workspace:
  log_level: debug
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden values
	if cfg.Budget.MaxCostPerStoryUSD != 5.00 {
		t.Errorf("expected 5.00, got %f", cfg.Budget.MaxCostPerStoryUSD)
	}
	if cfg.Budget.MaxCostPerDayUSD != 100.00 {
		t.Errorf("expected 100.00, got %f", cfg.Budget.MaxCostPerDayUSD)
	}
	if cfg.Workspace.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.Workspace.LogLevel)
	}

	// Defaults preserved for non-overridden values
	if cfg.Budget.WarningThresholdPct != 80 {
		t.Errorf("expected default 80, got %d", cfg.Budget.WarningThresholdPct)
	}
	if cfg.Sessions.StaleThresholdS != 180 {
		t.Errorf("expected default 180, got %d", cfg.Sessions.StaleThresholdS)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.config.yaml")

	content := []byte(`
budget:
  max_cost_per_story_usd: -1
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for negative budget")
	}
}
