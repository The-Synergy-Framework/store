package adapter

import (
	"fmt"
	"sync"
)

var (
	globalRegistry = NewRegistry()
)

// Registry manages available KV adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]func() Adapter
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	r := &Registry{
		adapters: make(map[string]func() Adapter),
	}

	// Register built-in adapters
	r.Register("memory", func() Adapter { return NewMemoryAdapter() })
	// r.Register("redis", func() Adapter { return NewRedisAdapter() }) // Future
	// r.Register("etcd", func() Adapter { return NewEtcdAdapter() })   // Future

	return r
}

// Register registers a new adapter factory.
func (r *Registry) Register(name string, factory func() Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = factory
}

// Get retrieves an adapter by name.
func (r *Registry) Get(name string) (Adapter, error) {
	r.mu.RLock()
	factory, exists := r.adapters[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", name)
	}

	return factory(), nil
}

// List returns all registered adapter names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}

	return names
}

// Exists checks if an adapter is registered.
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.adapters[name]
	return exists
}

// Global registry functions

// Register registers an adapter in the global registry.
func Register(name string, factory func() Adapter) {
	globalRegistry.Register(name, factory)
}

// Get retrieves an adapter from the global registry.
func Get(name string) (Adapter, error) {
	return globalRegistry.Get(name)
}

// List returns all registered adapters from the global registry.
func List() []string {
	return globalRegistry.List()
}

// Exists checks if an adapter exists in the global registry.
func Exists(name string) bool {
	return globalRegistry.Exists(name)
}
