package config

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectExistingConfigs(t *testing.T) {
	dir := t.TempDir()

	// Create docker-compose.yml and Dockerfile
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)

	found := DetectExistingConfigs(dir)
	if len(found) != 2 {
		t.Fatalf("expected 2 detected configs, got %d", len(found))
	}

	types := map[string]bool{}
	for _, d := range found {
		types[d.Type] = true
	}
	if !types["compose"] {
		t.Error("expected compose to be detected")
	}
	if !types["dockerfile"] {
		t.Error("expected dockerfile to be detected")
	}
}

func TestDetectExistingConfigs_Devcontainer(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
	os.WriteFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"), []byte("{}"), 0644)

	found := DetectExistingConfigs(dir)
	if len(found) != 1 {
		t.Fatalf("expected 1 detected config, got %d", len(found))
	}
	if found[0].Type != "devcontainer" {
		t.Errorf("expected devcontainer, got %s", found[0].Type)
	}
}

func TestDetectExistingConfigs_Empty(t *testing.T) {
	dir := t.TempDir()
	found := DetectExistingConfigs(dir)
	if len(found) != 0 {
		t.Fatalf("expected 0 detected configs, got %d", len(found))
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	cfg := &DevboxConfig{
		Name:     "testproject",
		Server:   "dev1",
		Services: []string{"mysql:8.0", "redis:7"},
		Ports:    map[string]int{"app": 8080, "mysql": 3306},
	}

	if err := WriteConfig(cfg, path); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify by loading
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "testproject" {
		t.Errorf("expected name testproject, got %s", loaded.Name)
	}
	if loaded.Server != "dev1" {
		t.Errorf("expected server dev1, got %s", loaded.Server)
	}
	if len(loaded.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(loaded.Services))
	}
	if loaded.Ports["app"] != 8080 {
		t.Errorf("expected port app:8080, got %d", loaded.Ports["app"])
	}
}

func TestWriteConfig_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte("existing"), 0644)

	cfg := &DevboxConfig{Name: "test", Server: "dev1"}
	err := WriteConfig(cfg, path)
	if err == nil {
		t.Fatal("expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestConvertFromCompose(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	composeContent := `services:
  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`
	os.WriteFile(composePath, []byte(composeContent), 0644)

	cfg, err := ConvertFromCompose(composePath)
	if err != nil {
		t.Fatalf("ConvertFromCompose failed: %v", err)
	}

	if len(cfg.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(cfg.Services))
	}

	serviceSet := map[string]bool{}
	for _, s := range cfg.Services {
		serviceSet[s] = true
	}
	if !serviceSet["mysql:8.0"] {
		t.Error("expected mysql:8.0 in services")
	}
	if !serviceSet["redis:7-alpine"] {
		t.Error("expected redis:7-alpine in services")
	}

	if cfg.Ports["mysql"] != 3306 {
		t.Errorf("expected mysql port 3306, got %d", cfg.Ports["mysql"])
	}
	if cfg.Ports["redis"] != 6379 {
		t.Errorf("expected redis port 6379, got %d", cfg.Ports["redis"])
	}

	// Name and Server should be empty
	if cfg.Name != "" {
		t.Errorf("expected empty name, got %s", cfg.Name)
	}
	if cfg.Server != "" {
		t.Errorf("expected empty server, got %s", cfg.Server)
	}
}

func TestConvertFromCompose_BuildOnly(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	composeContent := `services:
  app:
    build: .
    ports:
      - "8080:80"
`
	os.WriteFile(composePath, []byte(composeContent), 0644)

	_, err := ConvertFromCompose(composePath)
	if err == nil {
		t.Fatal("expected error for build-only services")
	}
	if !strings.Contains(err.Error(), "no image-based services") {
		t.Errorf("expected 'no image-based services' error, got: %v", err)
	}
}

func TestConvertFromCompose_Invalid(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	os.WriteFile(composePath, []byte("{{invalid yaml"), 0644)

	_, err := ConvertFromCompose(composePath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestPromptString(t *testing.T) {
	input := strings.NewReader("myproject\n")
	scanner := bufio.NewScanner(input)
	var out bytes.Buffer

	result := PromptString(&out, scanner, "Project name", "default")
	if result != "myproject" {
		t.Errorf("expected myproject, got %s", result)
	}
}

func TestPromptString_Default(t *testing.T) {
	input := strings.NewReader("\n")
	scanner := bufio.NewScanner(input)
	var out bytes.Buffer

	result := PromptString(&out, scanner, "Project name", "default")
	if result != "default" {
		t.Errorf("expected default, got %s", result)
	}
}

func TestPromptRequired(t *testing.T) {
	input := strings.NewReader("\n\ndev1\n")
	scanner := bufio.NewScanner(input)
	var out bytes.Buffer

	result := PromptRequired(&out, scanner, "Server")
	if result != "dev1" {
		t.Errorf("expected dev1, got %s", result)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"mysql:8.0, redis:7", 2},
		{"single", 1},
		{"", 0},
		{"  ", 0},
		{"a, , b", 2},
	}

	for _, tt := range tests {
		result := ParseCommaSeparated(tt.input)
		if len(result) != tt.expected {
			t.Errorf("ParseCommaSeparated(%q) = %d items, want %d", tt.input, len(result), tt.expected)
		}
	}
}

func TestParsePortMappings(t *testing.T) {
	ports := ParsePortMappings("app:8080, db:3306")
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if ports["app"] != 8080 {
		t.Errorf("expected app:8080, got %d", ports["app"])
	}
	if ports["db"] != 3306 {
		t.Errorf("expected db:3306, got %d", ports["db"])
	}
}

func TestParsePortMappings_Empty(t *testing.T) {
	ports := ParsePortMappings("")
	if ports != nil {
		t.Errorf("expected nil for empty input, got %v", ports)
	}
}
