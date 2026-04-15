// Resolves the latest CodeQL release versions from GitHub's
// redirect headers, avoiding the rate-limited JSON API.
package codeql

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	FallbackCLIVersion    = "2.25.1"
	FallbackBundleVersion = "codeql-bundle-v2.25.1"

	cliLatestURL    = "https://github.com/github/codeql-cli-binaries/releases/latest"
	bundleLatestURL = "https://github.com/github/codeql-action/releases/latest"
)

// noRedirectClient follows zero redirects so we can read the Location header directly.
var noRedirectClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// LatestCLIVersion returns the latest CodeQL CLI version string (e.g. "2.25.1").
// Falls back to FallbackCLIVersion on any error.
func LatestCLIVersion() string {
	tag, err := latestTagFromRedirect(cliLatestURL)
	if err != nil {
		return FallbackCLIVersion
	}
	return strings.TrimPrefix(tag, "v")
}

// LatestBundleVersion returns the latest bundle tag (e.g. "codeql-bundle-v2.25.1").
// Falls back to FallbackBundleVersion on any error.
func LatestBundleVersion() string {
	tag, err := latestTagFromRedirect(bundleLatestURL)
	if err != nil {
		return FallbackBundleVersion
	}
	return tag
}

// latestTagFromRedirect hits the /releases/latest page and extracts the version
// tag from the redirect Location header without calling the JSON API.
// GitHub redirects to .../releases/tag/<tag>, so we parse the last path segment.
func latestTagFromRedirect(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "qlt/1.0")

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no Location header in response from %s (status %d)", url, resp.StatusCode)
	}

	// Location is like: https://github.com/.../releases/tag/v2.25.1
	idx := strings.LastIndex(loc, "/")
	if idx < 0 || idx == len(loc)-1 {
		return "", fmt.Errorf("unexpected Location header format: %s", loc)
	}
	return loc[idx+1:], nil
}
