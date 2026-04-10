package config

import (
	"fmt"
	"os"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"gopkg.in/yaml.v3"
)

// DevboxConfig represents the per-project devbox.yaml configuration.
type DevboxConfig struct {
	Name     string            `yaml:"name"`
	Server   string            `yaml:"server"`
	Repo     string            `yaml:"repo"`
	Branch   string            `yaml:"branch,omitempty"`
	Services []string          `yaml:"services,omitempty"`
	Ports    map[string]int    `yaml:"ports,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
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

	return &cfg, nil
}

// LoadFromDir looks for devbox.yaml in the given directory and loads it.
func LoadFromDir(dir string) (*DevboxConfig, error) {
	path := dir + "/" + DefaultConfigFile
	return Load(path)
}
