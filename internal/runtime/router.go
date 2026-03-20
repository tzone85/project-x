package runtime

import (
	"context"
	"fmt"
	"log/slog"
)

// RuntimeEntry registers a runtime with its cost tier.
type RuntimeEntry struct {
	Runtime  Runtime
	CostTier CostTier
}

// Router selects the best runtime for a given role and story context.
// It supports cost-aware routing, capability matching, and fallback chains.
type Router struct {
	runtimes map[string]RuntimeEntry
	config   RoutingConfig
	logger   *slog.Logger
}

// NewRouter creates a runtime router.
func NewRouter(config RoutingConfig, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		runtimes: make(map[string]RuntimeEntry),
		config:   config,
		logger:   logger,
	}
}

// Register adds a runtime to the router.
func (r *Router) Register(entry RuntimeEntry) {
	r.runtimes[entry.Runtime.Name()] = entry
}

// RegisteredNames returns the names of all registered runtimes.
func (r *Router) RegisteredNames() []string {
	names := make([]string, 0, len(r.runtimes))
	for name := range r.runtimes {
		names = append(names, name)
	}
	return names
}

// Select picks the best runtime for the given role and model requirement.
// It checks the role-based preference first, then falls back, then tries
// any healthy runtime as a last resort.
func (r *Router) Select(ctx context.Context, role, model string) (Runtime, error) {
	// Find role-specific preference
	pref := r.preferenceFor(role)

	// Try preferred runtime
	if pref.Prefer != "" {
		if rt, err := r.tryRuntime(ctx, pref.Prefer, model); err == nil {
			return rt, nil
		}
	}

	// Try fallback runtime
	if pref.Fallback != "" {
		if rt, err := r.tryRuntime(ctx, pref.Fallback, model); err == nil {
			return rt, nil
		}
	}

	// Cost-optimized: try subscription-tier runtimes first
	if r.config.Strategy == "cost_optimized" {
		if rt := r.tryByTier(ctx, model, TierSubscription); rt != nil {
			return rt, nil
		}
	}

	// Last resort: any healthy runtime that supports the model
	if rt := r.tryAnyHealthy(ctx, model); rt != nil {
		return rt, nil
	}

	return nil, fmt.Errorf("no available runtime for role=%q model=%q", role, model)
}

// preferenceFor finds the routing preference for a role.
func (r *Router) preferenceFor(role string) RoutingPreference {
	for _, pref := range r.config.Preferences {
		if pref.Role == role {
			return pref
		}
	}
	return RoutingPreference{}
}

// tryRuntime attempts to use a specific runtime by name.
func (r *Router) tryRuntime(ctx context.Context, name, model string) (Runtime, error) {
	entry, ok := r.runtimes[name]
	if !ok {
		return nil, fmt.Errorf("runtime %q not registered", name)
	}

	if model != "" && !supportsModel(entry.Runtime.Capabilities(), model) {
		return nil, fmt.Errorf("runtime %q does not support model %q", name, model)
	}

	health, err := entry.Runtime.Health(ctx, "")
	if err != nil || health != HealthHealthy {
		r.logger.Warn("runtime unhealthy, skipping",
			"runtime", name, "health", health, "error", err)
		return nil, fmt.Errorf("runtime %q unhealthy", name)
	}

	return entry.Runtime, nil
}

// tryByTier tries all runtimes in a given cost tier.
func (r *Router) tryByTier(ctx context.Context, model string, tier CostTier) Runtime {
	for _, entry := range r.runtimes {
		if entry.CostTier != tier {
			continue
		}
		if rt, err := r.tryRuntime(ctx, entry.Runtime.Name(), model); err == nil {
			return rt
		}
	}
	return nil
}

// tryAnyHealthy tries all registered runtimes.
func (r *Router) tryAnyHealthy(ctx context.Context, model string) Runtime {
	for _, entry := range r.runtimes {
		if rt, err := r.tryRuntime(ctx, entry.Runtime.Name(), model); err == nil {
			return rt
		}
	}
	return nil
}

// supportsModel checks if a runtime supports a given model.
func supportsModel(caps RuntimeCapabilities, model string) bool {
	if len(caps.SupportedModels) == 0 {
		return true // no model restriction
	}
	for _, m := range caps.SupportedModels {
		if m == model {
			return true
		}
	}
	return false
}
