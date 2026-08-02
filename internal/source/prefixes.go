package source

import (
	"path/filepath"
	"strings"

	"github.com/dierodfer/cliOne/internal/model"
)

// Prefix maps a filesystem path fragment to the package manager that owns
// binaries living under it. By default the fragment must be a path prefix;
// set Contains for fragments that appear mid-path with a variable segment
// before them (e.g. an nvm-managed Node version directory).
type Prefix struct {
	Path     string
	Kind     model.SourceKind
	Contains bool
}

// DefaultPrefixes returns the manager path-prefix table for the given home
// directory.
func DefaultPrefixes(home string) []Prefix {
	return []Prefix{
		{Path: "/opt/homebrew/Cellar/", Kind: model.SourceHomebrew},
		{Path: "/usr/local/Cellar/", Kind: model.SourceHomebrew},
		{Path: filepath.Join(home, ".cargo", "bin") + "/", Kind: model.SourceCargo},
		{Path: filepath.Join(home, ".local", "share", "uv", "tools") + "/", Kind: model.SourceUvTool},
		{Path: filepath.Join(home, ".npm-global") + "/", Kind: model.SourceNpmGlobal},
		{Path: "/usr/lib/node_modules/", Kind: model.SourceNpmGlobal},
		// Homebrew's own npm prefix for `npm install -g`, distinct from the
		// Cellar path used for the `node` formula itself.
		{Path: "/opt/homebrew/lib/node_modules/", Kind: model.SourceNpmGlobal},
		{Path: "/usr/local/lib/node_modules/", Kind: model.SourceNpmGlobal},
		// nvm installs each Node version under a variable version segment
		// (~/.nvm/versions/node/vX.Y.Z/...), so this fragment can't be
		// matched as a fixed prefix.
		{Path: filepath.Join(home, ".nvm", "versions", "node") + "/", Kind: model.SourceNpmGlobal, Contains: true},
		// asdf puts a shim on $PATH rather than the real binary. Shims are
		// shell scripts, not symlinks, so path resolution stops at the shim
		// directory and never reaches ~/.asdf/installs — both are matched so
		// either entry point classifies correctly.
		{Path: filepath.Join(home, ".asdf", "shims") + "/", Kind: model.SourceAsdf},
		{Path: filepath.Join(home, ".asdf", "installs") + "/", Kind: model.SourceAsdf},
	}
}

// ClassifyPath matches a fully symlink-resolved binary path against the prefix
// table. Paths owned by no known manager come back as SourceManual; system
// paths like /usr/bin need the dpkg confirmation step in Resolve to be
// promoted to SourceAptDnf.
func ClassifyPath(resolved string, prefixes []Prefix) model.SourceKind {
	for _, p := range prefixes {
		if p.Contains {
			if strings.Contains(resolved, p.Path) {
				return p.Kind
			}
			continue
		}
		if strings.HasPrefix(resolved, p.Path) {
			return p.Kind
		}
	}
	return model.SourceManual
}

// isAmbiguousSystemPath reports whether the path is in a system location where
// both distro packages and manual installs commonly live, so distro-package
// ownership must be confirmed via dpkg.
func isAmbiguousSystemPath(resolved string) bool {
	return strings.HasPrefix(resolved, "/usr/bin/")
}
