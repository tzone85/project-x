package runtime

import (
	"fmt"
	"sort"
)

// Registry maps runtime names to their implementations.
// It provides lookup and enumeration of available runtimes.
type Registry struct {
	runtimes map[string]Runtime
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		runtimes: make(map[string]Runtime),
	}
}

// Register adds or replaces a runtime under the given name.
func (r *Registry) Register(name string, rt Runtime) {
	updated := make(map[string]Runtime, len(r.runtimes)+1)
	for k, v := range r.runtimes {
		updated[k] = v
	}
	updated[name] = rt
	r.runtimes = updated
}

// Get returns the runtime registered under name, or an error if not found.
func (r *Registry) Get(name string) (Runtime, error) {
	rt, ok := r.runtimes[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime: %q", name)
	}
	return rt, nil
}

// List returns the names of all registered runtimes in sorted order.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.runtimes))
	for name := range r.runtimes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
