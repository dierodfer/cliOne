// Package tui implements the CLIOne terminal user interface: a collapsible
// tree of tool categories showing installed vs. latest versions, a doctor
// view, and profile/text filtering. It reports versions only and never
// installs or upgrades anything.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

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
	version  string
	cats     []model.CategoryState
	profiles []model.Profile

	keys keyMap

	mode         viewMode
	cursor       int
	scroll       int             // index of the first tree row rendered (viewport top)
	doctorScroll int             // index of the first doctor line rendered (viewport top)
	expanded     map[string]bool // category ID -> open

	refreshPending int // outstanding versionDoneMsg replies from a manual `r` refresh

	filter     string
	filtering  bool // typing after `/`
	profileIdx int  // -1 = All

	scanned bool
	status  string
	width   int
	height  int
}

// Run starts the TUI and blocks until quit. version is shown in the header.
func Run(version string) error {
	scanner, err := scan.New()
	if err != nil {
		return err
	}
	app := NewApp(scanner, version)
	_, err = tea.NewProgram(app, tea.WithAltScreen()).Run()
	return err
}

// NewApp builds the initial model. Exposed for tests.
func NewApp(scanner *scan.Scanner, version string) *App {
	a := &App{
		scanner:    scanner,
		version:    version,
		keys:       defaultKeyMap(),
		expanded:   map[string]bool{},
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
		a.clampDoctorScroll()
		return a, nil

	case scanDoneMsg:
		return a, a.applyScanResult(msg)

	case versionDoneMsg:
		a.setLatest(msg.ToolID, msg.Result)
		a.noteRefreshDone()
		return a, nil

	case openURLDoneMsg:
		if msg.err != nil {
			a.status = "could not open browser: " + msg.err.Error()
		} else {
			a.status = "opened official page for " + msg.toolID
		}
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

// applyScanResult stores a finished scan and kicks off the async latest-version
// refreshes for it. Cached rows render as-is until those land.
func (a *App) applyScanResult(msg scanDoneMsg) tea.Cmd {
	a.scanned = true
	if msg.err != nil {
		a.status = "scan failed: " + msg.err.Error()
		return nil
	}
	a.cats = msg.cats
	// Categories render expanded by default; only seed IDs the user has not
	// already toggled so a re-scan never reopens a collapsed one.
	for _, cs := range a.cats {
		if _, set := a.expanded[cs.Category.ID]; !set {
			a.expanded[cs.Category.ID] = true
		}
	}
	cmd, _ := a.refreshCmds(false)
	return cmd
}

// refreshCmds turns the scanner's pending latest-version lookups into a single
// batched command, and reports how many lookups it covers. force ignores the
// cache TTL.
func (a *App) refreshCmds(force bool) (tea.Cmd, int) {
	funcs := a.scanner.RefreshFuncs(context.Background(), a.cats, force)
	cmds := make([]tea.Cmd, 0, len(funcs))
	for _, f := range funcs {
		f := f
		cmds = append(cmds, func() tea.Msg { return versionDoneMsg(f()) })
	}
	return tea.Batch(cmds...), len(funcs)
}

// noteRefreshDone counts down a manual `r` refresh and reports when the last
// reply lands. Replies from the automatic post-scan refresh are not counted.
func (a *App) noteRefreshDone() {
	if a.refreshPending == 0 {
		return
	}
	a.refreshPending--
	if a.refreshPending == 0 {
		a.status = "latest versions refreshed"
	}
}

// handleKey routes a keypress to the handler for the current input context:
// filter-typing captures almost everything, the doctor view is read-only, and
// otherwise the tree handles it.
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case a.filtering:
		a.handleFilterKey(msg)
		return a, nil
	case a.mode == modeDoctor:
		return a.handleDoctorKey(msg)
	default:
		return a.handleTreeKey(msg)
	}
}

// handleFilterKey edits the text filter while the user types after `/`.
func (a *App) handleFilterKey(msg tea.KeyMsg) {
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
}

// handleDoctorKey scrolls the doctor view or leaves it; the view never acts on
// rows.
func (a *App) handleDoctorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Doctor), key.Matches(msg, a.keys.Back), key.Matches(msg, a.keys.Quit):
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		a.mode = modeTree
	case key.Matches(msg, a.keys.Up):
		a.doctorScroll--
		a.clampDoctorScroll()
	case key.Matches(msg, a.keys.Down):
		a.doctorScroll++
		a.clampDoctorScroll()
	}
	return a, nil
}

// handleTreeKey handles navigation and row actions in the main tree view.
func (a *App) handleTreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case key.Matches(msg, a.keys.Enter):
		return a, a.activateRow(rows)

	case key.Matches(msg, a.keys.Back):
		a.goBack(rows)

	case key.Matches(msg, a.keys.Filter):
		a.filtering = true

	case key.Matches(msg, a.keys.Doctor):
		a.mode = modeDoctor
		a.doctorScroll = 0

	case key.Matches(msg, a.keys.Profile):
		a.cycleProfile()
		a.clampCursor(a.visibleRows())

	case key.Matches(msg, a.keys.Refresh):
		return a, a.startFullRefresh()
	}
	return a, nil
}

// rowAtCursor returns the row under the cursor, if the cursor is on one.
func (a *App) rowAtCursor(rows []row) (row, bool) {
	if a.cursor < 0 || a.cursor >= len(rows) {
		return row{}, false
	}
	return rows[a.cursor], true
}

// activateRow handles enter: category rows toggle open, tool rows open their
// official page.
func (a *App) activateRow(rows []row) tea.Cmd {
	r, ok := a.rowAtCursor(rows)
	if !ok {
		return nil
	}
	if !r.isCategory {
		return a.actOnTool(r.tool)
	}
	a.toggleExpand(r.catID)
	a.clampCursor(a.visibleRows())
	return nil
}

// goBack handles esc: a committed filter forces categories open, so backing
// out of the filter takes precedence over collapsing the category under the
// cursor.
func (a *App) goBack(rows []row) {
	if a.filter != "" {
		a.filter = ""
	} else if r, ok := a.rowAtCursor(rows); ok && r.isCategory && a.expanded[r.catID] {
		a.toggleExpand(r.catID)
	}
	a.clampCursor(a.visibleRows())
}

// startFullRefresh forces a live latest-version re-check for every installed
// tool, ignoring the cache's TTL. Ignored if a previous refresh is still in
// flight, mirroring the row-level "already updating" guard in actOnTool.
func (a *App) startFullRefresh() tea.Cmd {
	if a.refreshPending > 0 {
		return nil
	}
	cmd, n := a.refreshCmds(true)
	if n == 0 {
		a.status = "nothing to refresh"
		return nil
	}
	a.refreshPending = n
	a.status = fmt.Sprintf("refreshing %d tool(s)...", n)
	return cmd
}

func (a *App) View() string {
	if !a.scanned {
		return a.headerBar() + "\n\n  scanning installed tools...\n"
	}

	var body string
	if a.mode == modeDoctor {
		body = a.renderDoctor()
	} else {
		body = a.renderTree(a.visibleRows())
	}

	// Wrap the body in a rounded panel once the terminal width is known.
	panel := body
	if a.width > 4 {
		panel = boxStyle.Width(a.width - 2).Render(body)
	}

	statusLine := ""
	if a.status != "" {
		statusLine = statusStyle.Render(a.status) + "\n"
	}
	legendLine := ""
	if legend := a.legendBar(); legend != "" {
		legendLine = legend + "\n"
	}
	return a.headerBar() + "\n\n" + panel + "\n\n" + statusLine + legendLine + a.footerBar() + "\n"
}

// legendBar decodes the per-manager colors used by the [source] tag on each
// tool row. The doctor view has no source tags, so it gets no legend.
func (a *App) legendBar() string {
	if a.mode == modeDoctor {
		return ""
	}
	var parts []string
	for _, k := range legendKinds {
		parts = append(parts, sourceKindStyle(k).Render("●")+" "+keyLabelStyle.Render(k.String()))
	}
	line := " " + navHintStyle.Render("source:") + "  " + strings.Join(parts, "  ")
	if a.width > 0 {
		line = ansi.Truncate(line, a.width, "…")
	}
	return line
}

// headerBar renders the top line: brand + tool count + profile/filter on the
// left, and the nav hint or scroll position on the right.
func (a *App) headerBar() string {
	return a.justify(" "+a.headerLeft(), a.headerRight()+" ")
}

// headerLeft renders brand, tool count, active profile, and the filter being
// typed.
func (a *App) headerLeft() string {
	title := "CLIOne"
	if a.version != "" {
		title += " " + a.version
	}
	left := titleStyle.Render(title) +
		toolCountStyle.Render(fmt.Sprintf("  %d tools", a.totalTools())) +
		brandDimStyle.Render("   ·  profile: "+a.profileName())
	if a.filter != "" || a.filtering {
		cur := ""
		if a.filtering {
			cur = "▏"
		}
		left += brandDimStyle.Render("  ·  /") + a.filter + cur
	}
	return left
}

// headerRight renders the scroll position when the current view is windowed,
// and the nav hint otherwise.
func (a *App) headerRight() string {
	h := a.treeBodyHeight()
	if a.mode == modeDoctor {
		if pos, ok := scrollIndicator(a.doctorScroll, h, len(a.doctorLines())); ok {
			return navHintStyle.Render(pos)
		}
		return navHintStyle.Render("doctor view")
	}
	if pos, ok := scrollIndicator(a.scroll, h, len(a.visibleRows())); ok {
		return navHintStyle.Render(pos)
	}
	return navHintStyle.Render("↑/↓ navigate · enter act · ? keys")
}

// scrollIndicator formats a "[start–end/total]" position for a viewport of
// height h. ok is false when the content fits, or the height is unknown.
func scrollIndicator(scroll, h, total int) (string, bool) {
	if h <= 0 || total <= h {
		return "", false
	}
	end := scroll + h
	if end > total {
		end = total
	}
	return fmt.Sprintf("[%d–%d/%d]", scroll+1, end, total), true
}

// footerBar renders the bottom line: key hints on the left, an installed /
// not-installed summary on the right. Hints are mode-specific: the doctor
// view is read-only and scrolls instead of acting on rows.
func (a *App) footerBar() string {
	hints := []struct{ key, label string }{
		{"↑/↓", "Navigate"},
		{"enter", "Open page"},
		{"/", "Filter"},
		{"p", "Profile"},
		{"d", "Doctor"},
		{"r", "Refresh"},
		{"q", "Quit"},
	}
	if a.mode == modeDoctor {
		hints = []struct{ key, label string }{
			{"↑/↓", "Scroll"},
			{"esc/d", "Back"},
			{"q", "Quit"},
		}
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
	if a.mode != modeDoctor {
		reserved++ // source-color legend, rendered above the footer
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

