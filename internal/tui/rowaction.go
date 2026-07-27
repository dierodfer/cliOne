package tui

import (
	"context"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/updater"
)

// actOnTool handles enter/u on a tool row. The action is chosen by
// updater.Decide, so a row is updated only when a real update command backs it:
//   - runnable update (native or manager): run it async with a row spinner;
//     resolve to green or an inline error state when it finishes
//   - otherwise (not installed, latest unknown, or update-available with no
//     updater): open the official page in the OS browser
//   - up to date: nothing
func (a *App) actOnTool(ts model.ToolState) tea.Cmd {
	if a.updating[ts.Def.ID] {
		return nil // an update is already running on this row
	}
	action := updater.Decide(ts.Def, ts.Source, ts.Status, a.scanner.Registry)
	switch action {
	case model.ActionRunNativeUpdate, model.ActionRunManagerUpdate:
		a.updating[ts.Def.ID] = true
		delete(a.updateErrs, ts.Def.ID)
		a.errOpen[ts.Def.ID] = false
		def, src := ts.Def, ts.Source
		return tea.Batch(
			a.spinner.Tick,
			func() tea.Msg {
				return updateDoneMsg{toolID: def.ID, result: updater.Execute(context.Background(), def, src)}
			},
		)
	case model.ActionOpenOfficialPage:
		url := ts.Def.OfficialURL
		id := ts.Def.ID
		return func() tea.Msg {
			return openURLDoneMsg{toolID: id, err: openBrowser(url)}
		}
	default:
		return nil
	}
}

// openBrowser opens a URL with the platform opener.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// applyUpdateResult resolves a finished update on a row.
func (a *App) applyUpdateResult(msg updateDoneMsg) tea.Cmd {
	delete(a.updating, msg.toolID)
	if !msg.result.Success {
		a.updateErrs[msg.toolID] = msg.result
		a.status = "update failed for " + msg.toolID + " (press l on the row for the log)"
		return nil
	}
	a.status = "updated " + msg.toolID
	// Re-detect to pick up the new version and recompute the row status.
	if ts, ok := a.findTool(msg.toolID); ok {
		def := ts.Def
		return func() tea.Msg {
			res, _ := detectRun(def)
			return detectDoneMsg{toolID: def.ID, result: res}
		}
	}
	return nil
}
