package config

import (
	"fmt"
	"os"

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
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg DevboxConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("config file %s: 'name' is required", path)
	}

	if cfg.Server == "" {
		return nil, fmt.Errorf("config file %s: 'server' is required", path)
	}

	return &cfg, nil
}

// LoadFromDir looks for devbox.yaml in the given directory and loads it.
func LoadFromDir(dir string) (*DevboxConfig, error) {
	path := dir + "/" + DefaultConfigFile
	return Load(path)
}
