package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"gopkg.in/yaml.v3"
)

// memPattern matches Docker-compatible memory strings like "512m", "4g".
var memPattern = regexp.MustCompile(`^[0-9]+[gGmM]$`)

// Resources defines CPU and memory limits for a workspace container.
type Resources struct {
	CPUs   float64 `yaml:"cpus,omitempty"`
	Memory string  `yaml:"memory,omitempty"`
}

// IsZero returns true if no resource limits are configured.
func (r Resources) IsZero() bool {
	return r.CPUs == 0 && r.Memory == ""
}

// Validate checks that resource values are sensible.
func (r Resources) Validate() error {
	if r.CPUs < 0 {
		return fmt.Errorf("resources.cpus must be positive, got %g", r.CPUs)
	}
	if r.Memory != "" && !memPattern.MatchString(r.Memory) {
		return fmt.Errorf("resources.memory must match <number>(g|m), got %q", r.Memory)
	}
	return nil
}

// ParseMemoryBytes converts a Docker memory string like "4g" or "512m" to bytes.
func ParseMemoryBytes(mem string) (int64, error) {
	if mem == "" {
		return 0, nil
	}
	mem = strings.ToLower(mem)
	suffix := mem[len(mem)-1]
	num, err := strconv.ParseInt(mem[:len(mem)-1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value %q: %w", mem, err)
	}
	switch suffix {
	case 'g':
		return num * 1024 * 1024 * 1024, nil
	case 'm':
		return num * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unsupported memory suffix %q in %q", string(suffix), mem)
	}
}

// MergeResources returns a Resources where workspace overrides take precedence
// over server defaults for each field that is set.
func MergeResources(serverDefaults, workspaceOverride *Resources) Resources {
	result := Resources{}
	if serverDefaults != nil {
		result = *serverDefaults
	}
	if workspaceOverride != nil {
		if workspaceOverride.CPUs > 0 {
			result.CPUs = workspaceOverride.CPUs
		}
		if workspaceOverride.Memory != "" {
			result.Memory = workspaceOverride.Memory
		}
	}
	return result
}

// DevboxConfig represents the per-project devbox.yaml configuration.
type DevboxConfig struct {
	Name     string            `yaml:"name"`
	Server   string            `yaml:"server"`
	Repo     string            `yaml:"repo"`
	Branch   string            `yaml:"branch,omitempty"`
	Services []string          `yaml:"services,omitempty"`
	Ports    map[string]int    `yaml:"ports,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	Resources      *Resources        `yaml:"resources,omitempty"`
	WorkspacesRoot string            `yaml:"workspaces_root,omitempty"`
}

// DefaultConfigFile is the default config filename looked up in the project root.

// DefaultWorkspacesRoot is the default base directory for workspaces on the server.
const DefaultWorkspacesRoot = "/workspaces"

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

	if cfg.Resources != nil {
		if err := cfg.Resources.Validate(); err != nil {
			return nil, devboxerr.NewConfigError(
				fmt.Sprintf("config file %s: %v", path, err),
				"Example: resources: {cpus: 2, memory: 4g}",
				nil,
			)
		}
	}

	if cfg.WorkspacesRoot == "" {
		cfg.WorkspacesRoot = DefaultWorkspacesRoot
	}

	return &cfg, nil
}

// ValidateForUp checks that the config has enough info to create a workspace.
// If poolConfigured is true, the server field is optional (auto-select from pool).
func (c *DevboxConfig) ValidateForUp(poolConfigured bool) error {
	if c.Server == "" && !poolConfigured {
		return devboxerr.NewConfigError(
			"'server' is required when no server pool is configured",
			"Add 'server: your-server' to devbox.yaml, use --server flag, or run 'devbox server add'",
			nil,
		)
	}
	return nil
}

// ServerDefaults holds per-server default settings.
type ServerDefaults struct {
	Resources Resources `yaml:"resources"`
}

// GlobalConfig represents the user-level ~/.devbox/config.yaml.
type GlobalConfig struct {
	Servers map[string]ServerDefaults `yaml:"servers"`
}

// LoadGlobal reads the global config from ~/.devbox/config.yaml.
// Returns an empty config (not an error) if the file doesn't exist.
func LoadGlobal() (*GlobalConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &GlobalConfig{}, nil
	}
	path := filepath.Join(home, ".devbox", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return &GlobalConfig{}, nil
	}
	var gc GlobalConfig
	if err := yaml.Unmarshal(data, &gc); err != nil {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("failed to parse global config %s", path),
			"Check YAML syntax in ~/.devbox/config.yaml",
			err,
		)
	}
	return &gc, nil
}

// ServerResourceDefaults returns the resource defaults for a given server,
// or nil if no defaults are configured.
func (gc *GlobalConfig) ServerResourceDefaults(server string) *Resources {
	if gc == nil || gc.Servers == nil {
		return nil
	}
	if sd, ok := gc.Servers[server]; ok {
		return &sd.Resources
	}
	return nil
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
