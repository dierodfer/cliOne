// Package registry provides per-manager adapters that resolve the latest
// available version of a package.
package registry

import (
	"context"
	"regexp"
	"strings"

	"github.com/dierodfer6/cliOne/internal/model"
)

// Manager is a package-manager adapter. New managers can be added without
// touching calling code: implement the interface and register it.
type Manager interface {
	Kind() model.SourceKind
	LatestVersion(ctx context.Context, pkgName string) (string, error)
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

// versionInTag matches the first dotted-numeric run in a release tag.
var versionInTag = regexp.MustCompile(`\d+(?:\.\d+)*`)

// normalizeVersion reduces a release tag to the bare version it carries, so it
// can be compared against a detected version. Projects tag releases in their
// own style — "v1.2.3", "jq-1.8.2", "azure-cli-2.88.0", "Helm v4.2.3" — and a
// tag left with its prefix would fail numeric comparison and silently read as
// "up to date".
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if m := versionInTag.FindString(v); m != "" {
		return m
	}
	return strings.TrimPrefix(v, "v")
}
