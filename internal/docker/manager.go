package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"github.com/junixlabs/devbox/internal/ssh"
)

// validName matches safe identifiers for workspace and service names.
// Only alphanumeric, hyphens, underscores, and dots are allowed.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// WorkspaceBaseDir is the root directory for workspaces on the remote server.
const WorkspaceBaseDir = "/workspaces"

// Manager defines the interface for Docker Compose operations on a remote server.
type Manager interface {
	// Deploy creates the workspace directory, copies the compose file, and starts services.
	Deploy(ctx context.Context, composeYAML []byte) error

	// Up starts services defined in the compose file.
	Up(ctx context.Context) error

	// Down stops services defined in the compose file.
	Down(ctx context.Context) error

	// PS returns the status of services.
	PS(ctx context.Context) (string, error)

	// Logs streams logs for a specific service.
	Logs(ctx context.Context, service string, stdout, stderr io.Writer) error

	// Destroy stops services, removes volumes, and deletes the workspace directory.
	Destroy(ctx context.Context) error
}

type dockerManager struct {
	ssh         ssh.Executor
	host        string
	name        string
	composeDir  string
	composePath string
}

// NewManager creates a Manager for the given workspace on the given host.
// Returns an error if name contains unsafe characters.
func NewManager(sshExec ssh.Executor, host string, name string) (Manager, error) {
	if !validName.MatchString(name) {
		return nil, devboxerr.NewDockerError(
			fmt.Sprintf("invalid workspace name %q", name),
			"Workspace names must contain only alphanumeric characters, hyphens, underscores, and dots",
			nil,
		)
	}
	// Use explicit "/" join — these are remote Linux paths, not local OS paths.
	dir := WorkspaceBaseDir + "/" + name
	return &dockerManager{
		ssh:         sshExec,
		host:        host,
		name:        name,
		composeDir:  dir,
		composePath: dir + "/docker-compose.yml",
	}, nil
}

func (m *dockerManager) Deploy(ctx context.Context, composeYAML []byte) error {
	// Create workspace directory on remote.
	if _, _, err := m.ssh.Run(ctx, m.host, fmt.Sprintf("mkdir -p %s", m.composeDir)); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("failed to create workspace directory %s on %s", m.composeDir, m.host),
			fmt.Sprintf("Check SSH access: ssh %s", m.host),
			err,
		)
	}

	// Write compose YAML to a local temp file for SCP.
	tmpFile, err := os.CreateTemp("", "devbox-compose-*.yml")
	if err != nil {
		return devboxerr.NewDockerError(
			"failed to create temp file for compose YAML",
			"Check disk space and /tmp permissions",
			err,
		)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(composeYAML); err != nil {
		tmpFile.Close()
		return devboxerr.NewDockerError(
			"failed to write compose YAML to temp file",
			"Check disk space",
			err,
		)
	}
	tmpFile.Close()

	// Copy compose file to remote.
	if err := m.ssh.CopyTo(ctx, m.host, tmpFile.Name(), m.composePath); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("failed to copy compose file to %s:%s", m.host, m.composePath),
			fmt.Sprintf("Check SSH access: ssh %s", m.host),
			err,
		)
	}

	// Start services.
	if _, _, err := m.ssh.Run(ctx, m.host, m.composeCmd("up -d")); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("docker compose up failed on %s", m.host),
			"Check Docker is running: ssh "+m.host+" docker info",
			err,
		)
	}

	return nil
}

func (m *dockerManager) Up(ctx context.Context) error {
	if _, _, err := m.ssh.Run(ctx, m.host, m.composeCmd("up -d")); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("docker compose up failed on %s", m.host),
			"Check Docker is running: ssh "+m.host+" docker info",
			err,
		)
	}
	return nil
}

func (m *dockerManager) Down(ctx context.Context) error {
	if _, _, err := m.ssh.Run(ctx, m.host, m.composeCmd("down")); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("docker compose down failed on %s", m.host),
			"Check Docker is running: ssh "+m.host+" docker info",
			err,
		)
	}
	return nil
}

func (m *dockerManager) PS(ctx context.Context) (string, error) {
	stdout, _, err := m.ssh.Run(ctx, m.host, m.composeCmd("ps"))
	if err != nil {
		return "", devboxerr.NewDockerError(
			fmt.Sprintf("docker compose ps failed on %s", m.host),
			"Check Docker is running: ssh "+m.host+" docker info",
			err,
		)
	}
	return stdout, nil
}

func (m *dockerManager) Logs(ctx context.Context, service string, stdout, stderr io.Writer) error {
	if !validName.MatchString(service) {
		return devboxerr.NewDockerError(
			fmt.Sprintf("invalid service name %q", service),
			"Service names must contain only alphanumeric characters, hyphens, underscores, and dots",
			nil,
		)
	}
	cmd := m.composeCmd(fmt.Sprintf("logs --follow %s", service))
	if err := m.ssh.RunStream(ctx, m.host, cmd, stdout, stderr); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("docker compose logs failed for %s on %s", service, m.host),
			"Check that the service exists: docker compose ps",
			err,
		)
	}
	return nil
}

func (m *dockerManager) Destroy(ctx context.Context) error {
	// Stop services and remove volumes.
	if _, _, err := m.ssh.Run(ctx, m.host, m.composeCmd("down -v")); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("docker compose down -v failed on %s", m.host),
			"Check Docker is running: ssh "+m.host+" docker info",
			err,
		)
	}

	// Remove workspace directory.
	if _, _, err := m.ssh.Run(ctx, m.host, fmt.Sprintf("rm -rf %s", m.composeDir)); err != nil {
		return devboxerr.NewDockerError(
			fmt.Sprintf("failed to remove workspace directory %s on %s", m.composeDir, m.host),
			fmt.Sprintf("Check SSH access: ssh %s", m.host),
			err,
		)
	}

	return nil
}

// composeCmd builds a docker compose command string with the project file path.
func (m *dockerManager) composeCmd(subcommand string) string {
	return fmt.Sprintf("docker compose -f %s %s", m.composePath, subcommand)
}
