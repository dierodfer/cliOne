package tui

import (
	"strings"

	"github.com/dierodfer6/cliOne/internal/model"
)

// filterText returns the active text filter (typed after `/`).
func (a *App) filterText() string { return a.filter }

// fuzzyMatch reports whether every rune of pattern appears in s in order
// (case-insensitive subsequence match).
func fuzzyMatch(pattern, s string) bool {
	p := []rune(strings.ToLower(pattern))
	i := 0
	for _, r := range strings.ToLower(s) {
		if i >= len(p) {
			return true
		}
		if p[i] == r {
			i++
		}
	}
	return i >= len(p)
}

// filteredCats applies the profile filter, then the fuzzy text filter, to the
// scanned category states. A category survives the text filter if any of its
// tools (or its own name) matches; only matching tools are kept.
func (a *App) filteredCats() []model.CategoryState {
	cats := a.cats

	if prof, ok := a.activeProfile(); ok {
		included := map[string]bool{}
		for _, id := range prof.Categories {
			included[id] = true
		}
		var kept []model.CategoryState
		for _, cs := range cats {
			if included[cs.Category.ID] {
				kept = append(kept, cs)
			}
		}
		cats = kept
	}

	q := a.filterText()
	if q == "" {
		return cats
	}

	var out []model.CategoryState
	for _, cs := range cats {
		if fuzzyMatch(q, cs.Category.Name) {
			out = append(out, cs)
			continue
		}
		var tools []model.ToolState
		for _, ts := range cs.Tools {
			if fuzzyMatch(q, ts.Def.Name) || fuzzyMatch(q, ts.Def.ID) {
				tools = append(tools, ts)
			}
		}
		if len(tools) > 0 {
			filtered := cs
			filtered.Tools = tools
			filtered.Installed = 0
			for _, ts := range tools {
				if ts.Detect.Installed {
					filtered.Installed++
				}
			}
			filtered.Total = len(tools)
			out = append(out, filtered)
		}
	}
	return out
}
