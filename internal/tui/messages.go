package tui

import (
	"github.com/dierodfer/cliOne/internal/model"
	"github.com/dierodfer/cliOne/internal/scan"
)

// scanDoneMsg carries the initial full scan result.
type scanDoneMsg struct {
	cats []model.CategoryState
	err  error
}

// versionDoneMsg carries one async latest-version refresh result.
type versionDoneMsg scan.LatestUpdate

// openURLDoneMsg reports an attempt to open a tool's official page.
type openURLDoneMsg struct {
	toolID string
	err    error
}
