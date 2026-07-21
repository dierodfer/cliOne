package scan

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dierodfer6/cliOne/internal/cache"
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
)

func TestRefreshFuncsForceIgnoresFreshCache(t *testing.T) {
	c := cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json"))
	if err := c.SetLatest("fresh-tool", model.VersionResult{Latest: "1.0.0"}); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{Cache: c, Registry: registry.Default()}
	cats := []model.CategoryState{
		{Tools: []model.ToolState{
			{Def: model.ToolDef{ID: "fresh-tool", OfficialURL: "https://example.com"}, Detect: model.DetectResult{Installed: true}},
			{Def: model.ToolDef{ID: "stale-tool", OfficialURL: "https://example.com"}, Detect: model.DetectResult{Installed: true}},
			{Def: model.ToolDef{ID: "not-installed", OfficialURL: "https://example.com"}, Detect: model.DetectResult{Installed: false}},
		}},
	}

	// Without force, the tool with a fresh cache entry is skipped; the
	// not-installed tool is always skipped regardless of force.
	funcs := s.RefreshFuncs(context.Background(), cats, false)
	if len(funcs) != 1 {
		t.Fatalf("expected 1 refresh func (stale-tool only), got %d", len(funcs))
	}

	// With force, every installed tool is re-queried regardless of cache freshness.
	funcs = s.RefreshFuncs(context.Background(), cats, true)
	if len(funcs) != 2 {
		t.Fatalf("expected 2 refresh funcs with force=true (fresh-tool + stale-tool), got %d", len(funcs))
	}
}
