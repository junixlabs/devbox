package docker

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/junixlabs/devbox/internal/config"
	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"gopkg.in/yaml.v3"
)

// --- Compose generation tests ---

func TestParseServiceName(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"mysql:8.0", "mysql"},
		{"redis", "redis"},
		{"redis:7-alpine", "redis"},
		{"bitnami/redis:7", "redis"},
		{"ghcr.io/org/myapp:latest", "myapp"},
		{"postgres:16", "postgres"},
	}
	for _, tt := range tests {
		got := parseServiceName(tt.image)
		if got != tt.want {
			t.Errorf("parseServiceName(%q) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

func TestGenerateCompose_FullConfig(t *testing.T) {
	cfg := &config.DevboxConfig{
		Name:     "test-ws",
		Server:   "devbox-vps",
		Services: []string{"mysql:8.0", "redis:7-alpine"},
		Ports:    map[string]int{"mysql": 3306, "redis": 6379},
		Env:      map[string]string{"APP_ENV": "local"},
	}

	data, err := GenerateCompose("test-ws", cfg)
	if err != nil {
		t.Fatalf("GenerateCompose() error: %v", err)
	}

	// Unmarshal back to verify structure.
	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v", err)
	}

	if cf.Name != "test-ws" {
		t.Errorf("name = %q, want %q", cf.Name, "test-ws")
	}

	if len(cf.Services) != 2 {
		t.Fatalf("services count = %d, want 2", len(cf.Services))
	}

	// Check mysql service.
	mysql, ok := cf.Services["mysql"]
	if !ok {
		t.Fatal("missing mysql service")
	}
	if mysql.Image != "mysql:8.0" {
		t.Errorf("mysql.Image = %q, want %q", mysql.Image, "mysql:8.0")
	}
	if mysql.Restart != "unless-stopped" {
		t.Errorf("mysql.Restart = %q, want %q", mysql.Restart, "unless-stopped")
	}
	if len(mysql.Ports) != 1 || mysql.Ports[0] != "3306:3306" {
		t.Errorf("mysql.Ports = %v, want [3306:3306]", mysql.Ports)
	}
	if mysql.Environment["APP_ENV"] != "local" {
		t.Errorf("mysql.Environment[APP_ENV] = %q, want %q", mysql.Environment["APP_ENV"], "local")
	}

	// Check mysql volume.
	foundVol := false
	for _, v := range mysql.Volumes {
		if strings.Contains(v, "/var/lib/mysql") {
			foundVol = true
		}
	}
	if !foundVol {
		t.Error("mysql service missing data volume mount")
	}

	// Check redis service.
	redis, ok := cf.Services["redis"]
	if !ok {
		t.Fatal("missing redis service")
	}
	if redis.Image != "redis:7-alpine" {
		t.Errorf("redis.Image = %q, want %q", redis.Image, "redis:7-alpine")
	}
	if len(redis.Ports) != 1 || redis.Ports[0] != "6379:6379" {
		t.Errorf("redis.Ports = %v, want [6379:6379]", redis.Ports)
	}

	// Check named volumes exist at top level.
	if _, ok := cf.Volumes["test-ws-mysql-data"]; !ok {
		t.Error("missing top-level volume test-ws-mysql-data")
	}
	if _, ok := cf.Volumes["test-ws-redis-data"]; !ok {
		t.Error("missing top-level volume test-ws-redis-data")
	}
}

func TestGenerateCompose_EmptyServices(t *testing.T) {
	cfg := &config.DevboxConfig{
		Name:   "empty",
		Server: "devbox-vps",
	}

	data, err := GenerateCompose("empty", cfg)
	if err != nil {
		t.Fatalf("GenerateCompose() error: %v", err)
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(cf.Services) != 0 {
		t.Errorf("services count = %d, want 0", len(cf.Services))
	}
}

func TestGenerateCompose_NoColonImage(t *testing.T) {
	cfg := &config.DevboxConfig{
		Name:     "test",
		Server:   "s",
		Services: []string{"redis"},
		Ports:    map[string]int{"redis": 6379},
	}

	data, err := GenerateCompose("test", cfg)
	if err != nil {
		t.Fatalf("GenerateCompose() error: %v", err)
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	redis, ok := cf.Services["redis"]
	if !ok {
		t.Fatal("missing redis service")
	}
	if redis.Image != "redis" {
		t.Errorf("redis.Image = %q, want %q", redis.Image, "redis")
	}
	if len(redis.Ports) != 1 || redis.Ports[0] != "6379:6379" {
		t.Errorf("redis.Ports = %v, want [6379:6379]", redis.Ports)
	}
}

func TestGenerateCompose_DuplicateServiceNames(t *testing.T) {
	cfg := &config.DevboxConfig{
		Name:     "test",
		Server:   "s",
		Services: []string{"mysql:8.0", "bitnami/mysql:8.0"},
	}

	data, err := GenerateCompose("test", cfg)
	if err != nil {
		t.Fatalf("GenerateCompose() error: %v", err)
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(cf.Services) != 2 {
		t.Fatalf("services count = %d, want 2 (dedup should rename)", len(cf.Services))
	}
	if _, ok := cf.Services["mysql"]; !ok {
		t.Error("missing mysql service")
	}
	if _, ok := cf.Services["mysql-2"]; !ok {
		t.Error("missing mysql-2 service (renamed duplicate)")
	}
}

// --- Mock SSH executor ---

type sshCall struct {
	method  string
	host    string
	command string
}

type mockExecutor struct {
	calls     []sshCall
	runOut    string
	runErr    error
	copyErr   error
	streamErr error
}

func (m *mockExecutor) Run(_ context.Context, host string, command string) (string, string, error) {
	m.calls = append(m.calls, sshCall{method: "Run", host: host, command: command})
	return m.runOut, "", m.runErr
}

func (m *mockExecutor) RunStream(_ context.Context, host string, command string, _ io.Writer, _ io.Writer) error {
	m.calls = append(m.calls, sshCall{method: "RunStream", host: host, command: command})
	return m.streamErr
}

func (m *mockExecutor) CopyTo(_ context.Context, host string, _ string, remotePath string) error {
	m.calls = append(m.calls, sshCall{method: "CopyTo", host: host, command: remotePath})
	return m.copyErr
}

func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockExecutor) Close() error { return nil }

// helper to create manager, failing test on error.
func newTestManager(t *testing.T, mock *mockExecutor, name string) Manager {
	t.Helper()
	mgr, err := NewManager(mock, "devbox-vps", name)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	return mgr
}

// --- Manager tests ---

func TestNewManager_InvalidName(t *testing.T) {
	mock := &mockExecutor{}
	_, err := NewManager(mock, "devbox-vps", "bad name; rm -rf /")
	if err == nil {
		t.Fatal("expected error for invalid name, got nil")
	}
	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestDeploy_Success(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Deploy(context.Background(), []byte("name: test-ws\n"))
	if err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}

	if len(mock.calls) != 3 {
		t.Fatalf("expected 3 SSH calls, got %d: %+v", len(mock.calls), mock.calls)
	}

	// 1. mkdir
	if !strings.Contains(mock.calls[0].command, "mkdir -p /workspaces/test-ws") {
		t.Errorf("call 0 = %q, want mkdir", mock.calls[0].command)
	}

	// 2. CopyTo
	if mock.calls[1].method != "CopyTo" {
		t.Errorf("call 1 method = %q, want CopyTo", mock.calls[1].method)
	}
	if !strings.Contains(mock.calls[1].command, "/workspaces/test-ws/docker-compose.yml") {
		t.Errorf("call 1 = %q, want compose path", mock.calls[1].command)
	}

	// 3. docker compose up
	if !strings.Contains(mock.calls[2].command, "docker compose") || !strings.Contains(mock.calls[2].command, "up -d") {
		t.Errorf("call 2 = %q, want docker compose up", mock.calls[2].command)
	}
}

func TestDeploy_MkdirError(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("permission denied")}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Deploy(context.Background(), []byte("name: test\n"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestDeploy_CopyError(t *testing.T) {
	mock := &mockExecutor{copyErr: errors.New("scp failed")}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Deploy(context.Background(), []byte("name: test\n"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
	if !strings.Contains(dockerErr.Message, "copy") {
		t.Errorf("error message = %q, want it to contain 'copy'", dockerErr.Message)
	}
}

func TestUp_Success(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	if err := mgr.Up(context.Background()); err != nil {
		t.Fatalf("Up() error: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	if !strings.Contains(mock.calls[0].command, "up -d") {
		t.Errorf("command = %q, want up -d", mock.calls[0].command)
	}
}

func TestUp_Error(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("docker not running")}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Up(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestDown_Success(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	if err := mgr.Down(context.Background()); err != nil {
		t.Fatalf("Down() error: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	if !strings.Contains(mock.calls[0].command, "down") {
		t.Errorf("command = %q, want down", mock.calls[0].command)
	}
}

func TestDown_Error(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("docker not running")}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Down(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestPS_Success(t *testing.T) {
	mock := &mockExecutor{runOut: "NAME    STATUS\nmysql   running\n"}
	mgr := newTestManager(t, mock, "test-ws")

	out, err := mgr.PS(context.Background())
	if err != nil {
		t.Fatalf("PS() error: %v", err)
	}
	if !strings.Contains(out, "mysql") {
		t.Errorf("output = %q, want it to contain 'mysql'", out)
	}
}

func TestPS_Error(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("docker not running")}
	mgr := newTestManager(t, mock, "test-ws")

	_, err := mgr.PS(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestLogs_Success(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	var stdout, stderr strings.Builder
	if err := mgr.Logs(context.Background(), "mysql", &stdout, &stderr); err != nil {
		t.Fatalf("Logs() error: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	if !strings.Contains(mock.calls[0].command, "logs --follow mysql") {
		t.Errorf("command = %q, want logs --follow mysql", mock.calls[0].command)
	}
}

func TestLogs_Error(t *testing.T) {
	mock := &mockExecutor{streamErr: errors.New("service not found")}
	mgr := newTestManager(t, mock, "test-ws")

	var stdout, stderr strings.Builder
	err := mgr.Logs(context.Background(), "nonexistent", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}

func TestLogs_InvalidServiceName(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	var stdout, stderr strings.Builder
	err := mgr.Logs(context.Background(), "mysql; rm -rf /", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid service name, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}

	// Should not have made any SSH calls.
	if len(mock.calls) != 0 {
		t.Errorf("expected 0 SSH calls for invalid name, got %d", len(mock.calls))
	}
}

func TestDestroy_Success(t *testing.T) {
	mock := &mockExecutor{}
	mgr := newTestManager(t, mock, "test-ws")

	if err := mgr.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.calls))
	}

	// 1. docker compose down -v
	if !strings.Contains(mock.calls[0].command, "down -v") {
		t.Errorf("call 0 = %q, want down -v", mock.calls[0].command)
	}

	// 2. rm -rf
	if !strings.Contains(mock.calls[1].command, "rm -rf /workspaces/test-ws") {
		t.Errorf("call 1 = %q, want rm -rf", mock.calls[1].command)
	}
}

func TestDestroy_DownError(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("compose not found")}
	mgr := newTestManager(t, mock, "test-ws")

	err := mgr.Destroy(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var dockerErr *devboxerr.DockerError
	if !errors.As(err, &dockerErr) {
		t.Errorf("expected DockerError, got %T", err)
	}
}
