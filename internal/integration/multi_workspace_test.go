//go:build integration

package integration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/testutil"
	"github.com/junixlabs/devbox/internal/workspace"
)

const testService = "nginx:alpine"

// testParams returns CreateParams for a test workspace with unique port.
func testParams(server, name string, port int, res *config.Resources) workspace.CreateParams {
	p := workspace.CreateParams{
		Name:     name,
		Server:   server,
		Services: []string{testService},
		Ports:    map[string]int{"nginx": port},
	}
	if res != nil {
		p.Resources = *res
	}
	return p
}

// containerName returns the expected Docker container name for a test workspace.
func containerName(wsName string) string {
	return wsName + "-nginx-1"
}

// TestConcurrentWorkspaceCreation creates 3 workspaces in parallel goroutines
// and verifies all reach running status with their containers active.
func TestConcurrentWorkspaceCreation(t *testing.T) {
	server := testutil.TestServer(t)
	mgr := testutil.NewManager()

	names := []string{"inttest-a", "inttest-b", "inttest-c"}
	ports := []int{18081, 18082, 18083}

	// Best-effort cleanup of any leftover workspaces.
	for _, name := range names {
		_ = mgr.Destroy(name)
	}

	var (
		mu      sync.Mutex
		results = make([]*workspace.Workspace, len(names))
		errs    = make([]error, len(names))
		wg      sync.WaitGroup
	)

	for i, name := range names {
		wg.Add(1)
		go func(idx int, n string) {
			defer wg.Done()
			params := testParams(server, n, ports[idx], &config.Resources{CPUs: 0.25, Memory: "128m"})
			ws, err := mgr.Create(params)
			mu.Lock()
			results[idx] = ws
			errs[idx] = err
			mu.Unlock()
		}(i, name)
	}
	wg.Wait()

	// Register cleanup for all successfully created workspaces.
	t.Cleanup(func() {
		for _, name := range names {
			if err := mgr.Destroy(name); err != nil {
				t.Logf("cleanup Destroy(%s): %v", name, err)
			}
		}
	})

	// Verify all created successfully.
	for i, name := range names {
		if errs[i] != nil {
			t.Fatalf("Create(%s) failed: %v", name, errs[i])
		}
		if results[i].Status != workspace.StatusRunning {
			t.Errorf("workspace %s status = %s, want running", name, results[i].Status)
		}
	}

	// Verify all containers are running on the server.
	for _, name := range names {
		testutil.WaitForContainer(t, server, containerName(name), 60*time.Second)
	}

	// Verify list returns all 3.
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	found := 0
	for _, ws := range list {
		for _, name := range names {
			if ws.Name == name {
				found++
			}
		}
	}
	if found != len(names) {
		t.Errorf("List() found %d of %d test workspaces", found, len(names))
	}
}

// TestPortAutoAllocation creates workspaces with different explicit ports
// and verifies each port is bound on the host without conflicts.
func TestPortAutoAllocation(t *testing.T) {
	server := testutil.TestServer(t)
	mgr := testutil.NewManager()

	type wsSpec struct {
		name string
		port int
	}
	specs := []wsSpec{
		{"inttest-port-a", 19091},
		{"inttest-port-b", 19092},
	}

	for _, s := range specs {
		params := testParams(server, s.name, s.port, nil)
		testutil.CreateWorkspace(t, mgr, params)
	}

	// Wait for containers.
	for _, s := range specs {
		testutil.WaitForContainer(t, server, containerName(s.name), 60*time.Second)
	}

	// Verify each port is listening.
	for _, s := range specs {
		testutil.AssertPortListening(t, server, s.port)
	}

	// Verify no port collision: each workspace's assigned port is unique.
	portSet := make(map[int]string)
	for _, s := range specs {
		ws, err := mgr.Get(s.name)
		if err != nil {
			t.Fatalf("Get(%s): %v", s.name, err)
		}
		for _, port := range ws.Ports {
			if existing, ok := portSet[port]; ok {
				t.Errorf("port %d used by both %s and %s", port, existing, s.name)
			}
			portSet[port] = s.name
		}
	}
}

// TestResourceLimitsEnforcement creates a workspace with resource limits
// and verifies Docker applies the constraints.
func TestResourceLimitsEnforcement(t *testing.T) {
	server := testutil.TestServer(t)
	mgr := testutil.NewManager()

	res := &config.Resources{CPUs: 0.5, Memory: "256m"}
	params := testParams(server, "inttest-reslimit", 19001, res)
	testutil.CreateWorkspace(t, mgr, params)

	container := containerName("inttest-reslimit")
	testutil.WaitForContainer(t, server, container, 60*time.Second)

	// Verify CPU limit via docker inspect (NanoCpus = CPUs * 1e9).
	cpuOut := testutil.DockerInspect(t, server, container, "{{.HostConfig.NanoCpus}}")
	nanoCpus, err := strconv.ParseInt(strings.TrimSpace(cpuOut), 10, 64)
	if err != nil {
		t.Fatalf("parse NanoCpus %q: %v", cpuOut, err)
	}
	expectedNano := int64(0.5 * 1e9)
	if nanoCpus != expectedNano {
		t.Errorf("NanoCpus = %d, want %d (0.5 CPUs)", nanoCpus, expectedNano)
	}

	// Verify memory limit (256m = 268435456 bytes).
	memOut := testutil.DockerInspect(t, server, container, "{{.HostConfig.Memory}}")
	memBytes, err := strconv.ParseInt(strings.TrimSpace(memOut), 10, 64)
	if err != nil {
		t.Fatalf("parse Memory %q: %v", memOut, err)
	}
	expectedMem := int64(256 * 1024 * 1024)
	if memBytes != expectedMem {
		t.Errorf("Memory = %d, want %d (256m)", memBytes, expectedMem)
	}
}

// TestUserIsolation verifies that workspaces are isolated from each other:
// filesystem separation and independent Docker Compose projects.
func TestUserIsolation(t *testing.T) {
	server := testutil.TestServer(t)
	mgr := testutil.NewManager()

	wsA := testutil.CreateWorkspace(t, mgr, testParams(server, "inttest-iso-a", 19011, nil))
	wsB := testutil.CreateWorkspace(t, mgr, testParams(server, "inttest-iso-b", 19012, nil))

	cA := containerName(wsA.Name)
	cB := containerName(wsB.Name)
	testutil.WaitForContainer(t, server, cA, 60*time.Second)
	testutil.WaitForContainer(t, server, cB, 60*time.Second)

	// Verify workspace directories are separate.
	testutil.AssertDirExists(t, server, "/workspaces/"+wsA.Name)
	testutil.AssertDirExists(t, server, "/workspaces/"+wsB.Name)

	// Verify containers run in separate Docker Compose projects.
	projectA := testutil.DockerInspect(t, server, cA, "{{index .Config.Labels \"com.docker.compose.project\"}}")
	projectB := testutil.DockerInspect(t, server, cB, "{{index .Config.Labels \"com.docker.compose.project\"}}")
	if strings.TrimSpace(projectA) == strings.TrimSpace(projectB) {
		t.Errorf("workspaces share compose project: %s", projectA)
	}

	// Verify container A cannot see container B's workspace directory.
	cmd := fmt.Sprintf("docker exec %s test -d /workspaces/%s 2>&1; echo $?", cA, wsB.Name)
	out, err := testutil.SSHRunE(server, cmd)
	if err != nil {
		t.Fatalf("isolation check: %v", err)
	}
	// exit code 1 means directory not found = isolated
	if strings.TrimSpace(out) == "0" {
		t.Errorf("container %s can see workspace dir of %s — isolation violation", wsA.Name, wsB.Name)
	}
}

// TestMultiServerDistribution is skipped when only one test server is available.
// Set DEVBOX_TEST_SERVER_2 to enable this test.
func TestMultiServerDistribution(t *testing.T) {
	server1 := testutil.TestServer(t)
	server2 := os.Getenv("DEVBOX_TEST_SERVER_2")
	if server2 == "" {
		t.Skip("DEVBOX_TEST_SERVER_2 not set — skipping multi-server test")
	}

	mgr := testutil.NewManager()
	wsA := testutil.CreateWorkspace(t, mgr, testParams(server1, "inttest-multi-a", 19021, nil))
	wsB := testutil.CreateWorkspace(t, mgr, testParams(server2, "inttest-multi-b", 19021, nil))

	// Same port on different servers should not conflict.
	if wsA.ServerHost == wsB.ServerHost {
		t.Fatal("expected workspaces on different servers")
	}

	testutil.WaitForContainer(t, server1, containerName(wsA.Name), 60*time.Second)
	testutil.WaitForContainer(t, server2, containerName(wsB.Name), 60*time.Second)

	testutil.AssertPortListening(t, server1, 19021)
	testutil.AssertPortListening(t, server2, 19021)
}

// TestCleanupAfterDestroy creates a workspace, destroys it, and verifies
// that containers, ports, and directories are fully cleaned up.
func TestCleanupAfterDestroy(t *testing.T) {
	server := testutil.TestServer(t)
	mgr := testutil.NewManager()

	port := 19031
	params := testParams(server, "inttest-cleanup", port, nil)

	// Best-effort pre-cleanup.
	_ = mgr.Destroy(params.Name)

	ws, err := mgr.Create(params)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	testutil.WaitForContainer(t, server, containerName(ws.Name), 60*time.Second)
	testutil.AssertDirExists(t, server, "/workspaces/"+ws.Name)

	// Destroy the workspace.
	if err := mgr.Destroy(ws.Name); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	// Wait for Docker cleanup.
	time.Sleep(3 * time.Second)

	// Verify container is gone.
	out, _ := testutil.SSHRunE(server, fmt.Sprintf("docker ps -q --filter name=%s", containerName(ws.Name)))
	if strings.TrimSpace(out) != "" {
		t.Errorf("container %s still exists after destroy", containerName(ws.Name))
	}

	// Verify port is free.
	testutil.AssertPortFree(t, server, port)

	// Verify workspace directory is gone.
	testutil.AssertDirNotExists(t, server, "/workspaces/"+ws.Name)

	// Verify workspace is removed from state.
	_, err = mgr.Get(ws.Name)
	if err == nil {
		t.Error("Get() returned no error after destroy — workspace still in state")
	}
}
