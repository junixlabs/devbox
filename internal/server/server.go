package server

import "time"

// Server represents a remote server in the pool.
type Server struct {
	Name    string    `yaml:"name"`
	Host    string    `yaml:"host"`
	User    string    `yaml:"user,omitempty"`
	Port    int       `yaml:"port,omitempty"`
	AddedAt time.Time `yaml:"added_at"`
}

// HealthStatus holds the result of health checks against a server.
type HealthStatus struct {
	SSH       bool      `yaml:"ssh"`
	Docker    bool      `yaml:"docker"`
	Tailscale bool      `yaml:"tailscale"`
	CheckedAt time.Time `yaml:"checked_at"`
}

// PoolConfig is the on-disk representation of the server pool.
type PoolConfig struct {
	Servers []Server `yaml:"servers"`
}

// Pool defines the interface for managing a pool of servers.
type Pool interface {
	Add(name, host string, opts ...AddOption) (*Server, error)
	Remove(name string) error
	List() ([]Server, error)
	HealthCheck(name string) (*HealthStatus, error)
	// HealthCheckAll checks all servers in a single pass (one file read).
	HealthCheckAll() (map[string]*HealthStatus, error)
}

// AddOption configures optional fields when adding a server.
type AddOption func(*Server)

// WithUser sets the SSH user for the server.
func WithUser(user string) AddOption {
	return func(s *Server) { s.User = user }
}

// WithPort sets the SSH port for the server.
func WithPort(port int) AddOption {
	return func(s *Server) { s.Port = port }
}
