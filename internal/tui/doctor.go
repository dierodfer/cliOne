package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// doctorLines builds the doctor view content as individual lines so it can be
// windowed the same way the tree is: every tool whose binary resolves at more
// than one location on $PATH, with the active one (first on $PATH) marked.
func (a *App) doctorLines() []string {
	lines := []string{doctorHeading.Render(" Doctor — PATH conflicts"), ""}

	found := false
	for _, cs := range a.cats {
		for _, ts := range cs.Tools {
			if len(ts.Source.AllPaths) <= 1 {
				continue
			}
			found = true
			lines = append(lines, " "+categoryStyle.Render(ts.Def.Name))
			for i, p := range ts.Source.AllPaths {
				if i == 0 {
					lines = append(lines, "   "+activePath.Render("● "+p+"  (active — first on $PATH)"))
				} else {
					lines = append(lines, "   "+shadowedPath.Render("○ "+p+"  (shadowed)"))
				}
			}
			lines = append(lines, "")
		}
	}
	if !found {
		lines = append(lines, dimStyle.Render("  no PATH conflicts detected: every tool resolves at a single location"))
	}
	return lines
}

// clampDoctorScroll keeps the doctor viewport's scroll offset within range
// after the terminal is resized or the underlying data changes.
func (a *App) clampDoctorScroll() {
	h := a.treeBodyHeight()
	lines := a.doctorLines()
	if h <= 0 || len(lines) <= h {
		a.doctorScroll = 0
		return
	}
	if max := len(lines) - h; a.doctorScroll > max {
		a.doctorScroll = max
	}
	if a.doctorScroll < 0 {
		a.doctorScroll = 0
	}
}

// renderDoctor renders the (possibly windowed) doctor view body.
func (a *App) renderDoctor() string {
	lines := a.doctorLines()

	start, end := 0, len(lines)
	if h := a.treeBodyHeight(); h > 0 && len(lines) > h {
		start = a.doctorScroll
		end = start + h
		if end > len(lines) {
			end = len(lines)
			start = end - h
		}
		if start < 0 {
			start = 0
		}
	}

	inner := a.innerWidth()
	visible := make([]string, 0, end-start)
	for _, l := range lines[start:end] {
		if inner > 0 {
			l = ansi.Truncate(l, inner, "…")
		}
		visible = append(visible, l)
	}
	return strings.Join(visible, "\n")
}
