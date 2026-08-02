package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dierodfer/cliOne/internal/model"
)

func newTestStore(t *testing.T) (*JSONStore, *time.Time) {
	t.Helper()
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	s := NewJSONStore(filepath.Join(t.TempDir(), "cache.json"))
	s.nowFunc = func() time.Time { return now }
	return s, &now
}

func TestSourceRoundTripAndPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s := NewJSONStore(path)

	in := model.SourceResult{
		Kind:     model.SourceCargo,
		BinPath:  "/home/u/.cargo/bin/rg",
		AllPaths: []string{"/home/u/.cargo/bin/rg", "/usr/bin/rg"},
	}
	if err := s.SetSource("ripgrep", in); err != nil {
		t.Fatal(err)
	}

	// Reload from disk in a fresh store.
	s2 := NewJSONStore(path)
	got, ok := s2.GetSource("ripgrep")
	if !ok {
		t.Fatal("expected cache hit after reload")
	}
	if got.Kind != in.Kind || got.BinPath != in.BinPath || len(got.AllPaths) != 2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLatestRoundTrip(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.SetLatest("jq", model.VersionResult{Latest: "1.7.1"}); err != nil {
		t.Fatal(err)
	}
	got, ok := s.GetLatest("jq")
	if !ok || got.Latest != "1.7.1" {
		t.Fatalf("expected hit with 1.7.1, got %+v ok=%v", got, ok)
	}
}

func TestTTLExpiry(t *testing.T) {
	s, now := newTestStore(t)

	if err := s.SetSource("git", model.SourceResult{Kind: model.SourceAptDnf, BinPath: "/usr/bin/git"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLatest("git", model.VersionResult{Latest: "2.50.0"}); err != nil {
		t.Fatal(err)
	}

	// Both fresh.
	if _, ok := s.GetSource("git"); !ok {
		t.Fatal("source should be fresh")
	}
	if _, ok := s.GetLatest("git"); !ok {
		t.Fatal("latest should be fresh")
	}

	// After 13h: latest expired (12h TTL), source still valid (7d TTL).
	*now = now.Add(13 * time.Hour)
	if _, ok := s.GetLatest("git"); ok {
		t.Fatal("latest should have expired after 13h")
	}
	if _, ok := s.GetSource("git"); !ok {
		t.Fatal("source should still be valid after 13h")
	}

	// After 8 days total: source expired too.
	*now = now.Add(8 * 24 * time.Hour)
	if _, ok := s.GetSource("git"); ok {
		t.Fatal("source should have expired after 8 days")
	}
}

func TestIndependentTTLKeysPerTool(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.SetSource("a", model.SourceResult{Kind: model.SourceManual, BinPath: "/x/a"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetLatest("a"); ok {
		t.Fatal("setting source must not create a latest entry")
	}
	if err := s.SetLatest("b", model.VersionResult{Latest: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetSource("b"); ok {
		t.Fatal("setting latest must not create a source entry")
	}
}

func TestCorruptFileRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, []byte("{not json at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewJSONStore(path) // must not panic
	if _, ok := s.GetSource("anything"); ok {
		t.Fatal("corrupt file should behave as empty cache")
	}
	// And writing over the corrupt file works.
	if err := s.SetLatest("x", model.VersionResult{Latest: "1.2.3"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetLatest("x"); !ok {
		t.Fatal("expected hit after rewriting corrupt cache")
	}
}

func TestSetSourcesBatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	s := NewJSONStore(path)

	batch := map[string]model.SourceResult{
		"git": {Kind: model.SourceAptDnf, BinPath: "/usr/bin/git"},
		"rg":  {Kind: model.SourceCargo, BinPath: "/home/u/.cargo/bin/rg"},
	}
	if err := s.SetSources(batch); err != nil {
		t.Fatal(err)
	}

	// Every entry persists and reloads from the single write.
	s2 := NewJSONStore(path)
	if got, ok := s2.GetSource("git"); !ok || got.Kind != model.SourceAptDnf {
		t.Fatalf("git not persisted by batch: %+v ok=%v", got, ok)
	}
	if got, ok := s2.GetSource("rg"); !ok || got.Kind != model.SourceCargo {
		t.Fatalf("rg not persisted by batch: %+v ok=%v", got, ok)
	}

	// An empty batch is a no-op and never errors.
	if err := s.SetSources(nil); err != nil {
		t.Fatalf("empty batch should be a no-op, got %v", err)
	}
}

func TestMissingFileStartsEmpty(t *testing.T) {
	s := NewJSONStore(filepath.Join(t.TempDir(), "nope", "cache.json"))
	if _, ok := s.GetLatest("x"); ok {
		t.Fatal("missing file should behave as empty cache")
	}
}

func TestFailedLookupsNeverCached(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.SetLatest("x", model.VersionResult{Err: os.ErrDeadlineExceeded}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetLatest("x"); ok {
		t.Fatal("errored lookups must not be cached")
	}
}
