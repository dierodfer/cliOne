package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/dierodfer6/cliOne/internal/model"
)

// row is one selectable line of the custom collapsible tree: either a
// category header or a tool.
type row struct {
	isCategory bool
	catID      string
	toolID     string
	tool       model.ToolState
	cat        model.CategoryState
}

// visibleRows computes the tree rows for the current data, profile filter,
// text filter, and expansion state. Categories are collapsed by default;
// while a text filter is active, matching categories are forced open so hits
// are visible.
func (a *App) visibleRows() []row {
	var rows []row
	for _, cs := range a.filteredCats() {
		rows = append(rows, row{isCategory: true, catID: cs.Category.ID, cat: cs})
		open := a.expanded[cs.Category.ID] || a.filterText() != ""
		if !open {
			continue
		}
		for _, ts := range cs.Tools {
			rows = append(rows, row{catID: cs.Category.ID, toolID: ts.Def.ID, tool: ts})
		}
	}
	return rows
}

// renderTree draws the tree with the cursor on rows[a.cursor], including any
// expanded inline error panels. When the tree is taller than the terminal it
// renders only the scrolled window rows[a.scroll : a.scroll+h].
func (a *App) renderTree(rows []row) string {
	var b strings.Builder
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render("  no tools match the current filters"))
		b.WriteByte('\n')
		return b.String()
	}

	start, end := 0, len(rows)
	if h := a.treeBodyHeight(); h > 0 && len(rows) > h {
		start = a.scroll
		end = start + h
		if end > len(rows) {
			end = len(rows)
			start = end - h
		}
		if start < 0 {
			start = 0
		}
	}

	inner := a.innerWidth()
	for i := start; i < end; i++ {
		r := rows[i]
		line := a.renderRow(r)
		if inner > 0 {
			line = ansi.Truncate(line, inner, "…")
		}
		if i == a.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if !r.isCategory && a.errOpen[r.toolID] {
			if panel := a.renderErrorPanel(r.toolID); panel != "" {
				b.WriteString(panel)
				b.WriteByte('\n')
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (a *App) renderRow(r row) string {
	if r.isCategory {
		arrow := "▸"
		if a.expanded[r.catID] || a.filterText() != "" {
			arrow = "▾"
		}
		icon := categoryIcon(r.cat.Category.ID)
		name := categoryStyle.Render(r.cat.Category.Name)
		pill := pillStyle.Render(fmt.Sprintf("%d/%d", r.cat.Installed, r.cat.Total))
		return fmt.Sprintf(" %s %s %s %s", arrow, icon, name, pill)
	}
	return toolIndent + a.renderToolLine(r.tool)
}

func (a *App) renderToolLine(ts model.ToolState) string {
	icon := statusIcon(ts.Status)
	if a.updating[ts.Def.ID] {
		icon = a.spinner.View()
	}

	name := fmt.Sprintf("%-22s", ts.Def.Name)
	var parts []string
	parts = append(parts, icon, name)

	switch {
	case !ts.Detect.Installed:
		parts = append(parts, dimStyle.Render("not installed  "+ts.Def.OfficialURL))
	default:
		v := ts.Detect.Version
		if v == "" {
			v = "?"
		}
		parts = append(parts, versionStyle.Render("v"+v))
		if ts.Status == model.StatusUpdateAvail && ts.Latest.Latest != "" {
			parts = append(parts, latestStyle.Render("→ v"+ts.Latest.Latest))
		}
		if ts.Source.Kind != model.SourceUnknown {
			parts = append(parts, sourceStyle.Render("["+ts.Source.Kind.String()+"]"))
		}
	}

	if _, failed := a.updateErrs[ts.Def.ID]; failed {
		parts = append(parts, errStyle.Render("✗ update failed (l: log)"))
	}
	return strings.Join(parts, " ")
}

// toggleExpand flips a category open/closed.
func (a *App) toggleExpand(catID string) {
	a.expanded[catID] = !a.expanded[catID]
}

// clampCursor keeps the cursor inside the visible row range after any change
// to filters, profiles, or expansion, and scrolls the viewport to follow it.
func (a *App) clampCursor(rows []row) {
	if a.cursor >= len(rows) {
		a.cursor = len(rows) - 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	a.ensureVisible(rows)
}
