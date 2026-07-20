// Package scan wires catalog, cache, detect, source, and registry together
// into the []model.CategoryState the TUI renders. It is the only package that
// composes those layers.
package scan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dierodfer6/cliOne/internal/cache"
	"github.com/dierodfer6/cliOne/internal/catalog"
	"github.com/dierodfer6/cliOne/internal/detect"
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
	"github.com/dierodfer6/cliOne/internal/source"
)

// Scanner holds the wired dependencies for a scan.
type Scanner struct {
	Catalog  *catalog.Catalog
	Cache    cache.Cache
	Resolver *source.Resolver
	Registry registry.Registry
	GitHub   *registry.GitHubRelease
}

// New builds a Scanner with the embedded catalog, the default cache location,
// and the default source resolver and manager registry.
func New() (*Scanner, error) {
	cat, err := catalog.Load()
	if err != nil {
		return nil, err
	}
	path, err := cache.DefaultPath()
	if err != nil {
		path = filepath.Join(os.TempDir(), "clione", "cache.json")
	}
	return &Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(path),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
		GitHub:   registry.NewGitHubRelease(),
	}, nil
}

// Scan detects every catalog tool (live, in parallel), resolves ownership
// (cache-or-live), reads the cached latest version (never the network — the
// async refresh in refresh.go does live fetches), computes each StatusState,
// and groups results into CategoryState in catalog order.
func (s *Scanner) Scan(ctx context.Context) ([]model.CategoryState, error) {
	states := make([]model.ToolState, len(s.Catalog.Tools))
	var wg sync.WaitGroup
	for i, def := range s.Catalog.Tools {
		wg.Add(1)
		go func(i int, def model.ToolDef) {
			defer wg.Done()
			states[i] = s.scanTool(ctx, def)
		}(i, def)
	}
	wg.Wait()

	byCat := map[string][]model.ToolState{}
	for _, ts := range states {
		byCat[ts.Def.Category] = append(byCat[ts.Def.Category], ts)
	}

	var out []model.CategoryState
	for _, cat := range s.Catalog.Categories {
		tools := byCat[cat.ID]
		if len(tools) == 0 {
			continue
		}
		cs := model.CategoryState{Category: cat, Tools: tools, Total: len(tools)}
		for _, ts := range tools {
			if ts.Detect.Installed {
				cs.Installed++
			}
		}
		out = append(out, cs)
	}
	return out, nil
}

func (s *Scanner) scanTool(ctx context.Context, def model.ToolDef) model.ToolState {
	ts := model.ToolState{Def: def}
	ts.Detect, _ = detect.RunDetect(ctx, def)

	if ts.Detect.Installed {
		if src, ok := s.Cache.GetSource(def.ID); ok {
			ts.Source = src
		} else if res, err := s.Resolver.Resolve(ctx, BinName(def)); err == nil {
			ts.Source = res
			_ = s.Cache.SetSource(def.ID, res)
		}
	}

	if latest, ok := s.Cache.GetLatest(def.ID); ok {
		ts.Latest = latest
	}

	ts.Status = ComputeStatus(def, ts.Detect, ts.Source, ts.Latest, s.Registry)
	return ts
}

// BinName is the binary a tool is detected with: the first word of its detect
// command (e.g. ripgrep -> "rg").
func BinName(def model.ToolDef) string {
	fields := strings.Fields(def.Detect.Cmd)
	if len(fields) == 0 {
		return def.ID
	}
	return fields[0]
}

// ComputeStatus derives the 4-state semaphore for one tool.
//
//   - not detected                          -> not installed (red)
//   - installed, no native or manager path  -> no updater (white)
//   - installed, known newer latest         -> update available (yellow)
//   - otherwise                             -> up to date (green); an async
//     latest refresh can flip it to yellow later
func ComputeStatus(def model.ToolDef, det model.DetectResult, src model.SourceResult, latest model.VersionResult, reg registry.Registry) model.StatusState {
	if !det.Installed {
		return model.StatusNotInstalled
	}
	canUpdate := def.Update != nil
	if !canUpdate {
		if m, ok := reg.ForKind(src.Kind); ok && len(m.UpdateCommand(def.ID)) > 0 {
			canUpdate = true
		}
	}
	if !canUpdate {
		return model.StatusNoUpdater
	}
	if latest.Latest != "" && det.Version != "" && latest.Latest != det.Version {
		return model.StatusUpdateAvail
	}
	return model.StatusUpToDate
}
