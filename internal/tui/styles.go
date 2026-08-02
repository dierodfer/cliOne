package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/dierodfer/cliOne/internal/model"
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

// categoryIcons maps a catalog category ID to a portable emoji glyph. These
// are plain Unicode emoji (not Nerd Font icons) so they render in any
// terminal/font without extra setup; unknown category IDs fall back to a
// generic folder glyph in categoryIcon below.
var categoryIcons = map[string]string{
	"ai":               "🤖",
	"languages":        "💻",
	"package_managers": "📦",
	"git":              "🔀",
	"kubernetes":       "☸️",
	"utilities":        "🛠️",
	"containers":       "🐳",
	"cloud":            "☁️",
	"infrastructure":   "🏗️",
	"editors":          "📝",
}

// categoryIcon returns the portable emoji for a category ID, falling back to
// a generic glyph for any category the catalog adds later without an entry
// in categoryIcons.
func categoryIcon(catID string) string {
	if icon, ok := categoryIcons[catID]; ok {
		return icon
	}
	return "📁"
}

// sourceKindStyles gives each package manager its own color so a row's owner
// is readable at a glance and matches the legend under the tree. The colors
// deliberately avoid those already carrying meaning elsewhere: 111 (version),
// 214 (update-available), 42/203 (status dots), 196 (error), 241/245 (dim).
var sourceKindStyles = map[model.SourceKind]lipgloss.Style{
	model.SourceHomebrew:  lipgloss.NewStyle().Foreground(lipgloss.Color("178")), // amber
	model.SourceCargo:     lipgloss.NewStyle().Foreground(lipgloss.Color("173")), // rust
	model.SourceUvTool:    lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // violet
	model.SourceNpmGlobal: lipgloss.NewStyle().Foreground(lipgloss.Color("168")), // rose
	model.SourceAptDnf:    lipgloss.NewStyle().Foreground(lipgloss.Color("73")),  // teal
	model.SourceAsdf:      lipgloss.NewStyle().Foreground(lipgloss.Color("108")), // sage
	model.SourceManual:    lipgloss.NewStyle().Foreground(lipgloss.Color("103")), // slate
}

// legendKinds is the fixed display order for the source legend, so the row of
// colors stays stable between renders rather than following map iteration.
var legendKinds = []model.SourceKind{
	model.SourceHomebrew,
	model.SourceCargo,
	model.SourceUvTool,
	model.SourceNpmGlobal,
	model.SourceAptDnf,
	model.SourceAsdf,
	model.SourceManual,
}

// sourceKindStyle returns the color for a package manager, falling back to the
// muted slate used for manual installs for any kind without its own entry.
func sourceKindStyle(k model.SourceKind) lipgloss.Style {
	if s, ok := sourceKindStyles[k]; ok {
		return s
	}
	return sourceKindStyles[model.SourceManual]
}

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
