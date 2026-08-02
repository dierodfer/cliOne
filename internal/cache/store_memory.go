package cache

import (
	"sync"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

// MemoryStore is a process-local Cache with the same TTL semantics as
// JSONStore but no file behind it. It is the fallback when no per-user cache
// directory can be determined: results are still reused within a run, and
// nothing is written to a shared, world-writable location.
type MemoryStore struct {
	sourceTTL time.Duration
	latestTTL time.Duration
	nowFunc   func() time.Time

	mu      sync.Mutex
	sources map[string]sourceEntry
	latest  map[string]latestEntry
}

var (
	_ Cache = (*MemoryStore)(nil)
)

// NewMemoryStore builds an empty in-memory cache with the default TTLs.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sourceTTL: DefaultSourceTTL,
		latestTTL: DefaultLatestTTL,
		nowFunc:   time.Now,
		sources:   map[string]sourceEntry{},
		latest:    map[string]latestEntry{},
	}
}

func (m *MemoryStore) GetSource(toolID string) (model.SourceResult, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.sources[toolID]
	if !ok || m.nowFunc().Sub(e.StoredAt) > m.sourceTTL {
		return model.SourceResult{}, false
	}
	return model.SourceResult{Kind: e.Kind, BinPath: e.BinPath, AllPaths: e.AllPaths}, true
}

func (m *MemoryStore) SetSource(toolID string, v model.SourceResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setSourceLocked(toolID, v)
	return nil
}

// SetSources stores several sources at once, mirroring the batched write the
// JSON store uses so scan.Scanner takes the same path for either cache.
func (m *MemoryStore) SetSources(vs map[string]model.SourceResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, v := range vs {
		m.setSourceLocked(id, v)
	}
	return nil
}

func (m *MemoryStore) setSourceLocked(toolID string, v model.SourceResult) {
	m.sources[toolID] = sourceEntry{
		Kind:     v.Kind,
		BinPath:  v.BinPath,
		AllPaths: v.AllPaths,
		StoredAt: m.nowFunc(),
	}
}

func (m *MemoryStore) GetLatest(toolID string) (model.VersionResult, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.latest[toolID]
	if !ok || m.nowFunc().Sub(e.FetchedAt) > m.latestTTL {
		return model.VersionResult{}, false
	}
	return model.VersionResult{Latest: e.Latest, FetchedAt: e.FetchedAt}, true
}

func (m *MemoryStore) SetLatest(toolID string, v model.VersionResult) error {
	if v.Err != nil || v.Latest == "" {
		return nil // never cache failures, same as the JSON store
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	at := v.FetchedAt
	if at.IsZero() {
		at = m.nowFunc()
	}
	m.latest[toolID] = latestEntry{Latest: v.Latest, FetchedAt: at}
	return nil
}
