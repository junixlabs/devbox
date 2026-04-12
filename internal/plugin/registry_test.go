package plugin

import (
	"testing"
)

// mockProvider is a minimal Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Create(string, string, CreateOpts) error { return nil }
func (m *mockProvider) Start(string) error                      { return nil }
func (m *mockProvider) Stop(string) error                       { return nil }
func (m *mockProvider) Destroy(string) error                    { return nil }
func (m *mockProvider) Status(string) (string, error)           { return "running", nil }
func (m *mockProvider) ProviderName() string                    { return m.name }

// mockHook is a minimal Hook for testing.
type mockHook struct {
	name   string
	events []Event
}

func (h *mockHook) Execute(HookContext) error { return nil }
func (h *mockHook) Events() []Event           { return h.events }
func (h *mockHook) HookName() string          { return h.name }

func TestRegisterAndGetProvider(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test-provider"}
	m := Manifest{Name: "test-provider", Version: "1.0", Type: TypeProvider, Entrypoint: "test"}

	if err := r.RegisterProvider("test-provider", p, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := r.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderName() != "test-provider" {
		t.Errorf("got provider name %q, want %q", got.ProviderName(), "test-provider")
	}
}

func TestRegisterDuplicateProvider(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "dup"}
	m := Manifest{Name: "dup", Version: "1.0", Type: TypeProvider, Entrypoint: "test"}

	if err := r.RegisterProvider("dup", p, m); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}

	err := r.RegisterProvider("dup", p, m)
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}
}

func TestGetNonExistentProvider(t *testing.T) {
	r := NewRegistry()

	_, err := r.GetProvider("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent provider, got nil")
	}
}

func TestListPlugins(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "p1"}
	h := &mockHook{name: "h1", events: []Event{PreCreate}}

	_ = r.RegisterProvider("p1", p, Manifest{Name: "p1", Version: "1.0", Type: TypeProvider, Entrypoint: "p1"})
	_ = r.RegisterHook("h1", h, Manifest{Name: "h1", Version: "1.0", Type: TypeHook, Entrypoint: "h1"})

	plugins := r.ListPlugins()
	if len(plugins) != 2 {
		t.Errorf("got %d plugins, want 2", len(plugins))
	}
}

func TestGetHooks(t *testing.T) {
	r := NewRegistry()
	h1 := &mockHook{name: "h1", events: []Event{PreCreate, PostCreate}}
	h2 := &mockHook{name: "h2", events: []Event{PreStop}}

	_ = r.RegisterHook("h1", h1, Manifest{Name: "h1", Version: "1.0", Type: TypeHook, Entrypoint: "h1"})
	_ = r.RegisterHook("h2", h2, Manifest{Name: "h2", Version: "1.0", Type: TypeHook, Entrypoint: "h2"})

	preCreateHooks := r.GetHooks(PreCreate)
	if len(preCreateHooks) != 1 {
		t.Errorf("got %d pre_create hooks, want 1", len(preCreateHooks))
	}

	preStopHooks := r.GetHooks(PreStop)
	if len(preStopHooks) != 1 {
		t.Errorf("got %d pre_stop hooks, want 1", len(preStopHooks))
	}

	postDestroyHooks := r.GetHooks(PostDestroy)
	if len(postDestroyHooks) != 0 {
		t.Errorf("got %d post_destroy hooks, want 0", len(postDestroyHooks))
	}
}

func TestRegisterDuplicateHook(t *testing.T) {
	r := NewRegistry()
	h := &mockHook{name: "dup", events: []Event{PreCreate}}
	m := Manifest{Name: "dup", Version: "1.0", Type: TypeHook, Entrypoint: "test"}

	if err := r.RegisterHook("dup", h, m); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}

	err := r.RegisterHook("dup", h, m)
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}
}
