package config

import (
	"strings"
	"testing"
)

func validConfig() Config {
	return Default()
}

func TestValidateBudgetNegativeValues(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
		errMsg string
	}{
		{
			name:   "negative story cost",
			modify: func(c *Config) { c.Budget.MaxCostPerStoryUSD = -1 },
			errMsg: "max_cost_per_story_usd must be > 0",
		},
		{
			name:   "zero requirement cost",
			modify: func(c *Config) { c.Budget.MaxCostPerRequirementUSD = 0 },
			errMsg: "max_cost_per_requirement_usd must be > 0",
		},
		{
			name:   "zero daily cost",
			modify: func(c *Config) { c.Budget.MaxCostPerDayUSD = 0 },
			errMsg: "max_cost_per_day_usd must be > 0",
		},
		{
			name:   "warning threshold too high",
			modify: func(c *Config) { c.Budget.WarningThresholdPct = 101 },
			errMsg: "warning_threshold_pct must be between 1 and 100",
		},
		{
			name:   "warning threshold zero",
			modify: func(c *Config) { c.Budget.WarningThresholdPct = 0 },
			errMsg: "warning_threshold_pct must be between 1 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateSessionsPolicies(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
		errMsg string
	}{
		{
			name:   "invalid on_dead",
			modify: func(c *Config) { c.Sessions.OnDead = "explode" },
			errMsg: "sessions.on_dead must be one of",
		},
		{
			name:   "invalid on_stale",
			modify: func(c *Config) { c.Sessions.OnStale = "panic" },
			errMsg: "sessions.on_stale must be one of",
		},
		{
			name:   "negative stale threshold",
			modify: func(c *Config) { c.Sessions.StaleThresholdS = -1 },
			errMsg: "stale_threshold_s must be > 0",
		},
		{
			name:   "negative recovery attempts",
			modify: func(c *Config) { c.Sessions.MaxRecoveryAttempts = -1 },
			errMsg: "max_recovery_attempts must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidatePipelineOnExhaust(t *testing.T) {
	cfg := validConfig()
	cfg.Pipeline.Stages["review"] = StageRetryConfig{
		MaxRetries: 2,
		OnExhaust:  "self_destruct",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid on_exhaust")
	}
	if !strings.Contains(err.Error(), "on_exhaust must be one of") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePipelineNegativeRetries(t *testing.T) {
	cfg := validConfig()
	cfg.Pipeline.Stages["qa"] = StageRetryConfig{
		MaxRetries: -1,
		OnExhaust:  "pause_requirement",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for negative retries")
	}
	if !strings.Contains(err.Error(), "max_retries must be >= 0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateRoutingStrategy(t *testing.T) {
	cfg := validConfig()
	cfg.Routing.Strategy = "random"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid strategy")
	}
	if !strings.Contains(err.Error(), "routing.strategy must be one of") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceLogLevel(t *testing.T) {
	cfg := validConfig()
	cfg.Workspace.LogLevel = "verbose"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid log level")
	}
	if !strings.Contains(err.Error(), "workspace.log_level must be one of") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePricingNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Pricing["test-model"] = ModelPricing{InputPer1M: -1, OutputPer1M: 5}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for negative pricing")
	}
	if !strings.Contains(err.Error(), "input_per_1m must be >= 0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePlanningBounds(t *testing.T) {
	cfg := validConfig()
	cfg.Planning.MaxStoryComplexity = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "max_story_complexity must be > 0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.Budget.MaxCostPerStoryUSD = -1
	cfg.Workspace.LogLevel = "invalid"
	cfg.Routing.Strategy = "invalid"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "max_cost_per_story_usd") {
		t.Error("expected budget error in combined output")
	}
	if !strings.Contains(errStr, "log_level") {
		t.Error("expected log level error in combined output")
	}
	if !strings.Contains(errStr, "routing.strategy") {
		t.Error("expected routing strategy error in combined output")
	}
}
