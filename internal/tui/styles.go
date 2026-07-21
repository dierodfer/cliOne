package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/dierodfer6/cliOne/internal/model"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	brandDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	toolCountStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("80"))
	navHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	categoryStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	pillStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("189")).Background(lipgloss.Color("238")).Padding(0, 1)
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	versionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	latestStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sourceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("60"))
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	panelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).PaddingLeft(6)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	statusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Padding(0, 1)
	doctorHeading  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activePath     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	shadowedPath   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Rounded panel around the tree.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	// Footer key hint: a highlighted key chip followed by its label.
	keyChipStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("238")).Padding(0, 1)
	keyLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// Status-dot colors, reused for the footer summary.
	dotGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dotRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	// Tree indentation.
	toolIndent = "   "
)

// statusIcon renders the 4-state semaphore for a tool row as a colored dot.
func statusIcon(s model.StatusState) string {
	switch s {
	case model.StatusUpToDate:
		return dotGreen.Render("●")
	case model.StatusUpdateAvail:
		return latestStyle.Render("●")
	case model.StatusNotInstalled:
		return dotRed.Render("●")
	case model.StatusLatestUnknown:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("●")
	default:
		return " "
	}
}
