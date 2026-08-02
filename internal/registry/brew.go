package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/dierodfer/cliOne/internal/model"
)

// Brew resolves latest versions via the local brew CLI (`brew info --json=v2`,
// which reflects the freshest formula data brew knows) and synthesizes
// `brew upgrade <pkg>`.
type Brew struct {
	// Run executes a command and returns its stdout. Injectable for tests.
	Run func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewBrew returns a Brew adapter backed by the real brew binary.
func NewBrew() *Brew {
	return &Brew{Run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if _, err := exec.LookPath(name); err != nil {
			return nil, fmt.Errorf("%s: not installed: %w", name, err)
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, name, args...).Output()
	}}
}

func (b *Brew) Kind() model.SourceKind { return model.SourceHomebrew }

func (b *Brew) LatestVersion(ctx context.Context, pkgName string) (string, error) {
	out, err := b.Run(ctx, "brew", "info", "--json=v2", pkgName)
	if err != nil {
		return "", fmt.Errorf("brew info %s: %w", pkgName, err)
	}
	var payload struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
		// Casks (GUI apps like vscode, temurin) have no "formulae" entry;
		// `brew info --json=v2` reports them under "casks" instead, with the
		// version given directly rather than nested under "versions".
		Casks []struct {
			Version string `json:"version"`
		} `json:"casks"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("brew info %s: parsing JSON: %w", pkgName, err)
	}
	if len(payload.Formulae) > 0 && payload.Formulae[0].Versions.Stable != "" {
		return normalizeVersion(payload.Formulae[0].Versions.Stable), nil
	}
	if len(payload.Casks) > 0 && payload.Casks[0].Version != "" {
		return normalizeVersion(payload.Casks[0].Version), nil
	}
	return "", fmt.Errorf("brew info %s: no stable version in output", pkgName)
}
