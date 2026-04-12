package identity

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/junixlabs/devbox/internal/tailscale"
)

// Identity represents a resolved user identity.
type Identity struct {
	Username string
	Source   string // "tailscale" or "env"
}

// Resolver resolves the current user identity.
type Resolver interface {
	Current() (*Identity, error)
}

type resolver struct {
	ts tailscale.Manager
}

// NewResolver creates a Resolver that tries Tailscale first, then DEVBOX_USER env var.
func NewResolver(ts tailscale.Manager) Resolver {
	return &resolver{ts: ts}
}

// unsafeChars matches anything that isn't lowercase alphanumeric or hyphen.
var unsafeChars = regexp.MustCompile(`[^a-z0-9-]`)

// Sanitize normalizes a username to lowercase alphanumeric + hyphens only.
func Sanitize(username string) string {
	s := strings.ToLower(username)
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = unsafeChars.ReplaceAllString(s, "")
	// Collapse multiple hyphens.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}

// UsernameFromLogin extracts the username portion from an email-like login.
// "user@example.com" -> "user", then sanitized.
func UsernameFromLogin(login string) string {
	if i := strings.Index(login, "@"); i != -1 {
		login = login[:i]
	}
	return Sanitize(login)
}

func (r *resolver) Current() (*Identity, error) {
	if r.ts != nil {
		status, err := r.ts.Status()
		if err == nil && status.UserLogin != "" {
			username := UsernameFromLogin(status.UserLogin)
			if username != "" {
				return &Identity{Username: username, Source: "tailscale"}, nil
			}
		}
	}

	if envUser := os.Getenv("DEVBOX_USER"); envUser != "" {
		username := Sanitize(envUser)
		if username != "" {
			return &Identity{Username: username, Source: "env"}, nil
		}
	}

	return nil, fmt.Errorf("unable to determine user identity: set DEVBOX_USER or ensure Tailscale is connected")
}
