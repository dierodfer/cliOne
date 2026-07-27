package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/dierodfer6/cliOne/internal/scan"
)

// Doctor prints every tool that resolves at more than one location on
// $PATH, in plain text — the headless equivalent of the TUI's doctor view,
// backed by the same scan.Conflicts. Returns 0 unless the scan itself fails.
func Doctor(ctx context.Context, s *scan.Scanner, stdout io.Writer) int {
	cats, err := s.Scan(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "clione: doctor: scan failed:", err)
		return 1
	}

	conflicts := scan.Conflicts(cats)
	if len(conflicts) == 0 {
		_, _ = fmt.Fprintln(stdout, "no PATH conflicts detected: every tool resolves at a single location")
		return 0
	}
	for _, c := range conflicts {
		_, _ = fmt.Fprintln(stdout, c.Tool.Name)
		for i, p := range c.AllPaths {
			if i == 0 {
				_, _ = fmt.Fprintf(stdout, "  * %s  (active - first on $PATH)\n", p)
			} else {
				_, _ = fmt.Fprintf(stdout, "  - %s  (shadowed)\n", p)
			}
		}
	}
	return 0
}
