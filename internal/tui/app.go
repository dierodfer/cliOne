// Package tui implements the CLIOne terminal user interface: a collapsible
// tree of tool categories with per-row update actions, a doctor view, inline
// error panels, and profile/text filtering.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dierodfer6/cliOne/internal/detect"
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/scan"
)

type viewMode int

const (
	modeTree viewMode = iota
	modeDoctor
)

// App is the Bubble Tea model for the whole TUI.
type App struct {
	scanner  *scan.Scanner
	cats     []model.CategoryState
	profiles []model.Profile

	keys    keyMap
	spinner spinner.Model

	mode       viewMode
	cursor     int
	scroll     int             // index of the first tree row rendered (viewport top)
	expanded   map[string]bool // category ID -> open
	updating   map[string]bool // tool ID -> update running
	updateErrs map[string]model.UpdateResult
	errOpen    map[string]bool // tool ID -> error panel open

	filter     string
	filtering  bool // typing after `/`
	profileIdx int  // -1 = All

	scanned bool
	status  string
	width   int
	height  int
}

// Run starts the TUI and blocks until quit.
func Run() error {
	scanner, err := scan.New()
	if err != nil {
		return err
	}
	app := NewApp(scanner)
	_, err = tea.NewProgram(app, tea.WithAltScreen()).Run()
	return err
}

// NewApp builds the initial model. Exposed for tests.
func NewApp(scanner *scan.Scanner) *App {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	a := &App{
		scanner:    scanner,
		keys:       defaultKeyMap(),
		spinner:    sp,
		expanded:   map[string]bool{},
		updating:   map[string]bool{},
		updateErrs: map[string]model.UpdateResult{},
		errOpen:    map[string]bool{},
		profileIdx: -1,
	}
	if scanner != nil && scanner.Catalog != nil {
		a.profiles = scanner.Catalog.Profiles
	}
	return a
}

func (a *App) Init() tea.Cmd {
	return func() tea.Msg {
		cats, err := a.scanner.Scan(context.Background())
		return scanDoneMsg{cats: cats, err: err}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.clampCursor(a.visibleRows())
		return a, nil

	case scanDoneMsg:
		a.scanned = true
		if msg.err != nil {
			a.status = "scan failed: " + msg.err.Error()
			return a, nil
		}
		a.cats = msg.cats
		// Categories render expanded by default; only seed IDs the user has
		// not already toggled so a re-scan never reopens a collapsed one.
		for _, cs := range a.cats {
			if _, set := a.expanded[cs.Category.ID]; !set {
				a.expanded[cs.Category.ID] = true
			}
		}
		// Kick off async latest-version refreshes; cached rows render as-is.
		funcs := a.scanner.RefreshFuncs(context.Background(), a.cats)
		cmds := make([]tea.Cmd, 0, len(funcs))
		for _, f := range funcs {
			f := f
			cmds = append(cmds, func() tea.Msg { return versionDoneMsg(f()) })
		}
		return a, tea.Batch(cmds...)

	case versionDoneMsg:
		a.setLatest(msg.ToolID, msg.Result)
		return a, nil

	case detectDoneMsg:
		a.setDetect(msg.toolID, msg.result)
		return a, nil

	case updateDoneMsg:
		return a, a.applyUpdateResult(msg)

	case openURLDoneMsg:
		if msg.err != nil {
			a.status = "could not open browser: " + msg.err.Error()
		} else {
			a.status = "opened official page for " + msg.toolID
		}
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		if len(a.updating) == 0 {
			return a, nil // stop ticking when nothing is running
		}
		return a, cmd

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Text-filter input mode captures almost everything.
	if a.filtering {
		switch msg.String() {
		case "enter":
			a.filtering = false
		case "esc":
			a.filtering = false
			a.filter = ""
		case "backspace":
			if len(a.filter) > 0 {
				a.filter = a.filter[:len(a.filter)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				a.filter += string(msg.Runes)
			}
		}
		a.clampCursor(a.visibleRows())
		return a, nil
	}

	if a.mode == modeDoctor {
		switch {
		case key.Matches(msg, a.keys.Doctor), key.Matches(msg, a.keys.Back), key.Matches(msg, a.keys.Quit):
			if msg.String() == "ctrl+c" {
				return a, tea.Quit
			}
			a.mode = modeTree
		}
		return a, nil
	}

	rows := a.visibleRows()
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Up):
		a.cursor--
		a.clampCursor(rows)

	case key.Matches(msg, a.keys.Down):
		a.cursor++
		a.clampCursor(rows)

	case key.Matches(msg, a.keys.Enter), key.Matches(msg, a.keys.Update):
		if a.cursor < len(rows) {
			r := rows[a.cursor]
			if r.isCategory {
				a.toggleExpand(r.catID)
				a.clampCursor(a.visibleRows())
			} else {
				return a, a.actOnTool(r.tool)
			}
		}

	case key.Matches(msg, a.keys.Back):
		// A committed filter forces categories open, so backing out of the
		// filter takes precedence over collapsing the category under the cursor.
		if a.filter != "" {
			a.filter = ""
		} else if a.cursor < len(rows) {
			r := rows[a.cursor]
			if r.isCategory && a.expanded[r.catID] {
				a.toggleExpand(r.catID)
			}
		}
		a.clampCursor(a.visibleRows())

	case key.Matches(msg, a.keys.Filter):
		a.filtering = true

	case key.Matches(msg, a.keys.Log):
		if a.cursor < len(rows) && !rows[a.cursor].isCategory {
			a.toggleErrorPanel(rows[a.cursor].toolID)
		}

	case key.Matches(msg, a.keys.Doctor):
		a.mode = modeDoctor

	case key.Matches(msg, a.keys.Profile):
		a.cycleProfile()
		a.clampCursor(a.visibleRows())
	}
	return a, nil
}

func (a *App) View() string {
	if !a.scanned {
		return a.headerBar() + "\n\n  scanning installed tools...\n"
	}
	if a.mode == modeDoctor {
		return a.headerBar() + "\n\n" + a.renderDoctor()
	}

	body := a.renderTree(a.visibleRows())

	// Wrap the tree in a rounded panel once the terminal width is known.
	panel := body
	if a.width > 4 {
		panel = boxStyle.Width(a.width - 2).Render(body)
	}

	statusLine := ""
	if a.status != "" {
		statusLine = statusStyle.Render(a.status) + "\n"
	}
	return a.headerBar() + "\n\n" + panel + "\n\n" + statusLine + a.footerBar() + "\n"
}

// headerBar renders the top line: brand + tool count + profile/filter on the
// left, and the nav hint or scroll position on the right.
func (a *App) headerBar() string {
	left := titleStyle.Render("CLIOne") +
		toolCountStyle.Render(fmt.Sprintf("  %d tools", a.totalTools()))
	left += brandDimStyle.Render("   ·  profile: " + a.profileName())
	if a.filter != "" || a.filtering {
		cur := ""
		if a.filtering {
			cur = "▏"
		}
		left += brandDimStyle.Render("  ·  /") + a.filter + cur
	}

	right := navHintStyle.Render("↑/↓ navigate · enter act · ? keys")
	rows := a.visibleRows()
	if h := a.treeBodyHeight(); h > 0 && len(rows) > h {
		start := a.scroll
		end := start + h
		if end > len(rows) {
			end = len(rows)
		}
		right = navHintStyle.Render(fmt.Sprintf("[%d–%d/%d]", start+1, end, len(rows)))
	}
	return a.justify(" "+left, right+" ")
}

// footerBar renders the bottom line: key hints on the left, an installed /
// not-installed summary on the right.
func (a *App) footerBar() string {
	hints := []struct{ key, label string }{
		{"↑/↓", "Navigate"},
		{"enter/u", "Update"},
		{"/", "Filter"},
		{"p", "Profile"},
		{"d", "Doctor"},
		{"q", "Quit"},
	}
	var parts []string
	for _, h := range hints {
		parts = append(parts, keyChipStyle.Render(h.key)+" "+keyLabelStyle.Render(h.label))
	}
	left := " " + strings.Join(parts, "  ")

	inst, notInst := a.installCounts()
	right := dotGreen.Render("●") + keyLabelStyle.Render(fmt.Sprintf(" %d installed", inst)) +
		keyLabelStyle.Render("   ") +
		dotRed.Render("●") + keyLabelStyle.Render(fmt.Sprintf(" %d not installed", notInst)) + " "

	line := a.justify(left, right)
	if a.width > 0 {
		line = ansi.Truncate(line, a.width, "…")
	}
	return line
}

// justify places left at the start and right at the end of a width-wide line.
func (a *App) justify(left, right string) string {
	if a.width <= 0 {
		return left + "  " + right
	}
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}

// totalTools counts every catalog tool (installed or not) across all categories.
func (a *App) totalTools() int {
	n := 0
	for _, cs := range a.cats {
		n += cs.Total
	}
	return n
}

// installCounts returns how many catalog tools are installed vs. not.
func (a *App) installCounts() (installed, notInstalled int) {
	for _, cs := range a.cats {
		installed += cs.Installed
		notInstalled += cs.Total - cs.Installed
	}
	return installed, notInstalled
}

// innerWidth is the usable text width inside the rounded panel (border +
// padding removed). Zero means the width is unknown, so no truncation.
func (a *App) innerWidth() int {
	if a.width <= 4 {
		return 0
	}
	return a.width - 4
}

// treeBodyHeight returns how many tree lines fit inside the panel under the
// header/footer chrome. Zero means the terminal size is unknown (no
// WindowSizeMsg yet), in which case the whole tree is rendered without
// windowing.
func (a *App) treeBodyHeight() int {
	if a.height <= 0 {
		return 0
	}
	// header + blank + panel top border + panel bottom border + blank +
	// footer + trailing newline.
	reserved := 7
	if a.status != "" {
		reserved++
	}
	h := a.height - reserved
	if h < 1 {
		h = 1
	}
	return h
}

// ensureVisible scrolls the viewport so the cursor row stays on screen.
func (a *App) ensureVisible(rows []row) {
	h := a.treeBodyHeight()
	if h <= 0 || len(rows) <= h {
		a.scroll = 0
		return
	}
	if a.cursor < a.scroll {
		a.scroll = a.cursor
	}
	if a.cursor >= a.scroll+h {
		a.scroll = a.cursor - h + 1
	}
	if max := len(rows) - h; a.scroll > max {
		a.scroll = max
	}
	if a.scroll < 0 {
		a.scroll = 0
	}
}

// setLatest updates one tool's latest-version result and recomputes its status.
func (a *App) setLatest(toolID string, res model.VersionResult) {
	a.mutateTool(toolID, func(ts *model.ToolState) {
		ts.Latest = res
		ts.Status = scan.ComputeStatus(ts.Def, ts.Detect, ts.Source, ts.Latest)
	})
}

// setDetect updates one tool's detection result (post-update re-detect),
// recomputes its status, and refreshes the category installed count.
func (a *App) setDetect(toolID string, res model.DetectResult) {
	a.mutateTool(toolID, func(ts *model.ToolState) {
		ts.Detect = res
		ts.Status = scan.ComputeStatus(ts.Def, ts.Detect, ts.Source, ts.Latest)
	})
}

// mutateTool applies fn to the tool with the given ID and refreshes its
// category's installed count.
func (a *App) mutateTool(toolID string, fn func(*model.ToolState)) {
	for ci := range a.cats {
		for ti := range a.cats[ci].Tools {
			if a.cats[ci].Tools[ti].Def.ID != toolID {
				continue
			}
			fn(&a.cats[ci].Tools[ti])
			installed := 0
			for _, ts := range a.cats[ci].Tools {
				if ts.Detect.Installed {
					installed++
				}
			}
			a.cats[ci].Installed = installed
			return
		}
	}
}

// findTool returns a copy of the tool state with the given ID.
func (a *App) findTool(toolID string) (model.ToolState, bool) {
	for _, cs := range a.cats {
		for _, ts := range cs.Tools {
			if ts.Def.ID == toolID {
				return ts, true
			}
		}
	}
	return model.ToolState{}, false
}

// detectRun re-runs detection for one tool definition.
func detectRun(def model.ToolDef) (model.DetectResult, error) {
	return detect.RunDetect(context.Background(), def)
}
