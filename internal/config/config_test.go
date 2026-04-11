package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	yaml := `name: myproject
server: devbox-vps
repo: git@github.com:org/repo.git
branch: develop
services:
  - mysql:8.0
  - redis:7
ports:
  app: 8080
  db: 3306
env:
  APP_ENV: local
  DEBUG: "true"
`
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Name != "myproject" {
		t.Errorf("Name = %q, want %q", cfg.Name, "myproject")
	}
	if cfg.Server != "devbox-vps" {
		t.Errorf("Server = %q, want %q", cfg.Server, "devbox-vps")
	}
	if cfg.Repo != "git@github.com:org/repo.git" {
		t.Errorf("Repo = %q, want %q", cfg.Repo, "git@github.com:org/repo.git")
	}
	if cfg.Branch != "develop" {
		t.Errorf("Branch = %q, want %q", cfg.Branch, "develop")
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("Services count = %d, want 2", len(cfg.Services))
	}
	if cfg.Services[0] != "mysql:8.0" {
		t.Errorf("Services[0] = %q, want %q", cfg.Services[0], "mysql:8.0")
	}
	if cfg.Ports["app"] != 8080 {
		t.Errorf("Ports[app] = %d, want 8080", cfg.Ports["app"])
	}
	if cfg.Ports["db"] != 3306 {
		t.Errorf("Ports[db] = %d, want 3306", cfg.Ports["db"])
	}
	if cfg.Env["APP_ENV"] != "local" {
		t.Errorf("Env[APP_ENV] = %q, want %q", cfg.Env["APP_ENV"], "local")
	}
	if cfg.Env["DEBUG"] != "true" {
		t.Errorf("Env[DEBUG] = %q, want %q", cfg.Env["DEBUG"], "true")
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte("name: minimal\nserver: s1\n"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Name != "minimal" {
		t.Errorf("Name = %q, want %q", cfg.Name, "minimal")
	}
	if cfg.Server != "s1" {
		t.Errorf("Server = %q, want %q", cfg.Server, "s1")
	}
	if cfg.Branch != "" {
		t.Errorf("Branch = %q, want empty", cfg.Branch)
	}
	if len(cfg.Services) != 0 {
		t.Errorf("Services = %v, want empty", cfg.Services)
	}
}

func TestLoad_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte("server: s1\n"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "'name' is required") {
		t.Errorf("error = %q, want it to contain 'name' is required", err.Error())
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if ce.GetSuggestion() == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestLoad_MissingServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte("name: myproject\n"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing server")
	}
	if !strings.Contains(err.Error(), "'server' is required") {
		t.Errorf("error = %q, want it to contain 'server' is required", err.Error())
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/devbox.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if !strings.Contains(ce.GetSuggestion(), "devbox init") {
		t.Errorf("suggestion = %q, want it to mention devbox init", ce.GetSuggestion())
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte("{{invalid yaml content"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	os.WriteFile(path, []byte(""), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestLoadFromDir_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigFile)

	os.WriteFile(path, []byte("name: dirtest\nserver: s1\n"), 0644)

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir() error: %v", err)
	}
	if cfg.Name != "dirtest" {
		t.Errorf("Name = %q, want %q", cfg.Name, "dirtest")
	}
}

func TestLoadFromDir_NoFile(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error for missing devbox.yaml in directory")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %T", err)
	}
}

func TestDefaultConfigFile(t *testing.T) {
	if DefaultConfigFile != "devbox.yaml" {
		t.Errorf("DefaultConfigFile = %q, want %q", DefaultConfigFile, "devbox.yaml")
	}
}
