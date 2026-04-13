package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	gomcp "github.com/mark3labs/mcp-go/mcp"

	"io"
	"net/http"
	"net/http/httptest"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/registry"
	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/snapshot"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/template"
	"github.com/junixlabs/devbox/internal/workspace"
)

// --- Mock types ---

type mockManager struct {
	createFunc      func(params workspace.CreateParams) (*workspace.Workspace, error)
	listFunc        func(opts workspace.ListOptions) ([]workspace.Workspace, error)
	getFunc         func(name string) (*workspace.Workspace, error)
	destroyFunc     func(name string) error
	execFunc        func(name string, command string) (*workspace.ExecResult, error)
	startFunc       func(name string) error
	stopFunc        func(name string) error
	sshFunc         func(name string) error
	dockerStatsFunc func(host string) (map[string]*workspace.ResourceUsage, error)
	serverResFunc   func(host string) (*workspace.ServerResourceInfo, error)
}

func (m *mockManager) Create(params workspace.CreateParams) (*workspace.Workspace, error) {
	if m.createFunc != nil {
		return m.createFunc(params)
	}
	return &workspace.Workspace{Name: params.Name, Status: workspace.StatusRunning}, nil
}
func (m *mockManager) List(opts workspace.ListOptions) ([]workspace.Workspace, error) {
	if m.listFunc != nil {
		return m.listFunc(opts)
	}
	return nil, nil
}
func (m *mockManager) Get(name string) (*workspace.Workspace, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return &workspace.Workspace{Name: name, Status: workspace.StatusRunning}, nil
}
func (m *mockManager) Destroy(name string) error {
	if m.destroyFunc != nil {
		return m.destroyFunc(name)
	}
	return nil
}
func (m *mockManager) Exec(name string, command string) (*workspace.ExecResult, error) {
	if m.execFunc != nil {
		return m.execFunc(name, command)
	}
	return &workspace.ExecResult{Stdout: "ok", ExitCode: 0}, nil
}
func (m *mockManager) Start(name string) error {
	if m.startFunc != nil {
		return m.startFunc(name)
	}
	return nil
}
func (m *mockManager) Stop(name string) error {
	if m.stopFunc != nil {
		return m.stopFunc(name)
	}
	return nil
}
func (m *mockManager) SSH(name string) error {
	if m.sshFunc != nil {
		return m.sshFunc(name)
	}
	return nil
}
func (m *mockManager) DockerStats(host string) (map[string]*workspace.ResourceUsage, error) {
	if m.dockerStatsFunc != nil {
		return m.dockerStatsFunc(host)
	}
	return nil, nil
}
func (m *mockManager) ServerResources(host string) (*workspace.ServerResourceInfo, error) {
	if m.serverResFunc != nil {
		return m.serverResFunc(host)
	}
	return nil, nil
}

type mockPool struct {
	listFunc     func() ([]server.Server, error)
	healthFunc   func(name string) (*server.HealthStatus, error)
	healthAllFn  func() (map[string]*server.HealthStatus, error)
	addFunc      func(name, host string, opts ...server.AddOption) (*server.Server, error)
	removeFunc   func(name string) error
}

func (m *mockPool) List() ([]server.Server, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}
func (m *mockPool) HealthCheck(name string) (*server.HealthStatus, error) {
	if m.healthFunc != nil {
		return m.healthFunc(name)
	}
	return &server.HealthStatus{SSH: true, Docker: true, Tailscale: true}, nil
}
func (m *mockPool) HealthCheckAll() (map[string]*server.HealthStatus, error) {
	if m.healthAllFn != nil {
		return m.healthAllFn()
	}
	return nil, nil
}
func (m *mockPool) Add(name, host string, opts ...server.AddOption) (*server.Server, error) {
	if m.addFunc != nil {
		return m.addFunc(name, host, opts...)
	}
	return &server.Server{Name: name, Host: host}, nil
}
func (m *mockPool) Remove(name string) error {
	if m.removeFunc != nil {
		return m.removeFunc(name)
	}
	return nil
}

type mockCollector struct {
	collectWSFunc  func(ctx context.Context, host, container string) (*metrics.WorkspaceMetrics, error)
	collectSrvFunc func(ctx context.Context, host string) (*metrics.ServerMetrics, error)
}

func (m *mockCollector) CollectWorkspace(ctx context.Context, host, container string) (*metrics.WorkspaceMetrics, error) {
	if m.collectWSFunc != nil {
		return m.collectWSFunc(ctx, host, container)
	}
	return &metrics.WorkspaceMetrics{Container: container, CPUPercent: 5.0, MemUsage: 1024}, nil
}
func (m *mockCollector) CollectServer(ctx context.Context, host string) (*metrics.ServerMetrics, error) {
	if m.collectSrvFunc != nil {
		return m.collectSrvFunc(ctx, host)
	}
	return &metrics.ServerMetrics{TotalCPUs: 4, TotalMem: 8192}, nil
}

type mockSnapshotMgr struct {
	createFunc  func(ctx context.Context, host, workspace, name string) (*snapshot.Snapshot, error)
	restoreFunc func(ctx context.Context, host, workspace, name string) error
	listFunc    func(ctx context.Context, host, workspace string) ([]snapshot.Snapshot, error)
}

func (m *mockSnapshotMgr) Create(ctx context.Context, host, ws, name string) (*snapshot.Snapshot, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, host, ws, name)
	}
	return &snapshot.Snapshot{Name: name, Workspace: ws, Size: 1024, CreatedAt: time.Now()}, nil
}
func (m *mockSnapshotMgr) Restore(ctx context.Context, host, ws, name string) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, host, ws, name)
	}
	return nil
}
func (m *mockSnapshotMgr) List(ctx context.Context, host, ws string) ([]snapshot.Snapshot, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, host, ws)
	}
	return nil, nil
}

type mockExecutor struct {
	runFunc func(ctx context.Context, host string, command string) (string, string, error)
}

func (m *mockExecutor) Run(ctx context.Context, host string, command string) (string, string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, host, command)
	}
	return "ok", "", nil
}
func (m *mockExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}
func (m *mockExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error  { return nil }
func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error { return nil }
func (m *mockExecutor) Close() error                                                    { return nil }

// Verify mockExecutor satisfies the interface at compile time.
var _ devboxssh.Executor = (*mockExecutor)(nil)

// --- Test helpers ---

func makeRequest(args map[string]any) gomcp.CallToolRequest {
	var req gomcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

func resultJSON(t *testing.T, result *gomcp.CallToolResult) map[string]any {
	t.Helper()
	text := extractText(t, result)
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, text)
	}
	return data
}

func resultJSONArray(t *testing.T, result *gomcp.CallToolResult) []any {
	t.Helper()
	text := extractText(t, result)
	var data []any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to parse JSON array: %v\nraw: %s", err, text)
	}
	return data
}

// --- Workspace handler tests ---

func TestHandleCreate_Success(t *testing.T) {
	mgr := &mockManager{
		createFunc: func(p workspace.CreateParams) (*workspace.Workspace, error) {
			return &workspace.Workspace{Name: p.Name, Status: workspace.StatusRunning, ServerHost: "dev1"}, nil
		},
	}
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
	}
	deps := Deps{Manager: mgr, Pool: pool}
	handler := handleCreate(deps)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":   "test-ws",
		"server": "dev1",
	}))
	if err != nil {
		t.Fatalf("handleCreate error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handleCreate returned error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["name"] != "test-ws" {
		t.Errorf("name = %v, want test-ws", data["name"])
	}
}

func TestHandleCreate_MissingName(t *testing.T) {
	deps := Deps{Manager: &mockManager{}}
	handler := handleCreate(deps)

	result, err := handler(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("handleCreate error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestHandleCreate_InvalidName(t *testing.T) {
	mgr := &mockManager{
		createFunc: func(p workspace.CreateParams) (*workspace.Workspace, error) {
			return nil, &workspace.WorkspaceError{Message: "invalid workspace name", Suggestion: "use lowercase alphanumeric"}
		},
	}
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
	}
	deps := Deps{Manager: mgr, Pool: pool}
	handler := handleCreate(deps)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":   "bad name!",
		"server": "dev1",
	}))
	if err != nil {
		t.Fatalf("handleCreate error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid name")
	}
	data := resultJSON(t, result)
	if data["error_code"] != ErrNotFound {
		t.Errorf("error_code = %v, want %v", data["error_code"], ErrNotFound)
	}
}

func TestHandleCreate_Duplicate(t *testing.T) {
	mgr := &mockManager{
		createFunc: func(p workspace.CreateParams) (*workspace.Workspace, error) {
			return nil, &workspace.WorkspaceError{Message: "workspace already exists", Suggestion: "choose a different name"}
		},
	}
	deps := Deps{Manager: mgr, Pool: &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
	}}
	handler := handleCreate(deps)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":   "existing",
		"server": "dev1",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for duplicate workspace")
	}
}

func TestHandleList_Success(t *testing.T) {
	mgr := &mockManager{
		listFunc: func(opts workspace.ListOptions) ([]workspace.Workspace, error) {
			return []workspace.Workspace{
				{Name: "ws-1", Status: workspace.StatusRunning},
				{Name: "ws-2", Status: workspace.StatusStopped},
			}, nil
		},
	}
	handler := handleList(mgr)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(arr))
	}
}

func TestHandleList_Empty(t *testing.T) {
	mgr := &mockManager{
		listFunc: func(opts workspace.ListOptions) ([]workspace.Workspace, error) {
			return []workspace.Workspace{}, nil
		},
	}
	handler := handleList(mgr)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(arr))
	}
}

func TestHandleList_FilterByUser(t *testing.T) {
	var capturedOpts workspace.ListOptions
	mgr := &mockManager{
		listFunc: func(opts workspace.ListOptions) ([]workspace.Workspace, error) {
			capturedOpts = opts
			return []workspace.Workspace{}, nil
		},
	}
	handler := handleList(mgr)

	handler(context.Background(), makeRequest(map[string]any{"user": "alice"}))
	if capturedOpts.User != "alice" {
		t.Errorf("expected User=alice, got %q", capturedOpts.User)
	}
	if capturedOpts.All {
		t.Error("expected All=false when user is specified")
	}
}

func TestHandleExec_Success(t *testing.T) {
	mgr := &mockManager{
		execFunc: func(name, cmd string) (*workspace.ExecResult, error) {
			return &workspace.ExecResult{Stdout: "hello world", Stderr: "", ExitCode: 0}, nil
		},
	}
	handler := handleExec(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":    "test-ws",
		"command": "echo hello world",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["stdout"] != "hello world" {
		t.Errorf("stdout = %v, want 'hello world'", data["stdout"])
	}
	if data["exit_code"] != float64(0) {
		t.Errorf("exit_code = %v, want 0", data["exit_code"])
	}
}

func TestHandleExec_NotFound(t *testing.T) {
	mgr := &mockManager{
		execFunc: func(name, cmd string) (*workspace.ExecResult, error) {
			return nil, &workspace.WorkspaceError{Message: "workspace not found", Suggestion: "check 'devbox list'"}
		},
	}
	handler := handleExec(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":    "nonexistent",
		"command": "ls",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestHandleExec_MissingCommand(t *testing.T) {
	handler := handleExec(&mockManager{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name": "test-ws",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing command")
	}
}

func TestHandleExec_MissingName(t *testing.T) {
	handler := handleExec(&mockManager{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"command": "ls",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestHandleDestroy_Success(t *testing.T) {
	destroyed := false
	mgr := &mockManager{
		destroyFunc: func(name string) error {
			destroyed = true
			return nil
		},
	}
	handler := handleDestroy(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name": "test-ws",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	if !destroyed {
		t.Error("destroy was not called")
	}
	data := resultJSON(t, result)
	if data["destroyed"] != "test-ws" {
		t.Errorf("destroyed = %v, want test-ws", data["destroyed"])
	}
}

func TestHandleDestroy_NotFound(t *testing.T) {
	mgr := &mockManager{
		destroyFunc: func(name string) error {
			return &workspace.WorkspaceError{Message: "workspace not found"}
		},
	}
	handler := handleDestroy(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name": "nonexistent",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestHandleDestroy_MissingName(t *testing.T) {
	handler := handleDestroy(&mockManager{})

	result, err := handler(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

// --- mapWorkspaceError tests ---

func TestMapWorkspaceError_WorkspaceError(t *testing.T) {
	err := &workspace.WorkspaceError{
		Message:    "workspace not found",
		Suggestion: "check 'devbox list'",
	}
	result := mapWorkspaceError(err)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	data := resultJSON(t, result)
	if data["error_code"] != ErrNotFound {
		t.Errorf("error_code = %v, want %v", data["error_code"], ErrNotFound)
	}
}

func TestMapWorkspaceError_GenericError(t *testing.T) {
	result := mapWorkspaceError(fmt.Errorf("unexpected failure"))
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	data := resultJSON(t, result)
	if data["error_code"] != ErrInternal {
		t.Errorf("error_code = %v, want %v", data["error_code"], ErrInternal)
	}
}

// --- Server & metrics handler tests ---

func TestHandleServerList_Success(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{
				{Name: "dev1", Host: "10.0.0.1"},
				{Name: "dev2", Host: "10.0.0.2"},
			}, nil
		},
		healthAllFn: func() (map[string]*server.HealthStatus, error) {
			return map[string]*server.HealthStatus{
				"dev1": {SSH: true, Docker: true, Tailscale: true},
				"dev2": {SSH: false, Docker: false, Tailscale: false},
			}, nil
		},
	}
	exec := &mockExecutor{
		runFunc: func(ctx context.Context, host string, command string) (string, string, error) {
			switch {
			case command == "nproc":
				return "4", "", nil
			case command == "free -b":
				return "              total        used        free      shared  buff/cache   available\nMem:     8000000000  4000000000  2000000000   100000000  2000000000  3900000000\nSwap:             0           0           0", "", nil
			case command == "cat /proc/loadavg":
				return "1.5 1.2 0.8 1/200 12345", "", nil
			default:
				return "ok", "", nil
			}
		},
	}
	handler := handleServerList(pool, exec)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 2 {
		t.Errorf("expected 2 servers, got %d", len(arr))
	}
}

func TestHandleServerList_Empty(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{}, nil
		},
	}
	handler := handleServerList(pool, nil)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 0 {
		t.Errorf("expected 0 servers, got %d", len(arr))
	}
}

func TestHandleServerList_OfflineServer(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
		healthAllFn: func() (map[string]*server.HealthStatus, error) {
			return map[string]*server.HealthStatus{
				"dev1": {SSH: false, Docker: false, Tailscale: false},
			}, nil
		},
	}
	handler := handleServerList(pool, nil)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 1 {
		t.Fatalf("expected 1 server, got %d", len(arr))
	}
	srv := arr[0].(map[string]any)
	if srv["status"] != "offline" {
		t.Errorf("status = %v, want offline", srv["status"])
	}
}

func TestHandleServerStatus_Success(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
		healthFunc: func(name string) (*server.HealthStatus, error) {
			return &server.HealthStatus{SSH: true, Docker: true, Tailscale: true}, nil
		},
	}
	collector := &mockCollector{
		collectSrvFunc: func(ctx context.Context, host string) (*metrics.ServerMetrics, error) {
			return &metrics.ServerMetrics{TotalCPUs: 8, TotalMem: 16384}, nil
		},
	}
	exec := &mockExecutor{
		runFunc: func(ctx context.Context, host string, command string) (string, string, error) {
			switch {
			case command == "nproc":
				return "8", "", nil
			case command == "free -b":
				return "              total        used        free      shared  buff/cache   available\nMem:    16000000000  8000000000  4000000000   200000000  4000000000  7800000000\nSwap:             0           0           0", "", nil
			case command == "cat /proc/loadavg":
				return "2.0 1.5 1.0 2/400 23456", "", nil
			default:
				return "ok", "", nil
			}
		},
	}
	handler := handleServerStatus(pool, collector, exec)

	result, err := handler(context.Background(), makeRequest(map[string]any{"name": "dev1"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["name"] != "dev1" {
		t.Errorf("name = %v, want dev1", data["name"])
	}
}

func TestHandleServerStatus_NotFound(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{}, nil
		},
	}
	handler := handleServerStatus(pool, &mockCollector{}, nil)

	result, err := handler(context.Background(), makeRequest(map[string]any{"name": "nonexistent"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent server")
	}
}

func TestHandleServerStatus_MissingName(t *testing.T) {
	handler := handleServerStatus(&mockPool{}, &mockCollector{}, nil)

	result, err := handler(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestHandleMetrics_ByWorkspace(t *testing.T) {
	mgr := &mockManager{
		getFunc: func(name string) (*workspace.Workspace, error) {
			return &workspace.Workspace{Name: name, ServerHost: "10.0.0.1"}, nil
		},
	}
	collector := &mockCollector{
		collectWSFunc: func(ctx context.Context, host, container string) (*metrics.WorkspaceMetrics, error) {
			return &metrics.WorkspaceMetrics{Container: container, CPUPercent: 25.5, MemUsage: 512000}, nil
		},
	}
	handler := handleMetrics(mgr, &mockPool{}, collector)

	result, err := handler(context.Background(), makeRequest(map[string]any{"workspace": "test-ws"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["cpu_percent"] != 25.5 {
		t.Errorf("cpu_percent = %v, want 25.5", data["cpu_percent"])
	}
}

func TestHandleMetrics_ByServer(t *testing.T) {
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
	}
	collector := &mockCollector{
		collectSrvFunc: func(ctx context.Context, host string) (*metrics.ServerMetrics, error) {
			return &metrics.ServerMetrics{TotalCPUs: 4, TotalMem: 8192}, nil
		},
	}
	handler := handleMetrics(&mockManager{}, pool, collector)

	result, err := handler(context.Background(), makeRequest(map[string]any{"server": "dev1"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["total_cpus"] != float64(4) {
		t.Errorf("total_cpus = %v, want 4", data["total_cpus"])
	}
}

func TestHandleMetrics_NoParam(t *testing.T) {
	handler := handleMetrics(&mockManager{}, &mockPool{}, &mockCollector{})

	result, err := handler(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when no workspace or server provided")
	}
	data := resultJSON(t, result)
	if data["error_code"] != ErrInvalidInput {
		t.Errorf("error_code = %v, want %v", data["error_code"], ErrInvalidInput)
	}
}

func TestHandleMetrics_WorkspaceNotFound(t *testing.T) {
	mgr := &mockManager{
		getFunc: func(name string) (*workspace.Workspace, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	handler := handleMetrics(mgr, &mockPool{}, &mockCollector{})

	result, err := handler(context.Background(), makeRequest(map[string]any{"workspace": "nope"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent workspace")
	}
}

// --- Snapshot handler tests ---

func TestHandleSnapshotCreate_Success(t *testing.T) {
	mgr := &mockSnapshotMgr{
		createFunc: func(ctx context.Context, host, ws, name string) (*snapshot.Snapshot, error) {
			return &snapshot.Snapshot{Name: "snap1", Workspace: ws, Size: 2048, CreatedAt: time.Now()}, nil
		},
	}
	handler := handleSnapshotCreate(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"server":    "10.0.0.1",
		"workspace": "test-ws",
		"name":      "snap1",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["name"] != "snap1" {
		t.Errorf("name = %v, want snap1", data["name"])
	}
}

func TestHandleSnapshotCreate_MissingServer(t *testing.T) {
	handler := handleSnapshotCreate(&mockSnapshotMgr{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"workspace": "test-ws",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing server")
	}
}

func TestHandleSnapshotRestore_Success(t *testing.T) {
	handler := handleSnapshotRestore(&mockSnapshotMgr{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"server":    "10.0.0.1",
		"workspace": "test-ws",
		"name":      "snap1",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	data := resultJSON(t, result)
	if data["restored"] != true {
		t.Errorf("restored = %v, want true", data["restored"])
	}
}

func TestHandleSnapshotRestore_NotFound(t *testing.T) {
	mgr := &mockSnapshotMgr{
		restoreFunc: func(ctx context.Context, host, ws, name string) error {
			return fmt.Errorf("snapshot not found")
		},
	}
	handler := handleSnapshotRestore(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"server":    "10.0.0.1",
		"workspace": "test-ws",
		"name":      "nonexistent",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent snapshot")
	}
}

func TestHandleSnapshotList_Success(t *testing.T) {
	mgr := &mockSnapshotMgr{
		listFunc: func(ctx context.Context, host, ws string) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{
				{Name: "snap1", Workspace: ws, Size: 1024, CreatedAt: time.Now()},
				{Name: "snap2", Workspace: ws, Size: 2048, CreatedAt: time.Now()},
			}, nil
		},
	}
	handler := handleSnapshotList(mgr)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"server":    "10.0.0.1",
		"workspace": "test-ws",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(arr))
	}
}

func TestHandleSnapshotRestore_MissingWorkspace(t *testing.T) {
	handler := handleSnapshotRestore(&mockSnapshotMgr{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"server": "10.0.0.1",
		"name":   "snap1",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing workspace")
	}
}

func TestHandleSnapshotList_MissingServer(t *testing.T) {
	handler := handleSnapshotList(&mockSnapshotMgr{})

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"workspace": "test-ws",
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing server")
	}
}

// --- Template handler tests ---

func TestHandleTemplateList_Success(t *testing.T) {
	dir := t.TempDir()
	reg := template.NewLocalRegistry(dir, nil)
	reg.Save(&template.Template{
		Name:        "golang",
		Description: "Go development workspace",
		Services:    []string{"golang:1.21"},
	})

	handler := handleTemplateList(reg)

	result, err := handler(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	arr := resultJSONArray(t, result)
	if len(arr) == 0 {
		t.Error("expected at least 1 template")
	}
}

func TestHandleTemplateSearch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte("templates:\n  - name: golang\n    version: \"1.0.0\"\n    description: Go dev workspace\n    url: https://example.com/golang.tar.gz\n    updated_at: 2026-01-01T00:00:00Z\n"))
	}))
	defer ts.Close()

	remote := registry.NewRemoteRegistry(ts.URL)
	handler := handleTemplateSearch(remote)

	result, err := handler(context.Background(), makeRequest(map[string]any{"query": "golang"}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	arr := resultJSONArray(t, result)
	if len(arr) != 1 {
		t.Errorf("expected 1 search result, got %d", len(arr))
	}
}

func TestHandleTemplateSearch_EmptyQuery(t *testing.T) {
	handler := handleTemplateSearch(nil)

	result, err := handler(context.Background(), makeRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty query")
	}
	data := resultJSON(t, result)
	if data["error_code"] != ErrInvalidInput {
		t.Errorf("error_code = %v, want %v", data["error_code"], ErrInvalidInput)
	}
}

func TestHandleCreate_WithServicesAndEnv(t *testing.T) {
	var capturedParams workspace.CreateParams
	mgr := &mockManager{
		createFunc: func(p workspace.CreateParams) (*workspace.Workspace, error) {
			capturedParams = p
			return &workspace.Workspace{Name: p.Name, Status: workspace.StatusRunning}, nil
		},
	}
	deps := Deps{Manager: mgr, Pool: &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
	}}
	handler := handleCreate(deps)

	result, err := handler(context.Background(), makeRequest(map[string]any{
		"name":     "test-ws",
		"server":   "dev1",
		"repo":     "https://github.com/example/repo",
		"branch":   "main",
		"services": []any{"redis:7", "postgres:15"},
		"env":      map[string]any{"DB_HOST": "localhost", "DEBUG": "true"},
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	if len(capturedParams.Services) != 2 {
		t.Errorf("services count = %d, want 2", len(capturedParams.Services))
	}
	if capturedParams.Env["DB_HOST"] != "localhost" {
		t.Errorf("env DB_HOST = %q, want localhost", capturedParams.Env["DB_HOST"])
	}
	if capturedParams.Repo != "https://github.com/example/repo" {
		t.Errorf("repo = %q, want https://github.com/example/repo", capturedParams.Repo)
	}
}
