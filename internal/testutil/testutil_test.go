package testutil

import (
	"os"
	"testing"
)

func TestTestServer_Default(t *testing.T) {
	os.Unsetenv("DEVBOX_TEST_SERVER")
	got := TestServer(t)
	if got != "devbox-vps" {
		t.Errorf("TestServer() = %q, want %q", got, "devbox-vps")
	}
}

func TestTestServer_CustomEnv(t *testing.T) {
	t.Setenv("DEVBOX_TEST_SERVER", "my-server")
	got := TestServer(t)
	if got != "my-server" {
		t.Errorf("TestServer() = %q, want %q", got, "my-server")
	}
}
