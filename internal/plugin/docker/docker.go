package docker

import (
	"fmt"
	"strings"

	"github.com/junixlabs/devbox/internal/plugin"
)

// CommandRunner executes a command with arguments and returns the output.
type CommandRunner func(command string, args ...string) ([]byte, error)

// DockerProvider implements plugin.Provider using the Docker CLI.
type DockerProvider struct {
	run CommandRunner
}

// New creates a DockerProvider that delegates to the Docker CLI via the given runner.
func New(run CommandRunner) *DockerProvider {
	return &DockerProvider{run: run}
}

func (d *DockerProvider) Create(name string, image string, opts plugin.CreateOpts) error {
	args := []string{"run", "-d", "--name", name}

	if opts.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.1f", opts.CPUs))
	}
	if opts.Memory != "" {
		args = append(args, "--memory", opts.Memory)
	}
	for k, v := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%s", v, k))
	}
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for _, vol := range opts.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, image)

	if opts.Command != "" {
		args = append(args, "sh", "-c", opts.Command)
	}

	_, err := d.run("docker", args...)
	if err != nil {
		return fmt.Errorf("docker create %s: %w", name, err)
	}
	return nil
}

func (d *DockerProvider) Start(name string) error {
	_, err := d.run("docker", "start", name)
	if err != nil {
		return fmt.Errorf("docker start %s: %w", name, err)
	}
	return nil
}

func (d *DockerProvider) Stop(name string) error {
	_, err := d.run("docker", "stop", name)
	if err != nil {
		return fmt.Errorf("docker stop %s: %w", name, err)
	}
	return nil
}

func (d *DockerProvider) Destroy(name string) error {
	_, err := d.run("docker", "rm", "-f", name)
	if err != nil {
		return fmt.Errorf("docker destroy %s: %w", name, err)
	}
	return nil
}

func (d *DockerProvider) Status(name string) (string, error) {
	out, err := d.run("docker", "inspect", "--format", "{{.State.Status}}", name)
	if err != nil {
		return "", fmt.Errorf("docker status %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *DockerProvider) ProviderName() string {
	return "docker"
}

// Manifest returns the built-in manifest for the Docker provider.
func Manifest() plugin.Manifest {
	return plugin.Manifest{
		Name:        "docker",
		Version:     "builtin",
		Type:        plugin.TypeProvider,
		Entrypoint:  "docker",
		Description: "Built-in Docker container provider",
		BuiltIn:     true,
	}
}
