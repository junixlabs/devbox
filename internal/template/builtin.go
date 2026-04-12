package template

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// LoadBuiltins reads all embedded built-in templates.
func LoadBuiltins() ([]Template, error) {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil, fmt.Errorf("failed to read built-in templates: %w", err)
	}

	var templates []Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := builtinFS.ReadFile(filepath.Join("builtin", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read built-in template %s: %w", entry.Name(), err)
		}
		var t Template
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("failed to parse built-in template %s: %w", entry.Name(), err)
		}
		if t.Name == "" {
			t.Name = strings.TrimSuffix(entry.Name(), ".yaml")
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// NewDefaultRegistry creates a LocalRegistry with built-in templates
// and the default custom templates directory.
func NewDefaultRegistry() (*LocalRegistry, error) {
	builtins, err := LoadBuiltins()
	if err != nil {
		return nil, err
	}
	dir, err := DefaultRegistryDir()
	if err != nil {
		return nil, err
	}
	return NewLocalRegistry(dir, builtins), nil
}
