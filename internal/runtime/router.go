package runtime

import (
	"fmt"

	"github.com/tzone85/project-x/internal/agent"
	"github.com/tzone85/project-x/internal/config"
)

// Router selects the best runtime for a given agent role based on
// configuration preferences and available runtimes.
type Router struct {
	registry    *Registry
	preferences map[string]config.RoutingPreference // role -> preference
}

// NewRouter creates a Router backed by the given registry and config.
// Preferences are indexed by role for O(1) lookup.
func NewRouter(reg *Registry, cfg config.Config) *Router {
	prefs := make(map[string]config.RoutingPreference, len(cfg.Routing.Preferences))
	for _, p := range cfg.Routing.Preferences {
		prefs[p.Role] = p
	}
	return &Router{registry: reg, preferences: prefs}
}

// SelectRuntime returns the best runtime for the given role.
// It checks configured preferences first (prefer → fallback),
// then falls back to the first available runtime.
func (r *Router) SelectRuntime(role agent.Role) (Runtime, error) {
	// Check preference for this role.
	if pref, ok := r.preferences[string(role)]; ok {
		if rt, err := r.registry.Get(pref.Prefer); err == nil {
			return rt, nil
		}
		if pref.Fallback != "" {
			if rt, err := r.registry.Get(pref.Fallback); err == nil {
				return rt, nil
			}
		}
	}

	// Default: first available (sorted for determinism).
	names := r.registry.List()
	if len(names) == 0 {
		return nil, fmt.Errorf("no runtimes registered")
	}
	return r.registry.Get(names[0])
}
