package scan

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

const (
	versionSourceUserAgent = "clione/0.1 (+https://github.com/dierodfer6/cliOne)"
	// Bound a single body read so a misconfigured URL pointing at something
	// huge cannot balloon memory; release endpoints are a few bytes to a few
	// hundred KB of HTML at most.
	versionSourceMaxBody = 1 << 20
)

var versionSourceClient = &http.Client{Timeout: 10 * time.Second}

// fetchVersionSource resolves a tool's declared authoritative version endpoint:
// fetch the URL and apply its single-capture-group regex to the raw response
// body. Keeping extraction regex-based (rather than per-format parsers) means
// one mechanism covers plain-text endpoints, JSON documents, and HTML download
// pages alike, mirroring how detect.ExtractVersion reads command output.
func fetchVersionSource(ctx context.Context, spec model.VersionSourceSpec) (string, error) {
	re, err := regexp.Compile(spec.Regex)
	if err != nil {
		return "", fmt.Errorf("version_source %s: regex does not compile: %w", spec.URL, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return "", fmt.Errorf("version_source %s: %w", spec.URL, err)
	}
	req.Header.Set("User-Agent", versionSourceUserAgent)

	resp, err := versionSourceClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("version_source %s: %w", spec.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("version_source %s: HTTP %d", spec.URL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, versionSourceMaxBody))
	if err != nil {
		return "", fmt.Errorf("version_source %s: reading body: %w", spec.URL, err)
	}

	m := re.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("version_source %s: no version match in response", spec.URL)
	}
	return string(m[1]), nil
}
