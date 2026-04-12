package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/junixlabs/devbox/internal/config"
)

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr bool
	}{
		{
			name:    "valid template",
			tmpl:    Template{Name: "laravel", Description: "PHP app"},
			wantErr: false,
		},
		{
			name:    "empty name",
			tmpl:    Template{},
			wantErr: true,
		},
		{
			name:    "invalid name with uppercase",
			tmpl:    Template{Name: "Laravel"},
			wantErr: true,
		},
		{
			name:    "invalid name with spaces",
			tmpl:    Template{Name: "my template"},
			wantErr: true,
		},
		{
			name:    "valid name with hyphens",
			tmpl:    Template{Name: "my-template"},
			wantErr: false,
		},
		{
			name:    "invalid resources",
			tmpl:    Template{Name: "bad", Resources: &config.Resources{CPUs: -1}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToDevboxConfig(t *testing.T) {
	tmpl := &Template{
		Name:     "laravel",
		Services: []string{"mysql:8.0"},
		Ports:    map[string]int{"app": 8000},
		Env:      map[string]string{"APP_ENV": "local"},
	}

	cfg := tmpl.ToDevboxConfig("my-project", "dev-server")
	if cfg.Name != "my-project" {
		t.Errorf("expected name %q, got %q", "my-project", cfg.Name)
	}
	if cfg.Server != "dev-server" {
		t.Errorf("expected server %q, got %q", "dev-server", cfg.Server)
	}
	if len(cfg.Services) != 1 || cfg.Services[0] != "mysql:8.0" {
		t.Errorf("expected services [mysql:8.0], got %v", cfg.Services)
	}
	if cfg.Ports["app"] != 8000 {
		t.Errorf("expected port app:8000, got %v", cfg.Ports)
	}
}

func TestMatchesQuery(t *testing.T) {
	tmpl := &Template{Name: "laravel", Description: "PHP framework with MySQL"}

	tests := []struct {
		query string
		want  bool
	}{
		{"laravel", true},
		{"LARAVEL", true},
		{"PHP", true},
		{"mysql", true},
		{"nonexistent", false},
		{"lara", true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := tmpl.MatchesQuery(tt.query); got != tt.want {
				t.Errorf("MatchesQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestVersionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	reg := NewLocalRegistry(dir, nil)

	tmpl := &Template{
		Name:    "versioned-app",
		Version: "2.1.0",
	}
	if err := reg.Save(tmpl); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := reg.Get("versioned-app")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Version != "2.1.0" {
		t.Errorf("expected version %q, got %q", "2.1.0", got.Version)
	}
}

func TestLoadBuiltins(t *testing.T) {
	templates, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins() error: %v", err)
	}
	if len(templates) < 7 {
		t.Errorf("expected at least 7 built-in templates, got %d", len(templates))
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}
	for _, expected := range []string{"laravel", "rails", "nextjs", "go", "python", "django", "rust"} {
		if !names[expected] {
			t.Errorf("expected built-in template %q not found", expected)
		}
	}
}

func TestLocalRegistryListAndGet(t *testing.T) {
	builtins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins() error: %v", err)
	}

	dir := t.TempDir()
	reg := NewLocalRegistry(dir, builtins)

	// List should return builtins.
	all, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(all) < 5 {
		t.Errorf("expected at least 5 templates, got %d", len(all))
	}

	// Get built-in.
	tmpl, err := reg.Get("laravel")
	if err != nil {
		t.Fatalf("Get(laravel) error: %v", err)
	}
	if tmpl.Name != "laravel" {
		t.Errorf("expected name laravel, got %q", tmpl.Name)
	}

	// Get non-existent.
	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestLocalRegistrySave(t *testing.T) {
	dir := t.TempDir()
	reg := NewLocalRegistry(dir, nil)

	tmpl := &Template{
		Name:        "custom-app",
		Description: "My custom app",
		Services:    []string{"redis:7"},
		Ports:       map[string]int{"web": 3000},
	}

	if err := reg.Save(tmpl); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "custom-app.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("template file not created: %v", err)
	}

	// Get it back.
	got, err := reg.Get("custom-app")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Description != "My custom app" {
		t.Errorf("expected description %q, got %q", "My custom app", got.Description)
	}
	if len(got.Services) != 1 || got.Services[0] != "redis:7" {
		t.Errorf("expected services [redis:7], got %v", got.Services)
	}
}

func TestCustomOverridesBuiltin(t *testing.T) {
	builtins := []Template{{Name: "laravel", Description: "built-in"}}
	dir := t.TempDir()
	reg := NewLocalRegistry(dir, builtins)

	// Save a custom template with the same name.
	custom := &Template{Name: "laravel", Description: "custom override"}
	if err := reg.Save(custom); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := reg.Get("laravel")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Description != "custom override" {
		t.Errorf("expected custom override, got %q", got.Description)
	}
}

func TestFromDevboxConfig(t *testing.T) {
	cfg := &config.DevboxConfig{
		Name:     "my-project",
		Server:   "server1",
		Services: []string{"redis:7"},
		Ports:    map[string]int{"web": 3000},
		Env:      map[string]string{"NODE_ENV": "dev"},
	}

	tmpl := FromDevboxConfig(cfg, "my-template", "From existing workspace")
	if tmpl.Name != "my-template" {
		t.Errorf("expected name %q, got %q", "my-template", tmpl.Name)
	}
	if tmpl.Description != "From existing workspace" {
		t.Errorf("expected description %q, got %q", "From existing workspace", tmpl.Description)
	}
	if len(tmpl.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(tmpl.Services))
	}
}
