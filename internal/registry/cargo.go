package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dierodfer6/cliOne/internal/model"
)

const cratesIOBaseURL = "https://crates.io"

// Cargo resolves latest versions from the crates.io HTTP API and synthesizes
// `cargo install <pkg> --force`.
type Cargo struct {
	BaseURL string
	Client  *http.Client
}

func NewCargo() *Cargo {
	return &Cargo{BaseURL: cratesIOBaseURL, Client: defaultHTTPClient()}
}

func (c *Cargo) Kind() model.SourceKind { return model.SourceCargo }

func (c *Cargo) LatestVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/crates/%s", c.BaseURL, pkgName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("crates.io %s: %w", pkgName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crates.io %s: HTTP %d", pkgName, resp.StatusCode)
	}
	var payload struct {
		Crate struct {
			MaxStableVersion string `json:"max_stable_version"`
			NewestVersion    string `json:"newest_version"`
		} `json:"crate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("crates.io %s: parsing JSON: %w", pkgName, err)
	}
	v := payload.Crate.MaxStableVersion
	if v == "" {
		v = payload.Crate.NewestVersion
	}
	if v == "" {
		return "", fmt.Errorf("crates.io %s: no version in response", pkgName)
	}
	return normalizeVersion(v), nil
}

func (c *Cargo) UpdateCommand(pkgName string) []string {
	return []string{"cargo", "install", pkgName, "--force"}
}
