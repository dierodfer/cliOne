package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dierodfer6/cliOne/internal/model"
)

const npmRegistryBaseURL = "https://registry.npmjs.org"

// Npm resolves latest versions from the npm registry HTTP API (preferred over
// shelling out to the npm CLI) and synthesizes `npm install -g <pkg>`.
type Npm struct {
	BaseURL string
	Client  *http.Client
}

func NewNpm() *Npm {
	return &Npm{BaseURL: npmRegistryBaseURL, Client: defaultHTTPClient()}
}

func (n *Npm) Kind() model.SourceKind { return model.SourceNpmGlobal }

func (n *Npm) LatestVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("%s/%s/latest", n.BaseURL, pkgName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := n.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("npm registry %s: %w", pkgName, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry %s: HTTP %d", pkgName, resp.StatusCode)
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("npm registry %s: parsing JSON: %w", pkgName, err)
	}
	if payload.Version == "" {
		return "", fmt.Errorf("npm registry %s: no version in response", pkgName)
	}
	return normalizeVersion(payload.Version), nil
}

func (n *Npm) UpdateCommand(pkgName string) []string {
	return []string{"npm", "install", "-g", pkgName}
}
