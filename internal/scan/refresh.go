package scan

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

// LatestUpdate is the message payload produced by an async latest-version
// refresh. The TUI wraps RefreshFunc results in its own tea.Msg type.
type LatestUpdate struct {
	ToolID string
	Result model.VersionResult
}

// ErrNoLatestSource means no resolver exists for this tool (not manager-owned
// and no GitHub official URL), so its latest version cannot be determined.
var ErrNoLatestSource = errors.New("scan: no latest-version source for tool")

// RefreshFuncs returns one closure per installed tool that fetches the live
// latest version, writes it through the cache, and reports a LatestUpdate.
// Each closure is designed to be wrapped in a tea.Cmd so rows update in place
// without blocking startup. Tools with fresh cached data are skipped.
func (s *Scanner) RefreshFuncs(ctx context.Context, cats []model.CategoryState) []func() LatestUpdate {
	var out []func() LatestUpdate
	for _, cs := range cats {
		for _, ts := range cs.Tools {
			ts := ts
			if !ts.Detect.Installed {
				continue
			}
			if _, ok := s.Cache.GetLatest(ts.Def.ID); ok {
				continue // cache still fresh
			}
			out = append(out, func() LatestUpdate {
				return s.FetchLatest(ctx, ts.Def, ts.Source)
			})
		}
	}
	return out
}

// FetchLatest resolves the live latest version for one tool: through its
// owning manager when it has one, otherwise via GitHub releases when the
// official URL is a GitHub repo. The result is written through the cache on
// success.
func (s *Scanner) FetchLatest(ctx context.Context, def model.ToolDef, src model.SourceResult) LatestUpdate {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var latest string
	var err error
	if m, ok := s.Registry.ForKind(src.Kind); ok {
		latest, err = m.LatestVersion(ctx, def.ID)
	} else if repo, ok := githubRepoFromURL(def.OfficialURL); ok && s.GitHub != nil {
		latest, err = s.GitHub.LatestVersion(ctx, repo)
	} else {
		err = ErrNoLatestSource
	}

	res := model.VersionResult{Latest: latest, Err: err, FetchedAt: time.Now()}
	if err == nil {
		_ = s.Cache.SetLatest(def.ID, res)
	}
	return LatestUpdate{ToolID: def.ID, Result: res}
}

// githubRepoFromURL extracts "org/repo" from an official URL like
// https://github.com/org/repo (with optional trailing path segments ignored).
func githubRepoFromURL(url string) (string, bool) {
	const prefix = "https://github.com/"
	if !strings.HasPrefix(url, prefix) {
		return "", false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(url, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}
