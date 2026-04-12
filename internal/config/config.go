package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"gopkg.in/yaml.v3"
)

// DefaultWorkspacesRoot is the default base directory for workspaces on the server.
const DefaultWorkspacesRoot = "/workspaces"

// DevboxConfig represents the per-project devbox.yaml configuration.
type DevboxConfig struct {
	Name           string            `yaml:"name"`
	Server         string            `yaml:"server"`
	Repo           string            `yaml:"repo"`
	Branch         string            `yaml:"branch,omitempty"`
	Services       []string          `yaml:"services,omitempty"`
	Ports          map[string]int    `yaml:"ports,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	WorkspacesRoot string            `yaml:"workspaces_root,omitempty"`
}

// DefaultConfigFile is the default config filename looked up in the project root.
const DefaultConfigFile = "devbox.yaml"

// Load reads and parses a devbox.yaml file from the given path.
func Load(path string) (*DevboxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("failed to read config file %s", path),
			"Run 'devbox init' to create a devbox.yaml",
			err,
		)
	}

	var cfg DevboxConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("failed to parse config file %s", path),
			"Check YAML syntax in devbox.yaml",
			err,
		)
	}

	if cfg.Name == "" {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("config file %s: 'name' is required", path),
			"Add 'name: your-project' to devbox.yaml",
			nil,
		)
	}

	if cfg.Server == "" {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("config file %s: 'server' is required", path),
			"Add 'server: your-server' to devbox.yaml",
			nil,
		)
	}

	if cfg.WorkspacesRoot == "" {
		cfg.WorkspacesRoot = DefaultWorkspacesRoot
	}

	return &cfg, nil
}

// LoadFromDir looks for devbox.yaml in the given directory and loads it.
// If devbox.yaml is not found, it falls back to .devcontainer/devcontainer.json.
func LoadFromDir(dir string) (*DevboxConfig, error) {
	path := dir + "/" + DefaultConfigFile
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}

	// Only fall back to devcontainer.json if devbox.yaml doesn't exist.
	// errors.Is walks the Unwrap chain, so it finds os.ErrNotExist
	// even when wrapped inside a ConfigError.
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	dcPath := filepath.Join(dir, devcontainerPath)
	dc, dcErr := LoadDevcontainer(dcPath)
	if dcErr != nil {
		// Neither config found — return the original devbox.yaml error.
		return nil, err
	}

	dirName := filepath.Base(dir)
	if dirName == "." || dirName == "/" {
		if abs, absErr := filepath.Abs(dir); absErr == nil {
			dirName = filepath.Base(abs)
		}
	}

	return dc.ToDevboxConfig(dirName), nil
}
