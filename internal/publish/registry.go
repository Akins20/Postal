package publish

import "fmt"

// Registry resolves platform adapters by Platform() key. It is shared by the
// composer (for compose-time Validate) and the publish worker (Phase 6).
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry builds a Registry from the given adapters.
func NewRegistry(adapters ...Adapter) *Registry {
	m := make(map[string]Adapter, len(adapters))
	for _, a := range adapters {
		m[a.Platform()] = a
	}
	return &Registry{adapters: m}
}

// Get returns the adapter for a platform, reporting whether it is registered.
func (r *Registry) Get(platform string) (Adapter, bool) {
	a, ok := r.adapters[platform]
	return a, ok
}

// Validate checks a variant against the platform's adapter constraints. An
// unregistered platform returns a terminal error.
func (r *Registry) Validate(platform string, v PostVariant) error {
	a, ok := r.adapters[platform]
	if !ok {
		return Terminal("unsupported_platform", fmt.Sprintf("no adapter for platform %q", platform), nil)
	}
	return a.Validate(v)
}
