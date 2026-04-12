package server

import (
	"math"
	"testing"
)

func TestParseFreeOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantTotal int64
		wantUsed  int64
		wantErr   bool
	}{
		{
			name: "typical free -b output",
			output: `              total        used        free      shared  buff/cache   available
Mem:     8351916032  2147483648  4194304000   134217728  2010128384  5905580032
Swap:    2147483648           0  2147483648`,
			wantTotal: 8351916032,
			wantUsed:  2147483648,
		},
		{
			name: "minimal output",
			output: `              total        used        free
Mem:     16000000000  4000000000  12000000000`,
			wantTotal: 16000000000,
			wantUsed:  4000000000,
		},
		{
			name:    "no Mem line",
			output:  "Swap:    2147483648           0  2147483648\n",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "malformed Mem line",
			output:  "Mem: abc xyz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, used, err := parseFreeOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFreeOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if used != tt.wantUsed {
				t.Errorf("used = %d, want %d", used, tt.wantUsed)
			}
		})
	}
}

func TestParseLoadAvg(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   float64
	}{
		{
			name:   "typical /proc/loadavg",
			output: "1.23 0.98 0.75 1/234 5678\n",
			want:   1.23,
		},
		{
			name:   "zero load",
			output: "0.00 0.00 0.00 1/100 1234\n",
			want:   0.0,
		},
		{
			name:   "high load",
			output: "8.50 7.20 6.10 5/300 9999\n",
			want:   8.50,
		},
		{
			name:   "empty output",
			output: "",
			want:   0.0,
		},
		{
			name:   "non-numeric",
			output: "abc def\n",
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLoadAvg(tt.output)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("parseLoadAvg() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestAvailableScore(t *testing.T) {
	tests := []struct {
		name string
		info *ResourceInfo
		want float64
	}{
		{
			name: "fully idle server",
			info: &ResourceInfo{
				TotalCPUs:        4,
				UsedCPUPercent:   0,
				TotalMemoryBytes: 8_000_000_000,
				UsedMemoryBytes:  0,
			},
			want: 1.0,
		},
		{
			name: "fully loaded server",
			info: &ResourceInfo{
				TotalCPUs:        4,
				UsedCPUPercent:   100,
				TotalMemoryBytes: 8_000_000_000,
				UsedMemoryBytes:  8_000_000_000,
			},
			want: 0.0,
		},
		{
			name: "half loaded server",
			info: &ResourceInfo{
				TotalCPUs:        4,
				UsedCPUPercent:   50,
				TotalMemoryBytes: 8_000_000_000,
				UsedMemoryBytes:  4_000_000_000,
			},
			want: 0.5,
		},
		{
			name: "80% CPU 30% memory",
			info: &ResourceInfo{
				TotalCPUs:        8,
				UsedCPUPercent:   80,
				TotalMemoryBytes: 16_000_000_000,
				UsedMemoryBytes:  4_800_000_000, // 30%
			},
			want: 0.5*0.2 + 0.5*0.7, // 0.45
		},
		{
			name: "nil info",
			info: nil,
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AvailableScore(tt.info)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("AvailableScore() = %f, want %f", got, tt.want)
			}
		})
	}
}
