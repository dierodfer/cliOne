package tui

import (
	"strings"
)

// renderDoctor renders the read-only doctor view: every tool whose binary
// resolves at more than one location on $PATH, with the active one (first on
// $PATH) marked.
func (a *App) renderDoctor() string {
	var b strings.Builder
	b.WriteString(doctorHeading.Render(" Doctor — PATH conflicts"))
	b.WriteString("\n\n")

	found := false
	for _, cs := range a.cats {
		for _, ts := range cs.Tools {
			if len(ts.Source.AllPaths) <= 1 {
				continue
			}
			found = true
			b.WriteString(" " + categoryStyle.Render(ts.Def.Name) + "\n")
			for i, p := range ts.Source.AllPaths {
				if i == 0 {
					b.WriteString("   " + activePath.Render("● "+p+"  (active — first on $PATH)") + "\n")
				} else {
					b.WriteString("   " + shadowedPath.Render("○ "+p+"  (shadowed)") + "\n")
				}
			}
			b.WriteString("\n")
		}
	}
	if !found {
		b.WriteString(dimStyle.Render("  no PATH conflicts detected: every tool resolves at a single location"))
		b.WriteString("\n")
	}
	b.WriteString("\n" + helpStyle.Render("d/esc/q back"))
	return b.String()
}
