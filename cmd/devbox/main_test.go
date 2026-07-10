package main

import (
	"testing"

	"github.com/junixlabs/devbox/internal/tailscale"
)

func TestInjectTailscaleHostname_NilStatus(t *testing.T) {
	env := map[string]string{"FOO": "bar"}
	got := injectTailscaleHostname(env, nil)
	if got["REACT_NATIVE_PACKAGER_HOSTNAME"] != "" {
		t.Errorf("expected no hostname injected for nil status, got %q", got["REACT_NATIVE_PACKAGER_HOSTNAME"])
	}
	if got["FOO"] != "bar" {
		t.Errorf("expected existing env to be preserved, got %v", got)
	}
}

func TestInjectTailscaleHostname_NilEnv(t *testing.T) {
	status := &tailscale.StatusInfo{Hostname: "devbox-vps", TailnetName: "example.com"}
	got := injectTailscaleHostname(nil, status)
	want := "devbox-vps.example.com"
	if got["REACT_NATIVE_PACKAGER_HOSTNAME"] != want {
		t.Errorf("REACT_NATIVE_PACKAGER_HOSTNAME = %q, want %q", got["REACT_NATIVE_PACKAGER_HOSTNAME"], want)
	}
}

func TestInjectTailscaleHostname_EmptyPlaceholderOverwritten(t *testing.T) {
	status := &tailscale.StatusInfo{Hostname: "devbox-vps", TailnetName: "example.com"}
	env := map[string]string{"REACT_NATIVE_PACKAGER_HOSTNAME": ""}
	got := injectTailscaleHostname(env, status)
	want := "devbox-vps.example.com"
	if got["REACT_NATIVE_PACKAGER_HOSTNAME"] != want {
		t.Errorf("REACT_NATIVE_PACKAGER_HOSTNAME = %q, want %q", got["REACT_NATIVE_PACKAGER_HOSTNAME"], want)
	}
}

func TestInjectTailscaleHostname_UserOverridePreserved(t *testing.T) {
	status := &tailscale.StatusInfo{Hostname: "devbox-vps", TailnetName: "example.com"}
	env := map[string]string{"REACT_NATIVE_PACKAGER_HOSTNAME": "custom.host"}
	got := injectTailscaleHostname(env, status)
	if got["REACT_NATIVE_PACKAGER_HOSTNAME"] != "custom.host" {
		t.Errorf("expected user override to be preserved, got %q", got["REACT_NATIVE_PACKAGER_HOSTNAME"])
	}
}
