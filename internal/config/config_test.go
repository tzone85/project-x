package config

import "testing"

func TestValidate_ValidConfig(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestValidate_InvalidBackend(t *testing.T) {
	cfg := Defaults()
	cfg.Workspace.Backend = "mysql"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid backend")
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := Defaults()
	cfg.Workspace.LogLevel = "trace"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestValidate_InvalidWorktreePrune(t *testing.T) {
	cfg := Defaults()
	cfg.Cleanup.WorktreePrune = "never"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid worktree prune value")
	}
}

func TestValidate_JuniorComplexityTooHigh(t *testing.T) {
	cfg := Defaults()
	cfg.Routing.JuniorMaxComplexity = 20
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for out-of-range complexity")
	}
}

func TestValidate_JuniorComplexityTooLow(t *testing.T) {
	cfg := Defaults()
	cfg.Routing.JuniorMaxComplexity = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for out-of-range complexity")
	}
}

func TestValidate_IntermediateBelowJunior(t *testing.T) {
	cfg := Defaults()
	cfg.Routing.JuniorMaxComplexity = 5
	cfg.Routing.IntermediateMaxComplexity = 3
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when intermediate < junior")
	}
}

func TestValidate_IntermediateAbove13(t *testing.T) {
	cfg := Defaults()
	cfg.Routing.IntermediateMaxComplexity = 14
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when intermediate > 13")
	}
}

func TestValidate_BudgetLimits(t *testing.T) {
	cfg := Defaults()
	cfg.Budget.MaxCostPerStoryUSD = -1.0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative budget")
	}
}

func TestValidate_NegativeBudgetPerRequirement(t *testing.T) {
	cfg := Defaults()
	cfg.Budget.MaxCostPerRequirementUSD = -5.0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative requirement budget")
	}
}

func TestValidate_NegativeBudgetPerDay(t *testing.T) {
	cfg := Defaults()
	cfg.Budget.MaxCostPerDayUSD = -10.0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative daily budget")
	}
}

func TestValidate_NegativeWarningThreshold(t *testing.T) {
	cfg := Defaults()
	cfg.Budget.WarningThresholdPct = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative warning threshold")
	}
}

func TestValidate_InvalidSessionOnDead(t *testing.T) {
	cfg := Defaults()
	cfg.Sessions.OnDead = "ignore"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid on_dead value")
	}
}

func TestValidate_InvalidSessionOnStale(t *testing.T) {
	cfg := Defaults()
	cfg.Sessions.OnStale = "ignore"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid on_stale value")
	}
}

func TestDefaults_ReturnsNewInstance(t *testing.T) {
	cfg1 := Defaults()
	cfg2 := Defaults()
	cfg1.Workspace.Backend = "dolt"
	if cfg2.Workspace.Backend == "dolt" {
		t.Fatal("Defaults() should return independent copies")
	}
}

func TestValidate_RuntimePreferenceProviderMismatch(t *testing.T) {
	cfg := Defaults()
	cfg.Models.Junior = ModelConfig{
		Provider: "anthropic",
		Model:    "claude-haiku-4-20250414",
	}
	cfg.Routing.Preferences = []RoutingPreference{
		{Role: "junior", Prefer: "codex"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for mismatched runtime/model provider")
	}
}

func TestValidate_RuntimePreferenceProviderMatch(t *testing.T) {
	cfg := Defaults()
	cfg.Models.Junior = ModelConfig{
		Provider: "openai",
		Model:    "gpt-5.4",
	}
	cfg.Routing.Preferences = []RoutingPreference{
		{Role: "junior", Prefer: "codex", Fallback: "claude-code"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected fallback mismatch to be rejected")
	}

	cfg.Routing.Preferences = []RoutingPreference{
		{Role: "junior", Prefer: "codex"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected matching provider/runtime config to validate, got %v", err)
	}
}

func TestValidate_FallbackRequiresModels(t *testing.T) {
	cfg := Defaults()
	cfg.Fallback.Enabled = true
	cfg.Fallback.LLMModel = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when fallback.llm_model is empty")
	}
}

func TestValidate_FallbackRequiresRuntimeModel(t *testing.T) {
	cfg := Defaults()
	cfg.Fallback.Enabled = true
	cfg.Fallback.RuntimeModel = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when fallback.runtime_model is empty")
	}
}
