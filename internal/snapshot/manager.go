package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/ssh"
)

const snapshotRoot = "/workspaces/.snapshots"

// validName matches safe names for shell interpolation.
var validName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func validateName(value, label string) error {
	if !validName.MatchString(value) {
		return fmt.Errorf("invalid %s %q: must match [a-zA-Z0-9._-]+", label, value)
	}
	return nil
}

type sshManager struct {
	exec ssh.Executor
}

// NewManager creates a snapshot Manager that operates on a remote server via SSH.
func NewManager(exec ssh.Executor) Manager {
	return &sshManager{exec: exec}
}

func (m *sshManager) Create(ctx context.Context, host, workspace, name string) (*Snapshot, error) {
	if err := validateName(workspace, "workspace"); err != nil {
		return nil, err
	}
	if name == "" {
		name = fmt.Sprintf("%s-%s", workspace, time.Now().UTC().Format("20060102-150405"))
	}
	if err := validateName(name, "snapshot name"); err != nil {
		return nil, err
	}

	dir := fmt.Sprintf("%s/%s", snapshotRoot, workspace)
	archive := fmt.Sprintf("%s/%s.tar.gz", dir, name)

	// Create snapshot directory.
	if _, _, err := m.exec.Run(ctx, host, fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		return nil, fmt.Errorf("snapshot create: mkdir: %w", err)
	}

	// Discover Docker volume mount paths for ALL workspace containers.
	// Container names follow the pattern: <workspace>-<service>-1
	// Iterate all matching containers, collect unique mount source paths.
	mountCmd := fmt.Sprintf(
		`docker ps -a --filter "name=^%s-" --format '{{.Names}}' | xargs -I{} docker inspect {} --format '{{range .Mounts}}{{.Source}}\n{{end}}' | sort -u | grep -v '^$' | tr '\n' ' '`,
		workspace,
	)
	mountOut, _, err := m.exec.Run(ctx, host, mountCmd)
	if err != nil {
		return nil, fmt.Errorf("snapshot create: discover mounts: %w", err)
	}

	// Build tar paths: volume mounts + workspace config files.
	tarPaths := strings.TrimSpace(mountOut)
	wsDir := fmt.Sprintf("/workspaces/%s", workspace)
	// Add workspace config files if they exist.
	configCmd := fmt.Sprintf(
		`for f in %s/devbox.yaml %s/.env; do [ -f "$f" ] && printf '%%s ' "$f"; done`,
		wsDir, wsDir,
	)
	configOut, _, err := m.exec.Run(ctx, host, configCmd)
	if err != nil {
		return nil, fmt.Errorf("snapshot create: check config files: %w", err)
	}
	if configOut != "" {
		if tarPaths != "" {
			tarPaths += " "
		}
		tarPaths += strings.TrimSpace(configOut)
	}

	if tarPaths == "" {
		return nil, fmt.Errorf("snapshot create: no files to snapshot for workspace %q", workspace)
	}

	// Create compressed archive.
	tarCmd := fmt.Sprintf("tar czf %s -C / %s", archive, tarPaths)
	if _, _, err := m.exec.Run(ctx, host, tarCmd); err != nil {
		return nil, fmt.Errorf("snapshot create: tar: %w", err)
	}

	// Get archive size.
	sizeOut, _, err := m.exec.Run(ctx, host, fmt.Sprintf("stat --printf='%%s' %s", archive))
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

func (m *sshManager) Restore(ctx context.Context, host, workspace, name string) error {
	if err := validateName(workspace, "workspace"); err != nil {
		return err
	}
	if err := validateName(name, "snapshot name"); err != nil {
		return err
	}

	archive := fmt.Sprintf("%s/%s/%s.tar.gz", snapshotRoot, workspace, name)

	// Verify snapshot exists.
	if _, _, err := m.exec.Run(ctx, host, fmt.Sprintf("test -f %s", archive)); err != nil {
		return fmt.Errorf("snapshot restore: snapshot %q not found for workspace %q", name, workspace)
	}

	// Stop workspace containers (best-effort).
	stopCmd := fmt.Sprintf(`docker ps -q --filter "name=^%s-" | xargs -r docker stop`, workspace)
	m.exec.Run(ctx, host, stopCmd) //nolint:errcheck // best-effort

	// Extract archive.
	if _, _, err := m.exec.Run(ctx, host, fmt.Sprintf("tar xzf %s -C /", archive)); err != nil {
		return fmt.Errorf("snapshot restore: tar extract: %w", err)
	}

	// Restart workspace containers (best-effort).
	startCmd := fmt.Sprintf(`docker ps -aq --filter "name=^%s-" | xargs -r docker start`, workspace)
	m.exec.Run(ctx, host, startCmd) //nolint:errcheck // best-effort

	return nil
}

func (m *sshManager) List(ctx context.Context, host, workspace string) ([]Snapshot, error) {
	if err := validateName(workspace, "workspace"); err != nil {
		return nil, err
	}

	dir := fmt.Sprintf("%s/%s", snapshotRoot, workspace)

	// Check if directory exists.
	if _, _, err := m.exec.Run(ctx, host, fmt.Sprintf("test -d %s", dir)); err != nil {
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
	out, _, err := m.exec.Run(ctx, host, listCmd)
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
	for _, line := range strings.Split(out, "\n") {
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