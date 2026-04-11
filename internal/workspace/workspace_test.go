package workspace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusRunning, "running"},
		{StatusStopped, "stopped"},
		{StatusCreating, "creating"},
		{StatusError, "error"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("Status = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestWorkspaceStruct_JSONRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	started := now.Add(-time.Hour)
	ws := Workspace{
		Name:       "test-ws",
		Project:    "myproject",
		Branch:     "main",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		Ports:      map[string]int{"app": 8080, "db": 3306},
		Env:        map[string]string{"APP_ENV": "local"},
		CreatedAt:  now,
		StartedAt:  &started,
		StoppedAt:  nil,
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Workspace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != ws.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, ws.Name)
	}
	if decoded.Project != ws.Project {
		t.Errorf("Project = %q, want %q", decoded.Project, ws.Project)
	}
	if decoded.Status != ws.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, ws.Status)
	}
	if decoded.ServerHost != ws.ServerHost {
		t.Errorf("ServerHost = %q, want %q", decoded.ServerHost, ws.ServerHost)
	}
	if decoded.Ports["app"] != 8080 {
		t.Errorf("Ports[app] = %d, want 8080", decoded.Ports["app"])
	}
	if decoded.Env["APP_ENV"] != "local" {
		t.Errorf("Env[APP_ENV] = %q, want %q", decoded.Env["APP_ENV"], "local")
	}
	if decoded.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
	if decoded.StoppedAt != nil {
		t.Error("StoppedAt should be nil")
	}

	// Verify JSON field names (snake_case).
	jsonStr := string(data)
	for _, field := range []string{"server_host", "created_at", "started_at"} {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing field %q", field)
		}
	}
	// stopped_at should be omitted when nil.
	if strings.Contains(jsonStr, "stopped_at") {
		t.Error("stopped_at should be omitted when nil")
	}
}

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestStubManager_AllMethodsReturnError(t *testing.T) {
	mgr := NewManager()

	t.Run("Create", func(t *testing.T) {
		ws, err := mgr.Create("test", "project", "main")
		if err == nil {
			t.Fatal("expected error")
		}
		if ws != nil {
			t.Error("expected nil workspace")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want it to contain 'not implemented'", err.Error())
		}
	})

	t.Run("Start", func(t *testing.T) {
		err := mgr.Start("test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})

	t.Run("Stop", func(t *testing.T) {
		err := mgr.Stop("test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})

	t.Run("Destroy", func(t *testing.T) {
		err := mgr.Destroy("test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})

	t.Run("List", func(t *testing.T) {
		ws, err := mgr.List()
		if err == nil {
			t.Fatal("expected error")
		}
		if ws != nil {
			t.Error("expected nil workspace list")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})

	t.Run("Get", func(t *testing.T) {
		ws, err := mgr.Get("test")
		if err == nil {
			t.Fatal("expected error")
		}
		if ws != nil {
			t.Error("expected nil workspace")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})

	t.Run("SSH", func(t *testing.T) {
		err := mgr.SSH("test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("error = %q, want 'not implemented'", err.Error())
		}
	})
}
