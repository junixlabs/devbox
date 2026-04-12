package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `name: my-plugin
version: "1.0.0"
type: provider
entrypoint: ./my-plugin
description: A test plugin
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Name != "my-plugin" {
		t.Errorf("got name %q, want %q", m.Name, "my-plugin")
	}
	if m.Version != "1.0.0" {
		t.Errorf("got version %q, want %q", m.Version, "1.0.0")
	}
	if m.Type != TypeProvider {
		t.Errorf("got type %q, want %q", m.Type, TypeProvider)
	}
	if m.Entrypoint != "./my-plugin" {
		t.Errorf("got entrypoint %q, want %q", m.Entrypoint, "./my-plugin")
	}
	if m.Description != "A test plugin" {
		t.Errorf("got description %q, want %q", m.Description, "A test plugin")
	}
}

func TestLoadManifest_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `version: "1.0.0"
type: provider
entrypoint: ./my-plugin
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestLoadManifest_InvalidType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `name: bad-plugin
version: "1.0.0"
type: invalid
entrypoint: ./bad
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/plugin.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadManifest_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `name: no-version
type: hook
entrypoint: ./hook
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
}

func TestLoadManifest_MissingEntrypoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `name: no-entry
version: "1.0.0"
type: hook
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for missing entrypoint, got nil")
	}
}

func TestLoadManifest_HookWithEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `name: my-hook
version: "1.0.0"
type: hook
entrypoint: ./my-hook
events:
  - pre_create
  - post_destroy
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Events) != 2 {
		t.Errorf("got %d events, want 2", len(m.Events))
	}
	if m.Events[0] != PreCreate {
		t.Errorf("got event[0] %q, want %q", m.Events[0], PreCreate)
	}
	if m.Events[1] != PostDestroy {
		t.Errorf("got event[1] %q, want %q", m.Events[1], PostDestroy)
	}
}
