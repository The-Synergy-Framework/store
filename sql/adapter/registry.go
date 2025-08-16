package adapter

import (
	"fmt"
	"sync"
)

var (
	globalRegistry = NewRegistry()
)

// Registry manages available SQL adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[AdapterName]func() Adapter
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	r := &Registry{
		adapters: make(map[AdapterName]func() Adapter),
	}

	// Register built-in adapters
	r.Register("postgresql", func() Adapter { return NewPostgreSQLAdapter() })
	r.Register("postgres", func() Adapter { return NewPostgreSQLAdapter() }) // Alias
	r.Register("mysql", func() Adapter { return NewMySQLAdapter() })
	r.Register("sqlite", func() Adapter { return NewSQLiteAdapter() })
	r.Register("sqlite3", func() Adapter { return NewSQLiteAdapter() }) // Alias

	return r
}

// Register registers a new adapter factory.
func (r *Registry) Register(name AdapterName, factory func() Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = factory
}

// Get retrieves an adapter by name.
func (r *Registry) Get(name AdapterName) (Adapter, error) {
	r.mu.RLock()
	factory, exists := r.adapters[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", name)
	}

	return factory(), nil
}

// List returns all registered adapter names.
func (r *Registry) List() []AdapterName {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]AdapterName, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}

	return names
}

// Exists checks if an adapter is registered.
func (r *Registry) Exists(name AdapterName) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.adapters[name]
	return exists
}

// Global registry functions

// Register registers an adapter in the global registry.
func Register(name AdapterName, factory func() Adapter) {
	globalRegistry.Register(name, factory)
}

// Get retrieves an adapter from the global registry.
func Get(name AdapterName) (Adapter, error) {
	return globalRegistry.Get(name)
}

// List returns all registered adapters from the global registry.
func List() []AdapterName {
	return globalRegistry.List()
}

// Exists checks if an adapter exists in the global registry.
func Exists(name AdapterName) bool {
	return globalRegistry.Exists(name)
}
