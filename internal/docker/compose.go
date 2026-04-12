package docker

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/junixlabs/devbox/internal/config"
	"gopkg.in/yaml.v3"
)

// composeFile represents a docker-compose.yml structure.
type composeFile struct {
	Name     string                    `yaml:"name"`
	Services map[string]composeService `yaml:"services,omitempty"`
	Volumes  map[string]struct{}       `yaml:"volumes,omitempty"`
}

// composeService represents a single service in a docker-compose.yml.
type composeService struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Restart     string            `yaml:"restart"`
	Deploy      *composeDeploy    `yaml:"deploy,omitempty"`
}

// composeDeploy holds deployment configuration for a compose service.
type composeDeploy struct {
	Resources composeResources `yaml:"resources"`
}

// composeResources holds resource constraints.
type composeResources struct {
	Limits composeResourceLimits `yaml:"limits"`
}

// composeResourceLimits holds CPU and memory limits for a service.
type composeResourceLimits struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// knownVolumes maps service name prefixes to their default data paths.
var knownVolumes = map[string]string{
	"mysql":    "/var/lib/mysql",
	"mariadb":  "/var/lib/mysql",
	"postgres": "/var/lib/postgresql/data",
	"redis":    "/data",
	"mongo":    "/data/db",
}

// parseServiceName extracts the service name from an image string.
// "mysql:8.0" → "mysql", "redis" → "redis", "bitnami/redis:7" → "redis".
func parseServiceName(image string) string {
	// Strip tag.
	name := image
	if i := strings.LastIndex(name, ":"); i != -1 {
		name = name[:i]
	}
	// Strip registry/org prefix.
	if i := strings.LastIndex(name, "/"); i != -1 {
		name = name[i+1:]
	}
	return name
}

// GenerateCompose produces docker-compose.yml YAML bytes from a DevboxConfig.
func GenerateCompose(workspaceName string, cfg *config.DevboxConfig) ([]byte, error) {
	cf := composeFile{
		Name:     workspaceName,
		Services: make(map[string]composeService),
		Volumes:  make(map[string]struct{}),
	}

	// Track which port names have been assigned to a service.
	assignedPorts := make(map[string]bool)

	for _, image := range cfg.Services {
		svcName := parseServiceName(image)

		// Handle duplicate service names by appending a suffix.
		if _, exists := cf.Services[svcName]; exists {
			for i := 2; ; i++ {
				candidate := fmt.Sprintf("%s-%d", svcName, i)
				if _, exists := cf.Services[candidate]; !exists {
					slog.Warn("duplicate service name, renaming", "original", svcName, "renamed", candidate)
					svcName = candidate
					break
				}
			}
		}

		svc := composeService{
			Image:   image,
			Restart: "unless-stopped",
		}

		// Assign matching ports.
		// Matches exact name ("redis") and prefixed names ("redis-2", "redis-3")
		// used when a devcontainer forwards multiple ports to the same service.
		for portName, portNum := range cfg.Ports {
			if portName == svcName || strings.HasPrefix(portName, svcName+"-") {
				svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", portNum, portNum))
				assignedPorts[portName] = true
			}
		}

		// Inject environment variables.
		if len(cfg.Env) > 0 {
			svc.Environment = make(map[string]string, len(cfg.Env))
			for k, v := range cfg.Env {
				svc.Environment[k] = v
			}
		}

		// Add named volume for known service types (exact name match).
		baseName := parseServiceName(image) // use original name before dedup suffix
		for knownName, mountPath := range knownVolumes {
			if baseName == knownName {
				volName := fmt.Sprintf("%s-%s-data", workspaceName, svcName)
				svc.Volumes = append(svc.Volumes, fmt.Sprintf("%s:%s", volName, mountPath))
				cf.Volumes[volName] = struct{}{}
				break
			}
		}

		// Apply resource limits if configured.
		if cfg.Resources != nil && !cfg.Resources.IsZero() {
			svc.Deploy = &composeDeploy{
				Resources: composeResources{
					Limits: composeResourceLimits{
						CPUs:   formatCPUs(cfg.Resources.CPUs),
						Memory: cfg.Resources.Memory,
					},
				},
			}
		}

		cf.Services[svcName] = svc
	}

	// Warn about unassigned ports.
	for portName := range cfg.Ports {
		if !assignedPorts[portName] {
			slog.Warn("port has no matching service", "port", portName)
		}
	}

	// Drop empty volumes map so it doesn't render in YAML.
	if len(cf.Volumes) == 0 {
		cf.Volumes = nil
	}

	data, err := yaml.Marshal(&cf)
	if err != nil {
		return nil, fmt.Errorf("marshal compose file: %w", err)
	}
	return data, nil
}

// formatCPUs converts a float64 CPU count to a string for compose deploy limits.
// Returns empty string for zero values.
func formatCPUs(cpus float64) string {
	if cpus == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", cpus)
}
