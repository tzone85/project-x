// Package config handles loading, validation, and defaults for px configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for px.
type Config struct {
	Budget    BudgetConfig    `yaml:"budget"`
	Sessions  SessionsConfig  `yaml:"sessions"`
	Pipeline  PipelineConfig  `yaml:"pipeline"`
	Planning  PlanningConfig  `yaml:"planning"`
	Routing   RoutingConfig   `yaml:"routing"`
	Pricing   PricingMap      `yaml:"pricing"`
	Workspace WorkspaceConfig `yaml:"workspace"`
}

// BudgetConfig defines cost protection limits.
type BudgetConfig struct {
	MaxCostPerStoryUSD       float64 `yaml:"max_cost_per_story_usd"`
	MaxCostPerRequirementUSD float64 `yaml:"max_cost_per_requirement_usd"`
	MaxCostPerDayUSD         float64 `yaml:"max_cost_per_day_usd"`
	WarningThresholdPct      int     `yaml:"warning_threshold_pct"`
	HardStop                 bool    `yaml:"hard_stop"`
}

// SessionsConfig defines tmux session health and recovery settings.
type SessionsConfig struct {
	StaleThresholdS     int    `yaml:"stale_threshold_s"`
	OnDead              string `yaml:"on_dead"`
	OnStale             string `yaml:"on_stale"`
	MaxRecoveryAttempts int    `yaml:"max_recovery_attempts"`
}

// StageRetryConfig defines retry policy for a single pipeline stage.
type StageRetryConfig struct {
	MaxRetries int    `yaml:"max_retries"`
	OnExhaust  string `yaml:"on_exhaust"`
}

// PipelineConfig defines pipeline stage retry policies.
type PipelineConfig struct {
	Stages map[string]StageRetryConfig `yaml:"stages"`
}

// PlanningConfig defines planner quality criteria.
type PlanningConfig struct {
	RequiredFields          []string `yaml:"required_fields"`
	MaxStoryComplexity      int      `yaml:"max_story_complexity"`
	MaxStoriesPerRequirement int     `yaml:"max_stories_per_requirement"`
	EnforceFileOwnership    bool     `yaml:"enforce_file_ownership"`
}

// RoutingPreference defines runtime routing for a specific agent role.
type RoutingPreference struct {
	Role     string `yaml:"role"`
	Prefer   string `yaml:"prefer"`
	Fallback string `yaml:"fallback"`
}

// RoutingConfig defines the runtime routing strategy.
type RoutingConfig struct {
	Strategy    string              `yaml:"strategy"`
	Preferences []RoutingPreference `yaml:"preferences"`
}

// ModelPricing holds per-model token pricing.
type ModelPricing struct {
	InputPer1M  float64 `yaml:"input_per_1m"`
	OutputPer1M float64 `yaml:"output_per_1m"`
}

// PricingMap maps model identifiers to their pricing.
type PricingMap map[string]ModelPricing

// WorkspaceConfig defines workspace-level settings.
type WorkspaceConfig struct {
	LogLevel string `yaml:"log_level"`
	DataDir  string `yaml:"data_dir"`
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		Budget: BudgetConfig{
			MaxCostPerStoryUSD:       2.00,
			MaxCostPerRequirementUSD: 20.00,
			MaxCostPerDayUSD:         50.00,
			WarningThresholdPct:      80,
			HardStop:                 true,
		},
		Sessions: SessionsConfig{
			StaleThresholdS:     180,
			OnDead:              "redispatch",
			OnStale:             "restart",
			MaxRecoveryAttempts: 2,
		},
		Pipeline: PipelineConfig{
			Stages: map[string]StageRetryConfig{
				"review": {MaxRetries: 2, OnExhaust: "escalate"},
				"qa":     {MaxRetries: 3, OnExhaust: "pause_requirement"},
				"rebase": {MaxRetries: 2, OnExhaust: "pause_requirement"},
				"merge":  {MaxRetries: 1, OnExhaust: "pause_requirement"},
			},
		},
		Planning: PlanningConfig{
			RequiredFields: []string{
				"title", "description", "acceptance_criteria",
				"owned_files", "complexity", "depends_on",
			},
			MaxStoryComplexity:       8,
			MaxStoriesPerRequirement: 15,
			EnforceFileOwnership:     true,
		},
		Routing: RoutingConfig{
			Strategy: "cost_optimized",
			Preferences: []RoutingPreference{
				{Role: "junior", Prefer: "codex", Fallback: "claude-code"},
				{Role: "senior", Prefer: "claude-code", Fallback: "gemini"},
			},
		},
		Pricing: PricingMap{
			"anthropic/claude-opus-4-20250514":   {InputPer1M: 15.00, OutputPer1M: 75.00},
			"anthropic/claude-sonnet-4-20250514": {InputPer1M: 3.00, OutputPer1M: 15.00},
			"openai/gpt-4o-mini":                {InputPer1M: 0.15, OutputPer1M: 0.60},
		},
		Workspace: WorkspaceConfig{
			LogLevel: "info",
			DataDir:  defaultDataDir(),
		},
	}
}

// defaultDataDir returns ~/.px as the default data directory.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".px"
	}
	return filepath.Join(home, ".px")
}

// Load reads a config file and merges it over defaults.
// If the file does not exist, defaults are returned without error.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// DefaultConfigPath returns the default config file search path.
func DefaultConfigPath() string {
	return "px.config.yaml"
}
