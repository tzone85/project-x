package runtime

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestRouterSelectPreferred(t *testing.T) {
	claudeRT := newMockRuntime("claude-code", HealthHealthy, RuntimeCapabilities{})
	codexRT := newMockRuntime("codex", HealthHealthy, RuntimeCapabilities{})

	r := NewRouter(DefaultRoutingConfig(), slog.Default())
	r.Register(RuntimeEntry{Runtime: claudeRT, CostTier: TierSubscription})
	r.Register(RuntimeEntry{Runtime: codexRT, CostTier: TierAPIBased})

	// Senior prefers claude-code
	rt, err := r.Select(context.Background(), "senior", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "claude-code" {
		t.Errorf("expected claude-code, got %s", rt.Name())
	}
}

func TestRouterSelectFallback(t *testing.T) {
	claudeRT := newMockRuntime("claude-code", HealthDead, RuntimeCapabilities{})
	geminiRT := newMockRuntime("gemini", HealthHealthy, RuntimeCapabilities{})

	r := NewRouter(DefaultRoutingConfig(), slog.Default())
	r.Register(RuntimeEntry{Runtime: claudeRT, CostTier: TierSubscription})
	r.Register(RuntimeEntry{Runtime: geminiRT, CostTier: TierAPIBased})

	// Senior prefers claude-code (unhealthy), falls back to gemini
	rt, err := r.Select(context.Background(), "senior", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "gemini" {
		t.Errorf("expected gemini fallback, got %s", rt.Name())
	}
}

func TestRouterSelectCostOptimized(t *testing.T) {
	// No preference match for "intern" role — cost optimization kicks in
	subRT := newMockRuntime("sub-runtime", HealthHealthy, RuntimeCapabilities{})
	apiRT := newMockRuntime("api-runtime", HealthHealthy, RuntimeCapabilities{})

	r := NewRouter(RoutingConfig{Strategy: "cost_optimized"}, slog.Default())
	r.Register(RuntimeEntry{Runtime: subRT, CostTier: TierSubscription})
	r.Register(RuntimeEntry{Runtime: apiRT, CostTier: TierAPIBased})

	rt, err := r.Select(context.Background(), "intern", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "sub-runtime" {
		t.Errorf("expected subscription-tier runtime, got %s", rt.Name())
	}
}

func TestRouterSelectModelCapability(t *testing.T) {
	claudeRT := newMockRuntime("claude-code", HealthHealthy, RuntimeCapabilities{
		SupportedModels: []string{"claude-3.5-sonnet", "claude-3-opus"},
	})
	codexRT := newMockRuntime("codex", HealthHealthy, RuntimeCapabilities{
		SupportedModels: []string{"gpt-4o"},
	})

	r := NewRouter(RoutingConfig{Strategy: "cost_optimized"}, slog.Default())
	r.Register(RuntimeEntry{Runtime: claudeRT, CostTier: TierSubscription})
	r.Register(RuntimeEntry{Runtime: codexRT, CostTier: TierAPIBased})

	// Need gpt-4o, only codex supports it
	rt, err := r.Select(context.Background(), "any", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "codex" {
		t.Errorf("expected codex for gpt-4o model, got %s", rt.Name())
	}
}

func TestRouterSelectNoAvailable(t *testing.T) {
	deadRT := newMockRuntime("dead-runtime", HealthDead, RuntimeCapabilities{})

	r := NewRouter(RoutingConfig{Strategy: "cost_optimized"}, slog.Default())
	r.Register(RuntimeEntry{Runtime: deadRT, CostTier: TierSubscription})

	_, err := r.Select(context.Background(), "senior", "")
	if err == nil {
		t.Error("expected error when no runtime available")
	}
}

func TestRouterSelectNoRegistered(t *testing.T) {
	r := NewRouter(DefaultRoutingConfig(), slog.Default())

	_, err := r.Select(context.Background(), "senior", "")
	if err == nil {
		t.Error("expected error when no runtime registered")
	}
}

func TestRouterSelectHealthError(t *testing.T) {
	errRT := newMockRuntime("err-runtime", HealthHealthy, RuntimeCapabilities{})
	errRT.SetHealthErr(errors.New("connection failed"))

	r := NewRouter(RoutingConfig{Strategy: "cost_optimized"}, slog.Default())
	r.Register(RuntimeEntry{Runtime: errRT, CostTier: TierSubscription})

	_, err := r.Select(context.Background(), "any", "")
	if err == nil {
		t.Error("expected error when health check fails")
	}
}

func TestRouterRegister(t *testing.T) {
	r := NewRouter(DefaultRoutingConfig(), slog.Default())
	if len(r.RegisteredNames()) != 0 {
		t.Error("expected 0 registered runtimes initially")
	}

	r.Register(RuntimeEntry{
		Runtime:  newMockRuntime("test", HealthHealthy, RuntimeCapabilities{}),
		CostTier: TierSubscription,
	})

	if len(r.RegisteredNames()) != 1 {
		t.Errorf("expected 1 registered runtime, got %d", len(r.RegisteredNames()))
	}
}

func TestRouterNilLogger(t *testing.T) {
	r := NewRouter(DefaultRoutingConfig(), nil)
	if r == nil {
		t.Fatal("expected non-nil router with nil logger")
	}
}

func TestDefaultRoutingConfig(t *testing.T) {
	cfg := DefaultRoutingConfig()
	if cfg.Strategy != "cost_optimized" {
		t.Errorf("expected cost_optimized strategy, got %s", cfg.Strategy)
	}
	if len(cfg.Preferences) != 2 {
		t.Errorf("expected 2 preferences, got %d", len(cfg.Preferences))
	}
}

func TestSupportsModelEmpty(t *testing.T) {
	caps := RuntimeCapabilities{SupportedModels: nil}
	if !supportsModel(caps, "any-model") {
		t.Error("empty model list should support any model")
	}
}

func TestSupportsModelMatch(t *testing.T) {
	caps := RuntimeCapabilities{SupportedModels: []string{"gpt-4", "gpt-3.5"}}
	if !supportsModel(caps, "gpt-4") {
		t.Error("should support gpt-4")
	}
	if supportsModel(caps, "claude-3") {
		t.Error("should not support claude-3")
	}
}

func TestRouterSelectAnyHealthyFallback(t *testing.T) {
	// Performance strategy (not cost_optimized), no preference match
	rt1 := newMockRuntime("rt1", HealthDead, RuntimeCapabilities{})
	rt2 := newMockRuntime("rt2", HealthHealthy, RuntimeCapabilities{})

	r := NewRouter(RoutingConfig{Strategy: "performance"}, slog.Default())
	r.Register(RuntimeEntry{Runtime: rt1, CostTier: TierSubscription})
	r.Register(RuntimeEntry{Runtime: rt2, CostTier: TierAPIBased})

	rt, err := r.Select(context.Background(), "unknown-role", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "rt2" {
		t.Errorf("expected rt2 as any-healthy fallback, got %s", rt.Name())
	}
}

func TestRuntimeTypes(t *testing.T) {
	// Verify enum values are distinct
	statuses := []AgentStatus{StatusIdle, StatusRunning, StatusFinished, StatusErrored, StatusUnknown}
	seen := make(map[AgentStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate agent status: %s", s)
		}
		seen[s] = true
	}

	healths := []HealthStatus{HealthHealthy, HealthStale, HealthDead, HealthMissing}
	seenH := make(map[HealthStatus]bool)
	for _, h := range healths {
		if seenH[h] {
			t.Errorf("duplicate health status: %s", h)
		}
		seenH[h] = true
	}
}
