//go:build integration

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpclient "github.com/mark3labs/mcp-go/client"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/snapshot"
	"github.com/junixlabs/devbox/internal/workspace"
)

// --- Integration test helpers ---

// testWorkspaceStore is a thread-safe in-memory workspace manager for integration tests.
type testWorkspaceStore struct {
	mu         sync.Mutex
	workspaces map[string]*workspace.Workspace
}

func newTestWorkspaceStore() *testWorkspaceStore {
	return &testWorkspaceStore{workspaces: make(map[string]*workspace.Workspace)}
}

func (s *testWorkspaceStore) Create(params workspace.CreateParams) (*workspace.Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.workspaces[params.Name]; exists {
		return nil, &workspace.WorkspaceError{Message: "workspace already exists", Suggestion: "choose a different name"}
	}
	if params.Name == "" {
		return nil, &workspace.WorkspaceError{Message: "name is required"}
	}
	ws := &workspace.Workspace{
		Name:       params.Name,
		User:       params.User,
		Status:     workspace.StatusRunning,
		ServerHost: params.Server,
		CreatedAt:  time.Now(),
	}
	s.workspaces[params.Name] = ws
	return ws, nil
}

func (s *testWorkspaceStore) List(opts workspace.ListOptions) ([]workspace.Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []workspace.Workspace
	for _, ws := range s.workspaces {
		if opts.User != "" && ws.User != opts.User {
			continue
		}
		result = append(result, *ws)
	}
	return result, nil
}

func (s *testWorkspaceStore) Get(name string) (*workspace.Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, ok := s.workspaces[name]
	if !ok {
		return nil, &workspace.WorkspaceError{Message: "workspace not found", Suggestion: "check 'devbox list'"}
	}
	return ws, nil
}

func (s *testWorkspaceStore) Destroy(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workspaces[name]; !ok {
		return &workspace.WorkspaceError{Message: "workspace not found", Suggestion: "check 'devbox list'"}
	}
	delete(s.workspaces, name)
	return nil
}

func (s *testWorkspaceStore) Exec(name, command string) (*workspace.ExecResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workspaces[name]; !ok {
		return nil, &workspace.WorkspaceError{Message: "workspace not found", Suggestion: "check 'devbox list'"}
	}
	return &workspace.ExecResult{Stdout: "output of: " + command, ExitCode: 0}, nil
}

func (s *testWorkspaceStore) Start(name string) error  { return nil }
func (s *testWorkspaceStore) Stop(name string) error   { return nil }
func (s *testWorkspaceStore) SSH(name string) error     { return nil }
func (s *testWorkspaceStore) DockerStats(host string) (map[string]*workspace.ResourceUsage, error) {
	return nil, nil
}
func (s *testWorkspaceStore) ServerResources(host string) (*workspace.ServerResourceInfo, error) {
	return nil, nil
}

func setupIntegrationClient(t *testing.T) (*mcpclient.Client, context.Context) {
	t.Helper()

	store := newTestWorkspaceStore()
	pool := &mockPool{
		listFunc: func() ([]server.Server, error) {
			return []server.Server{{Name: "dev1", Host: "10.0.0.1"}}, nil
		},
		healthFunc: func(name string) (*server.HealthStatus, error) {
			return &server.HealthStatus{SSH: true, Docker: true, Tailscale: true}, nil
		},
		healthAllFn: func() (map[string]*server.HealthStatus, error) {
			return map[string]*server.HealthStatus{
				"dev1": {SSH: true, Docker: true, Tailscale: true},
			}, nil
		},
	}
	collector := &mockCollector{
		collectWSFunc: func(ctx context.Context, host, container string) (*metrics.WorkspaceMetrics, error) {
			return &metrics.WorkspaceMetrics{Container: container, CPUPercent: 10.0, MemUsage: 512000}, nil
		},
		collectSrvFunc: func(ctx context.Context, host string) (*metrics.ServerMetrics, error) {
			return &metrics.ServerMetrics{TotalCPUs: 4, TotalMem: 8192}, nil
		},
	}
	snapMgr := &mockSnapshotMgr{
		createFunc: func(ctx context.Context, host, ws, name string) (*snapshot.Snapshot, error) {
			return &snapshot.Snapshot{Name: name, Workspace: ws, Size: 1024, CreatedAt: time.Now()}, nil
		},
		listFunc: func(ctx context.Context, host, ws string) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{}, nil
		},
	}

	deps := Deps{
		Manager:     store,
		Pool:        pool,
		Collector:   collector,
		SSHExec:     &mockExecutor{},
		SnapshotMgr: snapMgr,
	}

	srv := NewServer(deps, "test")
	client, err := mcpclient.NewInProcessClient(srv)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}

	ctx := context.Background()

	// Initialize the client.
	_, err = client.Initialize(ctx, gomcp.InitializeRequest{
		Params: gomcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo:      gomcp.Implementation{Name: "test-client", Version: "1.0.0"},
		},
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client, ctx
}

func callTool(t *testing.T, client *mcpclient.Client, ctx context.Context, name string, args map[string]any) *gomcp.CallToolResult {
	t.Helper()
	var req gomcp.CallToolRequest
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := client.CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", name, err)
	}
	return result
}

func extractResultText(t *testing.T, result *gomcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(gomcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// --- Integration tests ---

func TestIntegration_FullWorkspaceLifecycle(t *testing.T) {
	client, ctx := setupIntegrationClient(t)

	// 1. Create workspace.
	result := callTool(t, client, ctx, "devbox_workspace_create", map[string]any{
		"name":   "int-test-ws",
		"server": "dev1",
	})
	if result.IsError {
		t.Fatalf("create error: %s", extractResultText(t, result))
	}
	var ws map[string]any
	json.Unmarshal([]byte(extractResultText(t, result)), &ws)
	if ws["name"] != "int-test-ws" {
		t.Errorf("created workspace name = %v, want int-test-ws", ws["name"])
	}

	// 2. List — should have 1 workspace.
	result = callTool(t, client, ctx, "devbox_workspace_list", nil)
	if result.IsError {
		t.Fatalf("list error: %s", extractResultText(t, result))
	}
	var wsList []any
	json.Unmarshal([]byte(extractResultText(t, result)), &wsList)
	if len(wsList) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(wsList))
	}

	// 3. Exec command.
	result = callTool(t, client, ctx, "devbox_workspace_exec", map[string]any{
		"name":    "int-test-ws",
		"command": "echo hello",
	})
	if result.IsError {
		t.Fatalf("exec error: %s", extractResultText(t, result))
	}
	var execRes map[string]any
	json.Unmarshal([]byte(extractResultText(t, result)), &execRes)
	if execRes["exit_code"] != float64(0) {
		t.Errorf("exit_code = %v, want 0", execRes["exit_code"])
	}

	// 4. Destroy workspace.
	result = callTool(t, client, ctx, "devbox_workspace_destroy", map[string]any{
		"name": "int-test-ws",
	})
	if result.IsError {
		t.Fatalf("destroy error: %s", extractResultText(t, result))
	}

	// 5. List — should be empty.
	result = callTool(t, client, ctx, "devbox_workspace_list", nil)
	if result.IsError {
		t.Fatalf("list error: %s", extractResultText(t, result))
	}
	json.Unmarshal([]byte(extractResultText(t, result)), &wsList)
	if len(wsList) != 0 {
		t.Errorf("expected 0 workspaces after destroy, got %d", len(wsList))
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	client, ctx := setupIntegrationClient(t)

	// Exec on nonexistent workspace.
	result := callTool(t, client, ctx, "devbox_workspace_exec", map[string]any{
		"name":    "nonexistent",
		"command": "ls",
	})
	if !result.IsError {
		t.Error("expected error for exec on nonexistent workspace")
	}
	var errBody map[string]any
	json.Unmarshal([]byte(extractResultText(t, result)), &errBody)
	if errBody["error_code"] != ErrNotFound {
		t.Errorf("error_code = %v, want %v", errBody["error_code"], ErrNotFound)
	}

	// Create with missing name.
	result = callTool(t, client, ctx, "devbox_workspace_create", map[string]any{})
	if !result.IsError {
		t.Error("expected error for create with missing name")
	}

	// Destroy nonexistent.
	result = callTool(t, client, ctx, "devbox_workspace_destroy", map[string]any{
		"name": "nonexistent",
	})
	if !result.IsError {
		t.Error("expected error for destroy of nonexistent workspace")
	}

	// Create duplicate.
	callTool(t, client, ctx, "devbox_workspace_create", map[string]any{
		"name":   "dup-ws",
		"server": "dev1",
	})
	result = callTool(t, client, ctx, "devbox_workspace_create", map[string]any{
		"name":   "dup-ws",
		"server": "dev1",
	})
	if !result.IsError {
		t.Error("expected error for duplicate workspace creation")
	}
}

func TestIntegration_ServerTools(t *testing.T) {
	client, ctx := setupIntegrationClient(t)

	// Server list.
	result := callTool(t, client, ctx, "devbox_server_list", nil)
	if result.IsError {
		t.Fatalf("server list error: %s", extractResultText(t, result))
	}
	var servers []any
	json.Unmarshal([]byte(extractResultText(t, result)), &servers)
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}

	// Metrics — requires workspace or server param.
	result = callTool(t, client, ctx, "devbox_metrics", map[string]any{})
	if !result.IsError {
		t.Error("expected error for metrics without params")
	}
}

func TestIntegration_ToolDiscovery(t *testing.T) {
	client, ctx := setupIntegrationClient(t)

	tools, err := client.ListTools(ctx, gomcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	// Verify expected tools are registered.
	expected := map[string]bool{
		"devbox_workspace_create":  false,
		"devbox_workspace_list":    false,
		"devbox_workspace_exec":    false,
		"devbox_workspace_destroy": false,
		"devbox_server_list":       false,
		"devbox_server_status":     false,
		"devbox_metrics":           false,
		"devbox_snapshot_create":   false,
		"devbox_snapshot_restore":  false,
		"devbox_snapshot_list":     false,
	}

	for _, tool := range tools.Tools {
		if _, ok := expected[tool.Name]; ok {
			expected[tool.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected tool %q not found in ListTools", name)
		}
	}
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	client, ctx := setupIntegrationClient(t)

	// Create 5 workspaces concurrently.
	errs := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			name := fmt.Sprintf("concurrent-ws-%d", idx)
			result := callTool(t, client, ctx, "devbox_workspace_create", map[string]any{
				"name":   name,
				"server": "dev1",
			})
			if result.IsError {
				errs <- fmt.Errorf("create %s: %s", name, extractResultText(t, result))
				return
			}
			errs <- nil
		}(i)
	}

	for i := 0; i < 5; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent create: %v", err)
		}
	}

	// List should show all 5.
	result := callTool(t, client, ctx, "devbox_workspace_list", nil)
	if result.IsError {
		t.Fatalf("list error: %s", extractResultText(t, result))
	}
	var wsList []any
	json.Unmarshal([]byte(extractResultText(t, result)), &wsList)
	if len(wsList) != 5 {
		t.Errorf("expected 5 workspaces, got %d", len(wsList))
	}
}
