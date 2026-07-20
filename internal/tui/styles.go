package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/dierodfer6/cliOne/internal/model"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	categoryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	countStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	versionStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	latestStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sourceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("60"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	panelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).PaddingLeft(6)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Padding(0, 1)
	doctorHeading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activePath    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	shadowedPath  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Tree indentation.
	toolIndent = "   "
)

// statusIcon renders the 4-state semaphore for a tool row.
func statusIcon(s model.StatusState) string {
	switch s {
	case model.StatusUpToDate:
		return "🟢"
	case model.StatusUpdateAvail:
		return "🟡"
	case model.StatusNotInstalled:
		return "🔴"
	case model.StatusNoUpdater:
		return "⚪"
	default:
		return "  "
	}
}
