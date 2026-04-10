package config

import (
	"fmt"
	"os"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"gopkg.in/yaml.v3"
)

// PortRangeConfig defines the allowed port range for auto-allocation.
type PortRangeConfig struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

// DevboxConfig represents the per-project devbox.yaml configuration.
type DevboxConfig struct {
	Name      string            `yaml:"name"`
	Server    string            `yaml:"server"`
	Repo      string            `yaml:"repo"`
	Branch    string            `yaml:"branch,omitempty"`
	Services  []string          `yaml:"services,omitempty"`
	Ports     map[string]int    `yaml:"ports,omitempty"`
	PortRange *PortRangeConfig  `yaml:"port_range,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
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

	if cfg.PortRange != nil {
		if cfg.PortRange.Min < 1024 {
			return nil, devboxerr.NewConfigError(
				fmt.Sprintf("config file %s: port_range.min must be >= 1024", path),
				"Set port_range.min to at least 1024",
				nil,
			)
		}
		if cfg.PortRange.Max <= cfg.PortRange.Min {
			return nil, devboxerr.NewConfigError(
				fmt.Sprintf("config file %s: port_range.max must be greater than port_range.min", path),
				"Ensure port_range.max > port_range.min (e.g. min: 10000, max: 60000)",
				nil,
			)
		}
	}

	return &cfg, nil
}

// LoadFromDir looks for devbox.yaml in the given directory and loads it.
func LoadFromDir(dir string) (*DevboxConfig, error) {
	path := dir + "/" + DefaultConfigFile
	return Load(path)
}
