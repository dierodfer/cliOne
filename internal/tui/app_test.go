package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dierodfer6/cliOne/internal/cache"
	"github.com/dierodfer6/cliOne/internal/catalog"
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
	"github.com/dierodfer6/cliOne/internal/scan"
	"github.com/dierodfer6/cliOne/internal/source"
)

func testApp(t *testing.T) *App {
	t.Helper()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
	}
	return NewApp(s, "test")
}

func syntheticCats() []model.CategoryState {
	tools := []model.ToolState{
		{
			Def:    model.ToolDef{ID: "git", Name: "Git", Category: "git", OfficialURL: "https://git-scm.com"},
			Status: model.StatusLatestUnknown,
			Detect: model.DetectResult{Installed: true, Version: "2.43.0"},
			Source: model.SourceResult{Kind: model.SourceAptDnf, BinPath: "/usr/bin/git", AllPaths: []string{"/usr/bin/git", "/usr/local/bin/git"}},
		},
		{
			Def:    model.ToolDef{ID: "lazygit", Name: "Lazygit", Category: "git", OfficialURL: "https://github.com/jesseduffield/lazygit"},
			Status: model.StatusNotInstalled,
		},
	}
	util := []model.ToolState{
		{
			Def:    model.ToolDef{ID: "ripgrep", Name: "ripgrep", Category: "utilities", OfficialURL: "https://github.com/BurntSushi/ripgrep"},
			Status: model.StatusUpToDate,
			Detect: model.DetectResult{Installed: true, Version: "14.1.0"},
			Source: model.SourceResult{Kind: model.SourceCargo, BinPath: "/h/.cargo/bin/rg", AllPaths: []string{"/h/.cargo/bin/rg"}},
		},
	}
	return []model.CategoryState{
		{Category: model.Category{ID: "git", Name: "Git"}, Tools: tools, Installed: 1, Total: 2},
		{Category: model.Category{ID: "utilities", Name: "Utilities"}, Tools: util, Installed: 1, Total: 1},
	}
}

func loadSynthetic(t *testing.T, a *App) {
	t.Helper()
	m, _ := a.Update(scanDoneMsg{cats: syntheticCats()})
	if m.(*App) != a {
		t.Fatal("Update should return the same app")
	}
}

func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
}

func TestCategoriesExpandedByDefaultWithCounts(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	out := a.View()
	if !strings.Contains(out, "Git") || !strings.Contains(out, "1/2") {
		t.Fatalf("expected category header with counts, got:\n%s", out)
	}
	// Categories are open by default, so child tools are visible immediately.
	if !strings.Contains(out, "Lazygit") {
		t.Fatalf("tools should be visible with categories expanded by default, got:\n%s", out)
	}
}

func TestEnterCollapsesThenExpands(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	// Child visible initially (expanded by default).
	if !strings.Contains(a.View(), "Lazygit") {
		t.Fatalf("expected Lazygit visible by default, got:\n%s", a.View())
	}
	// enter on the git category (cursor starts there) collapses it.
	a.Update(keyMsg("enter"))
	if strings.Contains(a.View(), "Lazygit") {
		t.Fatalf("expected collapse on enter, got:\n%s", a.View())
	}
	// enter again re-expands.
	a.Update(keyMsg("enter"))
	out := a.View()
	if !strings.Contains(out, "Lazygit") || !strings.Contains(out, "not installed") {
		t.Fatalf("expected re-expanded rows with a not-installed row, got:\n%s", out)
	}
}

func TestViewportScrolling(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	// Small terminal: header/footer chrome (4) + 3 body rows.
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 7})
	rows := a.visibleRows()
	if len(rows) <= a.treeBodyHeight() {
		t.Fatalf("test needs more rows (%d) than the body height (%d)", len(rows), a.treeBodyHeight())
	}
	lastTool := rows[len(rows)-1].tool.Def.Name

	// Drive the cursor to the bottom; the viewport must scroll and the last
	// row must be rendered.
	for i := 0; i < len(rows); i++ {
		a.Update(keyMsg("down"))
	}
	if a.scroll == 0 {
		t.Fatalf("viewport should have scrolled down, scroll=%d", a.scroll)
	}
	if !strings.Contains(a.View(), lastTool) {
		t.Fatalf("cursor row %q should be visible after scrolling down, got:\n%s", lastTool, a.View())
	}

	// Drive back to the top; the viewport must return and the first row render.
	firstCat := rows[0].cat.Category.Name
	for i := 0; i < len(rows); i++ {
		a.Update(keyMsg("up"))
	}
	if a.scroll != 0 {
		t.Fatalf("viewport should return to top, scroll=%d", a.scroll)
	}
	if !strings.Contains(a.View(), firstCat) {
		t.Fatalf("first row %q should be visible after scrolling up, got:\n%s", firstCat, a.View())
	}
}

func TestRefreshKeyForcesFullRefresh(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)

	_, cmd := a.Update(keyMsg("r"))
	if cmd == nil {
		t.Fatal("expected a refresh command")
	}
	// Synthetic data has two installed tools: git and ripgrep.
	if a.refreshPending != 2 {
		t.Fatalf("expected 2 pending refreshes, got %d", a.refreshPending)
	}
	if !strings.Contains(a.status, "refreshing 2") {
		t.Fatalf("expected a refreshing status, got %q", a.status)
	}

	// Simulate the results streaming back one at a time.
	a.Update(versionDoneMsg{ToolID: "git", Result: model.VersionResult{Latest: "2.44.0"}})
	if a.refreshPending != 1 {
		t.Fatalf("expected 1 pending after first result, got %d", a.refreshPending)
	}
	a.Update(versionDoneMsg{ToolID: "ripgrep", Result: model.VersionResult{Latest: "14.1.0"}})
	if a.refreshPending != 0 {
		t.Fatalf("expected 0 pending after both results, got %d", a.refreshPending)
	}
	if a.status != "latest versions refreshed" {
		t.Fatalf("expected completion status, got %q", a.status)
	}

	// A refresh already in flight ignores a second `r` press.
	a.refreshPending = 1
	_, cmd2 := a.Update(keyMsg("r"))
	if cmd2 != nil {
		t.Fatal("expected refresh to be a no-op while one is already in flight")
	}
}

func TestDoctorViewportScrolling(t *testing.T) {
	a := testApp(t)
	var tools []model.ToolState
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("tool%02d", i)
		tools = append(tools, model.ToolState{
			Def:    model.ToolDef{ID: id, Name: id, OfficialURL: "https://example.com"},
			Status: model.StatusLatestUnknown,
			Detect: model.DetectResult{Installed: true, Version: "1.0.0"},
			Source: model.SourceResult{Kind: model.SourceManual, AllPaths: []string{"/a/" + id, "/b/" + id}},
		})
	}
	a.Update(scanDoneMsg{cats: []model.CategoryState{
		{Category: model.Category{ID: "utilities", Name: "Utilities"}, Tools: tools, Installed: 20, Total: 20},
	}})
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	a.Update(keyMsg("d"))
	if a.mode != modeDoctor {
		t.Fatal("expected doctor mode")
	}

	lines := a.doctorLines()
	h := a.treeBodyHeight()
	if len(lines) <= h {
		t.Fatalf("test needs more doctor lines (%d) than the body height (%d)", len(lines), h)
	}
	lastToolID := tools[len(tools)-1].Def.ID
	if strings.Contains(a.View(), lastToolID) {
		t.Fatalf("last tool %q should not be visible before scrolling, got:\n%s", lastToolID, a.View())
	}

	// Scroll down past the window: the viewport must move and the last
	// conflict entry must become visible.
	for i := 0; i < len(lines); i++ {
		a.Update(keyMsg("down"))
	}
	if a.doctorScroll == 0 {
		t.Fatal("expected the doctor viewport to have scrolled down")
	}
	if !strings.Contains(a.View(), lastToolID) {
		t.Fatalf("expected last tool %q visible after scrolling down, got:\n%s", lastToolID, a.View())
	}

	// Scroll back up to the top: this is the bug being fixed — earlier
	// entries must become visible again.
	for i := 0; i < len(lines); i++ {
		a.Update(keyMsg("up"))
	}
	if a.doctorScroll != 0 {
		t.Fatalf("expected the doctor viewport to return to the top, got scroll=%d", a.doctorScroll)
	}
	if !strings.Contains(a.View(), "Doctor — PATH conflicts") {
		t.Fatalf("expected the doctor heading visible after scrolling back up, got:\n%s", a.View())
	}
}

func TestNavigationAndClamping(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.Update(keyMsg("up")) // clamp at 0
	if a.cursor != 0 {
		t.Fatalf("cursor should clamp at 0, got %d", a.cursor)
	}
	for i := 0; i < 20; i++ {
		a.Update(keyMsg("down"))
	}
	rows := a.visibleRows()
	if a.cursor != len(rows)-1 {
		t.Fatalf("cursor should clamp at %d, got %d", len(rows)-1, a.cursor)
	}
}

func TestVersionDoneFlipsStatusToUpdateAvail(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.Update(versionDoneMsg{ToolID: "ripgrep", Result: model.VersionResult{Latest: "15.0.0"}})
	ts, ok := a.findTool("ripgrep")
	if !ok {
		t.Fatal("ripgrep not found")
	}
	if ts.Status != model.StatusUpdateAvail {
		t.Fatalf("expected update-available after newer latest, got %v", ts.Status)
	}
	if ts.Latest.Latest != "15.0.0" {
		t.Fatalf("latest not stored: %+v", ts.Latest)
	}
}

func TestVersionDoneSameVersionStaysGreen(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.Update(versionDoneMsg{ToolID: "ripgrep", Result: model.VersionResult{Latest: "14.1.0"}})
	ts, _ := a.findTool("ripgrep")
	if ts.Status != model.StatusUpToDate {
		t.Fatalf("expected up-to-date, got %v", ts.Status)
	}
}

func TestUpdateFailureShowsErrorAndPanelToggles(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	res := model.UpdateResult{
		Action:   model.ActionRunManagerUpdate,
		Success:  false,
		ExitCode: 1,
		Stderr:   "line1\nline2\npermission denied",
		Err:      errors.New("exit status 1"),
	}
	a.Update(updateDoneMsg{toolID: "ripgrep", result: res})
	if _, ok := a.updateErrs["ripgrep"]; !ok {
		t.Fatal("expected error recorded for ripgrep")
	}

	// Navigate: expand utilities, move onto the ripgrep row, press l.
	a.expanded["utilities"] = true
	rows := a.visibleRows()
	for i, r := range rows {
		if r.toolID == "ripgrep" {
			a.cursor = i
		}
	}
	a.Update(keyMsg("l"))
	out := a.View()
	if !strings.Contains(out, "permission denied") {
		t.Fatalf("expected stderr tail in error panel, got:\n%s", out)
	}
	if !strings.Contains(out, "✗ update failed") {
		t.Fatalf("expected inline failure marker, got:\n%s", out)
	}
	// Toggle closed.
	a.Update(keyMsg("l"))
	if strings.Contains(a.View(), "permission denied") {
		t.Fatal("panel should close on second l")
	}
}

func TestUpdateSuccessTriggersRedetect(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.updating["ripgrep"] = true
	_, cmd := a.Update(updateDoneMsg{toolID: "ripgrep", result: model.UpdateResult{Action: model.ActionRunManagerUpdate, Success: true}})
	if a.updating["ripgrep"] {
		t.Fatal("spinner flag should clear on completion")
	}
	if cmd == nil {
		t.Fatal("expected a re-detect command after a successful update")
	}
	// Simulate the re-detect result coming back with the new version.
	a.Update(versionDoneMsg{ToolID: "ripgrep", Result: model.VersionResult{Latest: "15.0.0"}})
	a.Update(detectDoneMsg{toolID: "ripgrep", result: model.DetectResult{Installed: true, Version: "15.0.0"}})
	ts, _ := a.findTool("ripgrep")
	if ts.Status != model.StatusUpToDate {
		t.Fatalf("expected green after re-detect matches latest, got %v", ts.Status)
	}
}

func TestYellowWithoutUpdaterOpensPageNotError(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	// Outdated (yellow) but manual source with no bespoke updater: acting on it
	// must open the official page, not attempt (and fail) an update.
	ts := model.ToolState{
		Def:    model.ToolDef{ID: "manualtool", Name: "ManualTool", OfficialURL: "https://example.com"},
		Status: model.StatusUpdateAvail,
		Source: model.SourceResult{Kind: model.SourceManual},
		Detect: model.DetectResult{Installed: true, Version: "1.0.0"},
		Latest: model.VersionResult{Latest: "2.0.0"},
	}
	cmd := a.actOnTool(ts)
	if a.updating["manualtool"] {
		t.Fatal("a yellow row with no updater must not start an update")
	}
	if cmd == nil {
		t.Fatal("expected an open-page command")
	}
	if _, ok := cmd().(openURLDoneMsg); !ok {
		t.Fatal("expected openURLDoneMsg (open official page) for a yellow row without an updater")
	}
}

func TestDoctorViewListsConflicts(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.Update(keyMsg("d"))
	out := a.View()
	if !strings.Contains(out, "Doctor") {
		t.Fatalf("expected doctor view, got:\n%s", out)
	}
	if !strings.Contains(out, "/usr/bin/git") || !strings.Contains(out, "active — first on $PATH") {
		t.Fatalf("expected git PATH conflict with active marker, got:\n%s", out)
	}
	if !strings.Contains(out, "/usr/local/bin/git") || !strings.Contains(out, "shadowed") {
		t.Fatalf("expected shadowed path listed, got:\n%s", out)
	}
	// Single-location ripgrep must not appear.
	if strings.Contains(out, ".cargo/bin/rg") {
		t.Fatalf("single-path tool should not appear in doctor, got:\n%s", out)
	}
	a.Update(keyMsg("esc"))
	if a.mode != modeTree {
		t.Fatal("esc should leave doctor view")
	}
}

func TestProfileCyclingFiltersCategories(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	if a.profileName() != "All" {
		t.Fatalf("default profile should be All, got %s", a.profileName())
	}
	// Cycle to backend (languages, package_managers): neither synthetic
	// category is included, so the tree empties.
	a.Update(keyMsg("p"))
	if a.profileName() != "Backend" {
		t.Fatalf("expected Backend after first p, got %s", a.profileName())
	}
	if len(a.filteredCats()) != 0 {
		t.Fatalf("backend profile should hide git/utilities, got %v", a.filteredCats())
	}
	// Cycle to devops (kubernetes, git, package_managers): git remains.
	a.Update(keyMsg("p"))
	a.Update(keyMsg("p"))
	if a.profileName() != "DevOps" {
		t.Fatalf("expected DevOps, got %s", a.profileName())
	}
	cats := a.filteredCats()
	if len(cats) != 1 || cats[0].Category.ID != "git" {
		t.Fatalf("devops profile should keep only git, got %v", cats)
	}
	// Cycle through the rest back to All.
	for i := 0; i < 3; i++ {
		a.Update(keyMsg("p"))
	}
	if a.profileName() != "All" {
		t.Fatalf("expected wrap back to All, got %s", a.profileName())
	}
}

func TestTextFilterFuzzyAndCombinesWithProfile(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	a.Update(keyMsg("/"))
	if !a.filtering {
		t.Fatal("/ should enter filter mode")
	}
	for _, r := range "lzgt" {
		a.Update(keyMsg(string(r)))
	}
	out := a.View()
	if !strings.Contains(out, "Lazygit") {
		t.Fatalf("fuzzy filter lzgt should match Lazygit, got:\n%s", out)
	}
	if strings.Contains(out, "ripgrep") {
		t.Fatalf("non-matching tools should be hidden, got:\n%s", out)
	}
	a.Update(keyMsg("enter")) // commit filter

	// Combine with profile: DevOps keeps git category, filter still applies.
	a.Update(keyMsg("p")) // Backend
	if len(a.filteredCats()) != 0 {
		t.Fatal("backend + lzgt should be empty")
	}
	a.Update(keyMsg("p"))
	a.Update(keyMsg("p")) // DevOps
	cats := a.filteredCats()
	if len(cats) != 1 || cats[0].Tools[0].Def.ID != "lazygit" {
		t.Fatalf("devops + lzgt should keep only lazygit, got %v", cats)
	}

	// Esc clears the filter.
	a.Update(keyMsg("esc"))
	if a.filterText() != "" {
		t.Fatalf("esc should clear filter, got %q", a.filterText())
	}
}

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		p, s string
		want bool
	}{
		{"", "anything", true},
		{"rg", "ripgrep", true},
		{"RG", "ripgrep", true},
		{"gpx", "ripgrep", false},
		{"lazygit", "lazy", false},
	}
	for _, tc := range cases {
		if got := fuzzyMatch(tc.p, tc.s); got != tc.want {
			t.Fatalf("fuzzyMatch(%q,%q)=%v, want %v", tc.p, tc.s, got, tc.want)
		}
	}
}

func TestQuit(t *testing.T) {
	a := testApp(t)
	loadSynthetic(t, a)
	_, cmd := a.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should produce a quit command")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Fatalf("expected tea.Quit, got %#v", msg)
	}
}
