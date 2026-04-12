package server

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

// Selector picks the best server from a list based on resource availability.
type Selector interface {
	Select(ctx context.Context, servers []Server) (*Server, error)
}

type serverScore struct {
	server Server
	score  float64
}

// leastLoadedSelector picks the server with the most available resources.
type leastLoadedSelector struct {
	sshExec devboxssh.Executor
}

// NewLeastLoaded creates a Selector that picks the least-loaded server.
func NewLeastLoaded(exec devboxssh.Executor) Selector {
	return &leastLoadedSelector{sshExec: exec}
}

func (s *leastLoadedSelector) Select(ctx context.Context, servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available")
	}
	if len(servers) == 1 {
		// Single server: still verify it's reachable.
		host := SSHHost(&servers[0])
		srvCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if _, err := QueryResources(srvCtx, s.sshExec, host); err != nil {
			return nil, fmt.Errorf("server %q is offline: %w", servers[0].Name, err)
		}
		return &servers[0], nil
	}

	var mu sync.Mutex
	var scores []serverScore
	var wg sync.WaitGroup

	for _, srv := range servers {
		wg.Add(1)
		go func(srv Server) {
			defer wg.Done()

			srvCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			host := SSHHost(&srv)
			info, err := QueryResources(srvCtx, s.sshExec, host)
			if err != nil {
				slog.Warn("server offline or unreachable, skipping",
					"server", srv.Name, "host", host, "error", err)
				return
			}

			score := AvailableScore(info)
			mu.Lock()
			scores = append(scores, serverScore{server: srv, score: score})
			mu.Unlock()
		}(srv)
	}

	wg.Wait()

	if len(scores) == 0 {
		return nil, fmt.Errorf("all servers are offline or unreachable")
	}

	// Sort by score descending, then by name for deterministic tie-breaking.
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].server.Name < scores[j].server.Name
	})

	best := scores[0]
	slog.Debug("server selected", "server", best.server.Name, "score", best.score)
	return &best.server, nil
}
