// Command clione is the CLIOne TUI entrypoint. Running it with no arguments
// opens the TUI, which remains the primary, full-featured surface. It also
// exposes a few non-interactive subcommands (doctor/list) for scripting and
// CI — see internal/cli — each a thin wrapper over the same engine the TUI
// uses, so behavior never diverges between the two.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dierodfer6/cliOne/internal/cli"
	"github.com/dierodfer6/cliOne/internal/scan"
	"github.com/dierodfer6/cliOne/internal/tui"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main's testable core: it never calls os.Exit itself, only returns
// the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "doctor":
			return dispatchDoctor(args[1:], stdout, stderr)
		case "list":
			return dispatchList(args[1:], stdout, stderr)
		default:
			_, _ = fmt.Fprintf(stderr, "clione: unknown command %q (want doctor or list)\n", args[0])
			return 2
		}
	}

	fs := flag.NewFlagSet("clione", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, "clione", version)
		return 0
	}
	if err := tui.Run(version); err != nil {
		_, _ = fmt.Fprintln(stderr, "clione:", err)
		return 1
	}
	return 0
}

func dispatchList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	refresh := fs.Bool("refresh", false, "force a live latest-version check for every installed tool, ignoring the cache")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, err := scan.New()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "clione: list:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	return cli.List(ctx, s, stdout, *refresh)
}

func dispatchDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, err := scan.New()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "clione: doctor:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return cli.Doctor(ctx, s, stdout)
}
