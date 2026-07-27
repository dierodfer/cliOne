package scan

import "github.com/dierodfer6/cliOne/internal/model"

// Conflict is one tool whose binary resolves at more than one location on
// $PATH.
type Conflict struct {
	Tool     model.ToolDef
	AllPaths []string // AllPaths[0] is active (first on $PATH); the rest are shadowed
}

// Conflicts returns every tool in cats that resolves at more than one
// location on $PATH, in catalog order. Shared by the TUI's doctor view and
// the `clione doctor` CLI command so both render the same underlying data.
func Conflicts(cats []model.CategoryState) []Conflict {
	var out []Conflict
	for _, cs := range cats {
		for _, ts := range cs.Tools {
			if len(ts.Source.AllPaths) <= 1 {
				continue
			}
			out = append(out, Conflict{Tool: ts.Def, AllPaths: ts.Source.AllPaths})
		}
	}
	return out
}
