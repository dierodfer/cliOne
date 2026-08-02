package tui

import "github.com/dierodfer/cliOne/internal/model"

// activeProfile returns the currently selected profile, or ok=false when the
// selection is "All" (no profile filtering).
func (a *App) activeProfile() (model.Profile, bool) {
	if a.profileIdx < 0 || a.profileIdx >= len(a.profiles) {
		return model.Profile{}, false
	}
	return a.profiles[a.profileIdx], true
}

// profileName is the label shown in the header.
func (a *App) profileName() string {
	if p, ok := a.activeProfile(); ok {
		return p.Name
	}
	return "All"
}

// cycleProfile advances All -> profile1 -> ... -> profileN -> All. It is
// purely a view filter: no actions change, and it combines with the `/` text
// filter.
func (a *App) cycleProfile() {
	a.profileIdx++
	if a.profileIdx >= len(a.profiles) {
		a.profileIdx = -1
	}
}
