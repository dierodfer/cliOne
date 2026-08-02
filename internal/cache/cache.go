// Package cache persists source-resolution and latest-version lookups between
// runs so the TUI can render instantly from cached data.
package cache

import (
	"time"

	"github.com/dierodfer/cliOne/internal/model"
)

// Default TTLs. Source ownership changes rarely (7 days); latest versions go
// stale quickly (12 hours, within the 6-24h design band).
const (
	DefaultSourceTTL = 7 * 24 * time.Hour
	DefaultLatestTTL = 12 * time.Hour
)

// Cache stores per-tool source and latest-version results under two
// independent TTLs. A miss (expired or absent) returns ok=false.
type Cache interface {
	GetSource(toolID string) (model.SourceResult, bool)
	SetSource(toolID string, v model.SourceResult) error
	GetLatest(toolID string) (model.VersionResult, bool) // honors the short TTL internally
	SetLatest(toolID string, v model.VersionResult) error
}
