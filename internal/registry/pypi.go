package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

const (
	pypiBaseURL = "https://pypi.org"
	userAgent   = "clione/0.1 (+https://github.com/dierodfer6/cliOne)"
)

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// PyPI resolves latest versions from the PyPI JSON API and synthesizes
// `uv tool upgrade <pkg>` for uv-managed tools.
type PyPI struct {
	BaseURL string
	Client  *http.Client
}

func NewPyPI() *PyPI {
	return &PyPI{BaseURL: pypiBaseURL, Client: defaultHTTPClient()}
}

func (p *PyPI) Kind() model.SourceKind { return model.SourceUvTool }

func (p *PyPI) LatestVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("%s/pypi/%s/json", p.BaseURL, pkgName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("pypi %s: %w", pkgName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pypi %s: HTTP %d", pkgName, resp.StatusCode)
	}
	var payload struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("pypi %s: parsing JSON: %w", pkgName, err)
	}
	if payload.Info.Version == "" {
		return "", fmt.Errorf("pypi %s: no version in response", pkgName)
	}
	return normalizeVersion(payload.Info.Version), nil
}

func (p *PyPI) UpdateCommand(pkgName string) []string {
	return []string{"uv", "tool", "upgrade", pkgName}
}
