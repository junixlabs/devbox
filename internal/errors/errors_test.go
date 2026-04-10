package errors_test

import (
	"errors"
	"fmt"
	"testing"

	devboxerr "github.com/junixlabs/devbox/internal/errors"
)

func TestConfigError(t *testing.T) {
	inner := fmt.Errorf("file not found")
	err := devboxerr.NewConfigError("invalid config", "Run devbox init", inner)

	if err.Error() != "invalid config: file not found" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	if !errors.Is(err, inner) {
		t.Error("errors.Is should match inner error")
	}

	var ce *devboxerr.ConfigError
	if !errors.As(err, &ce) {
		t.Error("errors.As should match ConfigError")
	}

	if ce.GetSuggestion() != "Run devbox init" {
		t.Errorf("unexpected suggestion: %s", ce.GetSuggestion())
	}
}

func TestConfigErrorNoInner(t *testing.T) {
	err := devboxerr.NewConfigError("missing name", "Add 'name' to devbox.yaml", nil)

	if err.Error() != "missing name" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	if err.Unwrap() != nil {
		t.Error("Unwrap should return nil")
	}
}

func TestConnectionError(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := devboxerr.NewConnectionError("ssh failed", "Check SSH connectivity", inner)

	if err.Error() != "ssh failed: connection refused" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var ce *devboxerr.ConnectionError
	if !errors.As(err, &ce) {
		t.Error("errors.As should match ConnectionError")
	}

	if ce.GetSuggestion() != "Check SSH connectivity" {
		t.Errorf("unexpected suggestion: %s", ce.GetSuggestion())
	}
}

func TestDockerError(t *testing.T) {
	err := devboxerr.NewDockerError("container failed", "Run: docker ps", nil)

	if err.Error() != "container failed" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var de *devboxerr.DockerError
	if !errors.As(err, &de) {
		t.Error("errors.As should match DockerError")
	}

	if de.GetSuggestion() != "Run: docker ps" {
		t.Errorf("unexpected suggestion: %s", de.GetSuggestion())
	}
}

func TestSuggestibleInterface(t *testing.T) {
	cases := []struct {
		name string
		err  devboxerr.Suggestible
		want string
	}{
		{"ConfigError", devboxerr.NewConfigError("x", "fix config", nil), "fix config"},
		{"ConnectionError", devboxerr.NewConnectionError("x", "fix conn", nil), "fix conn"},
		{"DockerError", devboxerr.NewDockerError("x", "fix docker", nil), "fix docker"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.GetSuggestion() != tc.want {
				t.Errorf("got %q, want %q", tc.err.GetSuggestion(), tc.want)
			}
		})
	}
}

func TestUnwrapChain(t *testing.T) {
	root := fmt.Errorf("root cause")
	wrapped := fmt.Errorf("middle: %w", root)
	err := devboxerr.NewConnectionError("top level", "suggestion", wrapped)

	if !errors.Is(err, root) {
		t.Error("errors.Is should traverse the full chain to root")
	}
}
