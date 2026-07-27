package tui

import (
	"strconv"
	"strings"
)

// maxPanelLines bounds the stderr tail shown in the inline accordion panel.
const maxPanelLines = 10

// toggleErrorPanel handles `l` on a row: it expands or collapses the inline
// error panel under a row whose last update failed, without leaving the tree.
func (a *App) toggleErrorPanel(toolID string) {
	if _, ok := a.updateErrs[toolID]; !ok {
		a.status = "no error log for this row"
		return
	}
	a.errOpen[toolID] = !a.errOpen[toolID]
}

// renderErrorPanel renders the accordion body: the tail of the failed update's
// stderr (falling back to stdout, then the Go error).
func (a *App) renderErrorPanel(toolID string) string {
	res, ok := a.updateErrs[toolID]
	if !ok {
		return ""
	}
	body := strings.TrimSpace(res.Stderr)
	if body == "" {
		body = strings.TrimSpace(res.Stdout)
	}
	if body == "" && res.Err != nil {
		body = res.Err.Error()
	}
	if body == "" {
		body = "(no output captured)"
	}
	lines := strings.Split(body, "\n")
	if len(lines) > maxPanelLines {
		lines = lines[len(lines)-maxPanelLines:]
	}
	header := "┌ error log (exit " + strconv.Itoa(res.ExitCode) + ") — last " + strconv.Itoa(len(lines)) + " line(s)"
	out := []string{panelStyle.Render(header)}
	for _, l := range lines {
		out = append(out, panelStyle.Render("│ "+l))
	}
	out = append(out, panelStyle.Render("└ press l to close"))
	return strings.Join(out, "\n")
}
