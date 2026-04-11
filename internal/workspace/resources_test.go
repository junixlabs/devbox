package workspace

import (
	"testing"
)

func TestParseDockerStats(t *testing.T) {
	output := "myapp-redis-1\t0.15%\t12.5MiB / 4GiB\t0.30%\n" +
		"myapp-mysql-1\t2.50%\t256MiB / 4GiB\t6.25%\n"

	stats, err := ParseDockerStats(output)
	if err != nil {
		t.Fatalf("ParseDockerStats() error: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("got %d entries, want 2", len(stats))
	}

	redis := stats["myapp-redis-1"]
	if redis == nil {
		t.Fatal("missing myapp-redis-1")
	}
	if redis.CPUPercent != 0.15 {
		t.Errorf("redis CPUPercent = %g, want 0.15", redis.CPUPercent)
	}
	if redis.MemoryPercent != 0.30 {
		t.Errorf("redis MemoryPercent = %g, want 0.30", redis.MemoryPercent)
	}

	mysql := stats["myapp-mysql-1"]
	if mysql == nil {
		t.Fatal("missing myapp-mysql-1")
	}
	if mysql.CPUPercent != 2.50 {
		t.Errorf("mysql CPUPercent = %g, want 2.50", mysql.CPUPercent)
	}
}

func TestParseDockerStats_Empty(t *testing.T) {
	stats, err := ParseDockerStats("")
	if err != nil {
		t.Fatalf("ParseDockerStats() error: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty map, got %d entries", len(stats))
	}
}

func TestParseServerResources(t *testing.T) {
	cpuOut := "8\n"
	memOut := "MemTotal:       16384000 kB\nMemFree:         8192000 kB\n"

	info, err := ParseServerResources(cpuOut, memOut)
	if err != nil {
		t.Fatalf("ParseServerResources() error: %v", err)
	}
	if info.TotalCPUs != 8 {
		t.Errorf("TotalCPUs = %d, want 8", info.TotalCPUs)
	}
	if info.TotalMemoryBytes != 16384000*1024 {
		t.Errorf("TotalMemoryBytes = %d, want %d", info.TotalMemoryBytes, 16384000*1024)
	}
}

func TestParseServerResources_BadCPU(t *testing.T) {
	_, err := ParseServerResources("abc", "MemTotal: 16384000 kB\n")
	if err == nil {
		t.Error("expected error for bad CPU output")
	}
}

func TestParseServerResources_NoMemTotal(t *testing.T) {
	_, err := ParseServerResources("4", "MemFree: 8192000 kB\n")
	if err == nil {
		t.Error("expected error for missing MemTotal")
	}
}

func TestCheckLowResources(t *testing.T) {
	info := &ServerResourceInfo{
		TotalCPUs:        8,
		TotalMemoryBytes: 16 * 1024 * 1024 * 1024,
		UsedCPUs:         7.2,
		UsedMemoryBytes:  14 * 1024 * 1024 * 1024,
	}

	warnings := CheckLowResources(info, 85.0)
	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d: %v", len(warnings), warnings)
	}

	// Below threshold — no warnings.
	info.UsedCPUs = 4
	info.UsedMemoryBytes = 8 * 1024 * 1024 * 1024
	warnings = CheckLowResources(info, 85.0)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestCheckLowResources_Nil(t *testing.T) {
	warnings := CheckLowResources(nil, 85.0)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for nil, got %d", len(warnings))
	}
}

func TestFormatResourceUsage(t *testing.T) {
	ru := &ResourceUsage{
		CPUPercent:    25.5,
		MemoryUsed:    512 * 1024 * 1024,
		MemoryLimit:   4 * 1024 * 1024 * 1024,
		MemoryPercent: 12.5,
	}
	cpu, mem := FormatResourceUsage(ru)
	if cpu != "25.5%" {
		t.Errorf("cpu = %q, want %q", cpu, "25.5%")
	}
	if mem == "-" {
		t.Error("mem should not be '-'")
	}

	// Nil usage returns dashes.
	cpu, mem = FormatResourceUsage(nil)
	if cpu != "-" || mem != "-" {
		t.Errorf("expected '-', '-', got %q, %q", cpu, mem)
	}
}

func TestParseMemValue(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"123.4MiB", 129394278},   // 123.4 * 1024 * 1024
		{"7.775GiB", 8348342681},  // 7.775 * 1024^3
		{"512KiB", 512 * 1024},
		{"100B", 100},
		{"", 0},
		{"bad", 0},
	}
	for _, tt := range tests {
		got := parseMemValue(tt.input)
		// Allow some tolerance for float conversion.
		diff := got - tt.want
		if diff < 0 {
			diff = -diff
		}
		if diff > 1024*1024 { // 1MB tolerance
			t.Errorf("parseMemValue(%q) = %d, want ~%d", tt.input, got, tt.want)
		}
	}
}
