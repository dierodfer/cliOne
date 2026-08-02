package tui

import (
	"fmt"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dierodfer/cliOne/internal/model"
)

// actOnTool handles enter on a tool row by opening the tool's official page in
// the OS browser. CLIOne reports versions but never installs or upgrades
// anything itself, so acting on a row is always just navigation.
func (a *App) actOnTool(ts model.ToolState) tea.Cmd {
	if ts.Def.OfficialURL == "" {
		return nil
	}
	url, id := ts.Def.OfficialURL, ts.Def.ID
	return func() tea.Msg {
		return openURLDoneMsg{toolID: id, err: openBrowser(url)}
	}
}

// openBrowser opens a URL with the platform opener. The opener is resolved to
// an absolute path first so the command run is decided by $PATH lookup at a
// single known point rather than implicitly at exec time.
func openBrowser(url string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	bin, err := exec.LookPath(opener)
	if err != nil {
		return fmt.Errorf("no %s on PATH: %w", opener, err)
	}
	return exec.Command(bin, url).Start()
}
