package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dierodfer/cliOne/internal/model"
)

const githubBaseURL = "https://github.com"

// GitHubRelease resolves the latest release tag of a GitHub project by
// requesting /<org>/<repo>/releases/latest and reading the redirect Location
// header instead of following it. This deliberately avoids the GitHub REST
// API and its 60 req/hr unauthenticated rate limit.
//
// pkgName is "org/repo".
type GitHubRelease struct {
	BaseURL string
	Client  *http.Client
}

func NewGitHubRelease() *GitHubRelease {
	return &GitHubRelease{
		BaseURL: githubBaseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (g *GitHubRelease) Kind() model.SourceKind { return model.SourceManual }

func (g *GitHubRelease) LatestVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("%s/%s/releases/latest", g.BaseURL, pkgName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := g.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github %s: %w", pkgName, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		return "", fmt.Errorf("github %s: expected redirect, got HTTP %d", pkgName, resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	tag, err := TagFromLocation(loc)
	if err != nil {
		return "", fmt.Errorf("github %s: %w", pkgName, err)
	}
	return tag, nil
}

// TagFromLocation extracts the release tag from a .../releases/tag/<tag>
// redirect Location URL and normalizes a leading "v".
func TagFromLocation(loc string) (string, error) {
	const marker = "/releases/tag/"
	i := strings.Index(loc, marker)
	if i < 0 {
		return "", fmt.Errorf("no release tag in redirect location %q", loc)
	}
	tag := loc[i+len(marker):]
	if j := strings.IndexAny(tag, "?#"); j >= 0 {
		tag = tag[:j]
	}
	if tag == "" {
		return "", fmt.Errorf("empty release tag in redirect location %q", loc)
	}
	return normalizeVersion(tag), nil
}
