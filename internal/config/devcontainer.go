package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
)

// DevcontainerConfig represents a subset of the devcontainer.json specification.
type DevcontainerConfig struct {
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	Features          map[string]any    `json:"features"`
	ForwardPorts      []int             `json:"forwardPorts"`
	PostCreateCommand string            `json:"postCreateCommand"`
	ContainerEnv      map[string]string `json:"containerEnv"`
	RemoteUser        string            `json:"remoteUser"`
}

// devcontainerPath is the standard location for devcontainer.json.
const devcontainerPath = ".devcontainer/devcontainer.json"

// stripJSONC converts JSON with Comments (JSONC) to standard JSON by removing
// // line comments, /* */ block comments, and trailing commas before } or ].
func stripJSONC(data []byte) []byte {
	src := string(data)
	var out strings.Builder
	out.Grow(len(src))

	i := 0
	for i < len(src) {
		ch := src[i]

		// String literal — copy verbatim.
		if ch == '"' {
			j := i + 1
			for j < len(src) {
				if src[j] == '\\' {
					j += 2
					if j >= len(src) {
						break
					}
					continue
				}
				if src[j] == '"' {
					j++
					break
				}
				j++
			}
			out.WriteString(src[i:j])
			i = j
			continue
		}

		// Line comment.
		if ch == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment.
		if ch == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) {
				if src[i] == '*' && src[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		out.WriteByte(ch)
		i++
	}

	// Remove trailing commas before } or ].
	result := out.String()
	var clean strings.Builder
	clean.Grow(len(result))
	for i := 0; i < len(result); i++ {
		if result[i] == ',' {
			// Look ahead past whitespace for } or ].
			j := i + 1
			for j < len(result) && (result[j] == ' ' || result[j] == '\t' || result[j] == '\n' || result[j] == '\r') {
				j++
			}
			if j < len(result) && (result[j] == '}' || result[j] == ']') {
				continue // skip trailing comma
			}
		}
		clean.WriteByte(result[i])
	}

	return []byte(clean.String())
}

// LoadDevcontainer reads and parses a devcontainer.json file from the given path.
func LoadDevcontainer(path string) (*DevcontainerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("failed to read devcontainer config %s", path),
			"Check that .devcontainer/devcontainer.json exists",
			err,
		)
	}

	cleaned := stripJSONC(data)

	var dc DevcontainerConfig
	if err := json.Unmarshal(cleaned, &dc); err != nil {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("failed to parse devcontainer config %s", path),
			"Check JSON syntax in devcontainer.json",
			err,
		)
	}

	if dc.Image == "" {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("devcontainer config %s: 'image' is required", path),
			"Add '\"image\": \"your-image\"' to devcontainer.json",
			nil,
		)
	}

	return &dc, nil
}

// ToDevboxConfig maps a DevcontainerConfig to a DevboxConfig.
// dirName is used as the workspace name if the devcontainer has no name field.
// Server is left empty — it must be provided via --server flag.
func (dc *DevcontainerConfig) ToDevboxConfig(dirName string) *DevboxConfig {
	name := dc.Name
	if name == "" {
		name = dirName
	}

	// Sanitize name: replace spaces/special chars with hyphens.
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)

	cfg := &DevboxConfig{
		Name:     name,
		Services: []string{dc.Image},
	}

	if len(dc.ForwardPorts) > 0 {
		// Extract service name from image (e.g. "node:20" → "node").
		svcName := dc.Image
		if i := strings.LastIndex(svcName, ":"); i != -1 {
			svcName = svcName[:i]
		}
		if i := strings.LastIndex(svcName, "/"); i != -1 {
			svcName = svcName[i+1:]
		}

		cfg.Ports = make(map[string]int, len(dc.ForwardPorts))
		for idx, port := range dc.ForwardPorts {
			key := svcName
			if idx > 0 {
				key = fmt.Sprintf("%s-%d", svcName, idx+1)
			}
			cfg.Ports[key] = port
		}
	}

	if len(dc.ContainerEnv) > 0 {
		cfg.Env = make(map[string]string, len(dc.ContainerEnv))
		for k, v := range dc.ContainerEnv {
			cfg.Env[k] = v
		}
	}

	return cfg
}
