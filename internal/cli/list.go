// Package cli implements clione's non-interactive subcommands (list, doctor,
// update). They exist alongside the TUI, not instead of it: each one is a
// thin, testable wrapper over the same internal/scan and internal/updater
// logic the TUI already uses, so behavior never diverges between the two.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/scan"
)

// List prints every catalog tool grouped by category (installed or not), one
// line per tool: id, detected version, source, latest (if known), and
// status. When refresh is set, latest versions are re-checked live first,
// ignoring the cache's TTL — the same force-refresh `RefreshFuncs` does for
// the TUI's `r` key. Returns 0 unless the scan itself fails to run.
func List(ctx context.Context, s *scan.Scanner, stdout io.Writer, refresh bool) int {
	cats, err := s.Scan(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "clione: list: scan failed:", err)
		return 1
	}

	if refresh {
		for _, f := range s.RefreshFuncs(ctx, cats, true) {
			u := f()
			applyLatest(cats, u.ToolID, u.Result)
		}
	}

	for _, cs := range cats {
		_, _ = fmt.Fprintf(stdout, "%s (%d/%d installed)\n", cs.Category.Name, cs.Installed, cs.Total)
		for _, ts := range cs.Tools {
			_, _ = fmt.Fprintln(stdout, "  "+formatToolLine(ts))
		}
	}
	return 0
}

func formatToolLine(ts model.ToolState) string {
	if !ts.Detect.Installed {
		return fmt.Sprintf("%-20s not installed  %s", ts.Def.ID, ts.Def.OfficialURL)
	}
	latest := "?"
	if ts.Latest.Latest != "" {
		latest = ts.Latest.Latest
	}
	return fmt.Sprintf("%-20s installed=%-12s latest=%-12s source=%-10s status=%s",
		ts.Def.ID, ts.Detect.Version, latest, ts.Source.Kind, ts.Status)
}

// applyLatest folds a live latest-version result back into cats in place and
// recomputes that tool's status, mirroring the TUI's setLatest.
func applyLatest(cats []model.CategoryState, toolID string, res model.VersionResult) {
	for ci := range cats {
		for ti := range cats[ci].Tools {
			if cats[ci].Tools[ti].Def.ID != toolID {
				continue
			}
			ts := &cats[ci].Tools[ti]
			ts.Latest = res
			ts.Status = scan.ComputeStatus(ts.Def, ts.Detect, ts.Source, ts.Latest)
			return
		}
	}
}
