package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

// sourceEntry and latestEntry are the on-disk shapes. Errors are never
// persisted: only successful lookups are worth caching.
type sourceEntry struct {
	Kind     model.SourceKind `json:"kind"`
	BinPath  string           `json:"bin_path"`
	AllPaths []string         `json:"all_paths,omitempty"`
	StoredAt time.Time        `json:"stored_at"`
}

type latestEntry struct {
	Latest    string    `json:"latest"`
	FetchedAt time.Time `json:"fetched_at"`
}

type fileData struct {
	Sources map[string]sourceEntry `json:"sources"`
	Latest  map[string]latestEntry `json:"latest"`
}

// JSONStore is a Cache backed by a single JSON file, written atomically via
// temp-file+rename. A missing or corrupt file never crashes anything: the
// store just starts empty.
type JSONStore struct {
	path      string
	sourceTTL time.Duration
	latestTTL time.Duration
	nowFunc   func() time.Time

	mu   sync.Mutex
	data fileData
}

var _ Cache = (*JSONStore)(nil)

// DefaultPath returns os.UserCacheDir()/clione/cache.json.
func DefaultPath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "clione", "cache.json"), nil
}

// NewJSONStore opens (or lazily creates) the store at path with default TTLs.
func NewJSONStore(path string) *JSONStore {
	s := &JSONStore{
		path:      path,
		sourceTTL: DefaultSourceTTL,
		latestTTL: DefaultLatestTTL,
		nowFunc:   time.Now,
		data:      fileData{Sources: map[string]sourceEntry{}, Latest: map[string]latestEntry{}},
	}
	s.load()
	return s
}

// load reads the file if present. Corruption or absence leaves the store empty.
func (s *JSONStore) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var d fileData
	if err := json.Unmarshal(raw, &d); err != nil {
		return // corrupt file: start empty rather than crash
	}
	if d.Sources == nil {
		d.Sources = map[string]sourceEntry{}
	}
	if d.Latest == nil {
		d.Latest = map[string]latestEntry{}
	}
	s.data = d
}

// save writes the file atomically (temp file in the same directory + rename).
func (s *JSONStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".cache-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path)
}

func (s *JSONStore) GetSource(toolID string) (model.SourceResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data.Sources[toolID]
	if !ok || s.nowFunc().Sub(e.StoredAt) > s.sourceTTL {
		return model.SourceResult{}, false
	}
	return model.SourceResult{Kind: e.Kind, BinPath: e.BinPath, AllPaths: e.AllPaths}, true
}

func (s *JSONStore) SetSource(toolID string, v model.SourceResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Sources[toolID] = sourceEntry{
		Kind: v.Kind, BinPath: v.BinPath, AllPaths: v.AllPaths, StoredAt: s.nowFunc(),
	}
	return s.save()
}

func (s *JSONStore) GetLatest(toolID string) (model.VersionResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data.Latest[toolID]
	if !ok || s.nowFunc().Sub(e.FetchedAt) > s.latestTTL {
		return model.VersionResult{}, false
	}
	return model.VersionResult{Latest: e.Latest, FetchedAt: e.FetchedAt}, true
}

func (s *JSONStore) SetLatest(toolID string, v model.VersionResult) error {
	if v.Err != nil || v.Latest == "" {
		return nil // never cache failures
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at := v.FetchedAt
	if at.IsZero() {
		at = s.nowFunc()
	}
	s.data.Latest[toolID] = latestEntry{Latest: v.Latest, FetchedAt: at}
	return s.save()
}
