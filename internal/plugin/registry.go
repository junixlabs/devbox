package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Registry manages the set of registered providers and hooks.
type Registry struct {
	providers map[string]Provider
	hooks     map[string]Hook
	manifests []Manifest
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		hooks:     make(map[string]Hook),
	}
}

// RegisterProvider adds a provider to the registry.
// Returns an error if a provider with the same name is already registered.
func (r *Registry) RegisterProvider(name string, p Provider, m Manifest) error {
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}
	r.providers[name] = p
	r.manifests = append(r.manifests, m)
	return nil
}

// RegisterHook adds a hook to the registry.
// Returns an error if a hook with the same name is already registered.
func (r *Registry) RegisterHook(name string, h Hook, m Manifest) error {
	if _, exists := r.hooks[name]; exists {
		return fmt.Errorf("hook %q already registered", name)
	}
	r.hooks[name] = h
	r.manifests = append(r.manifests, m)
	return nil
}

// GetProvider returns the provider with the given name.
func (r *Registry) GetProvider(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

// GetHooks returns all hooks that subscribe to the given event.
func (r *Registry) GetHooks(event Event) []Hook {
	var result []Hook
	for _, h := range r.hooks {
		for _, e := range h.Events() {
			if e == event {
				result = append(result, h)
				break
			}
		}
	}
	return result
}

// ListPlugins returns the manifests of all registered plugins.
func (r *Registry) ListPlugins() []Manifest {
	return r.manifests
}

// Discover scans a directory for plugin subdirectories containing plugin.yaml files.
// Each valid plugin is loaded and registered. Invalid plugins are logged and skipped.
func (r *Registry) Discover(pluginDir string) error {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no plugin dir is fine
		}
		return fmt.Errorf("discover plugins in %s: %w", pluginDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.yaml")
		m, err := LoadManifest(manifestPath)
		if err != nil {
			slog.Warn("skipping plugin", "dir", entry.Name(), "error", err)
			continue
		}

		// Resolve entrypoint relative to plugin directory.
		// Manifest validation already rejects absolute paths and ".." in entrypoints.
		entrypoint := filepath.Join(pluginDir, entry.Name(), m.Entrypoint)

		switch m.Type {
		case TypeProvider:
			p := NewExternalProvider(*m, entrypoint)
			if err := r.RegisterProvider(m.Name, p, *m); err != nil {
				slog.Warn("skipping plugin", "name", m.Name, "error", err)
			}
		case TypeHook:
			h := NewExternalHook(*m, entrypoint)
			if err := r.RegisterHook(m.Name, h, *m); err != nil {
				slog.Warn("skipping plugin", "name", m.Name, "error", err)
			}
		}
	}

	return nil
}

// DefaultPluginDir returns the default plugin discovery directory.
func DefaultPluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".devbox", "plugins")
	}
	return filepath.Join(home, ".config", "devbox", "plugins")
}
