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
	cats := a.byProfile(a.cats)

	q := a.filterText()
	if q == "" {
		return cats
	}

	var out []model.CategoryState
	for _, cs := range cats {
		if matched, ok := matchCategory(q, cs); ok {
			out = append(out, matched)
		}
	}
	return out
}

// byProfile keeps only the categories the active profile includes. With no
// active profile (the "All" pseudo-profile) every category survives.
func (a *App) byProfile(cats []model.CategoryState) []model.CategoryState {
	prof, ok := a.activeProfile()
	if !ok {
		return cats
	}
	included := make(map[string]bool, len(prof.Categories))
	for _, id := range prof.Categories {
		included[id] = true
	}
	var kept []model.CategoryState
	for _, cs := range cats {
		if included[cs.Category.ID] {
			kept = append(kept, cs)
		}
	}
	return kept
}

// matchCategory applies the text filter to one category. A category whose own
// name matches is kept whole; otherwise it survives only if some of its tools
// match, and then it carries just those tools with recomputed counts.
func matchCategory(q string, cs model.CategoryState) (model.CategoryState, bool) {
	if fuzzyMatch(q, cs.Category.Name) {
		return cs, true
	}
	var tools []model.ToolState
	for _, ts := range cs.Tools {
		if fuzzyMatch(q, ts.Def.Name) || fuzzyMatch(q, ts.Def.ID) {
			tools = append(tools, ts)
		}
	}
	if len(tools) == 0 {
		return model.CategoryState{}, false
	}
	cs.Tools = tools
	cs.Total = len(tools)
	cs.Installed = 0
	for _, ts := range tools {
		if ts.Detect.Installed {
			cs.Installed++
		}
	}
	return cs, true
}
