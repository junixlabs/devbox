//go:build integration

// Package integration contains end-to-end tests that exercise real SSH
// connections and Docker operations against a live VPS.
//
// Run with: go test -tags integration ./internal/integration/ -v
//
// Requires:
//   - SSH access to the test server (default: devbox-vps)
//   - Docker + Compose installed on the test server
//   - Set DEVBOX_TEST_SERVER env to override the target host
//   - Set DEVBOX_TEST_SERVER=skip to skip all integration tests
package integration
