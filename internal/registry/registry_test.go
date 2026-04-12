package registry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/junixlabs/devbox/internal/template"
)

const testIndex = `templates:
  - name: django
    version: "1.0.0"
    description: Django Python application
    url: templates/django.yaml
  - name: rails
    version: "2.0.0"
    description: Ruby on Rails application
    url: templates/rails.yaml
  - name: nextjs
    version: "1.0.0"
    description: Next.js React application
    url: templates/nextjs.yaml
`

const testDjangoTemplate = `name: django
description: Django Python application with PostgreSQL and Redis
version: "1.0.0"
services:
  - postgres:16
  - redis:7
ports:
  app: 8000
env:
  DJANGO_SETTINGS_MODULE: config.settings.local
setup:
  - pip install -r requirements.txt
`

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testIndex))
	})
	mux.HandleFunc("/templates/django.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testDjangoTemplate))
	})
	mux.HandleFunc("/templates/rails.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return httptest.NewServer(mux)
}

func TestFetchIndex(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	reg := NewRemoteRegistry(srv.URL)
	entries, err := reg.FetchIndex()
	if err != nil {
		t.Fatalf("FetchIndex() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Name != "django" {
		t.Errorf("expected first entry name %q, got %q", "django", entries[0].Name)
	}
}

func TestFetchIndexHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reg := NewRemoteRegistry(srv.URL)
	_, err := reg.FetchIndex()
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestSearch(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	reg := NewRemoteRegistry(srv.URL)

	tests := []struct {
		query    string
		expected int
	}{
		{"django", 1},
		{"DJANGO", 1}, // case-insensitive
		{"application", 3},
		{"react", 1},
		{"nonexistent", 0},
		{"", 0}, // empty query returns nothing
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := reg.Search(tt.query)
			if err != nil {
				t.Fatalf("Search(%q) error: %v", tt.query, err)
			}
			if len(results) != tt.expected {
				t.Errorf("Search(%q) = %d results, want %d", tt.query, len(results), tt.expected)
			}
		})
	}
}

func TestPull(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	dir := t.TempDir()
	localReg := template.NewLocalRegistry(dir, nil)
	reg := NewRemoteRegistry(srv.URL)

	tmpl, err := reg.Pull("django", localReg)
	if err != nil {
		t.Fatalf("Pull() error: %v", err)
	}
	if tmpl.Name != "django" {
		t.Errorf("expected name %q, got %q", "django", tmpl.Name)
	}
	if tmpl.Version != "1.0.0" {
		t.Errorf("expected version %q, got %q", "1.0.0", tmpl.Version)
	}
	if len(tmpl.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(tmpl.Services))
	}

	// Verify saved to local registry.
	got, err := localReg.Get("django")
	if err != nil {
		t.Fatalf("local Get() error: %v", err)
	}
	if got.Name != "django" {
		t.Errorf("expected saved name %q, got %q", "django", got.Name)
	}
}

func TestPullNotFound(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	dir := t.TempDir()
	localReg := template.NewLocalRegistry(dir, nil)
	reg := NewRemoteRegistry(srv.URL)

	_, err := reg.Pull("nonexistent", localReg)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestPush(t *testing.T) {
	dir := t.TempDir()
	localReg := template.NewLocalRegistry(dir, nil)

	// Save a template locally first.
	tmpl := &template.Template{
		Name:        "my-app",
		Description: "My custom app",
		Version:     "1.0.0",
	}
	if err := localReg.Save(tmpl); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reg := NewRemoteRegistry("")
	output, err := reg.Push("my-app", localReg)
	if err != nil {
		t.Fatalf("Push() error: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty YAML output")
	}
}

func TestPullUnsafeURL(t *testing.T) {
	unsafeIndex := `templates:
  - name: evil
    version: "1.0.0"
    description: Malicious template
    url: "../../etc/passwd"
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(unsafeIndex))
	}))
	defer srv.Close()

	dir := t.TempDir()
	localReg := template.NewLocalRegistry(dir, nil)
	reg := NewRemoteRegistry(srv.URL)

	_, err := reg.Pull("evil", localReg)
	if err == nil {
		t.Fatal("expected error for unsafe URL with path traversal")
	}
}

func TestPushNotFound(t *testing.T) {
	dir := t.TempDir()
	localReg := template.NewLocalRegistry(dir, nil)

	reg := NewRemoteRegistry("")
	_, err := reg.Push("nonexistent", localReg)
	if err == nil {
		t.Fatal("expected error for nonexistent local template")
	}
}
