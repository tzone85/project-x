package runtime

import (
	"testing"

	"github.com/tzone85/project-x/internal/agent"
	"github.com/tzone85/project-x/internal/config"
)

func TestRouter_CostOptimized_PrefersConfigured(t *testing.T) {
	reg := NewRegistry()
	reg.Register("claude-code", NewClaudeCodeRuntime(false))
	reg.Register("codex", NewCodexRuntime())

	cfg := config.Config{
		Routing: config.RoutingConfig{
			Strategy: "cost_optimized",
			Preferences: []config.RoutingPreference{
				{Role: "junior", Prefer: "codex", Fallback: "claude-code"},
				{Role: "senior", Prefer: "claude-code", Fallback: "codex"},
			},
		},
	}

	router := NewRouter(reg, cfg)

	// Junior should prefer codex
	rt, err := router.SelectRuntime(agent.RoleJunior)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rt.Name() != "codex" {
		t.Errorf("expected codex for junior, got %s", rt.Name())
	}

	// Senior should prefer claude-code
	rt, err = router.SelectRuntime(agent.RoleSenior)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rt.Name() != "claude-code" {
		t.Errorf("expected claude-code for senior, got %s", rt.Name())
	}
}

func TestRouter_FallbackWhenPreferredMissing(t *testing.T) {
	reg := NewRegistry()
	reg.Register("claude-code", NewClaudeCodeRuntime(false))
	// codex NOT registered

	cfg := config.Config{
		Routing: config.RoutingConfig{
			Preferences: []config.RoutingPreference{
				{Role: "junior", Prefer: "codex", Fallback: "claude-code"},
			},
		},
	}

	router := NewRouter(reg, cfg)
	rt, err := router.SelectRuntime(agent.RoleJunior)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rt.Name() != "claude-code" {
		t.Errorf("expected fallback claude-code, got %s", rt.Name())
	}
}

func TestRouter_DefaultsToFirstRuntime(t *testing.T) {
	reg := NewRegistry()
	reg.Register("claude-code", NewClaudeCodeRuntime(false))

	cfg := config.Config{} // no routing preferences

	router := NewRouter(reg, cfg)
	rt, err := router.SelectRuntime(agent.RoleIntermediate)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rt.Name() != "claude-code" {
		t.Errorf("expected claude-code (only registered), got %s", rt.Name())
	}
}

func TestRouter_ErrorWhenNoRuntimes(t *testing.T) {
	reg := NewRegistry()
	cfg := config.Config{}
	router := NewRouter(reg, cfg)

	_, err := router.SelectRuntime(agent.RoleJunior)
	if err == nil {
		t.Error("expected error when no runtimes registered")
	}
}

func TestRouter_NoPreferenceForRole_UsesDefault(t *testing.T) {
	reg := NewRegistry()
	reg.Register("claude-code", NewClaudeCodeRuntime(false))
	reg.Register("codex", NewCodexRuntime())

	cfg := config.Config{
		Routing: config.RoutingConfig{
			Preferences: []config.RoutingPreference{
				{Role: "senior", Prefer: "claude-code", Fallback: "codex"},
			},
		},
	}

	router := NewRouter(reg, cfg)
	// QA has no preference configured — should get first available
	rt, err := router.SelectRuntime(agent.RoleQA)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	// Should get one of the registered runtimes
	if rt.Name() != "claude-code" && rt.Name() != "codex" {
		t.Errorf("expected a registered runtime, got %s", rt.Name())
	}
}
