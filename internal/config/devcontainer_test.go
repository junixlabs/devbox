package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
)

func TestLoadDevcontainer_FullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	content := `{
  "name": "My Go Project",
  "image": "mcr.microsoft.com/devcontainers/go:1",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/git:1": {"version": "latest"}
  },
  "forwardPorts": [8080, 3000],
  "postCreateCommand": "go mod download",
  "containerEnv": {
    "GO111MODULE": "on",
    "GOFLAGS": "-v"
  },
  "remoteUser": "vscode"
}`
	os.WriteFile(path, []byte(content), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.Name != "My Go Project" {
		t.Errorf("Name = %q, want %q", dc.Name, "My Go Project")
	}
	if dc.Image != "mcr.microsoft.com/devcontainers/go:1" {
		t.Errorf("Image = %q, want %q", dc.Image, "mcr.microsoft.com/devcontainers/go:1")
	}
	if len(dc.Features) != 2 {
		t.Errorf("Features count = %d, want 2", len(dc.Features))
	}
	if len(dc.ForwardPorts) != 2 {
		t.Fatalf("ForwardPorts count = %d, want 2", len(dc.ForwardPorts))
	}
	if dc.ForwardPorts[0] != 8080 {
		t.Errorf("ForwardPorts[0] = %d, want 8080", dc.ForwardPorts[0])
	}
	if dc.ForwardPorts[1] != 3000 {
		t.Errorf("ForwardPorts[1] = %d, want 3000", dc.ForwardPorts[1])
	}
	if dc.PostCreateCommand != "go mod download" {
		t.Errorf("PostCreateCommand = %q, want %q", dc.PostCreateCommand, "go mod download")
	}
	if dc.ContainerEnv["GO111MODULE"] != "on" {
		t.Errorf("ContainerEnv[GO111MODULE] = %q, want %q", dc.ContainerEnv["GO111MODULE"], "on")
	}
	if dc.RemoteUser != "vscode" {
		t.Errorf("RemoteUser = %q, want %q", dc.RemoteUser, "vscode")
	}
}

func TestLoadDevcontainer_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	os.WriteFile(path, []byte(`{"image": "node:20"}`), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.Image != "node:20" {
		t.Errorf("Image = %q, want %q", dc.Image, "node:20")
	}
	if dc.Name != "" {
		t.Errorf("Name = %q, want empty", dc.Name)
	}
	if len(dc.ForwardPorts) != 0 {
		t.Errorf("ForwardPorts = %v, want empty", dc.ForwardPorts)
	}
}

func TestLoadDevcontainer_MissingImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	os.WriteFile(path, []byte(`{"name": "no-image"}`), 0644)

	_, err := LoadDevcontainer(path)
	if err == nil {
		t.Fatal("expected error for missing image")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoadDevcontainer_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	os.WriteFile(path, []byte(`{invalid json`), 0644)

	_, err := LoadDevcontainer(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoadDevcontainer_FileNotFound(t *testing.T) {
	_, err := LoadDevcontainer("/nonexistent/devcontainer.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoadDevcontainer_WithComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	content := `{
  // This is the dev container config
  "image": "python:3.12",
  "forwardPorts": [8000], // Django port
  "containerEnv": {
    // Python settings
    "PYTHONDONTWRITEBYTECODE": "1"
  }
}`
	os.WriteFile(path, []byte(content), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.Image != "python:3.12" {
		t.Errorf("Image = %q, want %q", dc.Image, "python:3.12")
	}
	if len(dc.ForwardPorts) != 1 || dc.ForwardPorts[0] != 8000 {
		t.Errorf("ForwardPorts = %v, want [8000]", dc.ForwardPorts)
	}
	if dc.ContainerEnv["PYTHONDONTWRITEBYTECODE"] != "1" {
		t.Errorf("ContainerEnv[PYTHONDONTWRITEBYTECODE] = %q, want %q", dc.ContainerEnv["PYTHONDONTWRITEBYTECODE"], "1")
	}
}

func TestToDevboxConfig_Mapping(t *testing.T) {
	dc := &DevcontainerConfig{
		Name:         "my-project",
		Image:        "golang:1.22",
		ForwardPorts: []int{8080, 3000},
		ContainerEnv: map[string]string{
			"APP_ENV": "dev",
		},
	}

	cfg := dc.ToDevboxConfig("fallback-name")

	if cfg.Name != "my-project" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-project")
	}
	if cfg.Server != "" {
		t.Errorf("Server = %q, want empty", cfg.Server)
	}
	if len(cfg.Services) != 1 || cfg.Services[0] != "golang:1.22" {
		t.Errorf("Services = %v, want [golang:1.22]", cfg.Services)
	}
	if cfg.Ports["port-8080"] != 8080 {
		t.Errorf("Ports[port-8080] = %d, want 8080", cfg.Ports["port-8080"])
	}
	if cfg.Ports["port-3000"] != 3000 {
		t.Errorf("Ports[port-3000] = %d, want 3000", cfg.Ports["port-3000"])
	}
	if cfg.Env["APP_ENV"] != "dev" {
		t.Errorf("Env[APP_ENV] = %q, want %q", cfg.Env["APP_ENV"], "dev")
	}
}

func TestToDevboxConfig_DefaultName(t *testing.T) {
	dc := &DevcontainerConfig{
		Image: "node:20",
	}

	cfg := dc.ToDevboxConfig("my-dir")

	if cfg.Name != "my-dir" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-dir")
	}
}

func TestToDevboxConfig_NameSanitization(t *testing.T) {
	dc := &DevcontainerConfig{
		Name:  "My Cool Project!",
		Image: "node:20",
	}

	cfg := dc.ToDevboxConfig("fallback")

	// Spaces and special chars should be replaced with hyphens.
	if cfg.Name != "My-Cool-Project-" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My-Cool-Project-")
	}
}

func TestToDevboxConfig_NoPorts(t *testing.T) {
	dc := &DevcontainerConfig{
		Image: "alpine:latest",
	}

	cfg := dc.ToDevboxConfig("test")

	if cfg.Ports != nil {
		t.Errorf("Ports = %v, want nil", cfg.Ports)
	}
}

func TestToDevboxConfig_NoEnv(t *testing.T) {
	dc := &DevcontainerConfig{
		Image: "alpine:latest",
	}

	cfg := dc.ToDevboxConfig("test")

	if cfg.Env != nil {
		t.Errorf("Env = %v, want nil", cfg.Env)
	}
}

func TestLoadFromDir_DevcontainerFallback(t *testing.T) {
	dir := t.TempDir()

	// Create .devcontainer/devcontainer.json but NO devbox.yaml.
	dcDir := filepath.Join(dir, ".devcontainer")
	os.MkdirAll(dcDir, 0755)
	os.WriteFile(
		filepath.Join(dcDir, "devcontainer.json"),
		[]byte(`{"image": "node:20", "forwardPorts": [3000]}`),
		0644,
	)

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir() error: %v", err)
	}

	if len(cfg.Services) != 1 || cfg.Services[0] != "node:20" {
		t.Errorf("Services = %v, want [node:20]", cfg.Services)
	}
	if cfg.Ports["port-3000"] != 3000 {
		t.Errorf("Ports[port-3000] = %d, want 3000", cfg.Ports["port-3000"])
	}
	// Server should be empty — must come from --server flag.
	if cfg.Server != "" {
		t.Errorf("Server = %q, want empty", cfg.Server)
	}
}

func TestLoadFromDir_DevboxYamlPriority(t *testing.T) {
	dir := t.TempDir()

	// Create BOTH devbox.yaml AND .devcontainer/devcontainer.json.
	os.WriteFile(
		filepath.Join(dir, "devbox.yaml"),
		[]byte("name: from-devbox\nserver: s1\nservices:\n  - redis:7\n"),
		0644,
	)

	dcDir := filepath.Join(dir, ".devcontainer")
	os.MkdirAll(dcDir, 0755)
	os.WriteFile(
		filepath.Join(dcDir, "devcontainer.json"),
		[]byte(`{"image": "node:20"}`),
		0644,
	)

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir() error: %v", err)
	}

	// Should use devbox.yaml, not devcontainer.json.
	if cfg.Name != "from-devbox" {
		t.Errorf("Name = %q, want %q", cfg.Name, "from-devbox")
	}
	if cfg.Server != "s1" {
		t.Errorf("Server = %q, want %q", cfg.Server, "s1")
	}
	if len(cfg.Services) != 1 || cfg.Services[0] != "redis:7" {
		t.Errorf("Services = %v, want [redis:7]", cfg.Services)
	}
}

func TestLoadFromDir_NeitherExists(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error when neither config exists")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoadFromDir_MalformedDevboxYaml_NoFallback(t *testing.T) {
	dir := t.TempDir()

	// Create a malformed devbox.yaml (missing server).
	os.WriteFile(
		filepath.Join(dir, "devbox.yaml"),
		[]byte("name: broken-project\n"),
		0644,
	)

	// Also create a valid devcontainer.json — it should NOT be used.
	dcDir := filepath.Join(dir, ".devcontainer")
	os.MkdirAll(dcDir, 0755)
	os.WriteFile(
		filepath.Join(dcDir, "devcontainer.json"),
		[]byte(`{"image": "node:20"}`),
		0644,
	)

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error for malformed devbox.yaml, should not fall back to devcontainer.json")
	}

	// Should report the devbox.yaml error, not silently fall back.
	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if !strings.Contains(err.Error(), "'server' is required") {
		t.Errorf("error = %q, want it to mention server is required", err.Error())
	}
}

func TestLoadDevcontainer_URLInsideString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	// Ensure // inside a JSON string is NOT treated as a comment.
	content := `{"image": "node:20", "postCreateCommand": "echo https://example.com"}`
	os.WriteFile(path, []byte(content), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.PostCreateCommand != "echo https://example.com" {
		t.Errorf("PostCreateCommand = %q, want %q", dc.PostCreateCommand, "echo https://example.com")
	}
}

func TestLoadDevcontainer_BlockComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	content := `{
  /* Main container config */
  "image": "python:3.12",
  "forwardPorts": [8000]
}`
	os.WriteFile(path, []byte(content), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.Image != "python:3.12" {
		t.Errorf("Image = %q, want %q", dc.Image, "python:3.12")
	}
}

func TestLoadDevcontainer_TrailingCommas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")

	content := `{
  "image": "node:20",
  "forwardPorts": [3000, 8080,],
}`
	os.WriteFile(path, []byte(content), 0644)

	dc, err := LoadDevcontainer(path)
	if err != nil {
		t.Fatalf("LoadDevcontainer() error: %v", err)
	}

	if dc.Image != "node:20" {
		t.Errorf("Image = %q, want %q", dc.Image, "node:20")
	}
	if len(dc.ForwardPorts) != 2 {
		t.Fatalf("ForwardPorts count = %d, want 2", len(dc.ForwardPorts))
	}
}
