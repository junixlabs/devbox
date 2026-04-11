package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"gopkg.in/yaml.v3"
)

type filePool struct {
	configPath string
	sshExec    devboxssh.Executor
}

// NewFilePool creates a Pool backed by a YAML file at the given path.
func NewFilePool(configPath string, sshExec devboxssh.Executor) (Pool, error) {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}
	return &filePool{configPath: configPath, sshExec: sshExec}, nil
}

// DefaultConfigPath returns ~/.config/devbox/servers.yaml.
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config dir: %w", err)
	}
	return filepath.Join(configDir, "devbox", "servers.yaml"), nil
}

func (p *filePool) load() (*PoolConfig, error) {
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &PoolConfig{}, nil
		}
		return nil, fmt.Errorf("reading server config: %w", err)
	}
	var cfg PoolConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing server config: %w", err)
	}
	return &cfg, nil
}

func (p *filePool) save(cfg *PoolConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling server config: %w", err)
	}
	return os.WriteFile(p.configPath, data, 0o644)
}

func (p *filePool) Add(name, host string, opts ...AddOption) (*Server, error) {
	cfg, err := p.load()
	if err != nil {
		return nil, err
	}
	for _, s := range cfg.Servers {
		if s.Name == name {
			return nil, fmt.Errorf("server %q already exists", name)
		}
	}
	srv := Server{Name: name, Host: host, AddedAt: time.Now()}
	for _, opt := range opts {
		opt(&srv)
	}
	cfg.Servers = append(cfg.Servers, srv)
	if err := p.save(cfg); err != nil {
		return nil, err
	}
	return &srv, nil
}

func (p *filePool) Remove(name string) error {
	cfg, err := p.load()
	if err != nil {
		return err
	}
	idx := -1
	for i, s := range cfg.Servers {
		if s.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("server %q not found", name)
	}
	cfg.Servers = append(cfg.Servers[:idx], cfg.Servers[idx+1:]...)
	return p.save(cfg)
}

func (p *filePool) List() ([]Server, error) {
	cfg, err := p.load()
	if err != nil {
		return nil, err
	}
	return cfg.Servers, nil
}

func (p *filePool) HealthCheck(name string) (*HealthStatus, error) {
	cfg, err := p.load()
	if err != nil {
		return nil, err
	}
	var srv *Server
	for i := range cfg.Servers {
		if cfg.Servers[i].Name == name {
			srv = &cfg.Servers[i]
			break
		}
	}
	if srv == nil {
		return nil, fmt.Errorf("server %q not found", name)
	}

	status := &HealthStatus{CheckedAt: time.Now()}
	if p.sshExec == nil {
		return status, nil
	}

	host := sshHost(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stdout, _, sshErr := p.sshExec.Run(ctx, host, "echo ok")
	if sshErr == nil && len(stdout) > 0 {
		status.SSH = true
	}

	stdout, _, sshErr = p.sshExec.Run(ctx, host, "docker info --format '{{.ID}}'")
	if sshErr == nil && len(stdout) > 0 {
		status.Docker = true
	}

	_, _, sshErr = p.sshExec.Run(ctx, host, "tailscale status --json")
	if sshErr == nil {
		status.Tailscale = true
	}

	return status, nil
}

func sshHost(s *Server) string {
	host := s.Host
	if s.User != "" {
		host = s.User + "@" + host
	}
	return host
}
