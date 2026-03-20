package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsWhenNoFile(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected defaults when no file: %v", err)
	}
	if cfg.Workspace.Backend != "sqlite" {
		t.Errorf("expected sqlite backend, got %q", cfg.Workspace.Backend)
	}
}

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.yaml")
	err := os.WriteFile(path, []byte(`
workspace:
  state_dir: /tmp/px-test
  backend: sqlite
  log_level: debug
routing:
  junior_max_complexity: 2
  intermediate_max_complexity: 5
`), 0o644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.Workspace.LogLevel != "debug" {
		t.Errorf("expected debug log level, got %q", cfg.Workspace.LogLevel)
	}
	if cfg.Routing.JuniorMaxComplexity != 2 {
		t.Errorf("expected junior complexity 2, got %d", cfg.Routing.JuniorMaxComplexity)
	}
}

func TestLoad_OverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.yaml")
	err := os.WriteFile(path, []byte(`
workspace:
  backend: dolt
budget:
  max_cost_per_story_usd: 5.0
`), 0o644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.Workspace.Backend != "dolt" {
		t.Errorf("expected dolt backend, got %q", cfg.Workspace.Backend)
	}
	if cfg.Budget.MaxCostPerStoryUSD != 5.0 {
		t.Errorf("expected max cost 5.0, got %f", cfg.Budget.MaxCostPerStoryUSD)
	}
	// Non-overridden defaults should still be present
	if cfg.Budget.HardStop != true {
		t.Error("expected HardStop default true to be preserved")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.yaml")
	err := os.WriteFile(path, []byte(`{invalid yaml`), 0o644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "px.yaml")
	err := os.WriteFile(path, []byte(`
workspace:
  backend: mysql
`), 0o644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected validation error for invalid backend")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/px.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFindConfigFile_NoFile(t *testing.T) {
	// Save and change working directory to a temp dir with no config
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	result := FindConfigFile()
	if result != "" {
		t.Errorf("expected empty string when no config found, got %q", result)
	}
}

func TestFindConfigFile_PxYaml(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var.
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	path := filepath.Join(dir, "px.yaml")
	if err := os.WriteFile(path, []byte("version: '1'"), 0o644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	result := FindConfigFile()
	if result != path {
		t.Errorf("expected %q, got %q", path, result)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", home},
	}

	for _, tt := range tests {
		got := expandHome(tt.input)
		if got != tt.expected {
			t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
