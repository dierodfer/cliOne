// Package source determines which package manager owns an installed binary.
package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dierodfer6/cliOne/internal/model"
)

// Resolver classifies binaries by owner. The zero value is not usable; call
// NewResolver, or construct one directly in tests with custom fields.
type Resolver struct {
	Prefixes []Prefix
	// PathEnv is the $PATH-style list to scan. Empty means use os.Getenv("PATH").
	PathEnv string
	// DpkgOwns reports whether the distro package database owns the path.
	// Nil means use the real `dpkg -S` query (a no-op on macOS or when dpkg
	// is not installed).
	DpkgOwns func(ctx context.Context, path string) bool
}

// NewResolver builds a Resolver with the default prefix table for the current
// user's home directory.
func NewResolver() *Resolver {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &Resolver{Prefixes: DefaultPrefixes(home)}
}

// Resolve finds binName on $PATH, resolves symlinks, and classifies the active
// binary's owner. A binary that is not on PATH at all yields SourceUnknown
// with no paths.
func (r *Resolver) Resolve(ctx context.Context, binName string) (model.SourceResult, error) {
	pathEnv := r.PathEnv
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	all := ScanPath(binName, pathEnv)
	if len(all) == 0 {
		return model.SourceResult{Kind: model.SourceUnknown}, nil
	}

	active := all[0]
	resolved, err := filepath.EvalSymlinks(active)
	if err != nil {
		resolved = active
	}

	kind := ClassifyPath(resolved, r.Prefixes)
	if kind == model.SourceManual && isAmbiguousSystemPath(resolved) {
		owns := r.DpkgOwns
		if owns == nil {
			owns = dpkgOwns
		}
		if owns(ctx, resolved) {
			kind = model.SourceAptDnf
		}
	}

	return model.SourceResult{Kind: kind, BinPath: resolved, AllPaths: all}, nil
}

// dpkgOwns asks the Debian package database whether it owns path. It is a
// no-op (false) on macOS and on systems without dpkg on PATH.
func dpkgOwns(ctx context.Context, path string) bool {
	if runtime.GOOS == "darwin" {
		return false
	}
	// Run the absolute path LookPath resolved rather than re-resolving "dpkg"
	// through $PATH inside exec.
	bin, err := exec.LookPath("dpkg")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, bin, "-S", path).Run() == nil
}
