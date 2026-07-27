package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/scan"
	"github.com/dierodfer6/cliOne/internal/updater"
)

// Exit codes for Update. Chosen so scripts can tell "nothing to do because
// it's fine" (0) apart from "tried and failed" (1) and "unscriptable" (2/3).
const (
	ExitOK           = 0
	ExitUpdateFailed = 1
	ExitUnknownTool  = 2
	ExitNoUpdater    = 3
)

// Update resolves and, if possible, runs the update for a single catalog
// tool by ID. It never installs anything from scratch: routing is decided by
// the same updater.Decide the TUI's row action uses, so when there is no
// runnable native/manager update command it prints the official page instead
// of attempting one.
func Update(ctx context.Context, s *scan.Scanner, stdout, stderr io.Writer, toolID string) int {
	def, ok := s.Catalog.ToolByID(toolID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "clione: update: unknown tool %q\n", toolID)
		return ExitUnknownTool
	}

	ts := s.ScanOne(ctx, def)
	if ts.Detect.Installed {
		latest := s.FetchLatest(ctx, def, ts.Source)
		ts.Latest = latest.Result
		ts.Status = scan.ComputeStatus(def, ts.Detect, ts.Source, ts.Latest)
	}

	switch action := updater.Decide(def, ts.Source, ts.Status, s.Registry); action {
	case model.ActionRunNativeUpdate, model.ActionRunManagerUpdate:
		res := updater.Execute(ctx, def, ts.Source)
		if res.Stdout != "" {
			_, _ = fmt.Fprint(stdout, res.Stdout)
		}
		if res.Stderr != "" {
			_, _ = fmt.Fprint(stderr, res.Stderr)
		}
		if !res.Success {
			_, _ = fmt.Fprintf(stderr, "clione: update: %s failed (exit %d)\n", toolID, res.ExitCode)
			return ExitUpdateFailed
		}
		_, _ = fmt.Fprintf(stdout, "clione: update: %s updated successfully\n", toolID)
		return ExitOK
	case model.ActionOpenOfficialPage:
		_, _ = fmt.Fprintf(stdout, "clione: update: no runnable update for %s; see %s\n", toolID, def.OfficialURL)
		return ExitNoUpdater
	default: // ActionNone: already up to date
		_, _ = fmt.Fprintf(stdout, "clione: update: %s is already up to date\n", toolID)
		return ExitOK
	}
}
