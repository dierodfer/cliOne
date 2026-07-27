package source

import (
	"path/filepath"
	"strings"

	"github.com/dierodfer6/cliOne/internal/model"
)

// Prefix maps a filesystem path prefix to the package manager that owns
// binaries living under it.
type Prefix struct {
	Path string
	Kind model.SourceKind
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
	}
}

// ClassifyPath matches a fully symlink-resolved binary path against the prefix
// table. Paths owned by no known manager come back as SourceManual; system
// paths like /usr/bin need the dpkg confirmation step in Resolve to be
// promoted to SourceAptDnf.
func ClassifyPath(resolved string, prefixes []Prefix) model.SourceKind {
	for _, p := range prefixes {
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
