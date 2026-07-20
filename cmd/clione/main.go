// Command clione is the CLIOne TUI entrypoint.
//
// v0.1 is TUI-only: running `clione` with no arguments opens the TUI.
// There are no subcommands.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dierodfer6/cliOne/internal/tui"
)

var version = "0.1.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("clione", version)
		return
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "clione:", err)
		os.Exit(1)
	}
}
