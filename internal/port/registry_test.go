package port

import (
	"path/filepath"
	"testing"
)

func newTestRegistry(t *testing.T, portRange PortRange) Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ports.json")
	return NewFileRegistry(path, portRange)
}

func TestAllocateAutoAssign(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	p1, err := r.Allocate("ws1", "web", nil)
	if err != nil {
		t.Fatalf("Allocate ws1/web: %v", err)
	}
	if p1 != 10000 {
		t.Errorf("expected port 10000, got %d", p1)
	}

	p2, err := r.Allocate("ws2", "web", nil)
	if err != nil {
		t.Fatalf("Allocate ws2/web: %v", err)
	}
	if p2 != 10001 {
		t.Errorf("expected port 10001, got %d", p2)
	}

	if p1 == p2 {
		t.Error("auto-assigned ports should be unique")
	}
}

func TestAllocateIdempotent(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	p1, err := r.Allocate("ws1", "web", nil)
	if err != nil {
		t.Fatalf("first allocate: %v", err)
	}

	p2, err := r.Allocate("ws1", "web", nil)
	if err != nil {
		t.Fatalf("second allocate: %v", err)
	}

	if p1 != p2 {
		t.Errorf("idempotent allocate: got %d then %d", p1, p2)
	}
}

func TestAllocateManualOverride(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	override := 8080
	p, err := r.Allocate("ws1", "web", &override)
	if err != nil {
		t.Fatalf("Allocate with override: %v", err)
	}
	if p != 8080 {
		t.Errorf("expected override port 8080, got %d", p)
	}
}

func TestAllocateManualConflict(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	override := 8080
	if _, err := r.Allocate("ws1", "web", &override); err != nil {
		t.Fatalf("first allocate: %v", err)
	}

	_, err := r.Allocate("ws2", "api", &override)
	if err == nil {
		t.Fatal("expected conflict error for duplicate manual port")
	}
}

func TestCheckConflicts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ports.json")
	r := NewFileRegistry(path, PortRange{Min: 10000, Max: 10010})

	// Allocate non-conflicting ports.
	if _, err := r.Allocate("ws1", "web", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws2", "web", nil); err != nil {
		t.Fatal(err)
	}

	conflicts, err := r.CheckConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestRelease(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10002})

	if _, err := r.Allocate("ws1", "web", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws1", "api", nil); err != nil {
		t.Fatal(err)
	}

	if err := r.Release("ws1"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	allocs, err := r.GetAllocations("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allocs) != 0 {
		t.Errorf("expected 0 allocations after release, got %d", len(allocs))
	}

	// Released port should be reusable.
	p, err := r.Allocate("ws2", "web", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p != 10000 {
		t.Errorf("expected released port 10000 to be reused, got %d", p)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ports.json")
	portRange := PortRange{Min: 10000, Max: 10010}

	r1 := NewFileRegistry(path, portRange)
	if _, err := r1.Allocate("ws1", "web", nil); err != nil {
		t.Fatal(err)
	}

	// Create a new registry from the same file — allocations should persist.
	r2 := NewFileRegistry(path, portRange)
	allocs, err := r2.GetAllocations("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if port, ok := allocs["web"]; !ok || port != 10000 {
		t.Errorf("expected persisted allocation web→10000, got %v", allocs)
	}
}

func TestRangeExhaustion(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10001})

	if _, err := r.Allocate("ws1", "a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws2", "b", nil); err != nil {
		t.Fatal(err)
	}

	_, err := r.Allocate("ws3", "c", nil)
	if err == nil {
		t.Fatal("expected range exhaustion error")
	}
}

func TestGetAllocations(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	if _, err := r.Allocate("ws1", "web", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws1", "api", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws2", "web", nil); err != nil {
		t.Fatal(err)
	}

	allocs, err := r.GetAllocations("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allocs) != 2 {
		t.Errorf("expected 2 allocations for ws1, got %d", len(allocs))
	}
	if allocs["web"] != 10000 {
		t.Errorf("expected web→10000, got %d", allocs["web"])
	}
	if allocs["api"] != 10001 {
		t.Errorf("expected api→10001, got %d", allocs["api"])
	}
}

func TestListAll(t *testing.T) {
	r := newTestRegistry(t, PortRange{Min: 10000, Max: 10010})

	if _, err := r.Allocate("ws1", "web", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate("ws2", "api", nil); err != nil {
		t.Fatal(err)
	}

	all, err := r.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 total allocations, got %d", len(all))
	}
}
