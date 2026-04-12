package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadManifest reads and parses a plugin.yaml file from the given path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load plugin manifest %s: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse plugin manifest %s: %w", path, err)
	}

	if err := validateManifest(&m); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest %s: %w", path, err)
	}

	return &m, nil
}

func validateManifest(m *Manifest) error {
	if m.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if err := validateSafeName(m.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}
	if m.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if !m.Type.IsValid() {
		return fmt.Errorf("invalid type %q: must be one of %v", m.Type, ValidTypes)
	}
	if m.Entrypoint == "" {
		return fmt.Errorf("missing required field: entrypoint")
	}
	if filepath.IsAbs(m.Entrypoint) {
		return fmt.Errorf("entrypoint must be a relative path, got %q", m.Entrypoint)
	}
	if strings.Contains(m.Entrypoint, "..") {
		return fmt.Errorf("entrypoint must not contain '..', got %q", m.Entrypoint)
	}
	return nil
}

// validateSafeName checks that a plugin name is safe for use as a directory component.
func validateSafeName(name string) error {
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("must not contain path separators")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("must not contain '..'")
	}
	if name == "." {
		return fmt.Errorf("must not be '.'")
	}
	return nil
}
