package executor

import (
	"context"
	"io"
	"strings"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/docker"
	"github.com/junixlabs/devbox/internal/ssh"
)

// dockerExecutor adapts the existing, unchanged docker.Manager to the
// Executor interface. The Docker path's behavior is preserved byte-for-byte.
type dockerExecutor struct {
	mgr     docker.Manager
	cfg     *config.DevboxConfig
	wsName  string
	service string
}

func newDockerExecutor(sshExec ssh.Executor, cfg *config.DevboxConfig, host, name string) (Executor, error) {
	mgr, err := docker.NewManager(sshExec, host, name)
	if err != nil {
		return nil, err
	}
	return &dockerExecutor{
		mgr:     mgr,
		cfg:     cfg,
		wsName:  name,
		service: firstService(cfg.Services),
	}, nil
}

func (d *dockerExecutor) Deploy(ctx context.Context) error {
	composeYAML, err := docker.GenerateCompose(d.wsName, d.cfg)
	if err != nil {
		return err
	}
	return d.mgr.Deploy(ctx, composeYAML)
}

func (d *dockerExecutor) Up(ctx context.Context) error {
	return d.mgr.Up(ctx)
}

func (d *dockerExecutor) Down(ctx context.Context) error {
	return d.mgr.Down(ctx)
}

func (d *dockerExecutor) Logs(ctx context.Context, follow bool, stdout, stderr io.Writer) error {
	// docker.Manager.Logs always follows (--follow); matches existing behavior.
	_ = follow
	return d.mgr.Logs(ctx, d.service, stdout, stderr)
}

func (d *dockerExecutor) Destroy(ctx context.Context) error {
	return d.mgr.Destroy(ctx)
}

// firstService returns the base name of the first service, or "app" as default.
// Mirrors workspace.firstService — kept local to avoid an import cycle
// (workspace imports executor).
func firstService(services []string) string {
	if len(services) == 0 {
		return "app"
	}
	svc := services[0]
	if i := strings.LastIndex(svc, ":"); i != -1 {
		svc = svc[:i]
	}
	if i := strings.LastIndex(svc, "/"); i != -1 {
		svc = svc[i+1:]
	}
	return svc
}
