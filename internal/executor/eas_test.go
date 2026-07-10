package executor

import "testing"

func TestParseEASBuildURL_PrefersApplicationArchive(t *testing.T) {
	out := []byte(`[{"id":"abc","artifacts":{"applicationArchiveUrl":"https://expo.dev/artifacts/app-abc.apk","buildUrl":"https://expo.dev/accounts/x/builds/abc"}}]`)
	if got := parseEASBuildURL(out); got != "https://expo.dev/artifacts/app-abc.apk" {
		t.Errorf("parseEASBuildURL = %q, want the application archive URL", got)
	}
}

func TestParseEASBuildURL_FallsBackToBuildURL(t *testing.T) {
	out := []byte(`[{"id":"abc","artifacts":{"buildUrl":"https://expo.dev/accounts/x/builds/abc"}}]`)
	if got := parseEASBuildURL(out); got != "https://expo.dev/accounts/x/builds/abc" {
		t.Errorf("parseEASBuildURL = %q, want the build details URL", got)
	}
}

func TestParseEASBuildURL_RegexFallback(t *testing.T) {
	// Non-JSON output (e.g. a CLI notice line) — extract the first https URL.
	out := []byte("Build finished. See https://expo.dev/accounts/x/builds/xyz for details.")
	if got := parseEASBuildURL(out); got != "https://expo.dev/accounts/x/builds/xyz" {
		t.Errorf("parseEASBuildURL = %q, want first https URL", got)
	}
}

func TestParseEASBuildURL_Empty(t *testing.T) {
	if got := parseEASBuildURL([]byte("no urls here")); got != "" {
		t.Errorf("parseEASBuildURL = %q, want empty", got)
	}
}
