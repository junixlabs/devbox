package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/junixlabs/devbox/internal/config"
	"gopkg.in/yaml.v3"
)

// namePattern validates template names: lowercase letters, digits, hyphens.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Template extends DevboxConfig with setup scripts and metadata.
type Template struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Services    []string          `yaml:"services,omitempty"`
	Ports       map[string]int    `yaml:"ports,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Resources   *config.Resources `yaml:"resources,omitempty"`
	Setup       []string          `yaml:"setup,omitempty"`
}

// MatchesQuery returns true if the template's name or description
// contains the query string (case-insensitive). Returns false for empty queries.
func (t *Template) MatchesQuery(query string) bool {
	if query == "" {
		return false
	}
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(t.Name), q) ||
		strings.Contains(strings.ToLower(t.Description), q)
}

// Validate checks that the template has a valid name and consistent fields.
func (t *Template) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if !namePattern.MatchString(t.Name) {
		return fmt.Errorf("template name %q is invalid: must be lowercase letters, digits, and hyphens", t.Name)
	}
	if t.Resources != nil {
		if err := t.Resources.Validate(); err != nil {
			return fmt.Errorf("template %q: %w", t.Name, err)
		}
	}
	return nil
}

// ToDevboxConfig converts a template into a DevboxConfig, applying the
// project name and server from the caller.
func (t *Template) ToDevboxConfig(projectName, server string) *config.DevboxConfig {
	cfg := &config.DevboxConfig{
		Name:      projectName,
		Server:    server,
		Services:  t.Services,
		Ports:     t.Ports,
		Env:       t.Env,
		Resources: t.Resources,
	}
	return cfg
}

// Registry provides access to workspace templates.
type Registry interface {
	List() ([]Template, error)
	Get(name string) (*Template, error)
	Save(t *Template) error
}

// LocalRegistry stores templates as YAML files in a local directory.
type LocalRegistry struct {
	dir      string
	builtins []Template
}

// NewLocalRegistry creates a registry backed by the given directory,
// with optional built-in templates that are always available.
func NewLocalRegistry(dir string, builtins []Template) *LocalRegistry {
	return &LocalRegistry{dir: dir, builtins: builtins}
}

// List returns all available templates (built-in + custom).
func (r *LocalRegistry) List() ([]Template, error) {
	all := make([]Template, len(r.builtins))
	copy(all, r.builtins)

	custom, err := r.listCustom()
	if err != nil {
		return all, nil // return builtins even if custom dir fails
	}
	all = append(all, custom...)
	return all, nil
}

// Get returns a template by name. Custom templates override built-ins.
func (r *LocalRegistry) Get(name string) (*Template, error) {
	// Check custom templates first (override built-ins).
	path := filepath.Join(r.dir, name+".yaml")
	if data, err := os.ReadFile(path); err == nil {
		var t Template
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
		}
		t.Name = name
		return &t, nil
	}

	// Check built-ins.
	for i := range r.builtins {
		if r.builtins[i].Name == name {
			t := r.builtins[i]
			return &t, nil
		}
	}

	return nil, fmt.Errorf("template %q not found", name)
}

// Save writes a template to the custom templates directory.
func (r *LocalRegistry) Save(t *Template) error {
	if err := t.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	path := filepath.Join(r.dir, t.Name+".yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write template %q: %w", t.Name, err)
	}

	return nil
}

// DefaultRegistryDir returns the default custom templates directory.
func DefaultRegistryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "devbox", "templates"), nil
}

// listCustom reads all .yaml files from the custom templates directory.
func (r *LocalRegistry) listCustom() ([]Template, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var templates []Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		t, err := r.Get(name)
		if err != nil {
			continue
		}
		templates = append(templates, *t)
	}
	return templates, nil
}

// FromDevboxConfig creates a template from an existing DevboxConfig.
func FromDevboxConfig(cfg *config.DevboxConfig, name, description string) *Template {
	return &Template{
		Name:        name,
		Description: description,
		Services:    cfg.Services,
		Ports:       cfg.Ports,
		Env:         cfg.Env,
		Resources:   cfg.Resources,
	}
}
