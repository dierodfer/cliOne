package source

import (
	"os"
	"path/filepath"
)

// ScanPath walks every directory in pathEnv (a $PATH-style list) and returns
// each location where binName exists as an executable regular file, in PATH
// order. The first entry is the active binary. This feeds the doctor view and
// SourceResult.AllPaths.
func ScanPath(binName, pathEnv string) []string {
	var out []string
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		p := filepath.Join(dir, binName)
		if seen[p] {
			continue
		}
		info, err := os.Stat(p) // follows symlinks: broken links are skipped
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
