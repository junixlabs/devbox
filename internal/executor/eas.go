package executor

import (
	"encoding/json"
	"regexp"
)

// httpsURLRe matches an https URL, used as a resilient fallback for extracting
// a build link when the EAS CLI JSON shape is unexpected.
var httpsURLRe = regexp.MustCompile(`https://[^\s"']+`)

// parseEASBuildURL extracts an installable artifact/build URL from the output
// of `eas build --json`. EAS prints a JSON array of build objects; we prefer
// the built application archive (the .apk/.aab), then the build details page.
// If the JSON can't be parsed (EAS CLI shape drift), it falls back to the
// first https URL found anywhere in the output. Returns "" when nothing found.
func parseEASBuildURL(out []byte) string {
	var builds []struct {
		Artifacts struct {
			ApplicationArchiveURL string `json:"applicationArchiveUrl"`
			BuildURL              string `json:"buildUrl"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(out, &builds); err == nil && len(builds) > 0 {
		a := builds[0].Artifacts
		if a.ApplicationArchiveURL != "" {
			return a.ApplicationArchiveURL
		}
		if a.BuildURL != "" {
			return a.BuildURL
		}
	}
	if m := httpsURLRe.Find(out); m != nil {
		return string(m)
	}
	return ""
}
