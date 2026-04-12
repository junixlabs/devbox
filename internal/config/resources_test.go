package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResources_Validate(t *testing.T) {
	tests := []struct {
		name    string
		res     Resources
		wantErr bool
	}{
		{"zero value is valid", Resources{}, false},
		{"valid cpus and memory", Resources{CPUs: 2, Memory: "4g"}, false},
		{"valid fractional cpus", Resources{CPUs: 0.5, Memory: "512m"}, false},
		{"valid uppercase memory", Resources{Memory: "4G"}, false},
		{"negative cpus", Resources{CPUs: -1}, true},
		{"bad memory format", Resources{Memory: "4gb"}, true},
		{"bad memory no unit", Resources{Memory: "4096"}, true},
		{"bad memory text", Resources{Memory: "lots"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.res.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResources_IsZero(t *testing.T) {
	if !(Resources{}).IsZero() {
		t.Error("zero Resources should be zero")
	}
	if (Resources{CPUs: 1}).IsZero() {
		t.Error("Resources with CPUs should not be zero")
	}
	if (Resources{Memory: "1g"}).IsZero() {
		t.Error("Resources with Memory should not be zero")
	}
}

func TestParseMemoryBytes(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		{"", 0, false},
		{"4g", 4 * 1024 * 1024 * 1024, false},
		{"512m", 512 * 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"256M", 256 * 1024 * 1024, false},
		{"bad", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMemoryBytes(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("ParseMemoryBytes(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("ParseMemoryBytes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMergeResources(t *testing.T) {
	tests := []struct {
		name     string
		defaults *Resources
		override *Resources
		wantCPUs float64
		wantMem  string
	}{
		{"both nil", nil, nil, 0, ""},
		{"defaults only", &Resources{CPUs: 1, Memory: "2g"}, nil, 1, "2g"},
		{"override only", nil, &Resources{CPUs: 4, Memory: "8g"}, 4, "8g"},
		{"override wins", &Resources{CPUs: 1, Memory: "2g"}, &Resources{CPUs: 4, Memory: "8g"}, 4, "8g"},
		{"partial override cpus", &Resources{CPUs: 1, Memory: "2g"}, &Resources{CPUs: 4}, 4, "2g"},
		{"partial override memory", &Resources{CPUs: 1, Memory: "2g"}, &Resources{Memory: "8g"}, 1, "8g"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeResources(tt.defaults, tt.override)
			if got.CPUs != tt.wantCPUs {
				t.Errorf("CPUs = %g, want %g", got.CPUs, tt.wantCPUs)
			}
			if got.Memory != tt.wantMem {
				t.Errorf("Memory = %q, want %q", got.Memory, tt.wantMem)
			}
		})
	}
}

func TestLoadWithResources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	content := `name: test
server: devbox-vps
resources:
  cpus: 2
  memory: 4g
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Resources == nil {
		t.Fatal("expected Resources to be set")
	}
	if cfg.Resources.CPUs != 2 {
		t.Errorf("CPUs = %g, want 2", cfg.Resources.CPUs)
	}
	if cfg.Resources.Memory != "4g" {
		t.Errorf("Memory = %q, want %q", cfg.Resources.Memory, "4g")
	}
}

func TestLoadWithInvalidResources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devbox.yaml")

	content := `name: test
server: devbox-vps
resources:
  cpus: -1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for negative CPUs")
	}
}

func TestLoadGlobal_Missing(t *testing.T) {
	// LoadGlobal returns empty config when file doesn't exist.
	gc, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal() error: %v", err)
	}
	if gc == nil {
		t.Fatal("expected non-nil GlobalConfig")
	}
}

func TestGlobalConfig_ServerResourceDefaults(t *testing.T) {
	gc := &GlobalConfig{
		Servers: map[string]ServerDefaults{
			"devbox-vps": {Resources: Resources{CPUs: 1, Memory: "2g"}},
		},
	}

	got := gc.ServerResourceDefaults("devbox-vps")
	if got == nil {
		t.Fatal("expected non-nil Resources")
	}
	if got.CPUs != 1 || got.Memory != "2g" {
		t.Errorf("got %+v, want {1, 2g}", got)
	}

	// Unknown server returns nil.
	if gc.ServerResourceDefaults("unknown") != nil {
		t.Error("expected nil for unknown server")
	}

	// Nil config.
	var nilGC *GlobalConfig
	if nilGC.ServerResourceDefaults("any") != nil {
		t.Error("expected nil for nil GlobalConfig")
	}
}
