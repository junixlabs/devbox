package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/junixlabs/devbox/internal/ssh"
)

const snapshotRoot = "/workspaces/.snapshots"

type sshManager struct {
	exec ssh.Executor
}

// NewManager creates a snapshot Manager that operates on a remote server via SSH.
func NewManager(exec ssh.Executor) Manager {
	return &sshManager{exec: exec}
}

func (m *sshManager) Create(host, workspace, name string) (*Snapshot, error) {
	if name == "" {
		name = fmt.Sprintf("%s-%s", workspace, time.Now().UTC().Format("20060102-150405"))
	}

	dir := fmt.Sprintf("%s/%s", snapshotRoot, workspace)
	archive := fmt.Sprintf("%s/%s.tar.gz", dir, name)

	// Create snapshot directory.
	if _, _, err := m.exec.Run(context.Background(), host, fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		return nil, fmt.Errorf("snapshot create: mkdir: %w", err)
	}

	// Discover Docker volume mount paths for the workspace.
	// Container names follow the pattern: <workspace>-<service>-1
	mountCmd := fmt.Sprintf(
		`docker ps -a --filter "name=^%s-" --format '{{.Names}}' | head -1 | xargs -I{} docker inspect {} --format '{{range .Mounts}}{{.Source}} {{end}}'`,
		workspace,
	)
	mountOut, _, err := m.exec.Run(context.Background(), host, mountCmd)
	if err != nil {
		return nil, fmt.Errorf("snapshot create: discover mounts: %w", err)
	}

	// Build tar paths: volume mounts + workspace config files.
	tarPaths := ""
	if mountOut != "" {
		tarPaths = mountOut
	}
	wsDir := fmt.Sprintf("/workspaces/%s", workspace)
	// Add workspace config files if they exist.
	configCmd := fmt.Sprintf(
		`for f in %s/devbox.yaml %s/.env; do [ -f "$f" ] && printf '%%s ' "$f"; done`,
		wsDir, wsDir,
	)
	configOut, _, _ := m.exec.Run(context.Background(), host, configCmd)
	if configOut != "" {
		tarPaths += " " + configOut
	}

	if tarPaths == "" {
		return nil, fmt.Errorf("snapshot create: no files to snapshot for workspace %q", workspace)
	}

	// Create compressed archive.
	tarCmd := fmt.Sprintf("tar czf %s -C / %s", archive, tarPaths)
	if _, _, err := m.exec.Run(context.Background(), host, tarCmd); err != nil {
		return nil, fmt.Errorf("snapshot create: tar: %w", err)
	}

	// Get archive size.
	sizeOut, _, err := m.exec.Run(context.Background(), host, fmt.Sprintf("stat --printf='%%s' %s", archive))
	if err != nil {
		return nil, fmt.Errorf("snapshot create: stat: %w", err)
	}

	var size int64
	fmt.Sscanf(sizeOut, "%d", &size)

	return &Snapshot{
		Name:      name,
		Workspace: workspace,
		Size:      size,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (m *sshManager) Restore(host, workspace, name string) error {
	archive := fmt.Sprintf("%s/%s/%s.tar.gz", snapshotRoot, workspace, name)

	// Verify snapshot exists.
	if _, _, err := m.exec.Run(context.Background(), host, fmt.Sprintf("test -f %s", archive)); err != nil {
		return fmt.Errorf("snapshot restore: snapshot %q not found for workspace %q", name, workspace)
	}

	// Stop workspace containers (best-effort).
	stopCmd := fmt.Sprintf(`docker ps -q --filter "name=^%s-" | xargs -r docker stop`, workspace)
	m.exec.Run(context.Background(), host, stopCmd) //nolint:errcheck // best-effort

	// Extract archive.
	if _, _, err := m.exec.Run(context.Background(), host, fmt.Sprintf("tar xzf %s -C /", archive)); err != nil {
		return fmt.Errorf("snapshot restore: tar extract: %w", err)
	}

	// Restart workspace containers (best-effort).
	startCmd := fmt.Sprintf(`docker ps -aq --filter "name=^%s-" | xargs -r docker start`, workspace)
	m.exec.Run(context.Background(), host, startCmd) //nolint:errcheck // best-effort

	return nil
}

func (m *sshManager) List(host, workspace string) ([]Snapshot, error) {
	dir := fmt.Sprintf("%s/%s", snapshotRoot, workspace)

	// Check if directory exists.
	if _, _, err := m.exec.Run(context.Background(), host, fmt.Sprintf("test -d %s", dir)); err != nil {
		return nil, nil // No snapshots directory = empty list.
	}

	// List archives with stat info as JSON lines.
	listCmd := fmt.Sprintf(
		`find %s -maxdepth 1 -name '*.tar.gz' -printf '%%f\n' | sort | while read f; do `+
			`size=$(stat --printf='%%s' "%s/$f"); `+
			`mtime=$(stat --printf='%%Y' "%s/$f"); `+
			`printf '{"name":"%%s","size":%%s,"mtime":%%s}\n' "${f%%.tar.gz}" "$size" "$mtime"; `+
			`done`,
		dir, dir, dir,
	)
	out, _, err := m.exec.Run(context.Background(), host, listCmd)
	if err != nil {
		return nil, fmt.Errorf("snapshot list: %w", err)
	}

	if out == "" {
		return nil, nil
	}

	type entry struct {
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		Mtime int64  `json:"mtime"`
	}

	var snapshots []Snapshot
	for _, line := range splitLines(out) {
		if line == "" {
			continue
		}
		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		snapshots = append(snapshots, Snapshot{
			Name:      e.Name,
			Workspace: workspace,
			Size:      e.Size,
			CreatedAt: time.Unix(e.Mtime, 0),
		})
	}

	return snapshots, nil
}

// splitLines splits a string into non-empty lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
