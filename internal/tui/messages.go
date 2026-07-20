package tui

import (
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/scan"
)

// scanDoneMsg carries the initial full scan result.
type scanDoneMsg struct {
	cats []model.CategoryState
	err  error
}

// versionDoneMsg carries one async latest-version refresh result.
type versionDoneMsg scan.LatestUpdate

// detectDoneMsg carries a re-detection result after a successful update.
type detectDoneMsg struct {
	toolID string
	result model.DetectResult
}

// updateDoneMsg carries the outcome of running an update command on a row.
type updateDoneMsg struct {
	toolID string
	result model.UpdateResult
}

// openURLDoneMsg reports an attempt to open a tool's official page.
type openURLDoneMsg struct {
	toolID string
	err    error
}
