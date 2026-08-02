package scan

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dierodfer/cliOne/internal/model"
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
// without blocking startup. Tools with fresh cached data are skipped unless
// force is set, in which case every installed tool is re-queried live
// regardless of the cache's TTL (used by the TUI's manual refresh key).
func (s *Scanner) RefreshFuncs(ctx context.Context, cats []model.CategoryState, force bool) []func() LatestUpdate {
	var out []func() LatestUpdate
	for _, cs := range cats {
		for _, ts := range cs.Tools {
			ts := ts
			if !ts.Detect.Installed {
				continue
			}
			if !force {
				if _, ok := s.Cache.GetLatest(ts.Def.ID); ok {
					continue // cache still fresh
				}
			}
			out = append(out, func() LatestUpdate {
				return s.FetchLatest(ctx, ts.Def, ts.Source)
			})
		}
	}
	return out
}

// FetchLatest resolves the live latest version for one tool, preferring what
// the project itself publishes over what any package manager happens to ship:
//
//  1. the catalog's version_source, i.e. the project's own release endpoint;
//  2. GitHub releases, which is that same upstream authority for projects
//     hosted there;
//  3. the owning package manager's registry, as a last resort.
//
// Manager registries come last on purpose. A manager reports the version it
// distributes, which routinely trails the real release — a Homebrew cask or an
// npm package can sit a version behind upstream — so trusting it first makes
// an out-of-date tool look current. Which manager installed a binary is still
// worth knowing, and is reported separately as its source; it just isn't the
// authority on what the newest release is.
//
// The result is written through the cache on success.
func (s *Scanner) FetchLatest(ctx context.Context, def model.ToolDef, src model.SourceResult) LatestUpdate {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var latest string
	var err error
	repo, isGitHub := githubRepo(def)
	switch {
	case def.VersionSource != nil:
		latest, err = fetchVersionSource(ctx, *def.VersionSource)
	case isGitHub && s.GitHub != nil:
		latest, err = s.GitHub.LatestVersion(ctx, repo)
	default:
		if m, ok := s.Registry.ForKind(src.Kind); ok {
			latest, err = m.LatestVersion(ctx, def.PkgName())
		} else {
			err = ErrNoLatestSource
		}
	}

	res := model.VersionResult{Latest: latest, Err: err, FetchedAt: time.Now()}
	if err == nil {
		_ = s.Cache.SetLatest(def.ID, res)
	}
	return LatestUpdate{ToolID: def.ID, Result: res}
}

// githubRepo resolves the GitHub project whose releases are this tool's
// upstream feed: the catalog's explicit repo when set, otherwise the official
// URL when it is itself a GitHub URL.
func githubRepo(def model.ToolDef) (string, bool) {
	if def.Repo != "" {
		return def.Repo, true
	}
	return githubRepoFromURL(def.OfficialURL)
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
