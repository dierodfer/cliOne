// Package registry provides per-manager adapters that resolve the latest
// available version of a package and synthesize the manager's update command.
package registry

import (
	"context"
	"strings"

	"github.com/dierodfer6/cliOne/internal/model"
)

// Manager is a package-manager adapter. New managers can be added without
// touching calling code: implement the interface and register it.
type Manager interface {
	Kind() model.SourceKind
	LatestVersion(ctx context.Context, pkgName string) (string, error)
	UpdateCommand(pkgName string) []string
}

// Registry maps a binary's SourceKind to the manager that owns it.
type Registry map[model.SourceKind]Manager

// Default returns the built-in manager set.
func Default() Registry {
	r := Registry{}
	for _, m := range []Manager{NewBrew(), NewCargo(), NewNpm(), NewPyPI()} {
		r[m.Kind()] = m
	}
	return r
}

// ForKind returns the manager for a source kind, if one exists.
func (r Registry) ForKind(k model.SourceKind) (Manager, bool) {
	m, ok := r[k]
	return m, ok
}

// normalizeVersion strips a leading "v" from version strings/tags so
// comparisons against detected versions are consistent.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}
